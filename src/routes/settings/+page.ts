import { api } from '$lib/apiClient';
import type { PersonalToken } from '$lib/types';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	return {
		tokens: await api.get<PersonalToken[]>('/api/tokens', { fetch })
	};
};
