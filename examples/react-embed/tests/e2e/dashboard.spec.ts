import { test, expect } from '@playwright/test';

const API_URL = 'http://localhost:8080';
const AUTH_HEADERS = {
  Authorization: 'Bearer dev-token',
  'Content-Type': 'application/json',
  Accept: 'application/json',
};

test.describe('Dashboard', () => {
  test('displays the dashboard page', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('[data-testid="dashboard"]')).toBeVisible();
    await expect(page.locator('h1')).toContainText('Graphiti Workflows');
  });

  test('shows create workflow button', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('[data-testid="create-workflow-btn"]')).toBeVisible();
  });

  test('create workflow and navigate to builder', async ({ page }) => {
    await page.goto('/');

    const createBtn = page.locator('[data-testid="create-workflow-btn"]');
    await createBtn.click();

    // Should navigate to builder
    await expect(page).toHaveURL(/\/workflows\/.+/);
    await expect(page.locator('[data-testid="builder"]')).toBeVisible();
  });

  test('delete a workflow', async ({ page, request }) => {
    // Create via API
    const res = await request.post(`${API_URL}/api/workflows`, {
      headers: AUTH_HEADERS,
      data: { name: 'E2E Delete Test' },
    });
    const { workflow } = await res.json();

    await page.goto('/');

    // Wait for workflow to appear
    const deleteBtn = page.locator(`[data-testid="delete-workflow-${workflow.ID}"]`);
    await expect(deleteBtn).toBeVisible({ timeout: 5000 });
    await deleteBtn.click();

    // Should be removed from the list
    await expect(page.locator(`[data-testid="workflow-${workflow.ID}"]`)).not.toBeVisible({
      timeout: 5000,
    });
  });
});
