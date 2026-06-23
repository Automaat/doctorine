import type { TestOrder } from '$lib/types';

/** testKeysLabel renders an order's requested markers as a readable list. */
export function testKeysLabel(order: TestOrder): string {
	return order.test_keys.join(', ');
}

/** isOverdue is true when the order has a due date that has already passed. */
export function isOverdue(order: TestOrder, now: Date = new Date()): boolean {
	if (!order.due_on) return false;
	const due = new Date(`${order.due_on}T23:59:59Z`);
	if (Number.isNaN(due.getTime())) return false;
	return due.getTime() < now.getTime();
}

/** dueLabel renders an order's due date, flagging overdue orders. */
export function dueLabel(order: TestOrder, now: Date = new Date()): string {
	if (!order.due_on) return 'No due date';
	return isOverdue(order, now) ? `Due ${order.due_on} (overdue)` : `Due ${order.due_on}`;
}
