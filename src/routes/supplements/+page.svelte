<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { Plus } from 'lucide-svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	let error = $state('');
	let saving = $state(false);

	function value(form: FormData, key: string): string {
		const raw = String(form.get(key) ?? '').trim();
		return raw;
	}

	function optionalValue(form: FormData, key: string): string | null {
		const raw = value(form, key);
		return raw === '' ? null : raw;
	}

	async function createSupplement(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		saving = true;
		const formEl = event.currentTarget as HTMLFormElement;
		const form = new FormData(formEl);
		const payload = {
			name: value(form, 'name'),
			value: value(form, 'value'),
			frequency: value(form, 'frequency'),
			notes: optionalValue(form, 'notes')
		};
		try {
			const response = await fetch('/api/supplements', {
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
</script>

<section class="space-y-6">
	<div>
		<h1 class="page-title">Supplements</h1>
		<p class="text-sm text-surface-700">Track supplements you currently take.</p>
	</div>

	<form
		class="grid gap-4 rounded-md border border-surface-200 bg-white p-4 md:grid-cols-3"
		onsubmit={createSupplement}
	>
		<label class="label">
			<span class="text-sm font-semibold">Name</span>
			<input name="name" class="input" maxlength="200" placeholder="Omega 3" required />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Value</span>
			<input name="value" class="input" maxlength="120" placeholder="1000mg" required />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">How often</span>
			<input name="frequency" class="input" maxlength="120" placeholder="Daily" required />
		</label>
		<label class="label md:col-span-3">
			<span class="text-sm font-semibold">Notes</span>
			<textarea name="notes" class="textarea" rows="3"></textarea>
		</label>
		<div class="flex items-center gap-3 md:col-span-3">
			<button type="submit" class="btn preset-filled-primary-500" disabled={saving}>
				<Plus size={18} />
				<span>{saving ? 'Saving...' : 'Add supplement'}</span>
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
					<th>Supplement</th>
					<th>Value</th>
					<th>How often</th>
					<th>Notes</th>
				</tr>
			</thead>
			<tbody>
				{#if data.supplements.length === 0}
					<tr>
						<td colspan="4" class="text-sm text-surface-700">No supplements recorded.</td>
					</tr>
				{:else}
					{#each data.supplements as supplement}
						<tr>
							<td data-label="Supplement">
								<div class="font-semibold">{supplement.name}</div>
							</td>
							<td data-label="Value">{supplement.value}</td>
							<td data-label="How often">{supplement.frequency}</td>
							<td data-label="Notes">{supplement.notes ?? '-'}</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>
</section>
