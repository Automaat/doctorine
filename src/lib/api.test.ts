import { describe, expect, it, vi } from 'vitest';

vi.mock('$app/environment', () => ({ browser: false }));
vi.mock('$env/dynamic/public', () => ({ env: { PUBLIC_API_URL: 'http://api.test' } }));

import { resolveApiUrl } from './api';

describe('resolveApiUrl', () => {
	it('returns the configured API URL on the server', () => {
		expect(resolveApiUrl()).toBe('http://api.test');
	});
});
