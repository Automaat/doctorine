import { api } from '$lib/apiClient';
import type { Examination, Overview, WeightEntry } from '$lib/types';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	const [overview, examinations, weights] = await Promise.all([
		api.get<Overview>('/api/overview', { fetch }),
		api.get<Examination[]>('/api/examinations', { fetch }),
		api.get<WeightEntry[]>('/api/weights', { fetch })
	]);
	return {
		overview,
		examinations,
		weights
	};
};
