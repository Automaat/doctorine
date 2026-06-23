import { describe, expect, it, vi } from 'vitest';

const get = vi.fn();
vi.mock('$lib/apiClient', () => ({ api: { get: (...args: unknown[]) => get(...args) } }));

import { load } from './+page';

describe('dashboard page load', () => {
	it('loads overview, examinations, weights, and pending test orders', async () => {
		const overview = { document_count: 2, recent_documents: [] };
		const examinations = [{ id: 1 }];
		const weights = [{ id: 1 }];
		const testOrders = [{ id: 1, status: 'requested' }];
		get.mockImplementation((path: string) => {
			if (path === '/api/overview') return Promise.resolve(overview);
			if (path === '/api/weights') return Promise.resolve(weights);
			if (path === '/api/test-orders') return Promise.resolve(testOrders);
			return Promise.resolve(examinations);
		});
		const fetchFn = vi.fn();

		const result = await load({ fetch: fetchFn } as unknown as Parameters<typeof load>[0]);

		expect(result).toEqual({ overview, examinations, weights, testOrders });
		expect(get).toHaveBeenCalledWith('/api/overview', { fetch: fetchFn });
		expect(get).toHaveBeenCalledWith('/api/examinations', { fetch: fetchFn });
		expect(get).toHaveBeenCalledWith('/api/weights', { fetch: fetchFn });
		expect(get).toHaveBeenCalledWith('/api/test-orders', {
			query: { status: 'requested' },
			fetch: fetchFn
		});
	});
});
