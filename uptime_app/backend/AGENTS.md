# Backend Guidelines

## Scope & Module Organization

Follow the shared rules in [../AGENTS.md](../AGENTS.md). This guide applies to `backend/`.

- `cmd/api/`: HTTP server entry point and graceful shutdown.
- `internal/config/`: configuration loaded from process environment variables.
- `internal/httpapi/`: routes, handlers, and colocated HTTP tests.
- `internal/auth/`: authentication service, cryptography and repository interfaces.
- `internal/storage/`: Bun repositories, database pool and transactional mail worker.
- `internal/mail/`: SMTP sender.
- `cmd/migrate/` and `migrations/`: explicit embedded SQL migrations.

The module path is `uptime_app/backend`. Keep business logic under `internal/`, separate from HTTP handlers. The API uses PostgreSQL through Bun and JWT authentication; keep ORM types inside storage.

## Development & Build Commands

Use Go 1.25+. Run these commands from `backend/`:

- `go run ./cmd/api`: start the API, defaulting to port 8080.
- `go build -o bin/api ./cmd/api`: build the server binary.
- `go test ./...`: run all backend tests.
- `go vet ./...`: run static checks.
- `go run ./cmd/migrate up|status|down`: manage migrations explicitly.
- `TEST_DATABASE_URL=... go test -race ./...`: include destructive integration tests on a dedicated database ending in `_test`; never use the application database.
- `gofmt -w <changed-files>`: format changed Go files.

From the application root, `make backend-dev` starts the API.

## Coding Style & Testing

Use `gofmt`, tab indentation, lowercase package names, and PascalCase exported identifiers. Keep HTTP routing in `internal/httpapi/` and configuration defaults in `internal/config/`.

Use Go's `testing` package and `net/http/httptest`. Place `*_test.go` beside the implementation and name test functions `TestBehavior`. Cover relevant response codes, JSON bodies, and error paths. No coverage threshold is configured.

Preserve server timeouts and graceful shutdown when changing server initialization. Coordinate API contract changes with the frontend.

## API Documentation (Swaggo)

Every HTTP API route must have Swaggo annotations on its handler describing the method and path, parameters and request body, response schemas and status codes, and authentication requirements. Add annotations when introducing routes and update them whenever a route or its API behavior changes, including shared request and response types. Remove obsolete annotations when deleting routes.

After any API contract change, run `make swagger` from `uptime_app/` to regenerate `backend/docs/docs.go`, `backend/docs/swagger.json`, and `backend/docs/swagger.yaml`. Include all regenerated files with the API changes; do not edit generated files manually. Keep the documented operation list in `internal/httpapi/router_test.go` synchronized when adding or removing routes, and run `go test ./internal/httpapi` from `backend/` to verify the served contract.

Then run `make redoc` from `uptime_app/` to regenerate the standalone `backend/docs/redoc.html` from `backend/docs/swagger.json` and include it with the changes. Use `make swagger redoc` to update the contract and HTML together.

## Configuration

Go reads process variables; `backend/.env` is not loaded automatically. Set `HTTP_ADDR` to change the listening address, for example `HTTP_ADDR=:8081 go run ./cmd/api`. Update `.env.example` for new settings and match the frontend's `API_URL` to the configured address.
