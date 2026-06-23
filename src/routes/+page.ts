import { api } from '$lib/apiClient';
import type { Examination, Overview, TestOrder, WeightEntry } from '$lib/types';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	const [overview, examinations, weights, testOrders] = await Promise.all([
		api.get<Overview>('/api/overview', { fetch }),
		api.get<Examination[]>('/api/examinations', { fetch }),
		api.get<WeightEntry[]>('/api/weights', { fetch }),
		api.get<TestOrder[]>('/api/test-orders', { query: { status: 'requested' }, fetch })
	]);
	return {
		overview,
		examinations,
		weights,
		testOrders
	};
};
