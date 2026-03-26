import { test, expect } from '@playwright/test';
import { createWorkflow, navigateToWorkflow } from '../helpers';

test.describe('Builder Canvas', () => {
  let workflowId: string;

  test.beforeEach(async ({ page }) => {
    workflowId = await createWorkflow(page);
  });

  test('canvas SVG loads on builder page', async ({ page }) => {
    await navigateToWorkflow(page, workflowId);

    const canvas = page.locator('[data-testid="workflow-canvas"]');
    await expect(canvas).toBeVisible();

    // The canvas should be an SVG element
    const tagName = await canvas.evaluate((el) => el.tagName.toLowerCase());
    expect(tagName).toBe('svg');
  });

  test('canvas has grid pattern', async ({ page }) => {
    await navigateToWorkflow(page, workflowId);

    const canvas = page.locator('[data-testid="workflow-canvas"]');
    await expect(canvas).toBeVisible();

    // Grid pattern should exist within the SVG defs or as a rendered group
    const gridPattern = canvas.locator('pattern, [class*="grid"], [id*="grid"]').first();
    await expect(gridPattern).toBeAttached();
  });

  test('zoom in button works and changes zoom level', async ({ page }) => {
    await navigateToWorkflow(page, workflowId);

    const zoomLevel = page.locator('#zoom-level');
    await expect(zoomLevel).toBeVisible();

    const initialZoom = await zoomLevel.textContent();

    const zoomInBtn = page.locator('[data-testid="zoom-in-btn"]');
    await expect(zoomInBtn).toBeVisible();
    await zoomInBtn.click();

    // Zoom level text should change after clicking zoom in
    await expect(zoomLevel).not.toHaveText(initialZoom!);
  });

  test('zoom out button works and changes zoom level', async ({ page }) => {
    await navigateToWorkflow(page, workflowId);

    const zoomLevel = page.locator('#zoom-level');
    await expect(zoomLevel).toBeVisible();

    // First zoom in so we have room to zoom out
    const zoomInBtn = page.locator('[data-testid="zoom-in-btn"]');
    await zoomInBtn.click();

    const zoomAfterIn = await zoomLevel.textContent();

    const zoomOutBtn = page.locator('[data-testid="zoom-out-btn"]');
    await expect(zoomOutBtn).toBeVisible();
    await zoomOutBtn.click();

    // Zoom level should change after clicking zoom out
    await expect(zoomLevel).not.toHaveText(zoomAfterIn!);
  });

  test('zoom fit button is clickable', async ({ page }) => {
    await navigateToWorkflow(page, workflowId);

    const zoomFitBtn = page.locator('[data-testid="zoom-fit-btn"]');
    await expect(zoomFitBtn).toBeVisible();
    await expect(zoomFitBtn).toBeEnabled();
    await zoomFitBtn.click();
  });

  test('node palette is visible with categories', async ({ page }) => {
    await navigateToWorkflow(page, workflowId);

    const palette = page.locator('[data-testid="node-palette-results"]');
    await expect(palette).toBeVisible();

    // Palette should have category groupings
    const categories = palette.locator('.palette-category, [data-category]');
    const count = await categories.count();
    expect(count).toBeGreaterThan(0);
  });
});
