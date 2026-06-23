import type { TestOrder } from '$lib/types';

/** testKeysLabel renders an order's requested markers as a readable list. */
export function testKeysLabel(order: TestOrder): string {
	return order.test_keys.join(', ');
}

/** localDate formats a Date as YYYY-MM-DD in the local timezone. */
function localDate(now: Date): string {
	const year = now.getFullYear();
	const month = String(now.getMonth() + 1).padStart(2, '0');
	const day = String(now.getDate()).padStart(2, '0');
	return `${year}-${month}-${day}`;
}

/**
 * isOverdue is true when the order's due date is before today (local). Comparing
 * calendar dates avoids timezone skew at the day boundary.
 */
export function isOverdue(order: TestOrder, now: Date = new Date()): boolean {
	if (!order.due_on) return false;
	return order.due_on < localDate(now);
}

/** dueLabel renders an order's due date, flagging overdue orders. */
export function dueLabel(order: TestOrder, now: Date = new Date()): string {
	if (!order.due_on) return 'No due date';
	return isOverdue(order, now) ? `Due ${order.due_on} (overdue)` : `Due ${order.due_on}`;
}
