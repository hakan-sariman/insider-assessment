# Messaging Assestment

- Go 1.24, Gorilla **mux**, **Viper** (YAML + env), **Zap** logging
- Postgres for persistence, Redis for 24h cache of sent messages
- Simple scheduler: **every 2m, send exactly 2 unsent messages** (no cron pkg)
- Start/Stop scheduler APIs; auto-start on boot

## Quick Start

```bash
make docker-up            # starts postgres, redis, api
# Or run locally:
# go run ./cmd/api
```

API:
- `POST /api/v1/scheduler/start`
- `POST /api/v1/scheduler/stop`
- `GET /api/v1/messages?status=sent&limit=50&offset=0`
- `POST /api/v1/messages` (create unsent message, 140 chars max)

Health:
- `GET /healthz`

### Config
- YAML at `config/config.yaml` (Viper), override via env, e.g.: `APP_SERVER_PORT=9090`
- Important keys: `postgres.url`, `redis.addr`, `scheduler.*`, `outbound.*`

### Swagger
```bash
go install github.com/swaggo/swag/cmd/swag@latest
make swag
# then build with:
go build -tags swagger ./cmd/api
# visit /swagger/index.html
```

### Tests
```bash
make test
```
