import { redirect, type Handle, type HandleFetch } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { env as privateEnv } from '$env/dynamic/private';

interface SessionUser {
	username: string;
	name: string;
	isAdmin: boolean;
}

function backendBase(): string {
	return privateEnv.API_PROXY_TARGET || env.PUBLIC_API_URL || 'http://localhost:8000';
}

type AuthResult = { status: 'ok'; user: SessionUser } | { status: 'invalid' } | { status: 'error' };

// resolveUser verifies the opaque session token against the backend instead of
// trusting client-decoded claims. It distinguishes a definitively invalid
// session (revoked/expired -> 401) from a transient backend failure so a blip
// does not discard an otherwise valid cookie.
async function resolveUser(token: string): Promise<AuthResult> {
	let response: Response;
	try {
		response = await fetch(`${backendBase()}/api/auth/me`, {
			headers: { authorization: `Bearer ${token}` }
		});
	} catch {
		return { status: 'error' };
	}
	if (response.status === 401) return { status: 'invalid' };
	if (!response.ok) return { status: 'error' };
	try {
		const data = (await response.json()) as {
			username?: unknown;
			display_name?: unknown;
			is_admin?: unknown;
		};
		if (typeof data.username !== 'string' || data.username === '') return { status: 'invalid' };
		if (typeof data.is_admin !== 'boolean') return { status: 'invalid' };
		const name =
			typeof data.display_name === 'string' && data.display_name !== ''
				? data.display_name
				: data.username;
		return { status: 'ok', user: { username: data.username, name, isAdmin: data.is_admin } };
	} catch {
		return { status: 'error' };
	}
}

export const handle: Handle = async ({ event, resolve }) => {
	const { pathname } = event.url;
	const isAsset = pathname.startsWith('/_app/') || /\.\w+$/.test(pathname);
	const token = event.cookies.get('doctorine_token');
	const auth = token && !isAsset ? await resolveUser(token) : null;
	const user = auth?.status === 'ok' ? auth.user : null;
	event.locals.user = user;

	if (user) return resolve(event);

	// Only discard the cookie when the backend says the session is invalid;
	// keep it on a transient backend error so users are not logged out by a blip.
	if (token && !isAsset && auth?.status === 'invalid') {
		event.cookies.delete('doctorine_token', { path: '/' });
	}

	const isLogin = pathname === '/login';
	const isPublicApi = pathname === '/api/auth/login' || pathname === '/api/auth/logout';
	if (isAsset || isLogin || isPublicApi) return resolve(event);

	if (pathname.startsWith('/api/')) {
		// A service may authenticate with a personal access token via the
		// Authorization header instead of the browser session cookie. Let those
		// requests through to the proxy; the backend is the authority on whether
		// the bearer token is valid.
		if (event.request.headers.has('authorization')) return resolve(event);
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
