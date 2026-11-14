import { describe, expect, it, vi } from 'vitest';

vi.mock('$env/dynamic/public', () => ({ env: {} }));
vi.mock('$env/dynamic/private', () => ({ env: {} }));
vi.mock('@sveltejs/kit', () => ({
	redirect: (status: number, location: string) => {
		throw new Error(`redirect ${status} ${location}`);
	}
}));

import { handle } from './hooks.server';

function apiEvent(opts: { cookie?: string; authHeader?: string }) {
	const headers = new Headers();
	if (opts.authHeader) headers.set('authorization', opts.authHeader);
	return {
		url: new URL('http://frontend.test/api/overview'),
		cookies: { get: () => opts.cookie, delete: vi.fn() },
		request: new Request('http://frontend.test/api/overview', { headers }),
		locals: {} as App.Locals
	};
}

describe('handle api auth gate', () => {
	it('passes a bearer-authenticated api request through to the proxy', async () => {
		const event = apiEvent({ authHeader: 'Bearer dpat_x' });
		const resolve = vi.fn().mockResolvedValue(new Response('proxied'));
		const res = await handle({ event, resolve } as unknown as Parameters<typeof handle>[0]);
		expect(resolve).toHaveBeenCalledOnce();
		expect(await res.text()).toBe('proxied');
	});

	it('rejects an unauthenticated api request with 401', async () => {
		const event = apiEvent({});
		const resolve = vi.fn();
		const res = await handle({ event, resolve } as unknown as Parameters<typeof handle>[0]);
		expect(resolve).not.toHaveBeenCalled();
		expect(res.status).toBe(401);
	});

	it('rejects a non-bearer authorization header with 401', async () => {
		const event = apiEvent({ authHeader: 'Basic Zm9vOmJhcg==' });
		const resolve = vi.fn();
		const res = await handle({ event, resolve } as unknown as Parameters<typeof handle>[0]);
		expect(resolve).not.toHaveBeenCalled();
		expect(res.status).toBe(401);
	});
});

describe('handle oidc endpoints', () => {
	function oidcEvent(path: string, cookie?: string) {
		const url = new URL(`http://frontend.test${path}`);
		return {
			url,
			cookies: { get: () => cookie, delete: vi.fn() },
			request: new Request(url),
			locals: {} as App.Locals
		};
	}

	it.each(['/api/auth/oidc/status', '/api/auth/oidc/login', '/api/auth/oidc/callback'])(
		'lets %s through without a session',
		async (path) => {
			const event = oidcEvent(path);
			const resolve = vi.fn().mockResolvedValue(new Response(null, { status: 302 }));
			const res = await handle({ event, resolve } as unknown as Parameters<typeof handle>[0]);
			expect(resolve).toHaveBeenCalledOnce();
			expect(res.status).toBe(302);
		}
	);

	it('keeps a stale session cookie on the callback so the new session is not deleted', async () => {
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 401 })));
		try {
			const event = oidcEvent('/api/auth/oidc/callback', 'stale-token');
			const resolve = vi.fn().mockResolvedValue(new Response(null, { status: 302 }));
			await handle({ event, resolve } as unknown as Parameters<typeof handle>[0]);
			expect(resolve).toHaveBeenCalledOnce();
			expect(event.cookies.delete).not.toHaveBeenCalled();

			const other = oidcEvent('/api/auth/oidc/status', 'stale-token');
			await handle({ event: other, resolve } as unknown as Parameters<typeof handle>[0]);
			expect(other.cookies.delete).toHaveBeenCalledWith('doctorine_token', { path: '/' });
		} finally {
			vi.unstubAllGlobals();
		}
	});
});
