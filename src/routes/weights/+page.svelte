<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { formatDate } from '$lib/format';
	import { Plus, Trash2 } from 'lucide-svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	let error = $state('');
	let saving = $state(false);
	let deletingId = $state<number | null>(null);

	const numberFormat = new Intl.NumberFormat('pl-PL', { maximumFractionDigits: 2 });
	const today = new Date().toISOString().slice(0, 10);

	function value(form: FormData, key: string): string {
		return String(form.get(key) ?? '').trim();
	}

	function optionalValue(form: FormData, key: string): string | null {
		const raw = value(form, key);
		return raw === '' ? null : raw;
	}

	async function createWeight(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		saving = true;
		const formEl = event.currentTarget as HTMLFormElement;
		const form = new FormData(formEl);
		const payload = {
			measured_on: value(form, 'measured_on'),
			weight_kg: Number(value(form, 'weight_kg')),
			notes: optionalValue(form, 'notes')
		};
		try {
			const response = await fetch('/api/weights', {
				method: 'POST',
				headers: { 'content-type': 'application/json' },
				body: JSON.stringify(payload)
			});
			if (!response.ok) {
				const body = (await response.json().catch(() => null)) as { detail?: string } | null;
				error = body?.detail ?? 'Save failed';
				return;
			}
			formEl.reset();
			await invalidateAll();
		} catch {
			error = 'Save failed';
		} finally {
			saving = false;
		}
	}

	async function deleteWeight(id: number) {
		error = '';
		deletingId = id;
		try {
			const response = await fetch(`/api/weights/${id}`, { method: 'DELETE' });
			if (!response.ok && response.status !== 204) {
				const body = (await response.json().catch(() => null)) as { detail?: string } | null;
				error = body?.detail ?? 'Delete failed';
				return;
			}
			await invalidateAll();
		} catch {
			error = 'Delete failed';
		} finally {
			deletingId = null;
		}
	}
</script>

<section class="space-y-6">
	<div>
		<h1 class="page-title">Weight</h1>
		<p class="text-sm text-surface-700">Log your weight and track the trend over time.</p>
	</div>

	<form
		class="grid gap-4 rounded-md border border-surface-200 bg-white p-4 md:grid-cols-3"
		onsubmit={createWeight}
	>
		<label class="label">
			<span class="text-sm font-semibold">Date</span>
			<input name="measured_on" type="date" class="input" value={today} max={today} required />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Weight (kg)</span>
			<input
				name="weight_kg"
				type="number"
				class="input"
				min="1"
				max="999"
				step="0.1"
				placeholder="78.5"
				required
			/>
		</label>
		<label class="label md:col-span-3">
			<span class="text-sm font-semibold">Notes</span>
			<textarea name="notes" class="textarea" rows="2"></textarea>
		</label>
		<div class="flex items-center gap-3 md:col-span-3">
			<button type="submit" class="btn preset-filled-primary-500" disabled={saving}>
				<Plus size={18} />
				<span>{saving ? 'Saving...' : 'Add measurement'}</span>
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
					<th>Date</th>
					<th>Weight</th>
					<th>Notes</th>
					<th class="text-right">Actions</th>
				</tr>
			</thead>
			<tbody>
				{#if data.weights.length === 0}
					<tr>
						<td colspan="4" class="text-sm text-surface-700">No weight recorded.</td>
					</tr>
				{:else}
					{#each data.weights as entry}
						<tr>
							<td data-label="Date">
								<div class="font-semibold">{formatDate(entry.measured_on)}</div>
							</td>
							<td data-label="Weight">{numberFormat.format(entry.weight_kg)} kg</td>
							<td data-label="Notes">{entry.notes ?? '-'}</td>
							<td data-label="Actions" class="text-right">
								<button
									type="button"
									class="btn-icon btn-icon-sm preset-tonal-error-500"
									aria-label={`Delete ${formatDate(entry.measured_on)} weight`}
									disabled={deletingId === entry.id}
									onclick={() => deleteWeight(entry.id)}
								>
									<Trash2 size={18} />
								</button>
							</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>
</section>
