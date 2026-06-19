<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { formatDate } from '$lib/format';
	import type { ExaminationResult } from '$lib/types';
	import { Plus, Trash2 } from 'lucide-svelte';
	import type { PageData } from './$types';

	type ResultDraft = {
		id: number;
		testKey: string;
		name: string;
		value: string;
		valuePrefix: string;
		unit: string;
		referenceMin: string;
		referenceMax: string;
	};

	type ResultPayload = {
		test_key: string;
		name: string;
		value_text: string | null;
		value_numeric: number | null;
		value_prefix: string | null;
		unit: string | null;
		reference_min: number | null;
		reference_max: number | null;
		display_order: number;
	};

	let { data }: { data: PageData } = $props();
	let error = $state('');
	let saving = $state(false);
	let resultDrafts = $state<ResultDraft[]>([]);
	let nextResultID = 1;
	const numberFormat = new Intl.NumberFormat('pl-PL', { maximumFractionDigits: 3 });
	const unitOptions = [
		'%',
		'g/dl',
		'fl',
		'pg',
		'tys/ul',
		'mln/ul',
		'mg/dl',
		'g/ml',
		'mmol/l',
		'pg/ml',
		'ng/ml',
		'U/l',
		'umol/l',
		'ml/min/1.73m2',
		'uIU/ml',
		'mg/l'
	];

	function formValue(form: FormData, key: string): string | null {
		const raw = String(form.get(key) ?? '').trim();
		return raw === '' ? null : raw;
	}

	function emptyResultDraft(): ResultDraft {
		const id = nextResultID;
		nextResultID += 1;
		return {
			id,
			testKey: '',
			name: '',
			value: '',
			valuePrefix: '',
			unit: '',
			referenceMin: '',
			referenceMax: ''
		};
	}

	function addResultRow() {
		resultDrafts = [...resultDrafts, emptyResultDraft()];
	}

	function removeResultRow(id: number) {
		resultDrafts = resultDrafts.filter((result) => result.id !== id);
	}

	function resetResults() {
		resultDrafts = [];
		nextResultID = 1;
	}

	function parseOptionalNumber(raw: string): number | null {
		const cleaned = raw.trim().replace(',', '.');
		if (cleaned === '') return null;
		const parsed = Number(cleaned);
		return Number.isFinite(parsed) ? parsed : null;
	}

	function parseNumericValue(raw: string): number | null {
		const cleaned = raw
			.trim()
			.replace(',', '.')
			.replace(/^[<>]=?/, '');
		if (cleaned === '') return null;
		const parsed = Number(cleaned);
		return Number.isFinite(parsed) ? parsed : null;
	}

	function inferredPrefix(raw: string, selected: string): string | null {
		if (selected !== '') return selected;
		const match = raw.trim().match(/^(<=|>=|<|>)/);
		return match?.[1] ?? null;
	}

	function slugifyKey(raw: string): string {
		return raw
			.normalize('NFD')
			.replace(/[\u0300-\u036f]/g, '')
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '_')
			.replace(/^_+|_+$/g, '');
	}

	function buildResults(): { results: ResultPayload[]; detail: string } {
		const results: ResultPayload[] = [];
		const keys = new Set<string>();
		for (const [index, draft] of resultDrafts.entries()) {
			const name = draft.name.trim();
			const rawValue = draft.value.trim();
			const hasAnyValue = [
				draft.testKey,
				draft.name,
				draft.value,
				draft.valuePrefix,
				draft.unit,
				draft.referenceMin,
				draft.referenceMax
			].some((field) => field.trim() !== '');
			if (!hasAnyValue) continue;
			if (name === '' || rawValue === '') {
				return { results: [], detail: 'Each result needs name and value' };
			}
			const testKey = slugifyKey(draft.testKey || name);
			if (testKey === '') return { results: [], detail: 'Each result needs a valid key' };
			if (keys.has(testKey)) return { results: [], detail: 'Result keys must be unique' };
			keys.add(testKey);
			const referenceMin = parseOptionalNumber(draft.referenceMin);
			const referenceMax = parseOptionalNumber(draft.referenceMax);
			if (draft.referenceMin.trim() !== '' && referenceMin === null) {
				return { results: [], detail: 'Result minimum must be numeric' };
			}
			if (draft.referenceMax.trim() !== '' && referenceMax === null) {
				return { results: [], detail: 'Result maximum must be numeric' };
			}
			results.push({
				test_key: testKey,
				name,
				value_text: rawValue,
				value_numeric: parseNumericValue(rawValue),
				value_prefix: inferredPrefix(rawValue, draft.valuePrefix),
				unit: draft.unit.trim() || null,
				reference_min: referenceMin,
				reference_max: referenceMax,
				display_order: index + 1
			});
		}
		return { results, detail: '' };
	}

	async function createExamination(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		const { results, detail } = buildResults();
		if (detail !== '') {
			error = detail;
			return;
		}
		saving = true;
		const formEl = event.currentTarget as HTMLFormElement;
		const form = new FormData(formEl);
		const payload = {
			title: formValue(form, 'title'),
			exam_date: formValue(form, 'exam_date'),
			category: formValue(form, 'category'),
			facility: formValue(form, 'facility'),
			result_status: formValue(form, 'result_status'),
			summary: formValue(form, 'summary'),
			notes: formValue(form, 'notes'),
			results
		};
		const response = await fetch('/api/examinations', {
			method: 'POST',
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify(payload)
		});
		saving = false;
		if (!response.ok) {
			const body = (await response.json().catch(() => null)) as { detail?: string } | null;
			error = body?.detail ?? 'Save failed';
			return;
		}
		formEl.reset();
		resetResults();
		await invalidateAll();
	}

	function resultValue(result: ExaminationResult): string {
		if (result.value_text) return result.value_text;
		if (result.value_numeric === null) return '-';
		return `${result.value_prefix ?? ''}${numberFormat.format(result.value_numeric)}`;
	}

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

	function resultRange(result: ExaminationResult): string {
		if (result.reference_min === null && result.reference_max === null) return '-';
		if (result.reference_min === null)
			return `<= ${numberFormat.format(result.reference_max ?? 0)}`;
		if (result.reference_max === null) return `>= ${numberFormat.format(result.reference_min)}`;
		return `${numberFormat.format(result.reference_min)}-${numberFormat.format(result.reference_max)}`;
	}

	function isBeyondNorm(result: ExaminationResult): boolean {
		return resultFlag(result) !== null;
	}
</script>

<section class="space-y-6">
	<div>
		<h1 class="page-title">Examinations</h1>
		<p class="text-sm text-surface-700">Lab work, imaging, consultations, and result status.</p>
	</div>

	<form
		class="grid gap-4 rounded-md border border-surface-200 bg-white p-4 md:grid-cols-2"
		onsubmit={createExamination}
	>
		<label class="label">
			<span class="text-sm font-semibold">Title</span>
			<input name="title" class="input" maxlength="200" required />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Exam date</span>
			<input name="exam_date" type="date" class="input" required />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Category</span>
			<input name="category" class="input" value="general" maxlength="80" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Facility</span>
			<input name="facility" class="input" maxlength="200" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Result status</span>
			<select name="result_status" class="select">
				<option value="unknown">Unknown</option>
				<option value="normal">Normal</option>
				<option value="attention">Attention</option>
				<option value="urgent">Urgent</option>
			</select>
		</label>
		<label class="label md:col-span-2">
			<span class="text-sm font-semibold">Summary</span>
			<textarea name="summary" class="textarea" rows="3"></textarea>
		</label>
		<label class="label md:col-span-2">
			<span class="text-sm font-semibold">Notes</span>
			<textarea name="notes" class="textarea" rows="3"></textarea>
		</label>
		<div class="space-y-3 border-t border-surface-200 pt-4 md:col-span-2">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<h2 class="section-title">Structured results</h2>
				<button type="button" class="btn preset-tonal-primary-500" onclick={addResultRow}>
					<Plus size={18} />
					<span>Add result</span>
				</button>
			</div>
			{#if resultDrafts.length > 0}
				<div class="space-y-3">
					{#each resultDrafts as result}
						<div class="grid gap-3 border-t border-surface-200 pt-3 md:grid-cols-12">
							<label class="label md:col-span-2">
								<span class="text-sm font-semibold">Key</span>
								<input
									bind:value={result.testKey}
									class="input"
									maxlength="120"
									placeholder="ast"
								/>
							</label>
							<label class="label md:col-span-3">
								<span class="text-sm font-semibold">Name</span>
								<input bind:value={result.name} class="input" maxlength="200" placeholder="AST" />
							</label>
							<label class="label md:col-span-2">
								<span class="text-sm font-semibold">Value</span>
								<input bind:value={result.value} class="input" maxlength="120" placeholder="44" />
							</label>
							<label class="label md:col-span-1">
								<span class="text-sm font-semibold">Prefix</span>
								<select bind:value={result.valuePrefix} class="select">
									<option value="">=</option>
									<option value="<">&lt;</option>
									<option value=">">&gt;</option>
									<option value="<=">&lt;=</option>
									<option value=">=">&gt;=</option>
								</select>
							</label>
							<label class="label md:col-span-2">
								<span class="text-sm font-semibold">Unit</span>
								<select bind:value={result.unit} class="select">
									<option value="">None</option>
									{#each unitOptions as unit}
										<option value={unit}>{unit}</option>
									{/each}
								</select>
							</label>
							<label class="label md:col-span-1">
								<span class="text-sm font-semibold">Min</span>
								<input bind:value={result.referenceMin} class="input" inputmode="decimal" />
							</label>
							<label class="label md:col-span-1">
								<span class="text-sm font-semibold">Max</span>
								<input bind:value={result.referenceMax} class="input" inputmode="decimal" />
							</label>
							<div class="flex items-end md:col-span-12">
								<button
									type="button"
									class="btn preset-tonal-error-500"
									aria-label="Remove result"
									onclick={() => removeResultRow(result.id)}
								>
									<Trash2 size={18} />
								</button>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
		<div class="flex items-center gap-3 md:col-span-2">
			<button type="submit" class="btn preset-filled-primary-500" disabled={saving}>
				<Plus size={18} />
				<span>{saving ? 'Saving...' : 'Add examination'}</span>
			</button>
			{#if error}
				<p class="text-sm text-error-700" aria-live="polite">{error}</p>
			{/if}
		</div>
	</form>

	<div class="table-cards overflow-hidden rounded-md border border-surface-200 bg-white">
		<table class="data-table">
			<thead>
				<tr>
					<th>Exam</th>
					<th>Date</th>
					<th>Status</th>
					<th>Facility</th>
				</tr>
			</thead>
			<tbody>
				{#if data.examinations.length === 0}
					<tr>
						<td colspan="4" class="text-sm text-surface-700">No examinations recorded.</td>
					</tr>
				{:else}
					{#each data.examinations as examination}
						<tr>
							<td data-label="Exam">
								<div class="font-semibold">{examination.title}</div>
								{#if examination.summary}
									<div class="text-sm text-surface-700">{examination.summary}</div>
								{/if}
								{#if examination.results.length > 0}
									<div class="mt-3 overflow-x-auto rounded-md border border-surface-200">
										<div
											class="grid min-w-[40rem] grid-cols-[minmax(12rem,1fr)_7rem_7rem_9rem_4rem] bg-surface-50 text-sm font-semibold"
										>
											<div class="px-3 py-2">Result</div>
											<div class="px-3 py-2">Value</div>
											<div class="px-3 py-2">Unit</div>
											<div class="px-3 py-2">Range</div>
											<div class="px-3 py-2">Flag</div>
										</div>
										{#each examination.results as result}
											<div
												class={[
													'grid min-w-[40rem] grid-cols-[minmax(12rem,1fr)_7rem_7rem_9rem_4rem] border-t border-surface-200 text-sm',
													isBeyondNorm(result) ? 'bg-error-50 text-error-900' : ''
												]}
											>
												<div class="px-3 py-2">{result.name}</div>
												<div class="px-3 py-2 font-semibold">{resultValue(result)}</div>
												<div class="px-3 py-2">{result.unit ?? '-'}</div>
												<div class="px-3 py-2">{resultRange(result)}</div>
												<div class="px-3 py-2">
													{#if resultFlag(result)}
														<span class="font-semibold text-error-700">{resultFlag(result)}</span>
													{:else}
														-
													{/if}
												</div>
											</div>
										{/each}
									</div>
								{/if}
							</td>
							<td data-label="Date">{formatDate(examination.exam_date)}</td>
							<td data-label="Status">{examination.result_status}</td>
							<td data-label="Facility">{examination.facility ?? '-'}</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>
</section>
