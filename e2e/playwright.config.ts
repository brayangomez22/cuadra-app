import { defineConfig, devices } from '@playwright/test';

// The end-to-end flows run against the docker-compose stack (`make e2e-up`:
// its own Postgres, the migrations and the API on :18080) and the production
// build of the frontend, served by `vite preview` with the /api proxy pointed
// to that API: same origin, as in production, so the refresh cookie works.
const CI = Boolean(process.env.CI);
const port = 4173;
export const apiURL = process.env.E2E_API_URL ?? 'http://localhost:18080';

export default defineConfig({
  testDir: './tests',
  globalSetup: './support/global-setup.ts',
  fullyParallel: true,
  forbidOnly: CI,
  retries: CI ? 1 : 0,
  workers: CI ? 2 : undefined,
  timeout: 60_000,
  expect: { timeout: 10_000 },
  reporter: CI ? [['github'], ['html', { open: 'never' }]] : [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: `http://localhost:${port}`,
    locale: 'es-CO',
    timezoneId: 'America/Bogota',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: `npm run build && npx vite preview --port ${port} --strictPort`,
    cwd: '../frontend',
    url: `http://localhost:${port}`,
    env: { API_PROXY_TARGET: apiURL },
    // Never reuse a server: one already on the port may proxy to another API.
    reuseExistingServer: false,
    timeout: 180_000,
  },
});
