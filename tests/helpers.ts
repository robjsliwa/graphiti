import { Page } from '@playwright/test';

export async function createWorkflow(page: Page): Promise<string> {
  await page.goto('/');
  const responsePromise = page.waitForResponse(
    (resp) => resp.url().includes('/workflows') && resp.request().method() === 'POST'
  );
  await page.locator('[data-testid="create-workflow-btn"]').click();
  const response = await responsePromise;
  // After creating, we get redirected to /workflows/{id}
  await page.waitForURL(/\/workflows\//);
  const url = page.url();
  const id = url.split('/workflows/')[1];
  return id;
}

export async function navigateToWorkflow(page: Page, workflowId: string) {
  await page.goto(`/workflows/${workflowId}`);
  await page.locator('[data-testid="workflow-canvas"]').waitFor();
}
