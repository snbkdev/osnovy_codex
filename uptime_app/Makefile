.PHONY: swagger redoc setup dev stop backend-dev frontend-dev build check infra-up infra-down migrate-up migrate-status migrate-down test-integration test-auth-browser
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
	bash scripts/backend-env.sh go run ./cmd/api

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
	bash scripts/backend-env.sh go run ./cmd/migrate up

migrate-status:
	bash scripts/backend-env.sh go run ./cmd/migrate status

migrate-down:
	bash scripts/backend-env.sh go run ./cmd/migrate down

test-integration:
	@test -n "$$TEST_DATABASE_URL" || (echo "Set TEST_DATABASE_URL to a dedicated database ending in _test"; exit 1)
	cd backend && go test -race -count=1 ./...

swagger:
	cd backend && go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/api/main.go --parseInternal --output docs

redoc:
	npx --yes @redocly/cli@1.34.3 build-docs backend/docs/swagger.json --output backend/docs/redoc.html --disableGoogleFont
