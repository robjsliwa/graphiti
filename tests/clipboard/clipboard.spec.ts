import { test, expect } from '@playwright/test';
import { createWorkflow, navigateToWorkflow } from '../helpers';

const modifier = process.platform === 'darwin' ? 'Meta' : 'Control';

test.describe('Clipboard Operations', () => {
  let workflowId: string;

  test.beforeEach(async ({ page }) => {
    workflowId = await createWorkflow(page);
    await navigateToWorkflow(page, workflowId);
  });

  test('copy with no selected nodes is a no-op', async ({ page }) => {
    // Focus the canvas
    await page.locator('[data-testid="workflow-canvas"]').click();

    let commandCalled = false;
    page.on('response', (resp) => {
      if (resp.url().includes('/clipboard') && resp.request().method() === 'POST') {
        commandCalled = true;
      }
    });

    // Copy with nothing selected should not trigger a clipboard command
    await page.keyboard.press(`${modifier}+c`);

    // Perform a round-trip action to flush any pending requests
    await page.evaluate(() => new Promise((r) => setTimeout(r, 300)));
    expect(commandCalled).toBe(false);
  });

  test('paste with empty clipboard is a no-op', async ({ page }) => {
    await page.locator('[data-testid="workflow-canvas"]').click();

    let commandCalled = false;
    page.on('response', (resp) => {
      if (resp.url().includes('/clipboard') && resp.request().method() === 'POST') {
        commandCalled = true;
      }
    });

    // Paste with nothing in clipboard should not send a command
    await page.keyboard.press(`${modifier}+v`);

    // Perform a round-trip action to flush any pending requests
    await page.evaluate(() => new Promise((r) => setTimeout(r, 300)));
    expect(commandCalled).toBe(false);
  });

  test('commands API endpoint exists', async ({ page }) => {
    // Verify the commands endpoint exists by POSTing to it
    const response = await page.request.post(
      `/api/workflows/${workflowId}/commands`,
      { data: {} }
    );

    // Should not be 404 (may be 400 for invalid command body)
    expect(response.status()).not.toBe(404);
  });
});
