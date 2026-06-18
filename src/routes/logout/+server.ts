import { redirect, type RequestHandler } from '@sveltejs/kit';

export const POST: RequestHandler = async ({ cookies, fetch }) => {
	await fetch('/api/auth/logout', { method: 'POST' }).catch(() => undefined);
	cookies.delete('doctorine_token', { path: '/' });
	redirect(303, '/login');
};
