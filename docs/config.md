made by claude
# 設定與環境變數

## Redis
- `REDIS_ADDR`
  - 格式：`host:port`

## PostgreSQL
- `DATABASE_URL`
  - 連線字串

## TDX
- `TDX_CLIENT_ID`
- `TDX_CLIENT_SECRET`

## TDX MQTT
- `MQTT_CLIENT_ID`
  - 由 TDX 會員中心 → 資料服務 → 存取金鑰取得
  - 若留空則略過 MQTT 訂閱
- `MQTT_USERNAME`
- `MQTT_PASSWORD`

## HTTP / PowerSync
- `POWERSYNC_URL`
  - Flutter build-time dart-define: `--dart-define=POWERSYNC_URL=http://your-debian-server:8080`
  - PowerSync service endpoint (Debian server)
- `API_BASE_URL`
  - Flutter build-time dart-define: `--dart-define=API_BASE_URL=http://your-go-server:8080`
  - Go backend HTTP server (JWT + embed endpoints)
  - Default: `http://localhost:8080`

## PowerSync server (powersync/.env)
- `DATABASE_URL`
  - Azure PostgreSQL connection string (same as Go backend)
- `POWERSYNC_JWKS_URL`
  - Full URL of Go backend JWKS endpoint, reachable from Debian server
  - e.g. `http://go-server-host:8080/api/.well-known/jwks.json`

## Bus ETA Prediction

- `CWA_API_KEY`
  - CWA Open Data API 金鑰，申請網址：opendata.cwa.gov.tw
  - 若留空則略過天氣同步（weatherSync 不執行）
- `BUS_ETA_MODEL_PATH`
  - XGBoost 模型檔案路徑
  - 預設：`./model/bus_eta.json`
  - 若檔案不存在則僅用班表 + travel avg（無 ML 修正）
