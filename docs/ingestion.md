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
- 完成後呼叫 `railPreFetch` 與 `railCleanup`

## railPreFetch（由 railStatic 呼叫）
- 預抓未來時刻表，每日 03:00 隨 railStatic 一起執行
- TRA：抓今日 +1 到 +60 天（`/v2/Rail/TRA/DailyTimetable/TrainDate/{date}`）
- THSR：抓今日 +1 到 +45 天（`/v2/Rail/THSR/DailyTimetable/TrainDate/{date}`）
- 每個日期使用獨立的 If-Modified-Since cache key（`tra_traindate_{date}`），避免跨日衝突
- 第一次執行為冷啟動（全量抓取），後續執行大部分回傳 304 不做更新
- 寫入表：`tra_timetable`、`thsr_timetable`

## railCleanup（由 railStatic 呼叫）
- 刪除 `train_date < CURRENT_DATE` 的過期時刻表資料
- 對象：`tra_timetable`、`thsr_timetable`

## traEta
- 來源 API
  - `/v2/Rail/TRA/LiveTrainDelay`
  - `/v2/Rail/TRA/LiveBoard`
- 寫入 Redis
  - `tra:delay` (hash)
  - `tra:delay_all`
  - `tra:liveboard:{station_id}`

## tra_timetable / thsr_timetable
- 由 `railPreFetch` 管理（見上方），不再單獨呼叫
- Router package 中的 `get_tra_timetable` / `get_thsr_timetable` 仍保留供 gRPC 查詢使用，但使用舊的單一 cache key，僅於 router 內部觸發時執行

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
  - `tra_timetable`（DISTINCT ON trainno，只取未來日期）→ type `tra_train`
  - `thsr_timetable`（DISTINCT ON trainno，只取未來日期）→ type `thsr_train`
- 目的表
  - `search_vector`
- 向量模型
  - `qwen3-embedding:0.6b`（Ollama 本機服務，`http://ollama:11434/api/embed`）
  - 維度：1024，pgvector `vector(1024)` 欄位，HNSW 索引
