import { api } from '$lib/apiClient';
import type { Examination } from '$lib/types';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params }) => {
	return {
		examination: await api.get<Examination>(`/api/examinations/${params.id}`, { fetch })
	};
};
