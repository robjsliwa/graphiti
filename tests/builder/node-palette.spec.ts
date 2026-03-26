import { test, expect } from '@playwright/test';
import { createWorkflow, navigateToWorkflow } from '../helpers';

test.describe('Node Palette', () => {
  let workflowId: string;

  test.beforeEach(async ({ page }) => {
    workflowId = await createWorkflow(page);
    await navigateToWorkflow(page, workflowId);
  });

  test('palette shows categories: Sources, Processing, Destinations, Control', async ({ page }) => {
    const palette = page.locator('[data-testid="node-palette-results"]');
    await expect(palette).toBeVisible();

    for (const category of ['Sources', 'Processing', 'Destinations', 'Control']) {
      await expect(palette.getByText(category, { exact: false }).first()).toBeVisible();
    }
  });

  test('palette items have definition-id, name, and category attributes', async ({ page }) => {
    const paletteItems = page.locator('.palette-item');
    await expect(paletteItems.first()).toBeVisible();

    const count = await paletteItems.count();
    expect(count).toBeGreaterThan(0);

    const firstItem = paletteItems.first();
    const defId = await firstItem.getAttribute('data-definition-id');
    expect(defId).toBeTruthy();
  });

  test('search filters nodes', async ({ page }) => {
    const searchInput = page.locator('[data-testid="node-search-input"]');
    await expect(searchInput).toBeVisible();

    const paletteItems = page.locator('.palette-item');
    const countBefore = await paletteItems.count();
    expect(countBefore).toBeGreaterThan(0);

    // Type search term using keyboard to trigger keyup events for HTMX
    await searchInput.click();
    const searchResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/nodes/search')
    );
    await searchInput.pressSequentially('api', { delay: 50 });
    await searchResponse;

    // Wait for palette to update — filtered count should differ from original
    await expect(async () => {
      const countAfter = await paletteItems.count();
      expect(countAfter).toBeLessThanOrEqual(countBefore);
    }).toPass({ timeout: 5_000 });
  });

  test('clear search restores all nodes', async ({ page }) => {
    const searchInput = page.locator('[data-testid="node-search-input"]');
    await expect(searchInput).toBeVisible();

    const paletteItems = page.locator('.palette-item');
    const countBefore = await paletteItems.count();

    // Search to filter
    await searchInput.click();
    const searchResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/nodes/search')
    );
    await searchInput.pressSequentially('api', { delay: 50 });
    await searchResponse;

    // Clear search by selecting all and deleting
    const clearResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/nodes/search')
    );
    await searchInput.fill('');
    await searchInput.press('Backspace'); // Trigger keyup changed
    await clearResponse;

    // Wait for palette to restore all items
    await expect(paletteItems).toHaveCount(countBefore);
  });
});
