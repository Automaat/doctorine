import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/api', () => ({ resolveApiUrl: () => 'http://api.test' }));

import {
	buildCreatePayload,
	createToken,
	expiryLabel,
	isExpired,
	lastUsedLabel,
	revokeToken,
	scopeLabel
} from './tokens';
import type { PersonalToken } from './types';

function form(entries: Record<string, string>): FormData {
	const fd = new FormData();
	for (const [k, v] of Object.entries(entries)) fd.set(k, v);
	return fd;
}

function token(overrides: Partial<PersonalToken> = {}): PersonalToken {
	return {
		id: 1,
		name: 'coach',
		scope: 'read',
		expires_at: null,
		last_used_at: null,
		created_at: '2026-06-01T00:00:00Z',
		...overrides
	};
}

describe('buildCreatePayload', () => {
	it('builds a payload, treating blank expiry as null', () => {
		const { payload, error } = buildCreatePayload(
			form({ name: '  coach ', scope: 'read', expires_at: '' })
		);
		expect(error).toBeUndefined();
		expect(payload).toEqual({ name: 'coach', scope: 'read', expires_at: null });
	});

	it('keeps a provided expiry and defaults scope to full', () => {
		const { payload } = buildCreatePayload(form({ name: 'coach', expires_at: '2027-01-01' }));
		expect(payload).toEqual({ name: 'coach', scope: 'full', expires_at: '2027-01-01' });
	});

	it('rejects an empty name', () => {
		expect(buildCreatePayload(form({ name: '   ' })).error).toBe('Name is required');
	});

	it('rejects an over-long name', () => {
		expect(buildCreatePayload(form({ name: 'x'.repeat(121) })).error).toBe('Name is too long');
	});

	it('rejects an unknown scope', () => {
		expect(buildCreatePayload(form({ name: 'coach', scope: 'admin' })).error).toBe('Invalid scope');
	});
});

describe('display helpers', () => {
	it('labels scopes', () => {
		expect(scopeLabel('read')).toBe('Read-only');
		expect(scopeLabel('full')).toBe('Full access');
	});

	it('detects expiry against a reference time', () => {
		const now = new Date('2026-06-23T00:00:00Z');
		expect(isExpired(token({ expires_at: null }), now)).toBe(false);
		expect(isExpired(token({ expires_at: '2026-06-01T00:00:00Z' }), now)).toBe(true);
		expect(isExpired(token({ expires_at: '2026-12-01T00:00:00Z' }), now)).toBe(false);
		expect(isExpired(token({ expires_at: 'nonsense' }), now)).toBe(false);
	});

	it('renders expiry labels', () => {
		const now = new Date('2026-06-23T00:00:00Z');
		expect(expiryLabel(token({ expires_at: null }), now)).toBe('Never');
		expect(expiryLabel(token({ expires_at: '2026-12-01T23:59:59Z' }), now)).toBe('2026-12-01');
		expect(expiryLabel(token({ expires_at: '2026-01-01T23:59:59Z' }), now)).toBe(
			'2026-01-01 (expired)'
		);
	});

	it('renders last-used labels', () => {
		expect(lastUsedLabel(token({ last_used_at: null }))).toBe('Never used');
		expect(lastUsedLabel(token({ last_used_at: '2026-06-20T08:00:00Z' }))).toBe('2026-06-20');
	});
});

describe('token api calls', () => {
	afterEach(() => vi.restoreAllMocks());

	it('POSTs the create payload', async () => {
		const fetchFn = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ id: 1, token: 'dpat_x' }), {
				status: 201,
				headers: { 'content-type': 'application/json' }
			})
		);
		const result = await createToken({ name: 'coach', scope: 'read', expires_at: null }, fetchFn);
		expect(fetchFn).toHaveBeenCalledWith(
			'http://api.test/api/tokens',
			expect.objectContaining({ method: 'POST' })
		);
		expect(result.token).toBe('dpat_x');
	});

	it('DELETEs a token by id', async () => {
		const fetchFn = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		await revokeToken(7, fetchFn);
		expect(fetchFn).toHaveBeenCalledWith(
			'http://api.test/api/tokens/7',
			expect.objectContaining({ method: 'DELETE' })
		);
	});
});
