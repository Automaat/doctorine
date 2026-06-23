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
