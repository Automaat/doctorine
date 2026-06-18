import { browser } from '$app/environment';
import { env } from '$env/dynamic/public';
import { error } from '@sveltejs/kit';

export function resolveApiUrl(): string {
	if (browser) return '';
	const url = env.PUBLIC_API_URL;
	if (!url) {
		error(500, 'PUBLIC_API_URL is not configured');
	}
	return url;
}
