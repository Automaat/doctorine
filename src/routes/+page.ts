import { api } from '$lib/apiClient';
import type { Overview } from '$lib/types';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	return {
		overview: await api.get<Overview>('/api/overview', { fetch })
	};
};
