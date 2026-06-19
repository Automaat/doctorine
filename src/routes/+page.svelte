<script lang="ts">
	import { formatBytes, formatDate } from '$lib/format';
	import type { Examination, ExaminationResult } from '$lib/types';
	import {
		Activity,
		AlertTriangle,
		CalendarCheck,
		CalendarDays,
		FileText,
		Gauge,
		Stethoscope,
		TrendingUp
	} from 'lucide-svelte';
	import type { PageData } from './$types';

	type ResultRow = {
		examination: Examination;
		result: ExaminationResult;
		flag: string | null;
	};

	type TrendPoint = {
		date: string;
		value: number;
		unit: string | null;
		flag: string | null;
		referenceMin: number | null;
		referenceMax: number | null;
		href: string;
	};

	type TrendCard = {
		key: string;
		label: string;
		points: TrendPoint[];
	};

	type ChartBounds = {
		min: number;
		max: number;
	};

	type ReminderRule = {
		label: string;
		testKeys: string[];
		cadenceMonths: number;
		reason: string;
	};

	type ReminderItem = ReminderRule & {
		lastDate: string | null;
		dueDate: string | null;
		daysRemaining: number | null;
		href: string | null;
		status: 'missing' | 'due' | 'soon' | 'ok';
	};

	const chartWidth = 360;
	const chartHeight = 150;
	const chartPaddingX = 24;
	const chartPaddingTop = 18;
	const chartPaddingBottom = 28;
	const trackedTests = [
		{ key: 'tsh', label: 'TSH' },
		{ key: 'glukoza', label: 'Glucose' },
		{ key: 'witamina_d_25_oh', label: 'Vitamin D' },
		{ key: 'ast', label: 'AST' },
		{ key: 'kreatynina', label: 'Creatinine' },
		{ key: 'hemoglobina', label: 'Hemoglobin' }
	];
	const reminderRules: ReminderRule[] = [
		{
			label: 'TSH',
			testKeys: ['tsh'],
			cadenceMonths: 12,
			reason: 'Niedoczynność tarczycy'
		},
		{
			label: 'Vitamin B12',
			testKeys: ['witamina_b12'],
			cadenceMonths: 12,
			reason: 'Vegetarian diet'
		},
		{
			label: 'Ferritin',
			testKeys: ['ferrytyna'],
			cadenceMonths: 12,
			reason: 'Iron stores'
		},
		{
			label: 'Blood count',
			testKeys: ['hemoglobina', 'erytrocyty', 'hematokryt', 'leukocyty'],
			cadenceMonths: 12,
			reason: 'CBC baseline'
		},
		{
			label: 'Vitamin D',
			testKeys: ['witamina_d_25_oh'],
			cadenceMonths: 12,
			reason: 'Vegetarian diet'
		}
	];
	const numberFormat = new Intl.NumberFormat('pl-PL', { maximumFractionDigits: 2 });
	const todayDate = isoDate(new Date());

	let { data }: { data: PageData } = $props();

	const examinations = $derived(
		[...(data.examinations ?? [])].sort((left, right) =>
			right.exam_date.localeCompare(left.exam_date)
		)
	);
	const rows = $derived.by<ResultRow[]>(() =>
		examinations.flatMap((examination) =>
			examination.results.map((result) => ({
				examination,
				result,
				flag: resultFlag(result)
			}))
		)
	);
	const flaggedRows = $derived(rows.filter((row) => row.flag !== null));
	const attentionExaminations = $derived(
		examinations.filter(
			(examination) =>
				examination.result_status === 'attention' ||
				examination.result_status === 'urgent' ||
				examination.results.some((result) => resultFlag(result) !== null)
		)
	);
	const latestExamination = $derived(examinations[0] ?? null);
	const normalRate = $derived(
		rows.length === 0 ? 100 : Math.round(((rows.length - flaggedRows.length) / rows.length) * 100)
	);
	const metrics = $derived([
		{
			label: 'Examinations',
			value: examinations.length,
			meta: latestExamination ? `Latest ${formatDate(latestExamination.exam_date)}` : 'No records',
			icon: Stethoscope
		},
		{
			label: 'Lab results',
			value: rows.length,
			meta: `${normalRate}% within range`,
			icon: Activity
		},
		{
			label: 'Flagged results',
			value: flaggedRows.length,
			meta: `${attentionExaminations.length} exams need attention`,
			icon: AlertTriangle
		},
		{
			label: 'Documents',
			value: data.overview.document_count,
			meta: `${data.overview.recent_documents.length} recent`,
			icon: FileText
		}
	]);
	const trendCards = $derived.by<TrendCard[]>(() =>
		trackedTests
			.map((test) => ({
				...test,
				points: trendPoints(test.key)
			}))
			.filter((card) => card.points.length > 0)
	);
	const yearlyBuckets = $derived.by(() => buildYearlyBuckets());
	const recentFlaggedCutoff = lastYearCutoffDate();
	const recentFlaggedRows = $derived.by(() =>
		[...flaggedRows]
			.filter((row) => row.examination.exam_date >= recentFlaggedCutoff)
			.sort((left, right) => right.examination.exam_date.localeCompare(left.examination.exam_date))
	);
	const flaggedByTest = $derived.by(() => buildFlaggedByTest(recentFlaggedRows));
	const reminders = $derived.by(() => buildReminders());
	const latestResults = $derived.by(() =>
		rows
			.filter((row) => row.result.value_numeric !== null)
			.sort((left, right) => right.examination.exam_date.localeCompare(left.examination.exam_date))
			.slice(0, 8)
	);

	function resultFlag(result: ExaminationResult): string | null {
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

	function isoDate(date: Date): string {
		const year = date.getFullYear();
		const month = String(date.getMonth() + 1).padStart(2, '0');
		const day = String(date.getDate()).padStart(2, '0');
		return `${year}-${month}-${day}`;
	}

	function addMonths(date: string, months: number): string {
		const [year, month, day] = date.split('-').map(Number);
		const target = new Date(Date.UTC(year, month - 1 + months, day));
		if (target.getUTCDate() !== day) target.setUTCDate(0);
		return [
			target.getUTCFullYear(),
			String(target.getUTCMonth() + 1).padStart(2, '0'),
			String(target.getUTCDate()).padStart(2, '0')
		].join('-');
	}

	function daysBetween(start: string, end: string): number {
		const startDate = new Date(`${start}T00:00:00Z`);
		const endDate = new Date(`${end}T00:00:00Z`);
		return Math.round((endDate.getTime() - startDate.getTime()) / 86_400_000);
	}

	function reminderStatus(
		lastDate: string | null,
		daysRemaining: number | null
	): ReminderItem['status'] {
		if (lastDate === null || daysRemaining === null) return 'missing';
		if (daysRemaining < 0) return 'due';
		if (daysRemaining <= 90) return 'soon';
		return 'ok';
	}

	function buildReminders(): ReminderItem[] {
		const statusOrder = { due: 0, missing: 1, soon: 2, ok: 3 };
		return reminderRules
			.map((rule) => {
				const latest = rows
					.filter((row) => rule.testKeys.includes(row.result.test_key))
					.sort((left, right) =>
						right.examination.exam_date.localeCompare(left.examination.exam_date)
					)[0];
				const lastDate = latest?.examination.exam_date ?? null;
				const dueDate = lastDate === null ? null : addMonths(lastDate, rule.cadenceMonths);
				const daysRemaining = dueDate === null ? null : daysBetween(todayDate, dueDate);
				return {
					...rule,
					lastDate,
					dueDate,
					daysRemaining,
					href: latest === undefined ? null : `/examinations/${latest.examination.id}`,
					status: reminderStatus(lastDate, daysRemaining)
				};
			})
			.sort(
				(left, right) =>
					statusOrder[left.status] - statusOrder[right.status] ||
					(left.dueDate ?? '9999-12-31').localeCompare(right.dueDate ?? '9999-12-31')
			);
	}

	function reminderStatusLabel(reminder: ReminderItem): string {
		if (reminder.status === 'missing') return 'Missing';
		if (reminder.daysRemaining === null) return 'Missing';
		if (reminder.daysRemaining < 0) return `${Math.abs(reminder.daysRemaining)}d overdue`;
		if (reminder.daysRemaining === 0) return 'Due today';
		if (reminder.daysRemaining <= 90) return `Due in ${reminder.daysRemaining}d`;
		return 'Scheduled';
	}

	function lastYearCutoffDate(): string {
		const cutoff = new Date();
		cutoff.setFullYear(cutoff.getFullYear() - 1);
		const year = cutoff.getFullYear();
		const month = String(cutoff.getMonth() + 1).padStart(2, '0');
		const day = String(cutoff.getDate()).padStart(2, '0');
		return `${year}-${month}-${day}`;
	}

	function trendPoints(key: string): TrendPoint[] {
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

	function buildYearlyBuckets() {
		const buckets = new Map<string, { year: string; total: number; flagged: number }>();
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

	function buildFlaggedByTest(sourceRows: ResultRow[]) {
		const buckets = new Map<
			string,
			{ key: string; name: string; count: number; high: number; low: number; unit: string | null }
		>();
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

	function trendBounds(points: TrendPoint[]): ChartBounds {
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

	function xFor(index: number, count: number): number {
		if (count <= 1) return chartWidth / 2;
		const plotWidth = chartWidth - chartPaddingX * 2;
		return chartPaddingX + (index / (count - 1)) * plotWidth;
	}

	function yFor(value: number, bounds: ChartBounds): number {
		const plotHeight = chartHeight - chartPaddingTop - chartPaddingBottom;
		const ratio = (value - bounds.min) / (bounds.max - bounds.min);
		return chartPaddingTop + (1 - ratio) * plotHeight;
	}

	function linePath(points: TrendPoint[]): string {
		if (points.length === 0) return '';
		const bounds = trendBounds(points);
		return points
			.map(
				(point, index) =>
					`${index === 0 ? 'M' : 'L'} ${xFor(index, points.length)} ${yFor(point.value, bounds)}`
			)
			.join(' ');
	}

	function latestPoint(points: TrendPoint[]): TrendPoint | null {
		return points[points.length - 1] ?? null;
	}

	function previousPoint(points: TrendPoint[]): TrendPoint | null {
		return points[points.length - 2] ?? null;
	}

	function deltaLabel(points: TrendPoint[]): string {
		const latest = latestPoint(points);
		const previous = previousPoint(points);
		if (!latest || !previous) return 'first record';
		const delta = latest.value - previous.value;
		const sign = delta > 0 ? '+' : '';
		return `${sign}${numberFormat.format(delta)} from prior`;
	}

	function resultValue(result: ExaminationResult): string {
		if (result.value_text) return result.value_text;
		if (result.value_numeric === null) return '-';
		return `${result.value_prefix ?? ''}${numberFormat.format(result.value_numeric)}`;
	}

	function rangeLabel(result: ExaminationResult): string {
		if (result.reference_min === null && result.reference_max === null) return '-';
		if (result.reference_min === null)
			return `<= ${numberFormat.format(result.reference_max ?? 0)}`;
		if (result.reference_max === null) return `>= ${numberFormat.format(result.reference_min)}`;
		return `${numberFormat.format(result.reference_min)}-${numberFormat.format(result.reference_max)}`;
	}

	function referenceLine(points: TrendPoint[], value: number | null): string | null {
		if (value === null) return null;
		const bounds = trendBounds(points);
		const y = yFor(value, bounds);
		return `${chartPaddingX},${y} ${chartWidth - chartPaddingX},${y}`;
	}

	function bucketWidth(value: number, max: number): string {
		if (max <= 0) return '0%';
		return `${Math.max(4, Math.round((value / max) * 100))}%`;
	}

	function normalBucketWidth(total: number, flagged: number, max: number): string {
		return bucketWidth(Math.max(total - flagged, 0), max);
	}

	function maxBucketTotal(): number {
		return Math.max(1, ...yearlyBuckets.map((bucket) => bucket.total));
	}
</script>

<section class="space-y-6">
	<div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
		<div>
			<h1 class="page-title">Dashboard</h1>
			<p class="text-sm text-surface-700">Lab trends, flagged results, and recent records.</p>
		</div>
		<a href="/examinations" class="btn preset-filled-primary-500 w-fit">
			<Stethoscope size={18} />
			<span>Add exam</span>
		</a>
	</div>

	<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
		{#each metrics as metric}
			<div class="metric-card">
				<div class="flex items-start justify-between gap-3">
					<div>
						<div class="text-sm text-surface-700">{metric.label}</div>
						<div class="mt-1 text-3xl font-bold">{metric.value}</div>
						<div class="mt-1 text-xs text-surface-600">{metric.meta}</div>
					</div>
					<metric.icon class="text-primary-600" size={28} />
				</div>
			</div>
		{/each}
	</div>

	<div class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_22rem]">
		<section class="space-y-3">
			<div class="flex items-center justify-between gap-3">
				<h2 class="section-title">Result trends</h2>
				<TrendingUp class="text-surface-500" size={18} />
			</div>
			{#if trendCards.length === 0}
				<div
					class="rounded-md border border-dashed border-surface-300 bg-white p-6 text-sm text-surface-700"
				>
					No numeric lab results yet.
				</div>
			{:else}
				<div class="grid gap-3 lg:grid-cols-2">
					{#each trendCards as trend}
						{@const latest = latestPoint(trend.points)}
						{@const minLine = latest ? referenceLine(trend.points, latest.referenceMin) : null}
						{@const maxLine = latest ? referenceLine(trend.points, latest.referenceMax) : null}
						<div class="dashboard-panel">
							<div class="flex items-start justify-between gap-3">
								<div>
									<h3 class="text-sm font-bold">{trend.label}</h3>
									<div class="text-xs text-surface-600">{trend.points.length} records</div>
								</div>
								{#if latest}
									<div class="text-right">
										<div class="text-lg font-bold">
											{numberFormat.format(latest.value)}
											<span class="text-xs font-medium text-surface-600">{latest.unit ?? ''}</span>
										</div>
										<div class="text-xs text-surface-600">{deltaLabel(trend.points)}</div>
									</div>
								{/if}
							</div>
							<svg
								class="trend-chart"
								viewBox={`0 0 ${chartWidth} ${chartHeight}`}
								role="img"
								aria-label={`${trend.label} trend`}
							>
								<rect
									x={chartPaddingX}
									y={chartPaddingTop}
									width={chartWidth - chartPaddingX * 2}
									height={chartHeight - chartPaddingTop - chartPaddingBottom}
									class="trend-plot"
								/>
								{#if minLine}
									<polyline points={minLine} class="trend-reference" />
								{/if}
								{#if maxLine}
									<polyline points={maxLine} class="trend-reference" />
								{/if}
								<path d={linePath(trend.points)} class="trend-line" />
								{#each trend.points as point, index}
									{@const bounds = trendBounds(trend.points)}
									<a href={point.href} aria-label={`${trend.label} ${formatDate(point.date)}`}>
										<circle
											cx={xFor(index, trend.points.length)}
											cy={yFor(point.value, bounds)}
											r={point.flag ? 4.8 : 4}
											class={point.flag ? 'trend-point trend-point-flagged' : 'trend-point'}
										/>
									</a>
								{/each}
							</svg>
							<div class="flex items-center justify-between text-xs text-surface-600">
								<span>{formatDate(trend.points[0]?.date)}</span>
								<span>{formatDate(trend.points[trend.points.length - 1]?.date)}</span>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</section>

		<aside class="space-y-4">
			<section class="dashboard-panel">
				<div class="mb-3 flex items-center justify-between gap-3">
					<div>
						<h2 class="section-title">Exam reminders</h2>
						<div class="text-xs text-surface-600">TSH + vegetarian labs</div>
					</div>
					<CalendarCheck class="text-surface-500" size={18} />
				</div>
				<div class="divide-y divide-surface-200">
					{#each reminders as reminder}
						<div class="grid gap-2 py-3">
							<div class="flex items-start justify-between gap-3">
								<div class="min-w-0">
									{#if reminder.href}
										<a href={reminder.href} class="font-semibold hover:underline"
											>{reminder.label}</a
										>
									{:else}
										<div class="font-semibold">{reminder.label}</div>
									{/if}
									<div class="text-xs text-surface-600">
										Every {reminder.cadenceMonths} months · {reminder.reason}
									</div>
								</div>
								<span class={['reminder-status', `reminder-status-${reminder.status}`]}>
									{reminderStatusLabel(reminder)}
								</span>
							</div>
							<div class="grid grid-cols-2 gap-2 text-xs text-surface-700">
								<div>
									<span class="font-semibold">Last</span>
									<span>{formatDate(reminder.lastDate)}</span>
								</div>
								<div>
									<span class="font-semibold">Due</span>
									<span>{formatDate(reminder.dueDate)}</span>
								</div>
							</div>
						</div>
					{/each}
				</div>
			</section>

			<section class="dashboard-panel">
				<div class="mb-3 flex items-center justify-between gap-3">
					<h2 class="section-title">Exam activity</h2>
					<CalendarDays class="text-surface-500" size={18} />
				</div>
				{#if yearlyBuckets.length === 0}
					<p class="text-sm text-surface-700">No examinations recorded.</p>
				{:else}
					<div class="space-y-3">
						{#each yearlyBuckets as bucket}
							<div class="grid grid-cols-[3.5rem_1fr_2rem] items-center gap-2 text-sm">
								<div class="font-semibold">{bucket.year}</div>
								<div class="activity-track">
									<span
										class="activity-bar activity-bar-normal"
										style={`width: ${normalBucketWidth(bucket.total, bucket.flagged, maxBucketTotal())}`}
									></span>
									<span
										class="activity-bar activity-bar-flagged"
										style={`width: ${bucketWidth(bucket.flagged, maxBucketTotal())}`}
									></span>
								</div>
								<div class="text-right text-surface-700">{bucket.total}</div>
							</div>
						{/each}
					</div>
				{/if}
			</section>

			<section class="dashboard-panel">
				<div class="mb-3 flex items-center justify-between gap-3">
					<div>
						<h2 class="section-title">Flagged mix</h2>
						<div class="text-xs text-surface-600">Since {formatDate(recentFlaggedCutoff)}</div>
					</div>
					<Gauge class="text-surface-500" size={18} />
				</div>
				{#if flaggedByTest.length === 0}
					<p class="text-sm text-surface-700">No flagged results in the last year.</p>
				{:else}
					<div class="space-y-3">
						{#each flaggedByTest as item}
							<div>
								<div class="flex items-center justify-between gap-3 text-sm">
									<span class="font-semibold">{item.name}</span>
									<span class="text-surface-700">{item.count}</span>
								</div>
								<div class="mt-1 flex h-2 overflow-hidden rounded-sm bg-surface-100">
									<span
										class="bg-red-600"
										style={`width: ${Math.round((item.high / item.count) * 100)}%`}
									></span>
									<span
										class="bg-amber-500"
										style={`width: ${Math.round((item.low / item.count) * 100)}%`}
									></span>
								</div>
								<div class="mt-1 text-xs text-surface-600">
									{item.high} high, {item.low} low {item.unit ? `(${item.unit})` : ''}
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</section>
		</aside>
	</div>

	<div class="grid gap-4 lg:grid-cols-2">
		<section class="dashboard-panel">
			<div class="mb-3 flex items-center justify-between gap-3">
				<div>
					<h2 class="section-title">Recent flagged results</h2>
					<div class="text-xs text-surface-600">Since {formatDate(recentFlaggedCutoff)}</div>
				</div>
				<AlertTriangle class="text-surface-500" size={18} />
			</div>
			{#if recentFlaggedRows.length === 0}
				<p class="text-sm text-surface-700">No flagged results in the last year.</p>
			{:else}
				<div class="table-cards overflow-hidden rounded-md border border-surface-200">
					<table class="data-table compact-table">
						<thead>
							<tr>
								<th>Result</th>
								<th>Value</th>
								<th>Range</th>
								<th>Date</th>
							</tr>
						</thead>
						<tbody>
							{#each recentFlaggedRows as row}
								<tr class="result-row-abnormal">
									<td data-label="Result">
										<a
											href={`/examinations/${row.examination.id}`}
											class="font-semibold hover:underline"
										>
											{row.result.name}
										</a>
									</td>
									<td data-label="Value">
										{resultValue(row.result)}
										{row.result.unit ?? ''}
										<span class="result-flag-abnormal ml-1">{row.flag}</span>
									</td>
									<td data-label="Range">{rangeLabel(row.result)}</td>
									<td data-label="Date">{formatDate(row.examination.exam_date)}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>

		<section class="dashboard-panel">
			<div class="mb-3 flex items-center justify-between gap-3">
				<h2 class="section-title">Latest numeric results</h2>
				<Activity class="text-surface-500" size={18} />
			</div>
			{#if latestResults.length === 0}
				<p class="text-sm text-surface-700">No numeric lab results.</p>
			{:else}
				<div class="divide-y divide-surface-200">
					{#each latestResults as row}
						<a
							href={`/examinations/${row.examination.id}`}
							class="grid grid-cols-[1fr_auto] gap-3 py-2 text-sm hover:bg-surface-50"
						>
							<span class="min-w-0">
								<span class="block truncate font-semibold">{row.result.name}</span>
								<span class="block text-xs text-surface-600"
									>{formatDate(row.examination.exam_date)}</span
								>
							</span>
							<span class="text-right font-semibold">
								{resultValue(row.result)}
								<span class="text-xs font-medium text-surface-600">{row.result.unit ?? ''}</span>
							</span>
						</a>
					{/each}
				</div>
			{/if}
		</section>
	</div>

	<section class="space-y-3">
		<div class="flex items-center justify-between gap-3">
			<h2 class="section-title">Recent documents</h2>
			<a href="/documents" class="text-sm font-semibold text-primary-700 hover:underline">
				All documents
			</a>
		</div>

		{#if data.overview.recent_documents.length === 0}
			<div
				class="rounded-md border border-dashed border-surface-300 bg-white p-6 text-sm text-surface-700"
			>
				No documents uploaded.
			</div>
		{:else}
			<div class="table-cards overflow-hidden rounded-md border border-surface-200 bg-white">
				<table class="data-table">
					<thead>
						<tr>
							<th>Title</th>
							<th>Issued</th>
							<th>Type</th>
							<th>Size</th>
						</tr>
					</thead>
					<tbody>
						{#each data.overview.recent_documents as document}
							<tr>
								<td data-label="Title">
									<a
										href={`/api/documents/${document.id}/download`}
										class="font-semibold text-primary-700 hover:underline"
									>
										{document.title}
									</a>
								</td>
								<td data-label="Issued">{formatDate(document.issued_at)}</td>
								<td data-label="Type">{document.document_type}</td>
								<td data-label="Size">{formatBytes(document.size_bytes)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
</section>
