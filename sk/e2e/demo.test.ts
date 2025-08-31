import { expect, test } from '@playwright/test';

test('home page has expected h1', async ({ page }) => {
	await page.goto('/');
	// Check for the main hero heading
	await expect(page.getByRole('heading', { name: 'Turn rambling into compelling' })).toBeVisible();
	// Check that other key sections are present
	await expect(page.getByRole('heading', { name: 'Watch it in action' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Simple workflow' })).toBeVisible();
});
