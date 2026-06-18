import { api } from '$lib/apiClient';
import type { Examination } from '$lib/types';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	return {
		examinations: await api.get<Examination[]>('/api/examinations', { fetch })
	};
};
