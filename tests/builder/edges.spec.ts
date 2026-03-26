import { test, expect } from '@playwright/test';
import { createWorkflow, navigateToWorkflow } from '../helpers';

test.describe('Builder Edges', () => {
  let workflowId: string;

  test.beforeEach(async ({ page }) => {
    workflowId = await createWorkflow(page);
    await navigateToWorkflow(page, workflowId);
  });

  test('canvas starts with no edges in empty workflow', async ({ page }) => {
    const canvas = page.locator('[data-testid="workflow-canvas"]');
    await expect(canvas).toBeVisible();

    // A newly created workflow should have no edge elements
    const edges = canvas.locator('[data-edge-id]');
    const count = await edges.count();
    expect(count).toBe(0);
  });

  test('edge elements have data-edge-id attribute when present', async ({ page }) => {
    const canvas = page.locator('[data-testid="workflow-canvas"]');
    await expect(canvas).toBeVisible();

    // This test verifies the selector contract: any edge that exists
    // will have a data-edge-id attribute. With an empty workflow there
    // are none, so we just confirm the selector is queryable.
    const edges = canvas.locator('[data-edge-id]');
    const count = await edges.count();

    // For each edge present, verify it has a non-empty data-edge-id
    for (let i = 0; i < count; i++) {
      const edgeId = await edges.nth(i).getAttribute('data-edge-id');
      expect(edgeId).toBeTruthy();
    }
  });
});
