import { test, expect } from '@playwright/test';
import { createWorkflow, navigateToWorkflow } from '../helpers';

test.describe('Deploy Controls', () => {
  let workflowId: string;

  test.beforeEach(async ({ page }) => {
    workflowId = await createWorkflow(page);
    await navigateToWorkflow(page, workflowId);
  });

  test('validate button is present and clickable', async ({ page }) => {
    const validateBtn = page.locator('[data-testid="validate-btn"]');
    await expect(validateBtn).toBeVisible();
    await expect(validateBtn).toBeEnabled();
  });

  test('deploy button is present', async ({ page }) => {
    const deployBtn = page.locator('[data-testid="deploy-btn"]');
    await expect(deployBtn).toBeVisible();
  });

  test('deploy menu opens on chevron click', async ({ page }) => {
    const deployMenu = page.locator('[data-testid="deploy-menu"]');

    // Menu should not be visible initially
    await expect(deployMenu).toBeHidden();

    // Click the chevron button (not the deploy button) to open the menu
    const chevron = page.locator('.deploy-chevron');
    await chevron.click();

    // Menu should become visible
    await expect(deployMenu).toBeVisible();
  });

  test('deploy menu has expected options', async ({ page }) => {
    // Open the deploy menu via chevron
    const chevron = page.locator('.deploy-chevron');
    await chevron.click();

    const deployMenu = page.locator('[data-testid="deploy-menu"]');
    await expect(deployMenu).toBeVisible();

    // Check for expected menu options
    await expect(deployMenu.getByText('Deploy to Production')).toBeVisible();
    await expect(deployMenu.getByText('Deploy to Staging')).toBeVisible();
    await expect(deployMenu.getByText('Save as Draft')).toBeVisible();
    await expect(deployMenu.getByText('Export as YAML')).toBeVisible();
    await expect(deployMenu.getByText('Export as JSON')).toBeVisible();
  });
});
