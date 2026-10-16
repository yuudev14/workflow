import path from "node:path";
import { defineConfig, devices } from "@playwright/test";
import { adminState } from "./e2e/helpers";

// The browser tests start the whole app: the api, the worker, the sandbox and
// this frontend. Postgres and Keycloak must already run. The ports are not the
// dev ports, so the tests can run next to the dev stack.
const env = process.env;
const backendDir = path.resolve(import.meta.dirname, "../ytsoar");
const db = {
  DB_HOST: env.DB_HOST ?? "localhost",
  DB_PORT: env.DB_PORT ?? "5432",
  DB_USER: env.DB_USER ?? "postgres",
  DB_PASSWORD: env.DB_PASSWORD ?? "postgres",
  DB_NAME: env.DB_NAME ?? "ytsoar_e2e",
};
const dbUrl = `postgres://${db.DB_USER}:${db.DB_PASSWORD}@${db.DB_HOST}:${db.DB_PORT}/${db.DB_NAME}?sslmode=disable`;
const backend = {
  ...env,
  ...db,
  HTTP_ADDR: ":8090",
  JWT_SECRET: "e2e-secret",
  ADMIN_USERNAME: "admin",
  ADMIN_PASSWORD: "admin123!",
  CORS_ORIGINS: "http://localhost:3000",
  FRONTEND_URL: "http://localhost:3000",
  AUTH_PUBLIC_URL: "http://localhost:3000/auth-api",
  CONNECTORS_DIR: path.resolve(import.meta.dirname, "../connectors"),
  SANDBOX_ADDR: "localhost:50062",
  SANDBOX_LISTEN_ADDR: ":50062",
  // Less log output.
  APP_ENV: "production",
  GIN_MODE: "release",
};

export default defineConfig({
  testDir: "e2e",
  // The tests share one database, so they run one at a time.
  workers: 1,
  retries: env.CI ? 1 : 0,
  reporter: env.CI ? [["github"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: "http://localhost:3000",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    // Signs in as the admin one time. The other tests start signed in.
    { name: "setup", testMatch: /auth\.setup\.ts/ },
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"], storageState: adminState },
      dependencies: ["setup"],
    },
  ],
  webServer: [
    {
      command: `go run github.com/pressly/goose/v3/cmd/goose@v3.29.0 -dir db/migrations postgres "${dbUrl}" up && go run ./cmd/api`,
      cwd: backendDir,
      env: backend,
      url: "http://localhost:8090/api/auth/v1/providers",
      timeout: 300_000,
      reuseExistingServer: !env.CI,
    },
    {
      command: "go run ./cmd/sandbox",
      cwd: backendDir,
      env: backend,
      port: 50062,
      timeout: 300_000,
      reuseExistingServer: !env.CI,
    },
    {
      command: "go run ./cmd/worker",
      cwd: backendDir,
      env: backend,
    },
    {
      command: "npm run dev",
      env: { ...env, API_URL: "http://localhost:8090" },
      url: "http://localhost:3000",
      reuseExistingServer: !env.CI,
    },
  ],
});
