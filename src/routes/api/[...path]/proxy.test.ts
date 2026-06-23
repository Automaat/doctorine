import { describe, expect, it, vi } from 'vitest';

vi.mock('$env/dynamic/private', () => ({ env: { API_PROXY_TARGET: 'http://backend.test' } }));

import { GET } from './+server';

function event(opts: { cookie?: string; authHeader?: string }) {
	const headers = new Headers();
	if (opts.authHeader) headers.set('authorization', opts.authHeader);
	const request = new Request('http://frontend.test/api/overview', { method: 'GET', headers });
	const fetchFn = vi.fn().mockResolvedValue(new Response('{}', { status: 200 }));
	return {
		request,
		params: { path: 'overview' },
		cookies: { get: () => opts.cookie },
		url: { search: '' },
		fetch: fetchFn
	};
}

function forwardedAuth(fetchFn: ReturnType<typeof vi.fn>): string | null {
	const init = fetchFn.mock.calls[0][1] as RequestInit;
	return new Headers(init.headers).get('authorization');
}

describe('api proxy authorization forwarding', () => {
	it('forwards the session cookie as a bearer token', async () => {
		const e = event({ cookie: 'sess-token', authHeader: 'Bearer dpat_inbound' });
		await GET(e as unknown as Parameters<typeof GET>[0]);
		expect(forwardedAuth(e.fetch)).toBe('Bearer sess-token');
	});

	it('forwards an inbound bearer token when there is no cookie', async () => {
		const e = event({ authHeader: 'Bearer dpat_inbound' });
		await GET(e as unknown as Parameters<typeof GET>[0]);
		expect(forwardedAuth(e.fetch)).toBe('Bearer dpat_inbound');
	});

	it('sends no authorization when neither is present', async () => {
		const e = event({});
		await GET(e as unknown as Parameters<typeof GET>[0]);
		expect(forwardedAuth(e.fetch)).toBeNull();
	});
});
