.PHONY: build test lint run docker-build clean templ-generate dev db-up db-down db-reset worker

templ-generate:
	$(HOME)/go/bin/templ generate

build: templ-generate
	go build -ldflags="-s -w" -o server ./cmd/server

dev: templ-generate
	@echo "Ensure Postgres and Redis are running: make db-up"
	@echo "Then source environment: source env.local"
	go run cmd/server/main.go

worker: templ-generate
	go run cmd/server/main.go --worker

test:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

run:
	go run ./cmd/server

docker-build:
	docker build -t jimdaga/first-sip:local .

db-up:
	docker compose up -d postgres redis asynqmon

db-down:
	docker compose down

db-reset:
	docker compose down -v && docker compose up -d

clean:
	rm -f server coverage.out
