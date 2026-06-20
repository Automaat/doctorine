import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/api', () => ({ resolveApiUrl: () => 'http://api.test' }));

import { api, ApiError } from './apiClient';

function jsonResponse(status: number, body?: unknown): Response {
	return new Response(body === undefined ? null : JSON.stringify(body), {
		status,
		headers: { 'content-type': 'application/json' }
	});
}

describe('api client', () => {
	afterEach(() => vi.restoreAllMocks());

	it('GETs and parses a JSON body', async () => {
		const fetch = vi.fn().mockResolvedValue(jsonResponse(200, { ok: true }));
		const result = await api.get('/api/x', { fetch });
		expect(fetch).toHaveBeenCalledWith(
			'http://api.test/api/x',
			expect.objectContaining({ method: 'GET' })
		);
		expect(result).toEqual({ ok: true });
	});

	it('appends query params, skipping empty values', async () => {
		const fetch = vi.fn().mockResolvedValue(jsonResponse(200, []));
		await api.get('/api/x', { fetch, query: { a: '1', b: undefined, c: '', d: 2 } });
		expect(fetch).toHaveBeenCalledWith('http://api.test/api/x?a=1&d=2', expect.anything());
	});

	it('serializes a POST body with a JSON content type', async () => {
		const fetch = vi.fn().mockResolvedValue(jsonResponse(201, { id: 1 }));
		await api.post('/api/x', { name: 'n' }, { fetch });
		const init = fetch.mock.calls[0][1] as RequestInit;
		expect(init.body).toBe(JSON.stringify({ name: 'n' }));
		expect((init.headers as Record<string, string>)['content-type']).toBe('application/json');
	});

	it('throws ApiError carrying the server detail', async () => {
		const fetch = vi.fn().mockResolvedValue(jsonResponse(422, { detail: 'bad input' }));
		await expect(api.post('/api/x', {}, { fetch })).rejects.toMatchObject({
			name: 'ApiError',
			status: 422,
			message: 'bad input'
		});
	});

	it('falls back to status text when no detail is present', async () => {
		const fetch = vi.fn().mockResolvedValue(new Response('boom', { status: 500 }));
		await expect(api.get('/api/x', { fetch })).rejects.toBeInstanceOf(ApiError);
	});

	it('returns undefined for a 204 response', async () => {
		const fetch = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		const result = await api.del('/api/x', { fetch });
		expect(result).toBeUndefined();
	});
});
