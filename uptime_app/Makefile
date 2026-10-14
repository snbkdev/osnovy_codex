.PHONY: setup dev stop backend-dev frontend-dev build check infra-up infra-down migrate-up migrate-status migrate-down test-integration test-auth-browser
.DEFAULT_GOAL := setup

dev:
	node scripts/dev.mjs

stop:
	node scripts/stop.mjs

test-auth-browser:
	cd frontend && npm run test:auth-browser

setup:
	./scripts/setup.sh

backend-dev:
	cd backend && go run ./cmd/api

frontend-dev:
	cd frontend && npm run dev

build:
	cd backend && go build -o bin/api ./cmd/api
	cd frontend && npm run build

check:
	cd backend && go test ./... && go vet ./...
	cd frontend && npm run lint && npm run typecheck

infra-up:
	docker compose up -d --wait

infra-down:
	docker compose down

migrate-up:
	cd backend && go run ./cmd/migrate up

migrate-status:
	cd backend && go run ./cmd/migrate status

migrate-down:
	cd backend && go run ./cmd/migrate down

test-integration:
	@test -n "$$TEST_DATABASE_URL" || (echo "Set TEST_DATABASE_URL to a dedicated database ending in _test"; exit 1)
	cd backend && go test -race -count=1 ./...
