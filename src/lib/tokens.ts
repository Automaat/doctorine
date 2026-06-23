import { api } from '$lib/apiClient';
import type { CreatedToken, PersonalToken, TokenScope } from '$lib/types';

export interface CreateTokenPayload {
	name: string;
	scope: TokenScope;
	expires_at: string | null;
}

export interface BuildResult {
	payload?: CreateTokenPayload;
	error?: string;
}

const SCOPES: TokenScope[] = ['full', 'read'];

/** buildCreatePayload validates a token-creation form into a request body. */
export function buildCreatePayload(form: FormData): BuildResult {
	const name = String(form.get('name') ?? '').trim();
	if (name === '') return { error: 'Name is required' };
	if (name.length > 120) return { error: 'Name is too long' };

	const rawScope = String(form.get('scope') ?? 'full').trim();
	if (!SCOPES.includes(rawScope as TokenScope)) return { error: 'Invalid scope' };

	const rawExpiry = String(form.get('expires_at') ?? '').trim();
	return {
		payload: {
			name,
			scope: rawScope as TokenScope,
			expires_at: rawExpiry === '' ? null : rawExpiry
		}
	};
}

export function createToken(
	payload: CreateTokenPayload,
	fetchFn?: typeof globalThis.fetch
): Promise<CreatedToken> {
	return api.post<CreatedToken>('/api/tokens', payload, { fetch: fetchFn });
}

export function revokeToken(id: number, fetchFn?: typeof globalThis.fetch): Promise<void> {
	return api.del(`/api/tokens/${id}`, { fetch: fetchFn });
}

export function scopeLabel(scope: string): string {
	return scope === 'read' ? 'Read-only' : 'Full access';
}

export function isExpired(token: PersonalToken, now: Date = new Date()): boolean {
	if (!token.expires_at) return false;
	const expiry = new Date(token.expires_at);
	if (Number.isNaN(expiry.getTime())) return false;
	return expiry.getTime() <= now.getTime();
}

/** dateOnly extracts the YYYY-MM-DD portion of an RFC3339 timestamp. */
function dateOnly(value: string): string {
	return value.slice(0, 10);
}

/** expiryLabel renders a token's expiry for display, marking lapsed tokens. */
export function expiryLabel(token: PersonalToken, now: Date = new Date()): string {
	if (!token.expires_at) return 'Never';
	const label = dateOnly(token.expires_at);
	return isExpired(token, now) ? `${label} (expired)` : label;
}

export function lastUsedLabel(token: PersonalToken): string {
	return token.last_used_at ? dateOnly(token.last_used_at) : 'Never used';
}
