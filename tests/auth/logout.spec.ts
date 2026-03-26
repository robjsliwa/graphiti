import { test, expect } from '@playwright/test';

test.describe('Logout', () => {
  test('logout button is visible on dashboard', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('Workflows');

    await expect(page.locator('[data-testid="logout-btn"]')).toBeVisible();
  });

  test('clicking logout triggers form submission', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('Workflows');

    const logoutBtn = page.locator('[data-testid="logout-btn"]');
    await expect(logoutBtn).toBeVisible();

    // Clicking logout submits the form — in fake auth mode, the redirect chain
    // ends back at the dashboard. Verify the POST happens.
    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/auth/logout') && resp.status() < 400
    );
    await logoutBtn.click();
    const response = await responsePromise;

    // Logout should respond with a redirect (302)
    expect(response.status()).toBe(302);
  });
});
