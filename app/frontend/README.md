# YTSoar frontend

React app built with [Vite](https://vite.dev) and [React Router](https://reactrouter.com).

## Run

```bash
npm install
npm run dev
```

The dev server listens on port 3000. Open the app through nginx at http://localhost:9999, so the API paths (`/playbook-api`, `/auth-api`, `/ws`) work.

## Scripts

- `npm run dev`: start the dev server.
- `npm run build`: type-check and build to `dist/`.
- `npm run preview`: serve the build.
- `npm run lint`: run ESLint.

## Layout

- `src/main.tsx`: the entry point and every route.
- `src/pages`: one folder for each area. Page-only parts go in its `components/` folder.
- `src/components`, `src/hooks`, `src/services`: code that pages share.

## Settings

Copy `env.sample` to `.env` to reach the API without nginx. Each `VITE_*` value overrides one API base URL in `src/settings/urls.ts`.
