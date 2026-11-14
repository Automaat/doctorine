// @vitest-environment node
import { createServer, type Server } from 'node:http';
import type { AddressInfo } from 'node:net';
import { afterAll, beforeAll, describe, expect, it, vi } from 'vitest';

const target = vi.hoisted(() => ({ base: '' }));
vi.mock('$env/dynamic/private', () => ({
	env: {
		get API_PROXY_TARGET() {
			return target.base;
		}
	}
}));

import { GET } from './+server';

let server: Server;

beforeAll(async () => {
	server = createServer((req, res) => {
		res.setHeader('set-cookie', [
			'doctorine_oidc_state=; Path=/api/auth/oidc; Max-Age=0; HttpOnly',
			'doctorine_token=sess; Path=/; HttpOnly; SameSite=Lax'
		]);
		res.writeHead(req.headers.cookie === 'doctorine_oidc_state=signed' ? 302 : 400, {
			location: '/'
		});
		res.end();
	});
	await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
	target.base = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
});

afterAll(() => {
	server.close();
});

describe('api proxy against a real HTTP backend', () => {
	it('returns the backend 302 and both Set-Cookie headers instead of following', async () => {
		const request = new Request('http://frontend.test/api/auth/oidc/callback', {
			headers: { cookie: 'doctorine_oidc_state=signed' }
		});
		const res = await GET({
			request,
			params: { path: 'auth/oidc/callback' },
			cookies: { get: () => undefined },
			url: { search: '?code=c&state=s' },
			fetch
		} as unknown as Parameters<typeof GET>[0]);

		expect(res.status).toBe(302);
		expect(res.headers.get('location')).toBe('/');
		expect(res.headers.getSetCookie()).toEqual([
			'doctorine_oidc_state=; Path=/api/auth/oidc; Max-Age=0; HttpOnly',
			'doctorine_token=sess; Path=/; HttpOnly; SameSite=Lax'
		]);
	});
});
