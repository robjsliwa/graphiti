import { test, expect } from '@playwright/test';

const API_URL = 'http://localhost:8080';
const AUTH_HEADERS = {
  Authorization: 'Bearer dev-token',
  'Content-Type': 'application/json',
  Accept: 'application/json',
};

test.describe('Builder', () => {
  let workflowId: string;

  test.beforeEach(async ({ request }) => {
    const res = await request.post(`${API_URL}/api/workflows`, {
      headers: AUTH_HEADERS,
      data: { name: 'E2E Builder Test' },
    });
    const body = await res.json();
    workflowId = body.workflow.ID;
  });

  test.afterEach(async ({ request }) => {
    if (workflowId) {
      await request.delete(`${API_URL}/api/workflows/${workflowId}`, {
        headers: AUTH_HEADERS,
      });
    }
  });

  test('renders the builder with canvas', async ({ page }) => {
    await page.goto(`/workflows/${workflowId}`);
    await expect(page.locator('[data-testid="builder"]')).toBeVisible();
    await expect(page.locator('[data-testid="canvas-area"]')).toBeVisible();
    await expect(page.locator('graphiti-canvas')).toBeVisible();
  });

  test('shows node palette', async ({ page }) => {
    await page.goto(`/workflows/${workflowId}`);
    await expect(page.locator('[data-testid="node-palette"]')).toBeVisible({ timeout: 10000 });
  });

  test('shows config placeholder when no node selected', async ({ page }) => {
    await page.goto(`/workflows/${workflowId}`);
    await expect(page.locator('.config-placeholder')).toContainText('Select a node');
  });

  test('toolbar buttons are visible', async ({ page }) => {
    await page.goto(`/workflows/${workflowId}`);
    await expect(page.locator('[data-testid="undo-btn"]')).toBeVisible();
    await expect(page.locator('[data-testid="redo-btn"]')).toBeVisible();
    await expect(page.locator('[data-testid="zoom-to-fit-btn"]')).toBeVisible();
    await expect(page.locator('[data-testid="deploy-btn"]')).toBeVisible();
  });

  test('back button navigates to dashboard', async ({ page }) => {
    await page.goto(`/workflows/${workflowId}`);
    await page.locator('[data-testid="back-btn"]').click();
    await expect(page).toHaveURL('/');
  });
});
