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

	async function createIllness(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		saving = true;
		const formEl = event.currentTarget as HTMLFormElement;
		const form = new FormData(formEl);
		const payload = {
			title: value(form, 'title'),
			status: value(form, 'status'),
			diagnosed_on: value(form, 'diagnosed_on'),
			resolved_on: value(form, 'resolved_on'),
			clinician: value(form, 'clinician'),
			notes: value(form, 'notes')
		};
		const response = await fetch('/api/illnesses', {
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
		<h1 class="page-title">Illnesses</h1>
		<p class="text-sm text-surface-700">Track active, monitored, and resolved conditions.</p>
	</div>

	<form
		class="grid gap-4 rounded-md border border-surface-200 bg-white p-4 md:grid-cols-2"
		onsubmit={createIllness}
	>
		<label class="label">
			<span class="text-sm font-semibold">Title</span>
			<input name="title" class="input" maxlength="200" required />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Status</span>
			<select name="status" class="select">
				<option value="active">Active</option>
				<option value="monitoring">Monitoring</option>
				<option value="resolved">Resolved</option>
			</select>
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Diagnosed on</span>
			<input name="diagnosed_on" type="date" class="input" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Resolved on</span>
			<input name="resolved_on" type="date" class="input" />
		</label>
		<label class="label md:col-span-2">
			<span class="text-sm font-semibold">Clinician</span>
			<input name="clinician" class="input" maxlength="200" />
		</label>
		<label class="label md:col-span-2">
			<span class="text-sm font-semibold">Notes</span>
			<textarea name="notes" class="textarea" rows="3"></textarea>
		</label>
		<div class="flex items-center gap-3 md:col-span-2">
			<button type="submit" class="btn preset-filled-primary-500" disabled={saving}>
				<Plus size={18} />
				<span>{saving ? 'Saving...' : 'Add illness'}</span>
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
					<th>Illness</th>
					<th>Status</th>
					<th>Diagnosed</th>
					<th>Clinician</th>
				</tr>
			</thead>
			<tbody>
				{#if data.illnesses.length === 0}
					<tr>
						<td colspan="4" class="text-sm text-surface-700">No illnesses recorded.</td>
					</tr>
				{:else}
					{#each data.illnesses as illness}
						<tr>
							<td data-label="Illness">
								<div class="font-semibold">{illness.title}</div>
								{#if illness.notes}
									<div class="text-sm text-surface-700">{illness.notes}</div>
								{/if}
							</td>
							<td data-label="Status">{illness.status}</td>
							<td data-label="Diagnosed">{formatDate(illness.diagnosed_on)}</td>
							<td data-label="Clinician">{illness.clinician ?? '-'}</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>
</section>
