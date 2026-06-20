import type { Examination, ExaminationResult } from '$lib/types';

export type ResultRow = {
	examination: Examination;
	result: ExaminationResult;
	flag: string | null;
};

export type TrendPoint = {
	date: string;
	value: number;
	unit: string | null;
	flag: string | null;
	referenceMin: number | null;
	referenceMax: number | null;
	href: string;
};

export type TrendCard = {
	key: string;
	label: string;
	points: TrendPoint[];
};

export type ChartBounds = {
	min: number;
	max: number;
};

export type ReminderKind = 'recurring' | 'one_time' | 'doctor_directed';
export type ReminderStatus = 'missing' | 'due' | 'soon' | 'ok' | 'complete' | 'directed';

export type ReminderRule = {
	label: string;
	testKeys: string[];
	kind: ReminderKind;
	cadenceMonths: number | null;
	reason: string;
};

export type ReminderItem = ReminderRule & {
	lastDate: string | null;
	dueDate: string | null;
	daysRemaining: number | null;
	href: string | null;
	status: ReminderStatus;
};

export type YearlyBucket = { year: string; total: number; flagged: number };

export type FlaggedTest = {
	key: string;
	name: string;
	count: number;
	high: number;
	low: number;
	unit: string | null;
};

export const chartWidth = 360;
export const chartHeight = 150;
export const chartPaddingX = 24;
export const chartPaddingTop = 18;
export const chartPaddingBottom = 28;

export const trackedTests = [
	{ key: 'tsh', label: 'TSH' },
	{ key: 'glukoza', label: 'Glucose' },
	{ key: 'witamina_d_25_oh', label: 'Vitamin D' },
	{ key: 'ast', label: 'AST' },
	{ key: 'kreatynina', label: 'Creatinine' },
	{ key: 'hemoglobina', label: 'Hemoglobin' }
];

export const reminderRules: ReminderRule[] = [
	{
		label: 'Blood pressure',
		testKeys: [
			'blood_pressure',
			'blood_pressure_systolic',
			'blood_pressure_diastolic',
			'cisnienie_tetnicze',
			'cisnienie_skurczowe',
			'cisnienie_rozkurczowe'
		],
		kind: 'recurring',
		cadenceMonths: 36,
		reason: 'Adult screening'
	},
	{
		label: 'Dental check',
		testKeys: ['dental_exam', 'stomatolog', 'przeglad_stomatologiczny'],
		kind: 'recurring',
		cadenceMonths: 12,
		reason: 'Preventive care'
	},
	{
		label: 'Eye exam',
		testKeys: ['eye_exam', 'badanie_wzroku', 'okulista'],
		kind: 'recurring',
		cadenceMonths: 60,
		reason: 'Vision screening'
	},
	{
		label: 'TSH',
		testKeys: ['tsh'],
		kind: 'recurring',
		cadenceMonths: 12,
		reason: 'Niedoczynność tarczycy'
	},
	{
		label: 'Vitamin B12',
		testKeys: ['witamina_b12'],
		kind: 'recurring',
		cadenceMonths: 12,
		reason: 'Vegetarian diet'
	},
	{
		label: 'Blood count',
		testKeys: ['hemoglobina', 'erytrocyty', 'hematokryt', 'leukocyty', 'plytki_krwi'],
		kind: 'recurring',
		cadenceMonths: 12,
		reason: 'CBC baseline'
	},
	{
		label: 'Vitamin D',
		testKeys: ['witamina_d_25_oh', '25_oh_witamina_d', 'witamina_d'],
		kind: 'recurring',
		cadenceMonths: 12,
		reason: 'Vegetarian diet'
	},
	{
		label: 'Folate',
		testKeys: ['kwas_foliowy', 'foliany'],
		kind: 'recurring',
		cadenceMonths: 24,
		reason: 'Nutrition check'
	},
	{
		label: 'Lipid panel',
		testKeys: [
			'cholesterol_calkowity',
			'cholesterol_ldl',
			'ldl_cholesterol',
			'cholesterol_hdl',
			'hdl_cholesterol',
			'cholesterol_nie_hdl',
			'triglicerydy'
		],
		kind: 'recurring',
		cadenceMonths: 48,
		reason: 'Cardiovascular risk'
	},
	{
		label: 'Glucose / HbA1c',
		testKeys: ['glukoza', 'hba1c', 'hemoglobina_glikowana'],
		kind: 'recurring',
		cadenceMonths: 36,
		reason: 'Metabolic screening'
	},
	{
		label: 'Kidney + electrolytes',
		testKeys: ['kreatynina', 'egfr', 'sod', 'potas', 'mocznik'],
		kind: 'recurring',
		cadenceMonths: 24,
		reason: 'Basic chemistry'
	},
	{
		label: 'Liver enzymes',
		testKeys: ['alt', 'alat', 'ast', 'aspat', 'ggtp', 'ggt', 'bilirubina_calkowita'],
		kind: 'recurring',
		cadenceMonths: 12,
		reason: 'Prior AST flag'
	},
	{
		label: 'Urinalysis',
		testKeys: [
			'mocz_ph',
			'mocz_ciezar_wlasciwy',
			'mocz_bialko',
			'mocz_glukoza',
			'mocz_leukocyty',
			'mocz_erytrocyty',
			'mocz_azotyny',
			'mocz_urobilinogen'
		],
		kind: 'recurring',
		cadenceMonths: 24,
		reason: 'Urine screen'
	},
	{
		label: 'HIV screen',
		testKeys: ['hiv', 'hiv_ag_ab', 'hiv_1_2'],
		kind: 'one_time',
		cadenceMonths: null,
		reason: 'Once, risk-based repeat'
	},
	{
		label: 'Hepatitis C screen',
		testKeys: ['hcv', 'anty_hcv', 'hcv_ab', 'przeciwciala_hcv'],
		kind: 'one_time',
		cadenceMonths: null,
		reason: 'Once adult screen'
	},
	{
		label: 'Inflammation markers',
		testKeys: ['crp', 'crp_ilosciowo', 'crp_met_immunochemiczna', 'ob'],
		kind: 'doctor_directed',
		cadenceMonths: null,
		reason: 'Symptoms/follow-up'
	},
	{
		label: 'Thyroid antibodies',
		testKeys: ['anty_tpo', 'anty_tg'],
		kind: 'doctor_directed',
		cadenceMonths: null,
		reason: 'Diagnosis context'
	},
	{
		label: 'Sex hormones',
		testKeys: ['testosteron', 'prolaktyna', 'estradiol', 'fsh', 'lh'],
		kind: 'doctor_directed',
		cadenceMonths: null,
		reason: 'Symptoms/follow-up'
	},
	{
		label: 'Colorectal screen',
		testKeys: ['kolonoskopia', 'fit', 'krew_utajona_w_kale'],
		kind: 'doctor_directed',
		cadenceMonths: null,
		reason: 'Age/risk-based'
	}
];

// resultFlag derives an out-of-range flag (H/L) from a result, honouring an
// explicit flag, numeric reference bounds, and prefix-qualified values.
export function resultFlag(result: ExaminationResult): string | null {
	if (result.flag) return result.flag;
	if (result.value_numeric === null) return null;
	if (result.reference_min !== null && result.value_numeric < result.reference_min) return 'L';
	if (result.reference_max !== null && result.value_numeric > result.reference_max) return 'H';
	if (
		(result.value_prefix === '<' || result.value_prefix === '<=') &&
		result.reference_min !== null &&
		result.value_numeric <= result.reference_min
	) {
		return 'L';
	}
	if (
		(result.value_prefix === '>' || result.value_prefix === '>=') &&
		result.reference_max !== null &&
		result.value_numeric >= result.reference_max
	) {
		return 'H';
	}
	return null;
}

export function isoDate(date: Date): string {
	return date.toISOString().slice(0, 10);
}

export function addMonths(date: string, months: number): string {
	const [year, month, day] = date.split('-').map(Number);
	const target = new Date(Date.UTC(year, month - 1 + months, day));
	if (target.getUTCDate() !== day) target.setUTCDate(0);
	return [
		target.getUTCFullYear(),
		String(target.getUTCMonth() + 1).padStart(2, '0'),
		String(target.getUTCDate()).padStart(2, '0')
	].join('-');
}

export function daysBetween(start: string, end: string): number {
	const startDate = new Date(`${start}T00:00:00Z`);
	const endDate = new Date(`${end}T00:00:00Z`);
	return Math.round((endDate.getTime() - startDate.getTime()) / 86_400_000);
}

export function lastYearCutoff(now: Date): string {
	const cutoff = new Date(now);
	cutoff.setUTCFullYear(cutoff.getUTCFullYear() - 1);
	return isoDate(cutoff);
}

// buildResultRows flattens examinations into per-result rows with derived flags,
// newest examinations first.
export function buildResultRows(examinations: Examination[]): ResultRow[] {
	return examinations.flatMap((examination) =>
		examination.results.map((result) => ({
			examination,
			result,
			flag: resultFlag(result)
		}))
	);
}

export function reminderStatus(
	rule: ReminderRule,
	lastDate: string | null,
	daysRemaining: number | null
): ReminderStatus {
	if (rule.kind === 'doctor_directed') return 'directed';
	if (rule.kind === 'one_time') return lastDate === null ? 'missing' : 'complete';
	if (lastDate === null || daysRemaining === null) return 'missing';
	if (daysRemaining < 0) return 'due';
	if (daysRemaining <= 90) return 'soon';
	return 'ok';
}

export function buildReminders(rows: ResultRow[], today: string): ReminderItem[] {
	const statusOrder: Record<ReminderStatus, number> = {
		due: 0,
		missing: 1,
		soon: 2,
		ok: 3,
		complete: 4,
		directed: 5
	};
	return reminderRules
		.map((rule) => {
			const latest = rows
				.filter((row) => rule.testKeys.includes(row.result.test_key))
				.sort((left, right) =>
					right.examination.exam_date.localeCompare(left.examination.exam_date)
				)[0];
			const lastDate = latest?.examination.exam_date ?? null;
			const dueDate =
				rule.kind === 'recurring' && rule.cadenceMonths !== null && lastDate !== null
					? addMonths(lastDate, rule.cadenceMonths)
					: null;
			const daysRemaining = dueDate === null ? null : daysBetween(today, dueDate);
			return {
				...rule,
				lastDate,
				dueDate,
				daysRemaining,
				href: latest === undefined ? null : `/examinations/${latest.examination.id}`,
				status: reminderStatus(rule, lastDate, daysRemaining)
			};
		})
		.sort(
			(left, right) =>
				statusOrder[left.status] - statusOrder[right.status] ||
				(left.dueDate ?? '9999-12-31').localeCompare(right.dueDate ?? '9999-12-31')
		);
}

export function trendPoints(rows: ResultRow[], key: string): TrendPoint[] {
	return rows
		.filter((row) => row.result.test_key === key && row.result.value_numeric !== null)
		.map((row) => ({
			date: row.examination.exam_date,
			value: row.result.value_numeric ?? 0,
			unit: row.result.unit,
			flag: row.flag,
			referenceMin: row.result.reference_min,
			referenceMax: row.result.reference_max,
			href: `/examinations/${row.examination.id}`
		}))
		.sort((left, right) => left.date.localeCompare(right.date));
}

export function buildYearlyBuckets(examinations: Examination[]): YearlyBucket[] {
	const buckets = new Map<string, YearlyBucket>();
	for (const examination of examinations) {
		const year = examination.exam_date.slice(0, 4);
		const bucket = buckets.get(year) ?? { year, total: 0, flagged: 0 };
		bucket.total += 1;
		if (
			examination.result_status === 'attention' ||
			examination.result_status === 'urgent' ||
			examination.results.some((result) => resultFlag(result) !== null)
		) {
			bucket.flagged += 1;
		}
		buckets.set(year, bucket);
	}
	return [...buckets.values()].sort((left, right) => left.year.localeCompare(right.year));
}

export function buildFlaggedByTest(sourceRows: ResultRow[]): FlaggedTest[] {
	const buckets = new Map<string, FlaggedTest>();
	for (const row of sourceRows) {
		const current = buckets.get(row.result.test_key) ?? {
			key: row.result.test_key,
			name: row.result.name,
			count: 0,
			high: 0,
			low: 0,
			unit: row.result.unit
		};
		current.count += 1;
		if (row.flag === 'H') current.high += 1;
		if (row.flag === 'L') current.low += 1;
		buckets.set(row.result.test_key, current);
	}
	return [...buckets.values()]
		.sort((left, right) => right.count - left.count || left.name.localeCompare(right.name))
		.slice(0, 8);
}

export function trendBounds(points: TrendPoint[]): ChartBounds {
	const values = points.flatMap((point) =>
		[point.value, point.referenceMin, point.referenceMax].filter((value) => value !== null)
	) as number[];
	if (values.length === 0) return { min: 0, max: 1 };
	let min = Math.min(...values);
	let max = Math.max(...values);
	if (min === max) {
		min -= 1;
		max += 1;
	}
	const padding = (max - min) * 0.08;
	return { min: min - padding, max: max + padding };
}

export function xFor(index: number, count: number): number {
	if (count <= 1) return chartWidth / 2;
	const plotWidth = chartWidth - chartPaddingX * 2;
	return chartPaddingX + (index / (count - 1)) * plotWidth;
}

export function yFor(value: number, bounds: ChartBounds): number {
	const plotHeight = chartHeight - chartPaddingTop - chartPaddingBottom;
	const ratio = (value - bounds.min) / (bounds.max - bounds.min);
	return chartPaddingTop + (1 - ratio) * plotHeight;
}

export function linePath(points: TrendPoint[]): string {
	if (points.length === 0) return '';
	const bounds = trendBounds(points);
	return points
		.map(
			(point, index) =>
				`${index === 0 ? 'M' : 'L'} ${xFor(index, points.length)} ${yFor(point.value, bounds)}`
		)
		.join(' ');
}

export function referenceLine(points: TrendPoint[], value: number | null): string | null {
	if (value === null) return null;
	const bounds = trendBounds(points);
	const y = yFor(value, bounds);
	return `${chartPaddingX},${y} ${chartWidth - chartPaddingX},${y}`;
}

export function latestPoint(points: TrendPoint[]): TrendPoint | null {
	return points[points.length - 1] ?? null;
}

export function previousPoint(points: TrendPoint[]): TrendPoint | null {
	return points[points.length - 2] ?? null;
}
