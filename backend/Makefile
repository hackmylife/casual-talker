.PHONY: build dev migrate-up migrate-down sqlc test lint

# Binary output name
BINARY := casual-talker

# Migration directory
MIGRATIONS_DIR := db/migrations

# Load DATABASE_URL from .env or config.env if present, fall back to environment
-include config.env
export

build:
	GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build -o $(BINARY) ./cmd/server/

dev:
	go run ./cmd/server/

migrate-up:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

sqlc:
	sqlc generate

test:
	go test ./...

lint:
	golangci-lint run
