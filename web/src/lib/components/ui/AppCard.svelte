<script lang="ts">
	let {
		title,
		children
	}: {
		title?: string;
		children?: import('svelte').Snippet;
	} = $props();
</script>

<div class="card bg-base-200 shadow-xl">
	<div class="card-body">
		{#if title}
			<h2 class="card-title text-2xl">{title}</h2>
		{/if}
		{@render children?.()}
	</div>
</div>
