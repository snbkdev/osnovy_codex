# Frontend Guidelines

## Scope & Project Organization

Follow the shared rules in [../AGENTS.md](../AGENTS.md). This guide applies to `frontend/`.

- `src/app/`: Next.js App Router pages, layout, and global CSS.
- `src/components/`: reusable React components, including the API status button.
- `src/lib/`: shared utilities and API client functions.
- `public/`: static assets.
- `src/pages/api/[...path].ts`: server-side API proxy with authenticated client IP forwarding.
- `next.config.ts`: Next.js configuration.

## Development & Build Commands

Use Node.js 20.9+ and npm. Run these commands from `frontend/`:

- `npm ci`: install dependencies from `package-lock.json`.
- `npm run dev`: start development on port 3000.
- `npm run lint`: run ESLint.
- `npm run typecheck`: check TypeScript without emitting files.
- `npm run build`: create the production build using webpack.
- `npm start`: serve the frontend after building.
- `npm run test:auth-browser`: check authentication in Chromium against local API, Next.js and Mailpit; install the browser with `npx playwright install chromium` or set `BROWSER_EXECUTABLE`.

From the application root, `make frontend-dev` starts development.

## Coding Style & Naming

Use strict TypeScript, two-space indentation, PascalCase React component names, and kebab-case component filenames such as `api-status.tsx`. Use `@/` imports for modules under `src/` and follow App Router file conventions such as `page.tsx` and `layout.tsx`.

Follow the Next.js ESLint configuration; no separate formatter is configured. Keep browser interaction in client components and shared API functions in `src/lib/`. Preserve accessible labels, keyboard focus, and loading/error states when changing controls.

## Validation & Configuration

The authentication browser check uses Playwright; no unit test runner or coverage threshold is configured. Run lint, typecheck, and a production build for relevant changes. For API integration changes, start the backend and verify the API status button (also covered by `npm run test:auth-browser`).

Next.js reads `.env.local`. `API_URL` defaults to `http://127.0.0.1:8080` and controls proxying of `/api/*`; browser requests should use relative API paths. Set the same server-only `PROXY_SECRET` as the API to authenticate the socket-derived client IP. Never trust browser-supplied forwarded headers. Restart development after changing these settings and rebuild for production. Keep secrets out of browser-exposed variables and update `.env.example` when settings change.

<!-- BEGIN:nextjs-agent-rules -->

## This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->
