import { describe, expect, it } from 'vitest';
import { dueLabel, isOverdue, testKeysLabel } from './testOrders';
import type { TestOrder } from './types';

function order(overrides: Partial<TestOrder> = {}): TestOrder {
	return {
		id: 1,
		source: 'coach',
		test_keys: ['ferrytyna', 'testosteron'],
		reason: 'baseline before volume',
		status: 'requested',
		requested_on: '2026-06-01',
		due_on: null,
		examination_id: null,
		notes: null,
		created_at: '2026-06-01T00:00:00Z',
		updated_at: '2026-06-01T00:00:00Z',
		...overrides
	};
}

describe('testOrders helpers', () => {
	it('joins test keys', () => {
		expect(testKeysLabel(order())).toBe('ferrytyna, testosteron');
	});

	it('detects overdue against a reference date', () => {
		const now = new Date('2026-06-23T12:00:00Z');
		expect(isOverdue(order({ due_on: null }), now)).toBe(false);
		expect(isOverdue(order({ due_on: '2026-06-01' }), now)).toBe(true);
		expect(isOverdue(order({ due_on: '2026-12-01' }), now)).toBe(false);
		// Same day is not overdue (due end of day).
		expect(isOverdue(order({ due_on: '2026-06-23' }), now)).toBe(false);
	});

	it('renders due labels', () => {
		const now = new Date('2026-06-23T12:00:00Z');
		expect(dueLabel(order({ due_on: null }), now)).toBe('No due date');
		expect(dueLabel(order({ due_on: '2026-12-01' }), now)).toBe('Due 2026-12-01');
		expect(dueLabel(order({ due_on: '2026-06-01' }), now)).toBe('Due 2026-06-01 (overdue)');
	});
});
