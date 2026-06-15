made by claude
# 儲存層與資料表

## 主要資料表

### bus
- `raw_bus_route`
  - 原始 API jsonb 快照
- `bus_subroutes`
  - 子路線與站點、班表整理後資料（含 `Direction.first_bus_time`/`last_bus_time`/`holiday_*`）
- `bus_static`
  - 序列化後的 protobuf bytes
- `bus_stations`
  - 公車站點資訊與座標
- `bus_station_stop_map`
  - 站點與路線對應（每次 busStatic 前先刪除舊資料再重新插入，避免殘留已刪路線的停靠點）
- `bus_schedule`
  - 班表與發車頻率；UPSERT ON CONFLICT `bus_schedule_natural_key (sub_route_uid, direction, type, service_day, tripid, "stop_uid/MinHeadwayMins")`；`updated_at timestamptz`；busStatic 完成後刪除 `updated_at < syncStart` 的舊資料
- `bus_fares`
  - 公車票價（sub_route_uid PK, fare_pricing_type, is_free_bus, section_fares jsonb, updated_at）

### bike
- `bike_stations`
  - 站點、容量、地址、座標

### mrt
- `mrt_station`
  - 捷運站點與座標
- `mrt_schedule`
  - 捷運首末班車時間；UPSERT ON CONFLICT `mrt_firstlast_natural_key (station_id, lineid, destinationstaionid, serviceday, system)`；`updated_at timestamptz`；mrtStatic 完成後刪除 `updated_at < syncStart` 的舊資料；serviceday bitmask 第 8 bit = NationalHolidays

### rail
- `tra_stations`
- `thsr_stations`
- `tra_fares`
- `thsr_fares`
- `tra_timetable`
- `thsr_timetable`

### bus ETA prediction
- `bus_eta_history`
  - 每 30 秒 `BusEta` 寫入，僅限 `stop_status == 0`（有 GPS 資料、正在行駛）
  - 欄位：sub_route_uid, stop_uid, direction, stop_sequence, total_stops, estimate, next_bus_time, src_update_time, city, hour, day_of_week, is_holiday, temperature, precipitation, wind_speed, humidity, plate_numb, bus_speed, bus_distance_m
  - 索引：`(sub_route_uid, stop_uid, direction, recorded_at DESC)`
  - 保留 30 天，每日 04:30 清理
- `bus_travel_avg`
  - 主鍵：`(sub_route_uid, direction, stop_uid, hour, day_of_week)`
  - `avg_seconds`：各路線、方向、站點、時段的旅行時間中位數
  - `sample_count = 0`：GTFS 冷啟動種子（`scripts/gtfs_seed.py`），會被觀測資料覆蓋
  - `sample_count > 0`：每日 04:00 `computeTravelAvg` 從 `bus_eta_history` 計算後 upsert

### vector
- `search_vector`
  - 欄位：`type`, `uid`, `name`, `city`, `depart`, `destin`, `geom`, `embedding vector(1024)`, `updated_at`
  - 唯一鍵：`(type, uid, city)`
  - `embedding` 欄位使用 pgvector，HNSW 索引（cosine），取代舊的 `blob bytea`
  - type 值：`bus_route`, `bus_station`, `bike_station`, `mrt_station`, `tra_station`, `thsr_station`, `tra_train`, `thsr_train`

## Redis key 格式

| Key | TTL | 說明 |
|-----|-----|------|
| `weather:{city}` | 15m | 天氣快照（JSON：temperature, precipitation, wind_speed, humidity）；`weatherSync` 寫入 |
| `bus_eta_station:{city}:{stationName}` | 180s | 公車站牌 ETA（protobuf Bus_StationArrival） |
| `bus_eta_route:{subRouteUID}` | 180s | 公車路線 ETA（protobuf Bus_RouteArrival）；有 Publish |
| `bus_fare:{subRouteUID}` | 24h | 公車票價快取（protobuf Bus_Fare） |
| `bike_availability:{stationUID}` | 2m | 自行車即時狀態（protobuf BikeEta） |
| `mrt_live:{system}:{stationId}:{lineId}` | 2m | 捷運即時班次，各線獨立存 |
| `mrt_live:{system}:{stationId}` | — | Pub/Sub channel，所有線更新都推到此頻道 |
| `tra:liveboard:{stationId}` | 3m | 台鐵即時動態（protobuf Tra_LiveBoards） |
| `tra:delay_all` | 3m | 全線誤點（protobuf TraDelays） |
| `tra:delay` | 3m | 各列車誤點（Redis Hash, trainNo → delayMins） |
| `thsr_seats:{date}:{trainNo}` | 15m | 高鐵即時座位（protobuf ThsrAvailableSeats）；有 Publish |
| `lock:bus_eta` | 28s | BusEta 分散式鎖（防止並行重複執行） |
| `lock:mrt_eta` | 9s | mrtEta 分散式鎖 |

## 索引

| 表 | 索引 | 欄位 |
|---|---|---|
| `tra_timetable` | `idx_tra_timetable_station` | `(stationid, train_date, arrivaltime)` |
| `thsr_timetable` | `idx_thsr_timetable_station` | `(stationid, train_date, arrivaltime)` |
| `bus_station_stop_map` | `idx_bssm_station_name` | `(station_name)` |
| `bus_eta_history` | `idx_eta_history_recorded` | `(recorded_at)` |
| `search_vector` | `idx_search_vector_name_trgm` | `USING gin (name gin_trgm_ops)`，需 `pg_trgm` |

## 常見讀寫行為
- `bus_static` 以 `sub_route_uid` 查詢 protobuf bytes
- `bus_fares` 以 `sub_route_uid` 查詢票價（先查 Redis 快取，再查 DB）
- `bus_stations` 以地理距離排序查詢近站
- `tra_timetable` 與 `thsr_timetable` 依 `train_date` 與 `station_id` 查詢
- `tra_fares` 與 `thsr_fares` 依起迄站查詢
- 高鐵即時座位：Router SCAN `thsr_seats:{date}:*` 取初始值，再 PSubscribe 接收更新
- 捷運 ETA：Router SCAN `mrt_live:{system}:{stationId}:*` 取初始值，再 Subscribe 接收更新
