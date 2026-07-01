import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

const svelteKitCookieImport = "import { parse, serialize } from 'cookie';";

function svelteKitCookieV2Compat() {
	return {
		name: 'doctorine-sveltekit-cookie-v2-compat',
		enforce: 'pre' as const,
		transform(code: string, id: string) {
			const normalizedId = id.replaceAll('\\', '/');
			if (!normalizedId.includes('/@sveltejs/kit/src/runtime/server/cookie.js')) return null;
			if (!code.includes(svelteKitCookieImport)) return null;

			// SvelteKit 2.68 still imports cookie's v1 API names; cookie v2 renamed them.
			return code.replace(
				svelteKitCookieImport,
				"import { parse, serialize } from '$lib/server/cookie-v2-compat';"
			);
		}
	};
}

export default defineConfig({
	plugins: [svelteKitCookieV2Compat(), tailwindcss(), sveltekit()],
	resolve: {
		conditions: ['browser']
	},
	test: {
		globals: true,
		environment: 'jsdom',
		include: ['src/**/*.{test,spec}.{js,ts}'],
		coverage: {
			provider: 'v8',
			reporter: ['text', 'json', 'html', 'lcov'],
			include: ['src/**/*.{ts,svelte}'],
			exclude: [
				'node_modules/**',
				'.svelte-kit/**',
				'build/**',
				'**/*.config.*',
				'**/.*rc.*',
				'src/**/*.{test,spec}.{js,ts}',
				'src/**/*.d.ts',
				'src/**/$types.d.ts'
			],
			// Baseline measured against the full src tree (not just imported
			// files). Raise these as coverage improves; they guard against
			// regression below the honest current level.
			thresholds: {
				statements: 8,
				branches: 17,
				functions: 10,
				lines: 11
			}
		}
	}
});
