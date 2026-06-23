<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import {
		Activity,
		FileText,
		HeartPulse,
		LogOut,
		Pill,
		Stethoscope,
		User,
		Weight
	} from 'lucide-svelte';
	import type { LayoutData } from './$types';

	let { children, data }: { children: import('svelte').Snippet; data: LayoutData } = $props();

	const nav = [
		{ href: '/', label: 'Dashboard', icon: Activity },
		{ href: '/documents', label: 'Documents', icon: FileText },
		{ href: '/examinations', label: 'Exams', icon: Stethoscope },
		{ href: '/supplements', label: 'Supplements', icon: Pill },
		{ href: '/weights', label: 'Weight', icon: Weight },
		{ href: '/illnesses', label: 'Illnesses', icon: HeartPulse }
	];

	const isLoginPage = $derived($page.url.pathname === '/login');

	function isActive(href: string): boolean {
		return href === '/' ? $page.url.pathname === '/' : $page.url.pathname.startsWith(href);
	}
</script>

{#if isLoginPage}
	{@render children?.()}
{:else}
	<div class="flex min-h-screen bg-surface-50 text-surface-950">
		<aside class="hidden w-60 shrink-0 border-r border-surface-200 bg-white md:flex md:flex-col">
			<div class="flex items-center gap-2 border-b border-surface-200 p-4 text-lg font-bold">
				<HeartPulse class="text-primary-600" size={24} />
				<span>Doctorine</span>
			</div>
			<nav class="flex-1 space-y-1 p-2">
				{#each nav as item}
					<a
						href={item.href}
						class="flex min-h-11 items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors
							{isActive(item.href) ? 'bg-primary-600 text-white' : 'text-surface-700 hover:bg-surface-100'}"
					>
						<item.icon size={18} />
						<span>{item.label}</span>
					</a>
				{/each}
			</nav>
			{#if data.user}
				<div class="space-y-2 border-t border-surface-200 p-3">
					<div class="flex items-center gap-2 px-1 text-sm text-surface-700">
						<User size={16} />
						<span class="truncate">{data.user.name || data.user.username}</span>
					</div>
					<form method="POST" action="/logout">
						<button
							type="submit"
							class="flex min-h-11 w-full items-center gap-3 rounded-md px-3 py-2 text-sm text-surface-700 hover:bg-surface-100"
						>
							<LogOut size={18} />
							<span>Logout</span>
						</button>
					</form>
				</div>
			{/if}
		</aside>

		<div class="flex min-w-0 flex-1 flex-col">
			<header
				class="sticky top-0 z-20 flex items-center justify-between border-b border-surface-200 bg-white px-4 py-3 md:hidden"
			>
				<span class="flex items-center gap-2 text-base font-bold">
					<HeartPulse class="text-primary-600" size={20} />
					<span>Doctorine</span>
				</span>
				{#if data.user}
					<form method="POST" action="/logout">
						<button type="submit" class="btn-icon btn-icon-sm" aria-label="Logout">
							<LogOut size={20} />
						</button>
					</form>
				{/if}
			</header>

			<main class="mx-auto w-full max-w-[1200px] flex-1 p-4 pb-24 md:p-6 lg:p-8">
				{@render children?.()}
			</main>

			<nav
				class="fixed bottom-0 left-0 right-0 z-30 grid grid-cols-6 border-t border-surface-200 bg-white pb-[env(safe-area-inset-bottom)] md:hidden"
				aria-label="Mobile navigation"
			>
				{#each nav as item}
					<a
						href={item.href}
						class="flex min-h-14 flex-col items-center justify-center gap-1 px-1 py-2 text-[11px]
							{isActive(item.href) ? 'text-primary-600 font-semibold' : 'text-surface-600'}"
					>
						<item.icon size={20} />
						<span class="max-w-full truncate whitespace-nowrap">{item.label}</span>
					</a>
				{/each}
			</nav>
		</div>
	</div>
{/if}
