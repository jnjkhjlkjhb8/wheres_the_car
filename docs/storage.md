# 儲存層與資料表

## 主要資料表

### bus
- `raw_bus_route`
  - 原始 API jsonb 快照
- `bus_subroutes`
  - 子路線與站點、班表整理後資料
- `bus_static`
  - 序列化後的 protobuf bytes
- `bus_stations`
  - 公車站點資訊與座標
- `bus_station_stop_map`
  - 站點與路線對應
- `bus_schedule`
  - 班表與發車頻率

### bike
- `bike_stations`
  - 站點、容量、地址、座標

### mrt
- `mrt_station`
  - 捷運站點與座標
- `mrt_firstlast`
  - 首末班車時間

### rail
- `tra_stations`
- `thsr_stations`
- `tra_fares`
- `thsr_fares`
- `tra_timetable`
- `thsr_timetable`

### vector
- `search_vector`
  - 類型、名稱、城市、幾何資訊與向量

## 常見讀寫行為
- `bus_static` 以 `sub_route_uid` 查詢 protobuf bytes
- `bus_stations` 以地理距離排序查詢近站
- `tra_timetable` 與 `thsr_timetable` 依 `train_date` 與 `station_id` 查詢
- `tra_fares` 與 `thsr_fares` 依起迄站查詢
