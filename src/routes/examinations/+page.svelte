<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { formatDate } from '$lib/format';
	import { Plus } from 'lucide-svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	let error = $state('');
	let saving = $state(false);

	function value(form: FormData, key: string): string | null {
		const raw = String(form.get(key) ?? '').trim();
		return raw === '' ? null : raw;
	}

	async function createExamination(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		saving = true;
		const formEl = event.currentTarget as HTMLFormElement;
		const form = new FormData(formEl);
		const payload = {
			title: value(form, 'title'),
			exam_date: value(form, 'exam_date'),
			category: value(form, 'category'),
			facility: value(form, 'facility'),
			result_status: value(form, 'result_status'),
			summary: value(form, 'summary'),
			notes: value(form, 'notes')
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
		await invalidateAll();
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
