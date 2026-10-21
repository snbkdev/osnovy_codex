# Repository Guidelines

## Scope & Project Organization

Run shared commands from `uptime_app/`, inside the parent Git repository.

- `backend/`: Go HTTP API; follow [backend/AGENTS.md](backend/AGENTS.md).
- `frontend/`: Next.js application; follow [frontend/AGENTS.md](frontend/AGENTS.md).
- `scripts/`: project setup scripts.
- `Makefile`: shared development and validation commands.

This guide applies to both applications. Each directory guide adds instructions for its own scope.

## Shared Development Commands

The full project requires Go 1.25+, Node.js 20.9+, and npm.

- `make setup`: create missing local environment files and install frontend dependencies.
- `make dev`: start both applications, load `backend/.env`, and stop both on Ctrl+C or process failure.
- `make stop`: stop this checkout's active `make dev` from another terminal; repeated stops are safe.
- `make backend-dev` and `make frontend-dev`: start the applications in separate terminals.
- `make check`: run Go tests, go vet, ESLint, and TypeScript checks.
- `make build`: build both applications for production.
- `make test-auth-browser`: run Playwright against local applications and Mailpit; use a separate API test database and a fresh API process to avoid mail rate limits.

Use tabs for Makefile recipes. Keep setup scripts repeatable and preserve existing local configuration.

## API Integration & Validation

For database schema changes, follow [docs/guides/migrations.md](docs/guides/migrations.md) for migration creation, execution, rollback, and verification.

The frontend proxies `/api/*` requests to the Go API through Next.js. Coordinate route, response, and configuration changes across both applications. The health endpoint is `/api/v1/health`.

Run checks appropriate to the changed code. For integration changes, run `make check` and `make build`, leave verification of the API status button in the browser to the user. No project-wide coverage threshold is configured.

## Browser Verification

Правило пользователя: «Не нужно запускать открытие браузера, запускаю и проверяю сам».

Do not open a browser or run browser checks (including Playwright MCP and browser test scripts) unless the user explicitly requests them. The user starts the application and verifies browser behavior and layout themselves. This rule overrides the automatic browser verification requirement in the playwright-verification skill.

Continue running appropriate non-browser checks. In the completion report, distinguish checks actually performed from browser verification left to the user; do not claim browser verification passed.

## Commit & Pull Request Guidelines

The initial Git history has no formal commit convention. Use concise, descriptive subjects and keep commits focused.

PRs should describe the problem, resulting behavior, and validation performed. Link relevant issues, include screenshots for visible UI changes, and document configuration or dependency changes.

## Configuration & Secrets

Keep secrets out of Git and update the relevant `.env.example` when adding settings. The main `.gitignore` lives one directory above this application; maintain application exclusions there. Commit dependency manifests and lockfiles when dependencies change.
