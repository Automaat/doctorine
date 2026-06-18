<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { formatBytes, formatDate } from '$lib/format';
	import { Download, Trash2, Upload } from 'lucide-svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
	let error = $state('');
	let uploading = $state(false);
	let deletingId = $state<number | null>(null);

	async function upload(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		uploading = true;
		const form = event.currentTarget as HTMLFormElement;
		const response = await fetch('/api/documents', {
			method: 'POST',
			body: new FormData(form)
		});
		uploading = false;
		if (!response.ok) {
			const body = (await response.json().catch(() => null)) as { detail?: string } | null;
			error = body?.detail ?? 'Upload failed';
			return;
		}
		form.reset();
		await invalidateAll();
	}

	async function removeDocument(id: number) {
		error = '';
		deletingId = id;
		const response = await fetch(`/api/documents/${id}`, { method: 'DELETE' });
		deletingId = null;
		if (!response.ok) {
			const body = (await response.json().catch(() => null)) as { detail?: string } | null;
			error = body?.detail ?? 'Delete failed';
			return;
		}
		await invalidateAll();
	}
</script>

<section class="space-y-6">
	<div>
		<h1 class="page-title">Documents</h1>
		<p class="text-sm text-surface-700">Medical PDFs, scans, lab reports, and discharge files.</p>
	</div>

	<form
		class="grid gap-4 rounded-md border border-surface-200 bg-white p-4 md:grid-cols-2"
		enctype="multipart/form-data"
		onsubmit={upload}
	>
		<label class="label">
			<span class="text-sm font-semibold">File</span>
			<input name="file" type="file" class="input" required />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Title</span>
			<input name="title" class="input" maxlength="240" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Document type</span>
			<input name="document_type" class="input" value="medical" maxlength="80" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Issued at</span>
			<input name="issued_at" type="date" class="input" />
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Illness</span>
			<select name="illness_id" class="select">
				<option value="">None</option>
				{#each data.illnesses as illness}
					<option value={illness.id}>{illness.title}</option>
				{/each}
			</select>
		</label>
		<label class="label">
			<span class="text-sm font-semibold">Examination</span>
			<select name="examination_id" class="select">
				<option value="">None</option>
				{#each data.examinations as examination}
					<option value={examination.id}>{examination.exam_date} - {examination.title}</option>
				{/each}
			</select>
		</label>
		<label class="label md:col-span-2">
			<span class="text-sm font-semibold">Notes</span>
			<textarea name="notes" class="textarea" rows="3"></textarea>
		</label>
		<div class="flex items-center gap-3 md:col-span-2">
			<button type="submit" class="btn preset-filled-primary-500" disabled={uploading}>
				<Upload size={18} />
				<span>{uploading ? 'Uploading...' : 'Upload'}</span>
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
					<th>Title</th>
					<th>Issued</th>
					<th>Linked to</th>
					<th>Size</th>
					<th>Actions</th>
				</tr>
			</thead>
			<tbody>
				{#if data.documents.length === 0}
					<tr>
						<td colspan="5" class="text-sm text-surface-700">No documents uploaded.</td>
					</tr>
				{:else}
					{#each data.documents as document}
						<tr>
							<td data-label="Title">
								<div class="font-semibold">{document.title}</div>
								<div class="text-xs text-surface-600">{document.original_filename}</div>
							</td>
							<td data-label="Issued">{formatDate(document.issued_at)}</td>
							<td data-label="Linked to">
								{document.illness_title ?? document.examination_title ?? '-'}
							</td>
							<td data-label="Size">{formatBytes(document.size_bytes)}</td>
							<td>
								<div class="flex justify-end gap-2">
									<a
										href={`/api/documents/${document.id}/download`}
										class="btn-icon btn-icon-sm"
										aria-label="Download"
									>
										<Download size={18} />
									</a>
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Delete"
										disabled={deletingId === document.id}
										onclick={() => removeDocument(document.id)}
									>
										<Trash2 size={18} />
									</button>
								</div>
							</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>
</section>
