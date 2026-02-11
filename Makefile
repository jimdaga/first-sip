.PHONY: build test lint run docker-build clean templ-generate dev

templ-generate:
	$(HOME)/go/bin/templ generate

build: templ-generate
	go build -ldflags="-s -w" -o server ./cmd/server

dev: templ-generate
	go run cmd/server/main.go

test:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

run:
	go run ./cmd/server

docker-build:
	docker build -t jimdaga/first-sip:local .

clean:
	rm -f server coverage.out
