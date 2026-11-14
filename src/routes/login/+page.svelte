<script lang="ts">
	import { enhance } from '$app/forms';
	import { HeartPulse } from 'lucide-svelte';
	import { OIDC_LOGIN_PATH } from '$lib/oidc';
	import type { ActionData, PageData } from './$types';

	let { data, form }: { data: PageData; form: ActionData } = $props();
	let submitting = $state(false);
</script>

<div class="flex min-h-screen items-center justify-center bg-surface-50 p-4 text-surface-950">
	<div
		class="w-full max-w-sm space-y-5 rounded-md border border-surface-200 bg-white p-6 shadow-sm"
	>
		<div class="flex items-center gap-2 text-lg font-bold">
			<HeartPulse class="text-primary-600" size={24} />
			<span>Doctorine</span>
		</div>

		<form
			method="POST"
			class="space-y-4"
			use:enhance={() => {
				submitting = true;
				return async ({ update }) => {
					await update();
					submitting = false;
				};
			}}
		>
			<label class="label">
				<span class="text-sm font-semibold">Username</span>
				<input
					name="username"
					type="text"
					class="input"
					autocomplete="username"
					value={form?.username ?? ''}
					required
				/>
			</label>

			<label class="label">
				<span class="text-sm font-semibold">Password</span>
				<input
					name="password"
					type="password"
					class="input"
					autocomplete="current-password"
					required
				/>
			</label>

			<label class="flex items-center gap-2 text-sm">
				<input name="remember_me" type="checkbox" class="checkbox" />
				<span>Remember me</span>
			</label>

			{#if form?.error ?? data.oidcError}
				<div role="alert" class="rounded-md bg-error-100 p-3 text-sm text-error-900">
					{form?.error ?? data.oidcError}
				</div>
			{/if}

			<button type="submit" class="btn preset-filled-primary-500 w-full" disabled={submitting}>
				{submitting ? 'Logging in...' : 'Login'}
			</button>
		</form>

		{#if data.oidcEnabled}
			<div class="flex items-center gap-3 text-xs text-surface-600">
				<span class="h-px flex-1 bg-surface-200"></span>
				<span>or</span>
				<span class="h-px flex-1 bg-surface-200"></span>
			</div>
			<a href={OIDC_LOGIN_PATH} class="btn preset-tonal-primary-500 w-full" data-sveltekit-reload>
				Sign in with Authelia
			</a>
		{/if}
	</div>
</div>
