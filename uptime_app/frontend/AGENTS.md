# Frontend Guidelines

## Scope & Project Organization

Follow the shared rules in [../AGENTS.md](../AGENTS.md). This guide applies to `frontend/`.

- `src/app/`: Next.js App Router pages, layout, and global CSS.
- `src/components/`: reusable React components, including the API status button.
- `src/lib/`: shared utilities and API client functions.
- `public/`: static assets.
- `next.config.ts`: Next.js configuration and API proxy rewrites.

## Development & Build Commands

Use Node.js 20.9+ and npm. Run these commands from `frontend/`:

- `npm ci`: install dependencies from `package-lock.json`.
- `npm run dev`: start development on port 3000.
- `npm run lint`: run ESLint.
- `npm run typecheck`: check TypeScript without emitting files.
- `npm run build`: create the production build using webpack.
- `npm start`: serve the frontend after building.

From the application root, `make frontend-dev` starts development.

## Coding Style & Naming

Use strict TypeScript, two-space indentation, PascalCase React component names, and kebab-case component filenames such as `api-status.tsx`. Use `@/` imports for modules under `src/` and follow App Router file conventions such as `page.tsx` and `layout.tsx`.

Follow the Next.js ESLint configuration; no separate formatter is configured. Keep browser interaction in client components and shared API functions in `src/lib/`. Preserve accessible labels, keyboard focus, and loading/error states when changing controls.

## Validation & Configuration

No frontend test runner or coverage threshold is configured. Run lint, typecheck, and a production build for relevant changes. For API integration changes, start the backend and manually verify the API status button.

Next.js reads `.env.local`. `API_URL` defaults to `http://127.0.0.1:8080` and controls proxying of `/api/*`; browser requests should use relative API paths. Restart development after changing this setting and rebuild for production. Keep secrets out of browser-exposed variables and update `.env.example` when settings change.
