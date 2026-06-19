<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import type { ResultDefinition } from '$lib/types';
	import { Plus, Trash2 } from 'lucide-svelte';
	import type { PageData } from './$types';

	type ResultDraft = {
		id: number;
		definitionID: number | null;
		testKey: string;
		name: string;
		value: string;
		unit: string;
	};

	type ResultPayload = {
		definition_id: number | null;
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
	let activeResultID = $state<number | null>(null);
	let deletingId = $state<number | null>(null);
	let nextResultID = 1;
	const fallbackUnitOptions = [
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
			definitionID: null,
			testKey: '',
			name: '',
			value: '',
			unit: ''
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

	function definitionOptions(): ResultDefinition[] {
		const byKey = new Map<string, ResultDefinition>();
		for (const examination of data.examinations) {
			for (const result of examination.results) {
				const definition =
					result.definition ??
					(result.definition_id === null
						? null
						: {
								id: result.definition_id,
								test_key: result.test_key,
								name: result.name,
								unit: result.unit,
								reference_min: result.reference_min,
								reference_max: result.reference_max,
								category: 'laboratory',
								created_at: result.created_at,
								updated_at: result.updated_at
							});
				if (definition !== null) byKey.set(definition.test_key, definition);
			}
		}
		return [...byKey.values()].sort((left, right) =>
			left.name.localeCompare(right.name, 'pl', { sensitivity: 'base' })
		);
	}

	function unitOptions(): string[] {
		const units = new Set(fallbackUnitOptions);
		for (const definition of definitionOptions()) {
			if (definition.unit) units.add(definition.unit);
		}
		return [...units].sort((left, right) => left.localeCompare(right, 'pl'));
	}

	function draftNumber(value: number | null): string {
		return value === null ? '' : String(value);
	}

	function findDefinition(raw: string): ResultDefinition | null {
		const cleaned = raw.trim().toLowerCase();
		if (cleaned === '') return null;
		return (
			definitionOptions().find(
				(definition) =>
					definition.name.toLowerCase() === cleaned || definition.test_key.toLowerCase() === cleaned
			) ?? null
		);
	}

	function definitionLabel(definition: ResultDefinition): string {
		return definition.unit ? `${definition.test_key} (${definition.unit})` : definition.test_key;
	}

	function definitionRange(definition: ResultDefinition): string | null {
		if (definition.reference_min === null && definition.reference_max === null) return null;
		if (definition.reference_min === null) return `<= ${draftNumber(definition.reference_max)}`;
		if (definition.reference_max === null) return `>= ${draftNumber(definition.reference_min)}`;
		return `${draftNumber(definition.reference_min)}-${draftNumber(definition.reference_max)}`;
	}

	function filteredDefinitions(result: ResultDraft): ResultDefinition[] {
		const cleaned = result.name.trim().toLowerCase();
		const options = definitionOptions();
		if (cleaned === '') return options.slice(0, 8);
		return options
			.filter(
				(definition) =>
					definition.name.toLowerCase().includes(cleaned) ||
					definition.test_key.toLowerCase().includes(cleaned)
			)
			.slice(0, 8);
	}

	function applyDefinition(result: ResultDraft, definition: ResultDefinition) {
		result.definitionID = definition.id;
		result.testKey = definition.test_key;
		result.name = definition.name;
		result.unit = definition.unit ?? '';
	}

	function selectDefinition(result: ResultDraft, definition: ResultDefinition, event?: Event) {
		event?.preventDefault();
		applyDefinition(result, definition);
		activeResultID = null;
	}

	function syncDefinition(result: ResultDraft) {
		const definition = findDefinition(result.name);
		if (definition === null) {
			result.definitionID = null;
			result.testKey = '';
			return;
		}
		applyDefinition(result, definition);
	}

	function syncDefinitionInput(result: ResultDraft, event: Event) {
		result.name = (event.currentTarget as HTMLInputElement).value;
		syncDefinition(result);
		activeResultID = result.id;
	}

	function closeDefinitionMenu() {
		setTimeout(() => {
			activeResultID = null;
		}, 120);
	}

	function handleDefinitionKeydown(result: ResultDraft, event: KeyboardEvent) {
		if (event.key === 'Escape') {
			activeResultID = null;
			return;
		}
		if (event.key !== 'Enter' || activeResultID !== result.id) return;
		const [firstDefinition] = filteredDefinitions(result);
		if (firstDefinition === undefined) return;
		event.preventDefault();
		selectDefinition(result, firstDefinition);
	}

	function openDefinitionMenu(result: ResultDraft) {
		activeResultID = result.id;
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

	function inferredPrefix(raw: string): string | null {
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
			const hasAnyValue = [draft.testKey, draft.name, draft.value, draft.unit].some(
				(field) => field.trim() !== ''
			);
			if (!hasAnyValue) continue;
			if (name === '' || rawValue === '') {
				return { results: [], detail: 'Each result needs name and value' };
			}
			const testKey = slugifyKey(draft.testKey || name);
			if (testKey === '') return { results: [], detail: 'Each result needs a valid key' };
			if (keys.has(testKey)) return { results: [], detail: 'Result keys must be unique' };
			keys.add(testKey);
			results.push({
				definition_id: draft.definitionID,
				test_key: testKey,
				name,
				value_text: rawValue,
				value_numeric: parseNumericValue(rawValue),
				value_prefix: inferredPrefix(rawValue),
				unit: draft.unit.trim() || null,
				reference_min: null,
				reference_max: null,
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

	async function removeExamination(id: number, title: string) {
		if (!window.confirm(`Remove examination "${title}"?`)) return;
		error = '';
		deletingId = id;
		const response = await fetch(`/api/examinations/${id}`, { method: 'DELETE' });
		deletingId = null;
		if (!response.ok) {
			const body = (await response.json().catch(() => null)) as { detail?: string } | null;
			error = body?.detail ?? 'Delete failed';
			return;
		}
		await invalidateAll();
	}

	function examinationSummary(summary: string | null): string {
		return summary ?? 'No summary.';
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
							<div class="label md:col-span-5">
								<span class="text-sm font-semibold">Result</span>
								<div class="autocomplete-field">
									<input
										value={result.name}
										class="input"
										maxlength="200"
										placeholder="Start typing"
										autocomplete="off"
										role="combobox"
										aria-autocomplete="list"
										aria-expanded={activeResultID === result.id}
										aria-controls={`result-definition-menu-${result.id}`}
										onfocus={() => openDefinitionMenu(result)}
										onblur={closeDefinitionMenu}
										onkeydown={(event) => handleDefinitionKeydown(result, event)}
										oninput={(event) => syncDefinitionInput(result, event)}
									/>
									{#if activeResultID === result.id && filteredDefinitions(result).length > 0}
										<div
											id={`result-definition-menu-${result.id}`}
											class="autocomplete-menu"
											role="listbox"
										>
											{#each filteredDefinitions(result) as definition}
												<button
													type="button"
													class="autocomplete-option"
													role="option"
													aria-selected={definition.id === result.definitionID}
													onmousedown={(event) => selectDefinition(result, definition, event)}
												>
													<span class="autocomplete-option-title">{definition.name}</span>
													<span class="autocomplete-option-meta">
														<span>{definitionLabel(definition)}</span>
														{#if definition.reference_min !== null || definition.reference_max !== null}
															<span>{definitionRange(definition)}</span>
														{/if}
													</span>
												</button>
											{/each}
										</div>
									{/if}
								</div>
							</div>
							<label class="label md:col-span-3">
								<span class="text-sm font-semibold">Value</span>
								<input
									bind:value={result.value}
									class="input"
									maxlength="120"
									placeholder="44 or <5"
								/>
							</label>
							<label class="label md:col-span-3">
								<span class="text-sm font-semibold">Unit</span>
								<select bind:value={result.unit} class="select">
									<option value="">None</option>
									{#each unitOptions() as unit}
										<option value={unit}>{unit}</option>
									{/each}
								</select>
							</label>
							<div class="flex items-end md:col-span-1">
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

	<div class="overflow-hidden rounded-md border border-surface-200 bg-white">
		{#if data.examinations.length === 0}
			<p class="p-4 text-sm text-surface-700">No examinations recorded.</p>
		{:else}
			{#each data.examinations as examination}
				<div class="flex items-center gap-3 border-b border-surface-200 last:border-b-0">
					<a href={`/examinations/${examination.id}`} class="block min-w-0 flex-1 p-4">
						<div class="font-semibold text-surface-950">{examination.title}</div>
						<div class="text-sm text-surface-700">{examinationSummary(examination.summary)}</div>
					</a>
					<button
						type="button"
						class="btn-icon btn-icon-sm mr-3"
						aria-label="Delete examination"
						disabled={deletingId === examination.id}
						onclick={() => removeExamination(examination.id, examination.title)}
					>
						<Trash2 size={18} />
					</button>
				</div>
			{/each}
		{/if}
	</div>
</section>
