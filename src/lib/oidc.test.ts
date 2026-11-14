import { describe, expect, it, vi } from 'vitest';
import { fetchOIDCEnabled, oidcErrorMessage } from './oidc';

describe('oidcErrorMessage', () => {
	it.each([
		['oidc_failed', /sign-in failed/i],
		['oidc_no_user', /not linked to a Doctorine user/i],
		['oidc_unavailable', /unreachable/i]
	])('maps %s to a readable message', (code, expected) => {
		expect(oidcErrorMessage(code)).toMatch(expected);
	});

	it.each([null, '', 'unknown_code', '<script>'])('ignores %s', (code) => {
		expect(oidcErrorMessage(code)).toBeNull();
	});
});

describe('fetchOIDCEnabled', () => {
	it.each([
		['enabled', () => Promise.resolve(Response.json({ enabled: true })), true],
		['disabled', () => Promise.resolve(Response.json({ enabled: false })), false],
		['backend without the route', () => Promise.resolve(new Response('', { status: 404 })), false],
		['malformed body', () => Promise.resolve(new Response('nope', { status: 200 })), false],
		['network failure', () => Promise.reject(new Error('down')), false]
	])('%s', async (_name, respond, expected) => {
		const fetchFn = vi.fn(respond);
		expect(await fetchOIDCEnabled(fetchFn as unknown as typeof fetch)).toBe(expected);
		expect(fetchFn).toHaveBeenCalledWith('/api/auth/oidc/status');
	});
});
