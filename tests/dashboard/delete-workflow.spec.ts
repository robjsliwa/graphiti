import { test, expect } from '@playwright/test';

test.describe('Delete Workflow', () => {
  test('delete button exists on workflow cards', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('Workflows');

    // Create a workflow to ensure there's a card with a delete button
    await page.locator('[data-testid="create-workflow-btn"]').click();

    // Wait for the redirect to builder page to complete
    await page.waitForURL(/\/workflows\//);

    // Navigate back to dashboard
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('Workflows');

    // At least one workflow card should have a delete button
    const deleteBtn = page.locator('[data-testid="delete-workflow-btn"]').first();
    await expect(deleteBtn).toBeVisible();
  });

  test('delete workflow removes card after confirming dialog', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('Workflows');

    // Create a workflow to delete
    await page.locator('[data-testid="create-workflow-btn"]').click();
    await page.waitForURL(/\/workflows\//);

    // Go back to dashboard
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('Workflows');

    const cardsBefore = await page.locator('.workflow-card').count();
    expect(cardsBefore).toBeGreaterThan(0);

    // Handle the hx-confirm browser dialog by accepting it
    page.on('dialog', async (dialog) => {
      await dialog.accept();
    });

    // Click delete on the first workflow card
    const deleteBtn = page.locator('[data-testid="delete-workflow-btn"]').first();
    await expect(deleteBtn).toBeVisible();

    const deleteResponse = page.waitForResponse(
      (resp) => resp.url().includes('/workflows') && resp.request().method() === 'DELETE'
    );
    await deleteBtn.click();
    await deleteResponse;

    // Wait for the card count to decrease (HTMX swaps the DOM after delete)
    await expect(page.locator('.workflow-card')).not.toHaveCount(cardsBefore);

    // After the delete, the page reloads — verify we're still on dashboard
    await expect(page.locator('h1')).toContainText('Workflows');
  });
});
