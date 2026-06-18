import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

function backendURL(path: string, search: string): string {
	const base = env.API_PROXY_TARGET ?? 'http://localhost:8000';
	return `${base}/api/${path}${search}`;
}

const proxy: RequestHandler = async ({ request, params, cookies, url, fetch }) => {
	const headers = new Headers();
	for (const name of ['content-type', 'accept']) {
		const value = request.headers.get(name);
		if (value) headers.set(name, value);
	}
	const token = cookies.get('doctorine_token');
	if (token) {
		headers.set('authorization', `Bearer ${token}`);
	}

	const init: RequestInit = { method: request.method, headers };
	if (request.method !== 'GET' && request.method !== 'HEAD') {
		init.body = await request.arrayBuffer();
	}

	const upstream = await fetch(backendURL(params.path, url.search), init);
	const body = await upstream.arrayBuffer();
	const responseHeaders = new Headers();
	for (const name of ['content-type', 'content-disposition', 'content-length']) {
		const value = upstream.headers.get(name);
		if (value) responseHeaders.set(name, value);
	}
	return new Response(body.byteLength > 0 ? body : null, {
		status: upstream.status,
		headers: responseHeaders
	});
};

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const PATCH = proxy;
export const DELETE = proxy;
