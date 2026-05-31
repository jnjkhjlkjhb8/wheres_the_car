# 資料擷取與排程

## 排程時間
- 每日 03:00
  - 執行 `busStatic`, `bikeStatic`, `mrtStatic`, `railStatic`
  - 執行向量更新 `changetovector`
- 每 2 分鐘
  - 執行 `traEta`
- 每 30 秒
  - 執行 `bikeEta`, `BusEta`
- 每 10 秒
  - 執行 `mrtEta`

## busStatic
- 來源 API
  - `/v2/Bus/Route/{City}`
  - `/v2/Bus/StopOfRoute/{City}`
  - `/v2/Bus/Shape/{City}`
  - `/v2/Bus/Schedule/{City}`
  - `/v2/Bus/Station/{City}`
- 寫入表
  - `raw_bus_route`
  - `bus_subroutes`
  - `bus_stations`
  - `bus_schedule`
  - `bus_static`
- 重要行為
  - InterCity 子路線 UID 會做正規化
  - 會重建路線與站點對應表

## busDailyroute
- 來源 API
  - `/v2/Bus/DailyTimeTable/City/{City}`
- 寫入 Redis
  - `bus_daily_timetable:{sub_route_uid}`

## bikeStatic
- 來源 API
  - `/v2/Bike/Station/City/{City}`
- 寫入表
  - `bike_stations`

## bikeEta
- 來源 API
  - `/v2/Bike/Availability/City/{City}`
- 寫入 Redis
  - `bike_availability:{station_uid}`

## mrtStatic
- 來源 API
  - `/v2/Rail/Metro/Station/{System}`
  - `/v2/Rail/Metro/FirstLastTimetable/{System}`
- 寫入表
  - `mrt_station`
  - `mrt_firstlast`

## mrtEta
- 來源 API
  - `/v2/Rail/Metro/LiveBoard/{System}`
- 寫入 Redis
  - `mrt_live:{system}:{station_id}`

## railStatic
- 來源 API
  - `/v2/Rail/TRA/Station`
  - `/v2/Rail/THSR/Station`
  - `/v2/Rail/TRA/ODFare`
  - `/v2/Rail/THSR/ODFare`
- 寫入表
  - `tra_stations`, `thsr_stations`
  - `tra_fares`, `thsr_fares`

## traEta
- 來源 API
  - `/v2/Rail/TRA/LiveTrainDelay`
  - `/v2/Rail/TRA/LiveBoard`
- 寫入 Redis
  - `tra:delay` (hash)
  - `tra:delay_all`
  - `tra:liveboard:{station_id}`

## tra_timetable / thsr_timetable
- 來源 API
  - `/v2/Rail/TRA/DailyTimetable/TrainDate/{Date}`
  - `/v2/Rail/THSR/DailyTimetable/TrainDate/{Date}`
- 寫入表
  - `tra_timetable`
  - `thsr_timetable`

## 向量更新 (changetovector)
- 來源
  - `bus_static`, `bus_stations`, `bike_stations`, `mrt_station`, `tra_stations`, `thsr_stations`
- 目的表
  - `search_vector`
- 向量模型
  - `BAAI/bge-large-zh-v1.5` (HuggingFace)
