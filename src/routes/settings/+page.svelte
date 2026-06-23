<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import {
		buildCreatePayload,
		createToken,
		expiryLabel,
		lastUsedLabel,
		revokeToken,
		scopeLabel
	} from '$lib/tokens';
	import type { CreatedToken } from '$lib/types';
	import { Copy, KeyRound, Plus, Trash2 } from 'lucide-svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	let error = $state('');
	let saving = $state(false);
	let revokingId = $state<number | null>(null);
	let created = $state<CreatedToken | null>(null);
	let copied = $state(false);

	const today = new Date().toISOString().slice(0, 10);

	async function submitToken(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		const formEl = event.currentTarget as HTMLFormElement;
		const result = buildCreatePayload(new FormData(formEl));
		if (!result.payload) {
			error = result.error ?? 'Invalid input';
			return;
		}
		saving = true;
		copied = false;
		try {
			created = await createToken(result.payload);
			formEl.reset();
			await invalidateAll();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Create failed';
		} finally {
			saving = false;
		}
	}

	async function revoke(id: number) {
		error = '';
		revokingId = id;
		try {
			await revokeToken(id);
			if (created?.id === id) created = null;
			await invalidateAll();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Revoke failed';
		} finally {
			revokingId = null;
		}
	}

	async function copyToken() {
		if (!created) return;
		try {
			await navigator.clipboard.writeText(created.token);
			copied = true;
		} catch {
			copied = false;
		}
	}
</script>

<section class="space-y-6">
	<div>
		<h1 class="page-title">Settings</h1>
		<p class="text-sm text-surface-700">
			Manage long-lived API tokens for service integrations (e.g. the coaching loop). A token never
			expires unless you set a date and stays valid until revoked.
		</p>
	</div>

	<form
		class="grid gap-4 rounded-md border border-surface-200 bg-white p-4 md:grid-cols-3"
		onsubmit={submitToken}
	>
		<label class="label">
			<span class="text-sm font-semibold">Name</span>
			<input
				name="name"
				type="text"
				class="input"
				maxlength="120"
				placeholder="coaching loop"
				required
			/>
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Scope</span>
			<select name="scope" class="select">
				<option value="full">Full access</option>
				<option value="read">Read-only</option>
			</select>
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Expires (optional)</span>
			<input name="expires_at" type="date" class="input" min={today} />
		</label>
		<div class="flex items-center gap-3 md:col-span-3">
			<button type="submit" class="btn preset-filled-primary-500" disabled={saving}>
				<Plus size={18} />
				<span>{saving ? 'Creating...' : 'Create token'}</span>
			</button>
			{#if error}
				<p class="text-sm text-error-700" aria-live="polite">{error}</p>
			{/if}
		</div>
	</form>

	{#if created}
		<div class="space-y-2 rounded-md border border-primary-300 bg-primary-50 p-4">
			<p class="text-sm font-semibold text-primary-900">
				Copy your new token now — it will not be shown again.
			</p>
			<div class="flex items-center gap-2">
				<code class="flex-1 overflow-x-auto rounded bg-white px-3 py-2 text-sm"
					>{created.token}</code
				>
				<button type="button" class="btn preset-tonal-primary-500" onclick={copyToken}>
					<Copy size={18} />
					<span>{copied ? 'Copied' : 'Copy'}</span>
				</button>
			</div>
		</div>
	{/if}

	<div class="table-cards overflow-hidden rounded-md border border-surface-200 bg-white">
		<table class="data-table">
			<thead>
				<tr>
					<th>Name</th>
					<th>Scope</th>
					<th>Expires</th>
					<th>Last used</th>
					<th class="text-right">Actions</th>
				</tr>
			</thead>
			<tbody>
				{#if data.tokens.length === 0}
					<tr>
						<td colspan="5" class="text-sm text-surface-700">
							<span class="inline-flex items-center gap-2"
								><KeyRound size={16} /> No API tokens yet.</span
							>
						</td>
					</tr>
				{:else}
					{#each data.tokens as token}
						<tr>
							<td data-label="Name"><div class="font-semibold">{token.name}</div></td>
							<td data-label="Scope">{scopeLabel(token.scope)}</td>
							<td data-label="Expires">{expiryLabel(token)}</td>
							<td data-label="Last used">{lastUsedLabel(token)}</td>
							<td data-label="Actions" class="text-right">
								<button
									type="button"
									class="btn-icon btn-icon-sm preset-tonal-error-500"
									aria-label={`Revoke ${token.name}`}
									disabled={revokingId === token.id}
									onclick={() => revoke(token.id)}
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
