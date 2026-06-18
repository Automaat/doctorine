import { resolveApiUrl } from '$lib/api';

export class ApiError extends Error {
	readonly status: number;

	constructor(status: number, message: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

type FetchFn = typeof globalThis.fetch;

export interface ApiOptions {
	fetch?: FetchFn;
	query?: Record<string, string | number | undefined | null>;
	signal?: AbortSignal;
}

function extractDetail(body: unknown, fallback: string): string {
	if (body && typeof body === 'object' && 'detail' in body) {
		const detail = (body as { detail: unknown }).detail;
		if (typeof detail === 'string' && detail) return detail;
	}
	return fallback;
}

function buildUrl(path: string, query?: ApiOptions['query']): string {
	const base = resolveApiUrl();
	if (!query) return `${base}${path}`;
	const params = new URLSearchParams();
	for (const [key, value] of Object.entries(query)) {
		if (value !== undefined && value !== null && value !== '') {
			params.set(key, String(value));
		}
	}
	const qs = params.toString();
	return qs ? `${base}${path}?${qs}` : `${base}${path}`;
}

async function request<T>(
	method: string,
	path: string,
	body: unknown,
	opts: ApiOptions = {}
): Promise<T> {
	const doFetch = opts.fetch ?? globalThis.fetch;
	const init: RequestInit = { method, signal: opts.signal };
	if (body !== undefined) {
		init.headers = { 'content-type': 'application/json' };
		init.body = JSON.stringify(body);
	}
	const res = await doFetch(buildUrl(path, opts.query), init);
	if (!res.ok) {
		const errBody = await res.json().catch(() => null);
		throw new ApiError(res.status, extractDetail(errBody, res.statusText));
	}
	if (res.status === 204) return undefined as T;
	const text = await res.text();
	return (text ? JSON.parse(text) : undefined) as T;
}

export const api = {
	get: <T>(path: string, opts?: ApiOptions) => request<T>('GET', path, undefined, opts),
	post: <T>(path: string, body?: unknown, opts?: ApiOptions) =>
		request<T>('POST', path, body, opts),
	del: <T = void>(path: string, opts?: ApiOptions) => request<T>('DELETE', path, undefined, opts)
};
