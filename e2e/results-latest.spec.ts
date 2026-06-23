import { expect, test } from '@playwright/test';

// Exercises the latest-result-by-test_key endpoint end-to-end against the real
// backend + Postgres, so the DISTINCT ON "newest exam_date wins" query is
// covered in CI (the Go unit job has no database).
test('latest results returns the newest value per test_key', async ({ request }) => {
	const key = 'e2e_marker_latest';

	const older = await request.post('/api/examinations', {
		data: {
			title: 'E2E older labs',
			exam_date: '2026-01-01',
			results: [
				{
					test_key: key,
					name: 'E2E Marker',
					value_numeric: 30,
					reference_min: 40,
					reference_max: 400
				}
			]
		}
	});
	expect(older.status()).toBe(201);

	const newer = await request.post('/api/examinations', {
		data: {
			title: 'E2E newer labs',
			exam_date: '2026-06-01',
			results: [
				{
					test_key: key,
					name: 'E2E Marker',
					value_numeric: 120,
					reference_min: 40,
					reference_max: 400
				}
			]
		}
	});
	expect(newer.status()).toBe(201);

	const res = await request.get(`/api/results/latest?test_keys=${key}`);
	expect(res.ok()).toBeTruthy();
	const body = (await res.json()) as Array<{
		test_key: string;
		exam_date: string;
		value_numeric: number;
		flag: string | null;
	}>;
	expect(body).toHaveLength(1);
	expect(body[0].test_key).toBe(key);
	expect(body[0].exam_date).toBe('2026-06-01');
	expect(body[0].value_numeric).toBe(120);
	expect(body[0].flag).toBe('H');
});
