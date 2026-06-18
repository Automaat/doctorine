import { redirect, type Handle, type HandleFetch } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';

interface SessionUser {
	username: string;
	name: string;
	isAdmin: boolean;
}

function decodeUser(token: string): SessionUser | null {
	try {
		const payload = token.split('.')[1];
		if (!payload) return null;
		const pad = payload.length % 4 === 0 ? '' : '='.repeat(4 - (payload.length % 4));
		const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/') + pad);
		const claims = JSON.parse(json) as {
			username?: unknown;
			name?: unknown;
			is_admin?: unknown;
			exp?: unknown;
		};
		if (typeof claims.exp !== 'number' || claims.exp * 1000 < Date.now()) return null;
		if (typeof claims.username !== 'string' || claims.username === '') return null;
		if (typeof claims.is_admin !== 'boolean') return null;
		return {
			username: claims.username,
			name: typeof claims.name === 'string' ? claims.name : '',
			isAdmin: claims.is_admin
		};
	} catch {
		return null;
	}
}

export const handle: Handle = async ({ event, resolve }) => {
	const { pathname } = event.url;
	const token = event.cookies.get('doctorine_token');
	const user = token ? decodeUser(token) : null;
	event.locals.user = user;

	if (user) return resolve(event);

	if (token) {
		event.cookies.delete('doctorine_token', { path: '/' });
	}

	const isAsset = pathname.startsWith('/_app/') || /\.\w+$/.test(pathname);
	const isLogin = pathname === '/login';
	const isPublicApi = pathname === '/api/auth/login' || pathname === '/api/auth/logout';
	if (isAsset || isLogin || isPublicApi) return resolve(event);

	if (pathname.startsWith('/api/')) {
		return new Response(JSON.stringify({ detail: 'Not authenticated' }), {
			status: 401,
			headers: { 'content-type': 'application/json' }
		});
	}
	redirect(303, '/login');
};

export const handleFetch: HandleFetch = async ({ event, request, fetch }) => {
	const backend = env.PUBLIC_API_URL;
	if (backend && request.url.startsWith(backend)) {
		const token = event.cookies.get('doctorine_token');
		if (token) {
			request.headers.set('authorization', `Bearer ${token}`);
		}
	}
	return fetch(request);
};
