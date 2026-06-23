import { describe, expect, it } from 'vitest';
import type { Examination, ExaminationResult, WeightEntry } from './types';
import {
	addMonths,
	buildReminders,
	buildResultRows,
	daysBetween,
	lastYearCutoff,
	reminderStatus,
	resultFlag,
	trendBounds,
	trendPoints,
	weightTrendPoints,
	xFor,
	type ReminderRule
} from './dashboard';

function makeWeight(overrides: Partial<WeightEntry> = {}): WeightEntry {
	return {
		id: 1,
		measured_on: '2026-01-05',
		weight_kg: 82,
		notes: null,
		created_at: '2026-01-05T00:00:00Z',
		updated_at: '2026-01-05T00:00:00Z',
		...overrides
	};
}

describe('weightTrendPoints', () => {
	it('sorts entries chronologically and tags them as kg', () => {
		const points = weightTrendPoints([
			makeWeight({ id: 2, measured_on: '2026-03-09', weight_kg: 80.1 }),
			makeWeight({ id: 1, measured_on: '2026-01-05', weight_kg: 82.4 })
		]);
		expect(points.map((point) => point.date)).toEqual(['2026-01-05', '2026-03-09']);
		expect(points[0]).toMatchObject({ value: 82.4, unit: 'kg', href: '/weights' });
	});

	it('drops entries without a finite weight', () => {
		const points = weightTrendPoints([
			makeWeight({ weight_kg: Number.NaN }),
			makeWeight({ id: 2, measured_on: '2026-02-01', weight_kg: 81 })
		]);
		expect(points).toHaveLength(1);
		expect(points[0].value).toBe(81);
	});
});

function makeResult(overrides: Partial<ExaminationResult> = {}): ExaminationResult {
	return {
		id: 1,
		examination_id: 1,
		definition_id: null,
		definition: null,
		test_key: 'glukoza',
		name: 'Glucose',
		value_text: null,
		value_numeric: null,
		value_prefix: null,
		unit: 'mg/dL',
		reference_min: null,
		reference_max: null,
		flag: null,
		display_order: 0,
		created_at: '2025-01-01T00:00:00Z',
		updated_at: '2025-01-01T00:00:00Z',
		...overrides
	};
}

function makeExam(overrides: Partial<Examination> = {}): Examination {
	return {
		id: 1,
		title: 'Exam',
		exam_date: '2025-01-01',
		category: 'general',
		facility: null,
		result_status: 'unknown',
		summary: null,
		notes: null,
		results: [],
		created_at: '2025-01-01T00:00:00Z',
		updated_at: '2025-01-01T00:00:00Z',
		...overrides
	};
}

describe('resultFlag', () => {
	it('prefers an explicit flag', () => {
		expect(resultFlag(makeResult({ flag: 'H', value_numeric: 1, reference_max: 100 }))).toBe('H');
	});

	it('returns null when there is no numeric value', () => {
		expect(resultFlag(makeResult({ value_numeric: null }))).toBeNull();
	});

	it('flags below the minimum and above the maximum', () => {
		expect(resultFlag(makeResult({ value_numeric: 3, reference_min: 4 }))).toBe('L');
		expect(resultFlag(makeResult({ value_numeric: 11, reference_max: 10 }))).toBe('H');
	});

	it('treats an exact bound as in range without a prefix', () => {
		expect(
			resultFlag(makeResult({ value_numeric: 4, reference_min: 4, reference_max: 10 }))
		).toBeNull();
		expect(
			resultFlag(makeResult({ value_numeric: 10, reference_min: 4, reference_max: 10 }))
		).toBeNull();
	});

	it('flags prefix-qualified values at the bound', () => {
		expect(resultFlag(makeResult({ value_numeric: 4, reference_min: 4, value_prefix: '<=' }))).toBe(
			'L'
		);
		expect(
			resultFlag(makeResult({ value_numeric: 10, reference_max: 10, value_prefix: '>=' }))
		).toBe('H');
	});
});

describe('addMonths', () => {
	it('advances whole months', () => {
		expect(addMonths('2025-01-15', 12)).toBe('2026-01-15');
		expect(addMonths('2025-01-15', 3)).toBe('2025-04-15');
	});

	it('clamps to the last day when the target month is shorter', () => {
		expect(addMonths('2025-01-31', 1)).toBe('2025-02-28');
		expect(addMonths('2024-01-31', 1)).toBe('2024-02-29');
	});
});

describe('daysBetween', () => {
	it('counts whole days in both directions', () => {
		expect(daysBetween('2025-01-01', '2025-01-31')).toBe(30);
		expect(daysBetween('2025-02-01', '2025-01-01')).toBe(-31);
	});
});

describe('lastYearCutoff', () => {
	it('returns the same day one year earlier', () => {
		expect(lastYearCutoff(new Date('2025-06-20T12:00:00Z'))).toBe('2024-06-20');
	});
});

describe('reminderStatus', () => {
	const recurring: ReminderRule = {
		label: 'x',
		testKeys: ['k'],
		kind: 'recurring',
		cadenceMonths: 12,
		reason: 'r'
	};
	const oneTime: ReminderRule = { ...recurring, kind: 'one_time', cadenceMonths: null };
	const directed: ReminderRule = { ...recurring, kind: 'doctor_directed', cadenceMonths: null };

	it('is directed for doctor-directed rules', () => {
		expect(reminderStatus(directed, null, null)).toBe('directed');
		expect(reminderStatus(directed, '2025-01-01', 5)).toBe('directed');
	});

	it('reports one-time completion', () => {
		expect(reminderStatus(oneTime, null, null)).toBe('missing');
		expect(reminderStatus(oneTime, '2025-01-01', null)).toBe('complete');
	});

	it('grades recurring rules by days remaining', () => {
		expect(reminderStatus(recurring, null, null)).toBe('missing');
		expect(reminderStatus(recurring, '2025-01-01', null)).toBe('missing');
		expect(reminderStatus(recurring, '2025-01-01', -1)).toBe('due');
		expect(reminderStatus(recurring, '2025-01-01', 90)).toBe('soon');
		expect(reminderStatus(recurring, '2025-01-01', 91)).toBe('ok');
	});
});

describe('buildResultRows', () => {
	it('flattens examinations into flagged rows', () => {
		const exams = [
			makeExam({
				id: 7,
				results: [
					makeResult({ value_numeric: 11, reference_max: 10 }),
					makeResult({ value_numeric: 5, reference_max: 10 })
				]
			})
		];
		const rows = buildResultRows(exams);
		expect(rows).toHaveLength(2);
		expect(rows[0].flag).toBe('H');
		expect(rows[1].flag).toBeNull();
		expect(rows[0].examination.id).toBe(7);
	});
});

describe('trendPoints', () => {
	it('keeps numeric points sorted ascending by date', () => {
		const rows = buildResultRows([
			makeExam({
				id: 1,
				exam_date: '2025-03-01',
				results: [makeResult({ test_key: 'tsh', value_numeric: 2 })]
			}),
			makeExam({
				id: 2,
				exam_date: '2025-01-01',
				results: [makeResult({ test_key: 'tsh', value_numeric: 1 })]
			}),
			makeExam({
				id: 3,
				exam_date: '2025-02-01',
				results: [makeResult({ test_key: 'tsh', value_numeric: null })]
			})
		]);
		const points = trendPoints(rows, 'tsh');
		expect(points.map((point) => point.date)).toEqual(['2025-01-01', '2025-03-01']);
	});
});

describe('buildReminders', () => {
	it('marks a recurring check overdue and links the latest exam', () => {
		const rows = buildResultRows([
			makeExam({
				id: 42,
				exam_date: '2020-01-01',
				results: [makeResult({ test_key: 'tsh', value_numeric: 2 })]
			})
		]);
		const reminders = buildReminders(rows, '2025-06-20');
		const tsh = reminders.find((reminder) => reminder.label === 'TSH');
		expect(tsh).toBeDefined();
		expect(tsh?.lastDate).toBe('2020-01-01');
		expect(tsh?.dueDate).toBe('2021-01-01');
		expect(tsh?.status).toBe('due');
		expect(tsh?.href).toBe('/examinations/42');
	});

	it('marks untouched recurring checks missing and directed checks directed', () => {
		const reminders = buildReminders([], '2025-06-20');
		const tsh = reminders.find((reminder) => reminder.label === 'TSH');
		const inflammation = reminders.find((reminder) => reminder.label === 'Inflammation markers');
		expect(tsh?.status).toBe('missing');
		expect(inflammation?.status).toBe('directed');
	});
});

describe('trendBounds', () => {
	it('returns a unit range when there are no points', () => {
		expect(trendBounds([])).toEqual({ min: 0, max: 1 });
	});

	it('pads a flat series so the line is visible', () => {
		const bounds = trendBounds([
			{
				date: '2025-01-01',
				value: 10,
				unit: null,
				flag: null,
				referenceMin: null,
				referenceMax: null,
				href: '/x'
			}
		]);
		// min===max collapses to 9..11, then 8% padding on the span of 2.
		expect(bounds.min).toBeCloseTo(8.84, 5);
		expect(bounds.max).toBeCloseTo(11.16, 5);
	});

	it('includes reference bounds and pads the span', () => {
		const bounds = trendBounds([
			{
				date: '2025-01-01',
				value: 15,
				unit: null,
				flag: null,
				referenceMin: 10,
				referenceMax: 20,
				href: '/x'
			}
		]);
		expect(bounds.min).toBeCloseTo(9.2, 5);
		expect(bounds.max).toBeCloseTo(20.8, 5);
	});
});

describe('xFor', () => {
	it('centres a single point and spans the plot otherwise', () => {
		expect(xFor(0, 1)).toBe(180);
		expect(xFor(0, 2)).toBe(24);
		expect(xFor(1, 2)).toBe(336);
	});
});
