# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Before changing code

- Read `docs/README.md` first to understand the doc index.
- Read `docs/requirements.md` before any implementation.
- All changes require a requirements doc update at `docs/requirements.md` and explicit confirmation before implementation.
- When instructions are unclear, ask first.

## Commands

### Backend (Go)

```bash
# Run the gRPC server (local dev)
go run ./server/router

# Run the data ingestion scheduler (local dev)
go run ./server/functions

# Run all backend tests
go test ./...

# Run tests in a specific package
go test ./server/router/...
go test ./server/functions/...
```

### Frontend (Flutter)

```bash
cd frontend

# Run the app
flutter run

# Run tests
flutter test

# Analyze
flutter analyze
```

### Protobuf / gRPC code generation

**Rule: only edit `models/*.proto`. Never manually edit generated files.**

- `models/*.pb.go` — generated inside Docker at build time, never commit, never hand-edit
- `frontend/lib/data/generated/*.dart` — generated locally, never commit, never hand-edit

When a `.proto` file changes, regenerate stubs:

```bash
# Regenerate Dart stubs (run from repo root)
export PATH="$PATH:$HOME/.pub-cache/bin"
protoc --dart_out=grpc:frontend/lib/data/generated -I models models/*.proto
```

Go stubs are rebuilt automatically on the next `docker compose up --build`.

Both `models/*.pb.go` and `frontend/lib/data/generated/` are gitignored.

### Deployment

All services run on a single Ubuntu machine via Docker Compose.

```bash
# First-time or full rebuild
docker compose up -d --build

# Restart a single service
docker compose restart <service>

# View logs
docker compose logs -f <service>
```

Automated deployment via GitHub Actions (`Manual Deploy` workflow — trigger manually from the Actions tab).

### Environment

Copy `docker.env.example` to `docker.env` and fill in values.

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | Azure PostgreSQL connection string |
| `REDIS_ADDR` | `redis:6379` (Docker internal — do not change) |
| `TDX_CLIENT_ID` | TDX API credentials |
| `TDX_CLIENT_SECRET` | TDX API credentials |
| `HF_TOKEN` | HuggingFace (vector embedding, server-side) |
| `OSRM_FILE` | Filename of pre-processed `.osrm` file in `osrm-data/` |
| `MQTT_CLIENT_ID` | TDX MQTT credentials (leave empty to skip) |
| `MQTT_USERNAME` | TDX MQTT credentials |
| `MQTT_PASSWORD` | TDX MQTT credentials |

## Architecture

### Deployment — single Ubuntu host (6 GB RAM)

All services are managed by `docker-compose.yaml` at the repo root.

| Service | Port(s) | Memory |
|---|---|---|
| redis | 127.0.0.1:6379 | 512 MB |
| router | 50051 (gRPC), 8080 (HTTP) | 256 MB |
| functions | — | 192 MB |
| powersync | 8081 | 512 MB |
| osrm | 127.0.0.1:5000 | 1536 MB |

PostgreSQL is hosted on **Azure** (external). Redis and OSRM are only bound to localhost.

### Backend — two Go binaries

**`server/functions/`** — data ingestion scheduler + TDX MQTT subscriber.
- `robfig/cron` schedules:
  - 03:00 daily: `busStatic`, `bikeStatic`, `mrtStatic`, `railStatic`, vector update
  - Every 30 s: `bikeEta`, `BusEta`
  - Every 10 s: `mrtEta`
  - Every 2 min: `traEta`
- TDX MQTT (`mqtt.go`): subscribes to `mqtt.transportdata.tw:8883`, stores alerts in Redis Pub/Sub.

**`server/router/`** — gRPC server on `:50051` + HTTP on `:8080`.
- gRPC: static queries hit PostgreSQL; real-time streams subscribe to Redis Pub/Sub.
- HTTP: `GET /api/token/powersync` (JWT), `GET /api/.well-known/jwks.json` (JWKS), `POST /api/embed` (HuggingFace proxy).

**`models/*.proto`** — source of truth. Go stubs generated in Docker; Dart stubs generated locally.

### Data flow

```
TDX REST API ──cron──→ functions ──→ PostgreSQL (static)
                                 └──→ Redis (ETA cache + Pub/Sub)

TDX MQTT ──push──→ functions ──→ Redis (alert cache + Pub/Sub)

Redis Pub/Sub ──→ router ──gRPC stream──→ Flutter

PostgreSQL ──→ PowerSync ──sync──→ Flutter SQLite (offline search)
```

### Frontend (Flutter)

Entry: `frontend/lib/main.dart`.

**Directory structure** (feature-first):
```
lib/
├── main.dart
├── app/      → app.dart, router/app_router.dart, theme/app_theme.dart
├── core/     → grpc/grpc_client.dart, powersync/, storage/hive_store.dart,
│               location/, haptics/
├── data/     → generated/ (protoc output), repositories/, decoders/, models/
├── features/ → home/ board/ map/ rail/ metro/ bus/ search/ alerts/
│               live_activity/ settings/ monitor/
└── shared/widgets/   # shared UI components
```

**Routing**: `go_router` `StatefulShellRoute.indexedStack` with 4 branches (首頁/地圖/捷運/雙鐵). Defined in `frontend/lib/app/router/app_router.dart`. Shell widget: `frontend/lib/shared/widgets/main_scaffold.dart` (floating pill nav + circular search FAB).

**State management**: `flutter_bloc` Bloc only (no Cubit). All bottom sheets use `smooth_sheets`.

**Board widget system**: Home screen is a draggable/resizable grid board. Board state in `frontend/lib/features/board/bloc/board_bloc.dart`. Widget picker sheet in `frontend/lib/features/board/widgets/widget_picker_sheet.dart`.

**Search**: `SpotlightOverlay.show(context)` (static method, `frontend/lib/features/search/view/spotlight_overlay.dart`). Queries PowerSync SQLite `search_vector` table. Driven by `SearchBloc`.

**Alerts**: `AlertBloc` is globally provided in `app.dart`. Subscribes to 4 alert streams on startup.

**PowerSync**: `POWERSYNC_URL` injected at build time via `--dart-define=POWERSYNC_URL=http://<host>:8081`.

**dart-define variables**:
| Variable | Default | Purpose |
|---|---|---|
| `API_BASE_URL` | `http://localhost:8080` | Go HTTP server |
| `POWERSYNC_URL` | — | PowerSync sync endpoint |
| `GRPC_HOST` | `localhost` | gRPC server host |
| `GRPC_PORT` | `50051` | gRPC server port |
| `GRPC_TLS` | `false` | Enable TLS for gRPC |


## Docs

All spec documents are in `docs/`:

| File | Contents |
|---|---|
| `docs/README.md` | Doc index and workflow rules |
| `docs/requirements.md` | Feature requirements (gitignored — edit freely) |
| `docs/architecture.md` | System architecture and deployment |
| `docs/grpc.md` | gRPC service specifications |
| `docs/redis.md` | Redis key/channel conventions |
| `docs/ingestion.md` | Data ingestion schedules and MQTT |
| `docs/storage.md` | PostgreSQL table schema |
| `docs/config.md` | Environment variables reference |
| `docs/widgets.md` | Frontend widget module system |
| `docs/repositories.md` | Frontend repository data layer (gRPC wrappers) |
