import { describe, expect, it, vi } from 'vitest';

const get = vi.fn();
vi.mock('$lib/apiClient', () => ({ api: { get: (...args: unknown[]) => get(...args) } }));

import { load } from './+page';

describe('settings page load', () => {
	it('loads tokens through the api client', async () => {
		const tokens = [{ id: 1, name: 'coach' }];
		get.mockResolvedValue(tokens);
		const fetchFn = vi.fn();

		const result = await load({ fetch: fetchFn } as unknown as Parameters<typeof load>[0]);

		expect(result).toEqual({ tokens });
		expect(get).toHaveBeenCalledWith('/api/tokens', { fetch: fetchFn });
	});
});
