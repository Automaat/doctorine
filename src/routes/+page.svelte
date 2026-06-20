<script lang="ts">
	import { formatBytes, formatDate } from '$lib/format';
	import type { ExaminationResult } from '$lib/types';
	import {
		buildFlaggedByTest,
		buildReminders,
		buildResultRows,
		buildYearlyBuckets,
		chartHeight,
		chartPaddingBottom,
		chartPaddingTop,
		chartPaddingX,
		chartWidth,
		isoDate,
		lastYearCutoff,
		latestPoint,
		linePath,
		previousPoint,
		referenceLine,
		resultFlag,
		trackedTests,
		trendBounds,
		trendPoints,
		xFor,
		yFor
	} from '$lib/dashboard';
	import type { ReminderItem, ResultRow, TrendCard, TrendPoint } from '$lib/dashboard';
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

	const numberFormat = new Intl.NumberFormat('pl-PL', { maximumFractionDigits: 2 });
	const todayDate = isoDate(new Date());

	let { data }: { data: PageData } = $props();

	const examinations = $derived(
		[...(data.examinations ?? [])].sort((left, right) =>
			right.exam_date.localeCompare(left.exam_date)
		)
	);
	const rows = $derived.by<ResultRow[]>(() => buildResultRows(examinations));
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
				points: trendPoints(rows, test.key)
			}))
			.filter((card) => card.points.length > 0)
	);
	const yearlyBuckets = $derived.by(() => buildYearlyBuckets(examinations));
	const recentFlaggedCutoff = lastYearCutoff(new Date());
	const recentFlaggedRows = $derived.by(() =>
		[...flaggedRows]
			.filter((row) => row.examination.exam_date >= recentFlaggedCutoff)
			.sort((left, right) => right.examination.exam_date.localeCompare(left.examination.exam_date))
	);
	const flaggedByTest = $derived.by(() => buildFlaggedByTest(recentFlaggedRows));
	const reminders = $derived.by(() => buildReminders(rows, todayDate));
	const latestResults = $derived.by(() =>
		rows
			.filter((row) => row.result.value_numeric !== null)
			.sort((left, right) => right.examination.exam_date.localeCompare(left.examination.exam_date))
			.slice(0, 8)
	);

	function reminderStatusLabel(reminder: ReminderItem): string {
		if (reminder.status === 'directed') return 'As needed';
		if (reminder.status === 'complete') return 'Recorded';
		if (reminder.status === 'missing')
			return reminder.kind === 'one_time' ? 'Not recorded' : 'Missing';
		if (reminder.daysRemaining === null) return 'Missing';
		if (reminder.daysRemaining < 0) return `${Math.abs(reminder.daysRemaining)}d overdue`;
		if (reminder.daysRemaining === 0) return 'Due today';
		if (reminder.daysRemaining <= 90) return `Due in ${reminder.daysRemaining}d`;
		return 'Scheduled';
	}

	function reminderScheduleLabel(reminder: ReminderItem): string {
		if (reminder.kind === 'recurring' && reminder.cadenceMonths !== null) {
			return `Every ${reminder.cadenceMonths} months`;
		}
		if (reminder.kind === 'one_time') return 'One-time';
		return 'Doctor-led';
	}

	function reminderStatusClass(reminder: ReminderItem): string {
		return `reminder-status reminder-status-${reminder.status}`;
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

	function bucketWidth(value: number, max: number): string {
		if (value <= 0) return '0%';
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
						{@const bounds = trendBounds(trend.points)}
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
						<div class="text-xs text-surface-600">Routine + doctor-directed checks</div>
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
										{reminderScheduleLabel(reminder)} · {reminder.reason}
									</div>
								</div>
								<span class={reminderStatusClass(reminder)}>
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
