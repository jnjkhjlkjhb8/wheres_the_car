# 架構說明

## 目標
- 提供交通資料的 gRPC 介面。
- 以排程方式擷取資料，並寫入 DB 與 Redis。
- 以 Redis Pub/Sub 提供串流即時資料。

## 程式結構
- `server/functions`
  - 排程執行器，負責擷取 TDX API 與寫入 DB/Redis。
  - 使用 `robfig/cron` 設定排程。
- `server/router`
  - gRPC 服務端，負責查詢 DB/Redis 並回傳 protobuf。
  - 多數串流以 Redis Pub/Sub 實作。

## 外部依賴
- PostgreSQL
  - 靜態資料、時刻表、站點、路線等持久化。
- Redis
  - 快取與 Pub/Sub 串流。
- TDX API
  - 交通資料來源。
- OSRM
  - 近站步行時間估算。
- HuggingFace
  - 向量嵌入計算。

## 執行流程
1. `server/functions` 透過排程寫入 DB 與 Redis。
2. `server/router` 依 gRPC 請求來源取得資料。
3. 串流請求以 Redis Pub/Sub 傳送更新資料。

## 專案路徑
- `server/functions/*.go`：排程與資料匯入。
- `server/router/*.go`：gRPC 服務。
- `models/*.proto`：proto 定義。
- `models/*_grpc.pb.go`：gRPC 介面與型別。
