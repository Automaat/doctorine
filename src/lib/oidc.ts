export const OIDC_LOGIN_PATH = '/api/auth/oidc/login';

const OIDC_ERROR_MESSAGES: Record<string, string> = {
	oidc_failed: 'Authelia sign-in failed. Try again or log in with your password.',
	oidc_no_user: 'Your Authelia account is not linked to a Doctorine user.',
	oidc_unavailable: 'Authelia is unreachable right now. Log in with your password instead.'
};

/** oidcErrorMessage maps a /login?error= code from the OIDC callback to readable text. */
export function oidcErrorMessage(code: string | null): string | null {
	if (!code) return null;
	return OIDC_ERROR_MESSAGES[code] ?? null;
}

/** fetchOIDCEnabled asks the backend whether OIDC login is configured; any failure means disabled. */
export async function fetchOIDCEnabled(fetchFn: typeof fetch): Promise<boolean> {
	try {
		const response = await fetchFn('/api/auth/oidc/status');
		if (!response.ok) return false;
		const data = (await response.json()) as { enabled?: unknown };
		return data.enabled === true;
	} catch {
		return false;
	}
}
