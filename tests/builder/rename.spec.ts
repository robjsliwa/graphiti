import { test, expect } from '@playwright/test';
import { createWorkflow, navigateToWorkflow } from '../helpers';

test.describe('Workflow Rename', () => {
  let workflowId: string;

  test.beforeEach(async ({ page }) => {
    workflowId = await createWorkflow(page);
    await navigateToWorkflow(page, workflowId);
  });

  test('workflow name is displayed in breadcrumb', async ({ page }) => {
    const workflowName = page.locator('[data-testid="workflow-name"]');
    await expect(workflowName).toBeVisible();

    const name = await workflowName.textContent();
    expect(name).toBeTruthy();
    expect(name!.trim().length).toBeGreaterThan(0);
  });

  test('clicking name enables edit mode with input', async ({ page }) => {
    const workflowName = page.locator('[data-testid="workflow-name"]');
    await expect(workflowName).toBeVisible();
    await workflowName.click();

    // An input field should appear for editing
    const nameEdit = page.locator('[data-testid="workflow-name-edit"]');
    await expect(nameEdit).toBeVisible();
  });

  test('pressing Escape cancels edit', async ({ page }) => {
    const workflowName = page.locator('[data-testid="workflow-name"]');
    await expect(workflowName).toBeVisible();

    const originalName = await workflowName.textContent();

    // Enter edit mode
    await workflowName.click();

    const nameEdit = page.locator('[data-testid="workflow-name-edit"]');
    await expect(nameEdit).toBeVisible();

    // Type something different then press Escape
    await nameEdit.fill('Changed Name');
    await nameEdit.press('Escape');

    // Edit input should be gone and name should be unchanged
    await expect(nameEdit).toBeHidden();
    await expect(workflowName).toBeVisible();
    await expect(workflowName).toHaveText(originalName!.trim());
  });
});
