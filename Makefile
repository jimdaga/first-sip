.PHONY: build test lint run docker-build clean

build:
	go build -ldflags="-s -w" -o server ./cmd/server

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
