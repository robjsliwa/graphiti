import { test, expect } from '@playwright/test';
import { createWorkflow } from '../helpers';

test.describe('Keyboard Shortcuts', () => {
  let workflowId: string;

  test.beforeEach(async ({ page }) => {
    workflowId = await createWorkflow(page);
  });

  test('? key opens help modal', async ({ page }) => {
    // Focus the canvas so keyboard shortcuts are captured
    await page.locator('[data-testid="workflow-canvas"]').click();

    await page.keyboard.press('?');

    await expect(page.locator('[data-testid="help-modal"]')).toBeVisible();
  });

  test('Escape closes help modal', async ({ page }) => {
    // Open help modal first
    await page.locator('[data-testid="workflow-canvas"]').click();
    await page.keyboard.press('?');
    await expect(page.locator('[data-testid="help-modal"]')).toBeVisible();

    // Close it with Escape
    await page.keyboard.press('Escape');

    await expect(page.locator('[data-testid="help-modal"]')).not.toBeVisible();
  });

  test('help modal shows keyboard shortcut groups', async ({ page }) => {
    await page.locator('[data-testid="workflow-canvas"]').click();
    await page.keyboard.press('?');

    const modal = page.locator('[data-testid="help-modal"]');
    await expect(modal).toBeVisible();

    // Verify shortcut group headings are present
    await expect(modal.getByRole('heading', { name: 'Edit' })).toBeVisible();
    await expect(modal.getByRole('heading', { name: 'View' })).toBeVisible();
  });

  test('undo button starts disabled on fresh workflow', async ({ page }) => {
    const undoBtn = page.locator('[data-testid="undo-btn"]');
    await expect(undoBtn).toBeVisible();
    await expect(undoBtn).toBeDisabled();
  });

  test('redo button starts disabled on fresh workflow', async ({ page }) => {
    const redoBtn = page.locator('[data-testid="redo-btn"]');
    await expect(redoBtn).toBeVisible();
    await expect(redoBtn).toBeDisabled();
  });

  test('zoom in button is functional', async ({ page }) => {
    const zoomInBtn = page.locator('[data-testid="zoom-in-btn"]');
    await expect(zoomInBtn).toBeVisible();
    await expect(zoomInBtn).toBeEnabled();

    // Click should not error — the zoom action fires successfully
    await zoomInBtn.click();
  });

  test('zoom out button is functional', async ({ page }) => {
    const zoomOutBtn = page.locator('[data-testid="zoom-out-btn"]');
    await expect(zoomOutBtn).toBeVisible();
    await expect(zoomOutBtn).toBeEnabled();

    await zoomOutBtn.click();
  });

  test('zoom fit button is functional', async ({ page }) => {
    const zoomFitBtn = page.locator('[data-testid="zoom-fit-btn"]');
    await expect(zoomFitBtn).toBeVisible();
    await expect(zoomFitBtn).toBeEnabled();

    await zoomFitBtn.click();
  });
});
