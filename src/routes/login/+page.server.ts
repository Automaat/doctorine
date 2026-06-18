import { fail, redirect } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import type { Actions } from './$types';

const cookieSecure = env.DOCTORINE_COOKIE_SECURE === 'true';

export const actions: Actions = {
	default: async ({ request, cookies, fetch }) => {
		const form = await request.formData();
		const username = String(form.get('username') ?? '').trim();
		const password = String(form.get('password') ?? '');
		const rememberMe = form.get('remember_me') === 'on';

		if (!username || !password) {
			return fail(400, { error: 'Username and password required', username });
		}

		const response = await fetch('/api/auth/login', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify({ username, password, remember_me: rememberMe })
		});
		if (response.status === 401) {
			return fail(401, { error: 'Invalid username or password', username });
		}
		if (!response.ok) {
			return fail(response.status, { error: 'Login unavailable', username });
		}

		const { token } = (await response.json()) as { token: string };
		cookies.set('doctorine_token', token, {
			path: '/',
			httpOnly: true,
			sameSite: 'lax',
			secure: cookieSecure,
			maxAge: rememberMe ? 5 * 24 * 60 * 60 : undefined
		});
		redirect(303, '/');
	}
};
