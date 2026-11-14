import { cleanup, render, screen } from '@testing-library/svelte';
import type { ComponentProps } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('$app/forms', () => ({ enhance: () => ({ destroy: () => undefined }) }));
vi.mock('$env/dynamic/private', () => ({ env: {} }));

import LoginPage from './+page.svelte';
import { load } from './+page.server';

afterEach(cleanup);

type LoginProps = ComponentProps<typeof LoginPage>;

function renderLogin(data: { oidcEnabled: boolean; oidcError: string | null }) {
	const props: LoginProps = { data: { user: null, ...data }, form: null };
	return render(LoginPage, { props });
}

describe('login page', () => {
	it('shows the Authelia button as a full-page link when OIDC is enabled', () => {
		renderLogin({ oidcEnabled: true, oidcError: null });
		const link = screen.getByRole('link', { name: 'Sign in with Authelia' });
		expect(link.getAttribute('href')).toBe('/api/auth/oidc/login');
		expect(link.hasAttribute('data-sveltekit-reload')).toBe(true);
	});

	it('hides the Authelia button when OIDC is disabled', () => {
		renderLogin({ oidcEnabled: false, oidcError: null });
		expect(screen.queryByRole('link', { name: 'Sign in with Authelia' })).toBeNull();
		expect(screen.getByRole('button', { name: 'Login' })).toBeTruthy();
	});

	it('shows the OIDC error message', () => {
		renderLogin({ oidcEnabled: true, oidcError: 'Authelia is unreachable right now.' });
		expect(screen.getByRole('alert').textContent).toContain('Authelia is unreachable right now.');
	});
});

describe('login page load', () => {
	async function runLoad(search: string, status: Response | Error) {
		const fetchFn = vi.fn(() =>
			status instanceof Error ? Promise.reject(status) : Promise.resolve(status)
		);
		const result = await load({
			fetch: fetchFn,
			url: new URL(`http://frontend.test/login${search}`)
		} as unknown as Parameters<typeof load>[0]);
		return result as { oidcEnabled: boolean; oidcError: string | null };
	}

	it.each([
		['?error=oidc_no_user', /not linked to a Doctorine user/i],
		['?error=oidc_failed', /sign-in failed/i],
		['?error=oidc_unavailable', /unreachable/i]
	])('maps %s to a message', async (search, expected) => {
		const result = await runLoad(search, Response.json({ enabled: true }));
		expect(result.oidcEnabled).toBe(true);
		expect(result.oidcError).toMatch(expected);
	});

	it('reports OIDC disabled and no error by default', async () => {
		const result = await runLoad('', Response.json({ enabled: false }));
		expect(result).toEqual({ oidcEnabled: false, oidcError: null });
	});

	it('treats an unreachable backend as OIDC disabled', async () => {
		const result = await runLoad('', new Error('down'));
		expect(result.oidcEnabled).toBe(false);
	});
});
