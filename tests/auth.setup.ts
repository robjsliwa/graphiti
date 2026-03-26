import { test as setup, expect } from '@playwright/test';
import { AUTH_FILE } from './constants';

setup('authenticate', async ({ page }) => {
  // Navigate to the app — fake auth auto-authenticates via redirect chain
  await page.goto('/');

  // Wait for the dashboard to load (confirms auth succeeded)
  await expect(page.locator('h1')).toContainText('Workflows');

  // Save the authenticated browser state (cookies + localStorage)
  await page.context().storageState({ path: AUTH_FILE });
});
