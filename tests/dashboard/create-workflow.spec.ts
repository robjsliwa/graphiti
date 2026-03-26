import { test, expect } from '@playwright/test';

test.describe('Create Workflow', () => {
  test('create workflow form is visible', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('Workflows');

    await expect(page.locator('[data-testid="create-workflow-form"]')).toBeVisible();
    await expect(page.locator('[data-testid="create-workflow-btn"]')).toBeVisible();
  });

  test('clicking New Workflow creates a workflow and navigates to builder', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('Workflows');

    // POST /workflows creates the workflow and redirects to the builder
    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/workflows') && resp.request().method() === 'POST'
    );
    await page.locator('[data-testid="create-workflow-btn"]').click();
    const response = await responsePromise;

    // The POST should succeed (redirect status or 200)
    expect(response.status()).toBeLessThan(400);

    // After redirect, should be on the builder page /workflows/{uuid}
    await page.waitForURL(/\/workflows\/[a-f0-9-]+/);
    expect(page.url()).toMatch(/\/workflows\/[a-f0-9-]+$/);
  });
});
