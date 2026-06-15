made by claude
# Redis 規格

## 連線
- 位址透過 `REDIS_ADDR` 設定
- DB 使用預設 index 0

## Pub/Sub 頻道
- Bus
  - `bus_eta_route:{sub_route_uid}`
  - `bus_eta_station:{city}:{station_name}`
- Bike
  - `bike_availability:{station_uid}`
- MRT
  - `mrt_live:{system}:{station_id}`
- TRA
  - `tra:delay:all`
  - `tra:delay:{train_no}`
  - `tra:liveboard:{station_id}`
- MQTT Alert（由 TDX MQTT 訂閱寫入）
  - `mqtt:v2:Bus:News:City:{city}`
  - `mqtt:v2:Bus:News:InterCity`
  - `mqtt:v2:Rail:Metro:Alert:{system}`
  - `mqtt:v3:Rail:TRA:Alert`
  - `mqtt:v2:Rail:THSR:AlertInfo`
- MQTT 即時公車（由 TDX MQTT 訂閱寫入）
  - `mqtt:v2:Bus:RealTimeNearStop:City:{city}:{routeId}`

## 快取 key
- Bus ETA Prediction
  - `weather:{city}`（天氣快照 JSON，`weatherSync` 寫入）
- Bus
  - `bus_daily_timetable:{sub_route_uid}`
- TRA
  - `TRA_Fare:{origin_station_id}:{destination_station_id}`
  - `TRA_timetable:{date}:{origin_station_id}:{destination_station_id}`
  - `TRA_Stoptimes:{date}:{train_no}`
- THSR
  - `THSR_Fare:{origin_station_id}:{destination_station_id}`
  - `THSR_timetable:{date}:{origin_station_id}:{destination_station_id}`
  - `THSR_Stoptimes:{date}:{train_no}`

## Hash key
- `tra:delay`
  - key：`train_no`
  - value：`delay` (秒)

## MQTT 快取 key
- `mqtt:v2:Bus:RealTimeNearStop:City:{city}:{routeId}`
- `mqtt:v2:Bus:News:City:{city}`
- `mqtt:v2:Bus:News:InterCity`
- `mqtt:v2:Rail:Metro:Alert:{system}`
- `mqtt:v3:Rail:TRA:Alert`
- `mqtt:v2:Rail:THSR:AlertInfo`

## TTL
- Bus ETA
  - `bus_eta_route:*`：180 秒
  - `bus_eta_station:*`：180 秒
- Bike ETA
  - `bike_availability:*`：120 秒
- MRT LiveBoard
  - `mrt_live:*`：120 秒
- TRA Delay/LiveBoard
  - `tra:delay:all`：180 秒  ← Pub/Sub channel（A5 已修正 _all → :all）
  - `tra:delay`：180 秒（hash，trainNo → delay 秒數）
  - `tra:liveboard:*`：180 秒
- Bus DailyTimetable
  - `bus_daily_timetable:*`：23.5 小時
- MQTT Alert
  - `mqtt:v2:Bus:News:*`：5 分鐘
  - `mqtt:v2:Bus:News:InterCity`：5 分鐘
  - `mqtt:v2:Rail:Metro:Alert:*`：5 分鐘
  - `mqtt:v3:Rail:TRA:Alert`：5 分鐘
  - `mqtt:v2:Rail:THSR:AlertInfo`：5 分鐘
- MQTT 即時公車
  - `mqtt:v2:Bus:RealTimeNearStop:*`：60 秒
- Fares/Timetables
  - `TRA_Fare:*`：8 小時
  - `THSR_Fare:*`：1 小時
  - `TRA_timetable:*`：1 小時
  - `THSR_timetable:*`：1 小時
  - `TRA_Stoptimes:*`：1 小時
  - `THSR_Stoptimes:*`：1 小時
- MaaS 路程規劃快取
  - `maas:plan:{sha256_hex8}`：90 秒
- Bus ETA Prediction
  - `weather:{city}`：15 分鐘
