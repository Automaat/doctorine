<script lang="ts">
	import { goto } from '$app/navigation';
	import { formatDate } from '$lib/format';
	import type { ExaminationResult } from '$lib/types';
	import { ArrowLeft, Trash2 } from 'lucide-svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	let deleting = $state(false);
	let error = $state('');
	const numberFormat = new Intl.NumberFormat('pl-PL', { maximumFractionDigits: 3 });

	async function removeExamination() {
		if (!window.confirm(`Remove examination "${data.examination.title}"?`)) return;
		error = '';
		deleting = true;
		const response = await fetch(`/api/examinations/${data.examination.id}`, { method: 'DELETE' });
		deleting = false;
		if (!response.ok) {
			const body = (await response.json().catch(() => null)) as { detail?: string } | null;
			error = body?.detail ?? 'Delete failed';
			return;
		}
		await goto('/examinations');
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
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div class="space-y-2">
			<a href="/examinations" class="btn preset-tonal-surface-500 w-fit">
				<ArrowLeft size={18} />
				<span>Back</span>
			</a>
			<div>
				<h1 class="page-title">{data.examination.title}</h1>
				{#if data.examination.summary}
					<p class="text-sm text-surface-700">{data.examination.summary}</p>
				{/if}
			</div>
		</div>
		<button
			type="button"
			class="btn preset-tonal-error-500"
			disabled={deleting}
			onclick={removeExamination}
		>
			<Trash2 size={18} />
			<span>{deleting ? 'Removing...' : 'Remove'}</span>
		</button>
	</div>

	{#if error}
		<p class="text-sm text-error-700" aria-live="polite">{error}</p>
	{/if}

	<div class="grid gap-4 rounded-md border border-surface-200 bg-white p-4 md:grid-cols-4">
		<div>
			<div class="text-xs font-semibold uppercase text-surface-700">Date</div>
			<div>{formatDate(data.examination.exam_date)}</div>
		</div>
		<div>
			<div class="text-xs font-semibold uppercase text-surface-700">Status</div>
			<div>{data.examination.result_status}</div>
		</div>
		<div>
			<div class="text-xs font-semibold uppercase text-surface-700">Category</div>
			<div>{data.examination.category}</div>
		</div>
		<div>
			<div class="text-xs font-semibold uppercase text-surface-700">Facility</div>
			<div>{data.examination.facility ?? '-'}</div>
		</div>
		{#if data.examination.notes}
			<div class="md:col-span-4">
				<div class="text-xs font-semibold uppercase text-surface-700">Notes</div>
				<div class="whitespace-pre-wrap">{data.examination.notes}</div>
			</div>
		{/if}
	</div>

	<div class="overflow-x-auto rounded-md border border-surface-200 bg-white">
		<div
			class="grid min-w-[40rem] grid-cols-[minmax(12rem,1fr)_7rem_7rem_9rem_4rem] bg-surface-50 text-sm font-semibold"
		>
			<div class="px-3 py-2">Result</div>
			<div class="px-3 py-2">Value</div>
			<div class="px-3 py-2">Unit</div>
			<div class="px-3 py-2">Range</div>
			<div class="px-3 py-2">Flag</div>
		</div>
		{#if data.examination.results.length === 0}
			<div class="px-3 py-2 text-sm text-surface-700">No structured results.</div>
		{:else}
			{#each data.examination.results as result}
				<div
					class={[
						'grid min-w-[40rem] grid-cols-[minmax(12rem,1fr)_7rem_7rem_9rem_4rem] border-t border-surface-200 text-sm',
						isBeyondNorm(result) ? 'result-row-abnormal' : ''
					]}
				>
					<div class="px-3 py-2">{result.name}</div>
					<div class="px-3 py-2 font-semibold">{resultValue(result)}</div>
					<div class="px-3 py-2">{result.unit ?? '-'}</div>
					<div class="px-3 py-2">{resultRange(result)}</div>
					<div class="px-3 py-2">
						{#if resultFlag(result)}
							<span class="result-flag-abnormal">{resultFlag(result)}</span>
						{:else}
							-
						{/if}
					</div>
				</div>
			{/each}
		{/if}
	</div>
</section>
