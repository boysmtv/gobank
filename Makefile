.PHONY: build run test lint migrate-up migrate-down docker-up docker-down seed

BINARY=gobank
MAIN=./cmd/server
MIGRATE_DSN?=postgres://gobank:gobank@localhost:5432/gobank?sslmode=disable

build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/$(BINARY) $(MAIN)

run:
	go run $(MAIN)

test:
	go test ./... -v -race -coverprofile=coverage.out -covermode=atomic

test-coverage:
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

migrate-up:
	migrate -path ./migrations -database "$(MIGRATE_DSN)" up

migrate-down:
	migrate -path ./migrations -database "$(MIGRATE_DSN)" down

migrate-drop:
	migrate -path ./migrations -database "$(MIGRATE_DSN)" drop -f

migrate-create:
	migrate create -ext sql -dir ./migrations -seq $(name)

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down -v

docker-logs:
	docker compose logs -f app

seed:
	go run ./scripts/seed.go

sqlc-generate:
	sqlc generate

tidy:
	go mod tidy

vet:
	go vet ./...

.DEFAULT_GOAL := build