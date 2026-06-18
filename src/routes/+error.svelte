<script lang="ts">
	import { page } from '$app/stores';
</script>

<section class="space-y-3">
	<h1 class="page-title">Something failed</h1>
	<p class="text-surface-700">{$page.error?.message ?? 'Unexpected error'}</p>
</section>
