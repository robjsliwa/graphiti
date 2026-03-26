import { test, expect } from '@playwright/test';

test.describe('Application Health', () => {
  test('server is running and dashboard loads', async ({ page }) => {
    // Fake auth auto-authenticates via redirect chain:
    // GET / -> 302 /auth/login -> auto-login -> 302 / -> dashboard
    await page.goto('/');

    // After fake auth redirect chain, we should see the dashboard
    await expect(page.locator('h1')).toContainText('Workflows');
  });

  test('dashboard has create workflow form', async ({ page }) => {
    await page.goto('/');

    await expect(page.locator('[data-testid="create-workflow-form"]')).toBeVisible();
    await expect(page.locator('[data-testid="create-workflow-btn"]')).toBeVisible();
  });

  test('theme toggle is present', async ({ page }) => {
    await page.goto('/');

    await expect(page.locator('[data-testid="theme-toggle"]')).toBeVisible();
  });
});
