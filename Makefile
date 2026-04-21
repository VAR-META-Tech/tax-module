.PHONY: build run test lint clean migrate-up migrate-down docker-up docker-down

APP_NAME=tax-module
BUILD_DIR=./bin

# Build the application
build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

# Run the application
run:
	go run ./cmd/server

# Run unit tests
test:
	go test ./... -v -count=1

# Run integration tests (requires Docker)
test-integration:
	go test ./... -v -count=1 -tags=integration

# Run all tests
test-all: test test-integration

# Run linter
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Database migrations
MIGRATE_DSN ?= "postgres://datpham:secret@localhost:5432/tax_module?sslmode=disable"
MIGRATIONS_PATH=internal/repository/postgres/migrations

migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database $(MIGRATE_DSN) up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database $(MIGRATE_DSN) down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $$name

# Docker
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-build:
	docker compose build
