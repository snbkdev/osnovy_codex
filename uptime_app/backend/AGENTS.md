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

## Database Inspection via MCP

Use the PostgreSQL MCP server (`postgres`) to inspect schemas and read data needed for development. Discover the available `mcp__postgres__*` tools in the current session; connection settings belong to the MCP server configuration, not tool arguments.

1. Confirm the connection with `mcp__postgres__execute_sql({"sql":"SELECT current_database() AS database_name, current_schema() AS schema_name;"})`. Do not assume MCP connects to the same database as the API or integration tests.
2. Discover schemas with `mcp__postgres__list_schemas({})`, then tables with `mcp__postgres__list_objects({"schema_name":"public","object_type":"table"})`.
3. Inspect columns, keys, and relationships with `mcp__postgres__get_object_details({"schema_name":"public","object_name":"monitors","object_type":"table"})`. Use actual schema and table names returned by discovery.
4. Read data with `mcp__postgres__execute_sql({"sql":"SELECT count(*) AS monitor_count FROM public.monitors;"})`. For row samples, select only needed columns, add relevant filters, use a deterministic `ORDER BY`, and bound results with `LIMIT` (usually 20–50 rows). Avoid `SELECT *`.

The SQL tool is read-only. Apply schema changes through [the migration workflow](../docs/guides/migrations.md). Do not expose password hashes, tokens, credentials, mail payloads, or unnecessary personal data in results, logs, documentation, or fixtures. Prefer counts and anonymized examples when real values are unnecessary.

If PostgreSQL MCP tools are unavailable or the connection fails, report the limitation instead of inventing results. Repository migrations and storage models describe the expected schema, but do not establish the current database state.

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
