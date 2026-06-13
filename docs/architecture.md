made by claude
# 架構說明

## 部署環境

單台 Ubuntu 主機（8 GB RAM），所有服務透過 Docker Compose 統一管理。

```
Flutter App
    │
    ├─ gRPC :50051      → router
    ├─ HTTP :8080       → router (JWT / JWKS / embed)
    └─ HTTP :8081       → PowerSync (資料同步)

router / functions / powersync / osrm / redis
    全部在同一台 Ubuntu 主機
    │
    └─ Azure PostgreSQL（外部）
```

## 服務一覽

| 服務 | 映像 | 對外端口 | 記憶體上限 |
|---|---|---|---|
| redis | redis:7-alpine | 127.0.0.1:6379 | 512 MB |
| router | bus-router (Go) | 50051, 8080 | 256 MB |
| functions | bus-functions (Go) | — | 192 MB |
| powersync | journeyapps/powersync-service | 8081 | 512 MB |
| osrm | osrm/osrm-backend | 127.0.0.1:5000 | 1536 MB |
| ollama | ollama/ollama (custom) | 127.0.0.1:11434 | 800 MB |

Redis 與 OSRM 僅對 localhost 開放，不對外暴露。

## 程式結構

- `server/functions`
  - 排程執行器，負責擷取 TDX REST API 與寫入 DB/Redis
  - TDX MQTT 訂閱（`mqtt.go`），接收即時告警並推送至 Redis Pub/Sub
  - 使用 `robfig/cron` 設定排程
- `server/router`
  - gRPC 服務端（:50051），查詢 DB/Redis 並回傳 protobuf
  - HTTP 服務端（:8080）：`/api/token/powersync`、`/api/.well-known/jwks.json`、`/api/embed`
  - 串流以 Redis Pub/Sub 實作

## 外部依賴

| 依賴 | 說明 |
|---|---|
| Azure PostgreSQL | 靜態資料、時刻表、站點、路線等持久化 |
| TDX REST API | 排程擷取交通靜態與即時資料 |
| TDX MQTT | 推送式即時告警（`mqtt.transportdata.tw:8883`） |
| Ollama (本機) | 向量嵌入計算（`qwen3-embedding:0.6b`，Docker 內部服務） |

## 資料流

```
TDX REST API ──排程──→ functions ──→ PostgreSQL（靜態）
                                 └──→ Redis（ETA 快取 + Pub/Sub）

TDX MQTT ──push──→ functions ──→ Redis（告警快取 + Pub/Sub）

Redis Pub/Sub ──→ router ──gRPC stream──→ Flutter App

PostgreSQL ──→ PowerSync ──sync──→ Flutter SQLite（離線搜尋）

Flutter App ──→ router /api/embed ──→ ollama:11434 ──→ 向量
           └──→ router /api/token/powersync ──→ JWT
```

## 專案路徑

- `server/functions/*.go`：排程、資料匯入、MQTT 訂閱
- `server/router/*.go`：gRPC 服務、HTTP 端點
- `models/*.proto`：proto 定義
- `models/*_grpc.pb.go`：gRPC 介面（已提交）
- `powersync/`：PowerSync 設定（`config.yaml`、`sync-rules.yaml`）
- `osrm-data/`：OSRM 預處理檔案（gitignored，手動放置）
