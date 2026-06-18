import { expect, test } from '@playwright/test';
import { ADMIN_PASSWORD, ADMIN_USERNAME, STORAGE_STATE } from './credentials';

test('login', async ({ page }) => {
	await page.goto('/login');
	await page.getByLabel('Username').fill(ADMIN_USERNAME);
	await page.getByLabel('Password').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Login' }).click();
	await expect(page).toHaveURL('/');
	await expect(page.getByRole('heading', { name: 'Health overview' })).toBeVisible();
	await page.context().storageState({ path: STORAGE_STATE });
});
