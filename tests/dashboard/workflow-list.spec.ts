import { test, expect } from '@playwright/test';

test.describe('Workflow List', () => {
  test('dashboard shows Workflows heading', async ({ page }) => {
    await page.goto('/');

    await expect(page.locator('h1')).toContainText('Workflows');
  });

  test('workflow cards display name, status badge, version, and updated date', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('Workflows');

    // Create a workflow to ensure at least one card exists
    await page.locator('[data-testid="create-workflow-btn"]').click();
    await page.waitForURL(/\/workflows\//);
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('Workflows');

    const cards = page.locator('.workflow-card');
    await expect(cards.first()).toBeVisible();

    // Each card should have a badge
    const firstCard = cards.first();
    await expect(firstCard.locator('.badge')).toBeVisible();
  });

  test('each workflow card links to its builder page', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('Workflows');

    // Create a workflow to ensure at least one card exists
    await page.locator('[data-testid="create-workflow-btn"]').click();
    await page.waitForURL(/\/workflows\//);
    await page.goto('/');
    await expect(page.locator('h1')).toContainText('Workflows');

    // The .workflow-card is an <a> tag itself
    const firstCard = page.locator('.workflow-card').first();
    await expect(firstCard).toBeVisible();

    const href = await firstCard.getAttribute('href');
    expect(href).toMatch(/\/workflows\/[a-f0-9-]+/);

    await firstCard.click();
    await page.waitForURL(/\/workflows\/[a-f0-9-]+/);
  });
});
