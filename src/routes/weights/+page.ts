import { api } from '$lib/apiClient';
import type { WeightEntry } from '$lib/types';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	return {
		weights: await api.get<WeightEntry[]>('/api/weights', { fetch })
	};
};
