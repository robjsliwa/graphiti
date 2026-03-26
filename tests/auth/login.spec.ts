import { test, expect } from '@playwright/test';

test.describe('Login', () => {
  test('login page auto-redirects to dashboard in fake auth mode', async ({ page }) => {
    // In fake auth mode, visiting /auth/login auto-authenticates and redirects
    await page.goto('/auth/login');

    // Should end up on the dashboard after the auto-login redirect chain
    await expect(page.locator('h1')).toContainText('Workflows');
  });

  test('dashboard is accessible after login', async ({ page }) => {
    // storageState from auth.setup.ts means we are already authenticated
    await page.goto('/');

    await expect(page.locator('h1')).toContainText('Workflows');
    await expect(page.locator('[data-testid="create-workflow-form"]')).toBeVisible();
  });

  test('session persists across page refreshes', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('Workflows');

    // Reload the page — session cookie should keep us authenticated
    await page.reload();
    await expect(page.locator('h1')).toContainText('Workflows');

    // Reload again to be thorough
    await page.reload();
    await expect(page.locator('h1')).toContainText('Workflows');
  });
});
