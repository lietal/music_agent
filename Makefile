.PHONY: dev test build lint clean test-backend lint-backend build-backend build-frontend

dev:
	docker compose up -d
	@echo "Backend running at http://localhost:8080"
	cd web && npm run dev

test: test-backend
	@echo "All tests passed"

test-backend:
	go test ./...

build: build-backend build-frontend

build-backend:
	go build -o bin/server ./cmd/server

build-frontend:
	cd web && npm ci && npm run build

lint: lint-backend
	golangci-lint run ./...

lint-backend:
	golangci-lint run ./...

clean:
	docker compose down -v
	rm -rf bin/
	rm -rf web/dist/
