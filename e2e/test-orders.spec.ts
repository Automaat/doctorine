import { expect, test } from '@playwright/test';

// Exercises the coach-ordered test loop end-to-end: the coach creates an order,
// the user enters a matching examination, and the order auto-completes.
test('a matching examination auto-completes a coach order', async ({ request }) => {
	const key = `e2e_marker_order_${test.info().retry}`;

	const created = await request.post('/api/test-orders', {
		data: { test_keys: [key], reason: 'baseline before volume' }
	});
	expect(created.status()).toBe(201);
	const order = (await created.json()) as { id: number; status: string };
	expect(order.status).toBe('requested');

	// It shows up as pending.
	const pending = await request.get('/api/test-orders?status=requested');
	const pendingList = (await pending.json()) as Array<{ id: number }>;
	expect(pendingList.some((o) => o.id === order.id)).toBe(true);

	// User enters results covering the order's keys.
	const exam = await request.post('/api/examinations', {
		data: {
			title: 'E2E order fulfillment',
			exam_date: '2026-06-03',
			results: [{ test_key: key, name: 'E2E Marker', value_numeric: 50 }]
		}
	});
	expect(exam.status()).toBe(201);
	const examId = ((await exam.json()) as { id: number }).id;

	// The order is now completed and linked to the examination.
	const completed = await request.get('/api/test-orders?status=completed');
	const completedList = (await completed.json()) as Array<{
		id: number;
		status: string;
		examination_id: number | null;
	}>;
	const fulfilled = completedList.find((o) => o.id === order.id);
	expect(fulfilled).toBeTruthy();
	expect(fulfilled?.status).toBe('completed');
	expect(fulfilled?.examination_id).toBe(examId);
});
