import { api } from '$lib/apiClient';
import type { Supplement } from '$lib/types';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	return {
		supplements: await api.get<Supplement[]>('/api/supplements', { fetch })
	};
};
