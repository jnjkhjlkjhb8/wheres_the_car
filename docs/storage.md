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
  - 班表與發車頻率（serial PK；每次 busStatic 前先 DELETE 同城市資料再插入，避免重複累積）
- `bus_fares`
  - 公車票價（sub_route_uid PK, fare_pricing_type, is_free_bus, section_fares jsonb, updated_at）

### bike
- `bike_stations`
  - 站點、容量、地址、座標

### mrt
- `mrt_station`
  - 捷運站點與座標
- `mrt_firstlast`
  - 首末班車時間（serial PK；每次 mrtStatic 前先 DELETE WHERE system=? 再插入；serviceday bitmask 第 8 bit = NationalHolidays）

### rail
- `tra_stations`
- `thsr_stations`
- `tra_fares`
- `thsr_fares`
- `tra_timetable`
- `thsr_timetable`

### vector
- `search_vector`
  - 欄位：`type`, `uid`, `name`, `city`, `depart`, `destin`, `geom`, `embedding vector(1024)`, `updated_at`
  - 唯一鍵：`(type, uid, city)`
  - `embedding` 欄位使用 pgvector，HNSW 索引（cosine），取代舊的 `blob bytea`
  - type 值：`bus_route`, `bus_station`, `bike_station`, `mrt_station`, `tra_station`, `thsr_station`, `tra_train`, `thsr_train`

## Redis key 格式

| Key | TTL | 說明 |
|-----|-----|------|
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

## 常見讀寫行為
- `bus_static` 以 `sub_route_uid` 查詢 protobuf bytes
- `bus_fares` 以 `sub_route_uid` 查詢票價（先查 Redis 快取，再查 DB）
- `bus_stations` 以地理距離排序查詢近站
- `tra_timetable` 與 `thsr_timetable` 依 `train_date` 與 `station_id` 查詢
- `tra_fares` 與 `thsr_fares` 依起迄站查詢
- 高鐵即時座位：Router SCAN `thsr_seats:{date}:*` 取初始值，再 PSubscribe 接收更新
- 捷運 ETA：Router SCAN `mrt_live:{system}:{stationId}:*` 取初始值，再 Subscribe 接收更新
