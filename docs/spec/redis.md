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

## 快取 key
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

## TTL
- Bus ETA
  - `bus_eta_route:*`：180 秒
  - `bus_eta_station:*`：180 秒
- Bike ETA
  - `bike_availability:*`：120 秒
- MRT LiveBoard
  - `mrt_live:*`：120 秒
- TRA Delay/LiveBoard
  - `tra:delay_all`：180 秒
  - `tra:delay`：180 秒
  - `tra:liveboard:*`：180 秒
- Bus DailyTimetable
  - `bus_daily_timetable:*`：23.5 小時
- Fares/Timetables
  - `TRA_Fare:*`：8 小時
  - `THSR_Fare:*`：1 小時
  - `TRA_timetable:*`：1 小時
  - `THSR_timetable:*`：1 小時
  - `TRA_Stoptimes:*`：1 小時
  - `THSR_Stoptimes:*`：1 小時
