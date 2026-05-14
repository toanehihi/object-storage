.PHONY: run lint test docker-up docker-down migrate-up migrate-down build vet

# --- Run ---
run:
	go run ./cmd

# --- Build ---
build:
	go build -o bin/server ./cmd

# --- Lint & Test ---
lint:
	golangci-lint run ./...

test:
	go test -v -race ./...

vet:
	go vet ./...

# --- Docker ---
docker-up:
	docker compose up -d

docker-down:
	docker compose down

# --- Migrations ---
migrate-up:
	migrate -path migrations -database "postgres://objectstorage:objectstorage@localhost:5432/objectstorage?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://objectstorage:objectstorage@localhost:5432/objectstorage?sslmode=disable" down
