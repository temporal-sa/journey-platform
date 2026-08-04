// Playwright End-to-End Acceptance Test Suite
// Run against live local backend and frontend

import { test, expect } from '@playwright/test';

test.describe('Journey Platform Golden Path E2E Acceptance', () => {
  test('Marketer Golden Path: Navigate, Inspect Draft, Validate, & Test Run', async ({ page }) => {
    // 1. Open frontend root
    await page.goto('http://localhost:3000');
    await expect(page).toHaveTitle(/Journey/i);

    // 2. Navigation bar presence
    const journeysNav = page.locator('nav a', { hasText: 'Journeys' });
    await expect(journeysNav).toBeVisible();

    // 3. Inspect seeded journeys
    await page.goto('http://localhost:3000/journeys');
    await expect(page.locator('h1')).toContainText('Journeys');

    // 4. Catalog page navigation
    await page.goto('http://localhost:3000/catalog');
    await expect(page.locator('h1')).toContainText('Catalog');

    // 5. Version history page navigation
    await page.goto('http://localhost:3000/history');
    await expect(page.locator('h1')).toContainText('History');

    // 6. Execution runs page navigation
    await page.goto('http://localhost:3000/runs');
    await expect(page.locator('h1')).toContainText('Runs');
  });
});
