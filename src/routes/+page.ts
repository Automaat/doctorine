import { api } from '$lib/apiClient';
import type { Examination, Overview } from '$lib/types';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	const [overview, examinations] = await Promise.all([
		api.get<Overview>('/api/overview', { fetch }),
		api.get<Examination[]>('/api/examinations', { fetch })
	]);
	return {
		overview,
		examinations
	};
};
