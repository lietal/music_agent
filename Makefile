.PHONY: dev test build lint clean

dev:
	docker compose up -d

test:
	go test ./...
	@if [ -d web ]; then cd web && npm test; fi

build:
	go build -o bin/server ./cmd/server

lint:
	golangci-lint run ./...

clean:
	docker compose down -v
	rm -rf bin/
