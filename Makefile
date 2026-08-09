.PHONY: test test-verbose build up down migrate-down

# Запуск всех тестов (юнит-тесты с sqlmock, без реальной БД)
test:
	cd backend && go test ./...

test-verbose:
	cd backend && go test -v ./...

build:
	cd backend && go build -o /tmp/freelance-btc-server ./cmd/api

up:
	docker compose up --build

down:
	docker compose down

# Откатить последнюю миграцию (нужен установленный CLI: go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest)
migrate-down:
	migrate -source "file://backend/internal/db/migrations" -database "$$DATABASE_URL" down 1
