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

function oidcEvent(path: string, upstream: Response, cookieHeader?: string) {
	const headers = new Headers();
	if (cookieHeader) headers.set('cookie', cookieHeader);
	const request = new Request(`http://frontend.test/api/${path}`, { method: 'GET', headers });
	return {
		request,
		params: { path },
		cookies: { get: () => undefined },
		url: { search: '?code=abc&state=xyz' },
		fetch: vi.fn().mockResolvedValue(upstream)
	};
}

describe('api proxy redirect and cookie passthrough', () => {
	it('does not follow redirects and forwards the inbound cookie header', async () => {
		const e = oidcEvent(
			'auth/oidc/callback',
			new Response(null, { status: 302, headers: { location: '/' } }),
			'doctorine_oidc_state=signed.value; other=1'
		);
		await GET(e as unknown as Parameters<typeof GET>[0]);
		const [target, init] = e.fetch.mock.calls[0] as [string, RequestInit];
		expect(target).toBe('http://backend.test/api/auth/oidc/callback?code=abc&state=xyz');
		expect(init.redirect).toBe('manual');
		expect(new Headers(init.headers).get('cookie')).toBe(
			'doctorine_oidc_state=signed.value; other=1'
		);
	});

	it('passes a 302 with its Location and every Set-Cookie through unchanged', async () => {
		const upstreamHeaders = new Headers({ location: '/' });
		const session = 'doctorine_token=sess; Path=/; HttpOnly; Secure; SameSite=Lax';
		const clearState =
			'doctorine_oidc_state=; Path=/api/auth/oidc; Max-Age=0; HttpOnly; Secure; SameSite=Lax';
		upstreamHeaders.append('set-cookie', clearState);
		upstreamHeaders.append('set-cookie', session);
		const e = oidcEvent(
			'auth/oidc/callback',
			new Response(null, { status: 302, headers: upstreamHeaders })
		);

		const res = await GET(e as unknown as Parameters<typeof GET>[0]);

		expect(res.status).toBe(302);
		expect(res.headers.get('location')).toBe('/');
		expect(res.headers.getSetCookie()).toEqual([clearState, session]);
	});

	it('passes an absolute provider redirect through for the login start', async () => {
		const authorize = 'https://auth.example.test/api/oidc/authorization?client_id=doctorine';
		const upstreamHeaders = new Headers({ location: authorize });
		upstreamHeaders.append(
			'set-cookie',
			'doctorine_oidc_state=v.s; Path=/api/auth/oidc; Max-Age=600; HttpOnly; SameSite=Lax'
		);
		const e = oidcEvent(
			'auth/oidc/login',
			new Response(null, { status: 302, headers: upstreamHeaders })
		);

		const res = await GET(e as unknown as Parameters<typeof GET>[0]);

		expect(res.status).toBe(302);
		expect(res.headers.get('location')).toBe(authorize);
		expect(res.headers.getSetCookie()).toHaveLength(1);
	});
});
