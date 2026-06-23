import { expect, test } from '@playwright/test';

// Exercises the result-trend endpoint end-to-end against the real backend +
// Postgres, covering the dated-series query (ordering + day window) in CI.
test('result trend returns the dated numeric series oldest first', async ({ request }) => {
	// Unique per retry so a retried run starts from a clean key (POSTs are not
	// idempotent and the trend query does not collapse duplicates).
	const key = `e2e_marker_trend_${test.info().retry}`;

	const jan = await request.post('/api/examinations', {
		data: {
			title: 'E2E trend jan',
			exam_date: '2026-01-01',
			results: [{ test_key: key, name: 'E2E Trend', value_numeric: 30 }]
		}
	});
	expect(jan.status()).toBe(201);

	const mar = await request.post('/api/examinations', {
		data: {
			title: 'E2E trend mar',
			exam_date: '2026-03-01',
			results: [{ test_key: key, name: 'E2E Trend', value_numeric: 80 }]
		}
	});
	expect(mar.status()).toBe(201);

	const res = await request.get(`/api/results/trend/${key}?days=36500`);
	expect(res.ok()).toBeTruthy();
	const body = (await res.json()) as Array<{ exam_date: string; value_numeric: number }>;
	expect(body).toHaveLength(2);
	expect(body[0].exam_date).toBe('2026-01-01');
	expect(body[0].value_numeric).toBe(30);
	expect(body[1].exam_date).toBe('2026-03-01');
	expect(body[1].value_numeric).toBe(80);
});
