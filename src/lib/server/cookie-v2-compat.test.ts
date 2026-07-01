import { describe, expect, it } from 'vitest';

import { parse, serialize } from './cookie-v2-compat';

describe('cookie v2 compatibility', () => {
	it('parses cookies through the old cookie.parse signature', () => {
		expect({ ...parse('doctorine_token=opaque%20token; theme=dark') }).toEqual({
			doctorine_token: 'opaque token',
			theme: 'dark'
		});
	});

	it('serializes cookies through the old cookie.serialize signature', () => {
		expect(
			serialize('doctorine_token', 'opaque token', {
				httpOnly: true,
				maxAge: 60,
				path: '/',
				sameSite: 'lax',
				secure: true
			})
		).toBe('doctorine_token=opaque%20token; Max-Age=60; Path=/; HttpOnly; Secure; SameSite=Lax');
	});
});
