.SILENT: help
.PHONY: help run dev test lint fmt vet tidy build clean swag docker-up docker-down docker-build

APP_NAME=messaging
BIN_DIR=bin
BIN=$(BIN_DIR)/api

help:
	echo "Targets:"
	echo "  run           - run api locally (uses config/config.yaml)"
	echo "  dev           - run via docker compose"
	echo "  build         - build api binary to $(BIN)"
	echo "  test          - run unit tests"
	echo "  fmt           - format code (gofmt -s -w)"
	echo "  vet           - run go vet"
	echo "  tidy          - go mod tidy"
	echo "  lint          - basic lint: fmt check + vet"
	echo "  clean         - remove build artifacts"
	echo "  swag          - generate swagger (requires swag CLI)"
	echo "  docker-up     - docker compose up"
	echo "  docker-down   - docker compose down -v"
	echo "  docker-build  - docker build image $(APP_NAME):latest"

run:
	go run ./cmd/api

dev: docker-up

test:
	go test ./... -v

fmt:
	gofmt -s -w .

vet:
	go vet ./...

tidy:
	go mod tidy

lint: fmt vet
	@echo "Lint OK"

build:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -o $(BIN) ./cmd/api

clean:
	rm -rf $(BIN_DIR)

# Swagger generation (requires swag CLI)
swag:
	@command -v swag >/dev/null 2>&1 || { \
		echo "swag not found. Install: go install github.com/swaggo/swag/cmd/swag@latest"; \
		exit 1; \
	}
	swag init -g cmd/api/main.go -o internal/api/swagger
	@echo "Swagger docs generated in internal/api/swagger"

docker-up:
	docker compose up --build

docker-down:
	docker compose down -v

docker-build:
	docker build -f docker/Dockerfile -t $(APP_NAME):latest .
