import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30000,
  retries: 0,
  use: {
    baseURL: 'http://localhost:5174',
    headless: true,
    screenshot: 'only-on-failure',
  },
  webServer: [
    {
      command: 'cd ../.. && task run',
      port: 8080,
      reuseExistingServer: true,
      timeout: 30000,
    },
    {
      command: 'npm run dev',
      port: 5174,
      reuseExistingServer: true,
      timeout: 15000,
    },
  ],
  workers: 1,
});
