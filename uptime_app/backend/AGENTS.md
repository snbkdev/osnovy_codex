# Backend Guidelines

## Scope & Module Organization

Follow the shared rules in [../AGENTS.md](../AGENTS.md). This guide applies to `backend/`.

- `cmd/api/`: HTTP server entry point and graceful shutdown.
- `internal/config/`: configuration loaded from process environment variables.
- `internal/httpapi/`: routes, handlers, and colocated HTTP tests.
- `migrations/`: reserved for future database migrations.

The module path is `uptime_app/backend`. Keep business logic under `internal/`, separate from HTTP handlers. The API currently uses the standard library; no database or authentication layer is configured.

## Development & Build Commands

Use Go 1.25+. Run these commands from `backend/`:

- `go run ./cmd/api`: start the API, defaulting to port 8080.
- `go build -o bin/api ./cmd/api`: build the server binary.
- `go test ./...`: run all backend tests.
- `go vet ./...`: run static checks.
- `gofmt -w <changed-files>`: format changed Go files.

From the application root, `make backend-dev` starts the API.

## Coding Style & Testing

Use `gofmt`, tab indentation, lowercase package names, and PascalCase exported identifiers. Keep HTTP routing in `internal/httpapi/` and configuration defaults in `internal/config/`.

Use Go's `testing` package and `net/http/httptest`. Place `*_test.go` beside the implementation and name test functions `TestBehavior`. Cover relevant response codes, JSON bodies, and error paths. No coverage threshold is configured.

Preserve server timeouts and graceful shutdown when changing server initialization. Coordinate API contract changes with the frontend.

## Configuration

Go reads process variables; `backend/.env` is not loaded automatically. Set `HTTP_ADDR` to change the listening address, for example `HTTP_ADDR=:8081 go run ./cmd/api`. Update `.env.example` for new settings and match the frontend's `API_URL` to the configured address.
