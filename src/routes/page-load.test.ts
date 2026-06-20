import { describe, expect, it, vi } from 'vitest';

const get = vi.fn();
vi.mock('$lib/apiClient', () => ({ api: { get: (...args: unknown[]) => get(...args) } }));

import { load } from './+page';

describe('dashboard page load', () => {
	it('loads overview and examinations through the api client', async () => {
		const overview = { document_count: 2, recent_documents: [] };
		const examinations = [{ id: 1 }];
		get.mockImplementation((path: string) =>
			path === '/api/overview' ? Promise.resolve(overview) : Promise.resolve(examinations)
		);
		const fetchFn = vi.fn();

		const result = await load({ fetch: fetchFn } as unknown as Parameters<typeof load>[0]);

		expect(result).toEqual({ overview, examinations });
		expect(get).toHaveBeenCalledWith('/api/overview', { fetch: fetchFn });
		expect(get).toHaveBeenCalledWith('/api/examinations', { fetch: fetchFn });
	});
});
