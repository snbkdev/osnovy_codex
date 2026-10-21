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

Run checks appropriate to the changed code. For integration changes, run `make check` and `make build`, then start both applications and verify the API status button. No project-wide coverage threshold is configured.

## Browser Work & Required Implementation Verification

Use **Playwright MCP** for all agent-driven browser work in this project: navigation, interaction, screenshots, console messages, and network inspection. Existing Playwright test scripts complement this verification and do not replace it.

After implementing a task that affects application behavior or UI, load and follow [playwright-verification](.codex/skills/playwright-verification/SKILL.md) before reporting completion. Verify the affected functionality and rendered layout through Playwright MCP, fix defects within the task scope, and repeat affected checks after fixes. This also applies to backend changes that affect browser-visible behavior.

For documentation-only changes or changes with no browser-observable effect, explicitly state why browser verification is not applicable. If Playwright MCP or a required local service is unavailable, report the blocker and unverified scenarios; do not claim verification passed or silently substitute another browser tool.

## Commit & Pull Request Guidelines

The initial Git history has no formal commit convention. Use concise, descriptive subjects and keep commits focused.

PRs should describe the problem, resulting behavior, and validation performed. Link relevant issues, include screenshots for visible UI changes, and document configuration or dependency changes.

## Configuration & Secrets

Keep secrets out of Git and update the relevant `.env.example` when adding settings. The main `.gitignore` lives one directory above this application; maintain application exclusions there. Commit dependency manifests and lockfiles when dependencies change.
