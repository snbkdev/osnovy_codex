.PHONY: setup backend-dev frontend-dev build check

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
