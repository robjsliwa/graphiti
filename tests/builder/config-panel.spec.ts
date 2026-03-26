import { test, expect } from '@playwright/test';
import { createWorkflow, navigateToWorkflow } from '../helpers';

test.describe('Config Panel', () => {
  let workflowId: string;

  test.beforeEach(async ({ page }) => {
    workflowId = await createWorkflow(page);
    await navigateToWorkflow(page, workflowId);
  });

  test('config panel shows "Select a node to configure" initially', async ({ page }) => {
    const configPanel = page.locator('[data-testid="config-panel"]');
    await expect(configPanel).toBeVisible();

    // With no node selected, the config panel should show a placeholder message
    await expect(configPanel.getByText('Select a node', { exact: false })).toBeVisible();
  });

  test('config panel has Configuration heading', async ({ page }) => {
    const configPanel = page.locator('[data-testid="config-panel"]');
    await expect(configPanel).toBeVisible();

    await expect(configPanel.getByText('Configuration', { exact: false })).toBeVisible();
  });
});
