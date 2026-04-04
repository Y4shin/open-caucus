<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		header,
		body
	}: {
		header: Snippet;
		body: Snippet;
	} = $props();
</script>

<table class="data-table table table-zebra w-full">
	<thead>
		{@render header()}
	</thead>
	<tbody>
		{@render body()}
	</tbody>
</table>
