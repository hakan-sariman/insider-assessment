.SILENT: help
.PHONY: help run dev test lint swag migrate-up migrate-down docker-up docker-down

APP_NAME=messaging

help:
	echo "Targets:"
	echo "  run           - run api locally (uses config/config.yaml)"
	echo "  dev           - run via docker compose"
	echo "  test          - run unit tests"
	echo "  swag          - generate swagger (requires swag CLI)"
	echo "  migrate-up    - apply SQL migrations (simple psql example)"
	echo "  migrate-down  - rollback latest migration (manual)"
	echo "  docker-up     - docker compose up"
	echo "  docker-down   - docker compose down -v"

run:
	go run ./cmd/api

dev: docker-up

test:
	go test ./... -v

# Swagger generation (annotations exist; you can add swag CLI later)
swag:
	echo "Install swag if missing: go install github.com/swaggo/swag/cmd/swag@latest"
	echo "Then run: swag init -g cmd/api/main.go -o internal/api/swagger"
	echo "To enable server route, build with: go build -tags swagger ./cmd/api"

migrate-up:
	echo "Apply migrations manually using psql, for example:"
	echo "psql $$DATABASE_URL -f internal/storage/migrations/001_init.sql"

migrate-down:
	echo "Create a down migration or revert manually."

docker-up:
	docker compose up --build

docker-down:
	docker compose down -v
