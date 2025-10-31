## Messaging Service

Lightweight message scheduling service in Go. Stores messages in Postgres, caches sent messages in Redis, and periodically sends unsent messages to an outbound webhook.

### Features
- Simple scheduler: sends a fixed batch of unsent messages each tick
- HTTP API to create messages, list sent messages, start/stop scheduler
- Auto DB migrations on startup
- Optional Swagger docs

---

## Requirements
- Go 1.24+ (for local development)
- Docker and Docker Compose (for containerized run)

---

## Quick Start (Docker)

This brings up Postgres, Redis, API, and a sample webhook server.

```bash
make docker-up
# or
docker compose up --build
```

Services:
- API: http://localhost:8080
- Webhook (sample): http://localhost:8090
- Postgres: localhost:5432
- Redis: localhost:6379

Stop and clean volumes:
```bash
make docker-down
# or
docker compose down -v
```

---

## Local Run (without Docker)

1) Start dependencies yourself (Postgres, Redis), or reuse those from Docker.

2) Copy and adjust config:
```bash
cp config/config.yaml.example config/config.yaml
```

3) Run the API:
```bash
make run
# or
go run ./cmd/api
```

By default the app reads `config/config.yaml`. You can override via env variables (Viper) or set a custom path:
```bash
export APP_CONFIG_PATH=/absolute/path/to/your/config.yaml
```

---

## Configuration

Config file: `config/config.yaml` (a ready-to-copy example is in `config/config.yaml.example`).

Key sections:
- `server`: port and timeouts
- `postgres`: `url` and connection limits
- `redis`: address, db, ttl for sent cache
- `scheduler`: `enabled`, `interval`, `batch_size`
- `outbound`: webhook `url`, `timeout`, `expect_status`, and auth header/value
- `swagger.enabled`: enable serving swagger docs when built with tag

Environment overrides example:
```bash
APP_SERVER_PORT=9090 APP_REDIS_ADDR=localhost:6379 make run
```

When using Docker Compose, `config/` is mounted into the API container and `APP_CONFIG_PATH` is set to `/app/config/config.yaml`.

---

## API Endpoints

- Health:
  - `GET /healthz`

- Messages:
  - `POST /api/v1/messages` — create a message
    - body: `{ "to": "string", "content": "string" }`
  - `GET /api/v1/messages?limit=50&offset=0` — list sent messages

- Scheduler:
  - `POST /api/v1/scheduler/start`
  - `POST /api/v1/scheduler/stop`

Default port: `8080`

---

## Swagger (optional)

Generate swagger files and build with the swagger tag, then open `/swagger/index.html`.

```bash
go install github.com/swaggo/swag/cmd/swag@latest
make swag
```

---

## Useful Make Targets

```bash
make run          # run API locally
make dev          # run via docker compose
make test         # run unit tests
make build        # build binary to bin/api
make docker-up    # compose up (builds images)
make docker-down  # compose down -v
make docker-build # build api image locally
```

---

## Notes
- Database migrations run automatically at API startup.
- The included `webhook` service simply accepts requests and returns HTTP 202.

---

## License
See `LICENSE`.
