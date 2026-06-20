import { expect, test } from '@playwright/test';

test('dashboard and primary pages load', async ({ page }) => {
	await page.goto('/');
	await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();

	await page.getByRole('link', { name: 'Documents' }).first().click();
	await expect(page.getByRole('heading', { name: 'Documents' })).toBeVisible();

	await page.getByRole('link', { name: 'Exams' }).first().click();
	await expect(page.getByRole('heading', { name: 'Examinations' })).toBeVisible();

	await page.getByRole('link', { name: 'Supplements' }).first().click();
	await expect(page.getByRole('heading', { name: 'Supplements' })).toBeVisible();

	await page.getByRole('link', { name: 'Illnesses' }).first().click();
	await expect(page.getByRole('heading', { name: 'Illnesses' })).toBeVisible();
});
