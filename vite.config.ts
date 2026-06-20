import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
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
