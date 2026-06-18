import { api } from '$lib/apiClient';
import type { DocumentRecord, Examination, Illness } from '$lib/types';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	const [documents, illnesses, examinations] = await Promise.all([
		api.get<DocumentRecord[]>('/api/documents', { fetch }),
		api.get<Illness[]>('/api/illnesses', { fetch }),
		api.get<Examination[]>('/api/examinations', { fetch })
	]);
	return { documents, illnesses, examinations };
};
