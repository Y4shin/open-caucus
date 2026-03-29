<script lang="ts">
	let {
		title,
		class: className = '',
		children
	}: {
		title?: string;
		class?: string;
		children?: import('svelte').Snippet;
	} = $props();
</script>

<section class={`rounded-box border border-base-300 bg-base-200 p-4 ${className}`.trim()}>
		{#if title}
			<h2 class="mb-3 text-lg font-semibold">{title}</h2>
		{/if}
		{@render children?.()}
	</section>
