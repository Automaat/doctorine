<script lang="ts">
	import { formatBytes, formatDate } from '$lib/format';
	import { AlertTriangle, FileText, HeartPulse, Stethoscope } from 'lucide-svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const metrics = $derived([
		{
			label: 'Documents',
			value: data.overview.document_count,
			icon: FileText
		},
		{
			label: 'Active illnesses',
			value: data.overview.illness_count,
			icon: HeartPulse
		},
		{
			label: 'Examinations',
			value: data.overview.examination_count,
			icon: Stethoscope
		},
		{
			label: 'Flagged results',
			value: data.overview.flagged_results,
			icon: AlertTriangle
		}
	]);
</script>

<section class="space-y-6">
	<div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
		<div>
			<h1 class="page-title">Health overview</h1>
			<p class="text-sm text-surface-700">Private medical records, exams, and illness notes.</p>
		</div>
		<a href="/documents" class="btn preset-filled-primary-500 w-fit">Upload document</a>
	</div>

	<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
		{#each metrics as metric}
			<div class="metric-card">
				<div class="flex items-center justify-between gap-3">
					<div>
						<div class="text-sm text-surface-700">{metric.label}</div>
						<div class="mt-1 text-3xl font-bold">{metric.value}</div>
					</div>
					<metric.icon class="text-primary-600" size={28} />
				</div>
			</div>
		{/each}
	</div>

	<section class="space-y-3">
		<div class="flex items-center justify-between gap-3">
			<h2 class="section-title">Recent documents</h2>
			<a href="/documents" class="text-sm font-semibold text-primary-700 hover:underline"
				>All documents</a
			>
		</div>

		{#if data.overview.recent_documents.length === 0}
			<div
				class="rounded-md border border-dashed border-surface-300 bg-white p-6 text-sm text-surface-700"
			>
				No documents uploaded.
			</div>
		{:else}
			<div class="table-cards overflow-hidden rounded-md border border-surface-200 bg-white">
				<table class="data-table">
					<thead>
						<tr>
							<th>Title</th>
							<th>Issued</th>
							<th>Type</th>
							<th>Size</th>
						</tr>
					</thead>
					<tbody>
						{#each data.overview.recent_documents as document}
							<tr>
								<td data-label="Title">
									<a
										href={`/api/documents/${document.id}/download`}
										class="font-semibold text-primary-700 hover:underline"
									>
										{document.title}
									</a>
								</td>
								<td data-label="Issued">{formatDate(document.issued_at)}</td>
								<td data-label="Type">{document.document_type}</td>
								<td data-label="Size">{formatBytes(document.size_bytes)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
</section>
