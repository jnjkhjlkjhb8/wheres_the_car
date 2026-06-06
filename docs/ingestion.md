made by claude
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

## TDX MQTT 訂閱

- 實作位置：`server/functions/mqtt.go`，啟動函數 `startMQTT(rc)`
- 在 `main.go` 於 cron 排程啟動後呼叫，程式結束時呼叫 `Disconnect(500)`
- 若 `MQTT_CLIENT_ID` / `MQTT_USERNAME` / `MQTT_PASSWORD` 任一為空則跳過，不影響其他排程

### 連線
- Broker：`mqtts://mqtt.transportdata.tw:8883`（MQTTS / TLS）
- 憑證：`MQTT_CLIENT_ID`、`MQTT_USERNAME`、`MQTT_PASSWORD`
- `SetAutoReconnect(true)` + `SetConnectRetry(true)` + 每 10 秒重連
- 所有訂閱於 `OnConnectHandler` 中重新建立（確保斷線重連後恢復）

### 訂閱主題與 Redis 行為

| MQTT topic（QoS 1）| Redis key | TTL |
|---|---|---|
| `v2/Bus/RealTimeNearStop/City/#` | `mqtt:v2:Bus:RealTimeNearStop:City:{city}:{routeId}` | 60 秒 |
| `v2/Bus/News/City/+` | `mqtt:v2:Bus:News:City:{city}` | 5 分鐘 |
| `v2/Bus/News/InterCity` | `mqtt:v2:Bus:News:InterCity` | 5 分鐘 |
| `v2/Rail/Metro/Alert/#` | `mqtt:v2:Rail:Metro:Alert:{system}` | 5 分鐘 |
| `v3/Rail/TRA/Alert` | `mqtt:v3:Rail:TRA:Alert` | 5 分鐘 |
| `v2/Rail/THSR/AlertInfo` | `mqtt:v2:Rail:THSR:AlertInfo` | 5 分鐘 |

- Redis key 推導規則：`"mqtt:" + topic.replace("/", ":")`
- 每筆訊息：`rc.Set(key, payload, ttl)` 存快取 + `rc.Publish(key, payload)` 推送至 Pub/Sub
- 訊息格式：TDX 標準 JSON，**不解析**，原文儲存

## 向量更新 (changetovector)
- 來源
  - `bus_static`, `bus_stations`, `bike_stations`, `mrt_station`, `tra_stations`, `thsr_stations`
- 目的表
  - `search_vector`
- 向量模型
  - `BAAI/bge-large-zh-v1.5` (HuggingFace)
