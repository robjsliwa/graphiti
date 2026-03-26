import { test, expect } from '@playwright/test';
import { createWorkflow, navigateToWorkflow } from '../helpers';

test.describe('Execution Mode', () => {
  let workflowId: string;

  test.beforeEach(async ({ page }) => {
    workflowId = await createWorkflow(page);
    await navigateToWorkflow(page, workflowId);
  });

  test('builder mode tab is active by default', async ({ page }) => {
    const builderTab = page.locator('[data-testid="mode-tab-builder"]');
    await expect(builderTab).toBeVisible();

    // Builder tab should have an active state
    const isActive = await builderTab.evaluate((el) => {
      return el.classList.contains('active') || el.getAttribute('aria-selected') === 'true';
    });
    expect(isActive).toBe(true);
  });

  test('clicking Execution tab switches mode', async ({ page }) => {
    const executionTab = page.locator('[data-testid="mode-tab-execution"]');
    await expect(executionTab).toBeVisible();
    await executionTab.click();

    // The execution panel should become visible (check by id and data-testid)
    const executionPanel = page.locator('#panel-left-execution');
    await expect(executionPanel).toBeVisible();

    // Verify the data-testid attribute exists on the element
    await expect(executionPanel).toHaveAttribute('data-testid', 'panel-left-execution');
  });

  test('clicking Builder tab switches back', async ({ page }) => {
    // Switch to execution mode first
    const executionTab = page.locator('[data-testid="mode-tab-execution"]');
    await executionTab.click();

    // Switch back to builder mode
    const builderTab = page.locator('[data-testid="mode-tab-builder"]');
    await builderTab.click();

    // The builder tab should be active again
    const isActive = await builderTab.evaluate((el) => {
      return el.classList.contains('active') || el.getAttribute('aria-selected') === 'true';
    });
    expect(isActive).toBe(true);
  });

  test('execution run list container exists', async ({ page }) => {
    // Switch to execution mode to see the run list
    const executionTab = page.locator('[data-testid="mode-tab-execution"]');
    await executionTab.click();

    const runList = page.locator('[data-testid="execution-run-list"]');
    await expect(runList).toBeAttached();
  });
});
