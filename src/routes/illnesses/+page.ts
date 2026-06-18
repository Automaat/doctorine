import { api } from '$lib/apiClient';
import type { Illness } from '$lib/types';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	return {
		illnesses: await api.get<Illness[]>('/api/illnesses', { fetch })
	};
};
