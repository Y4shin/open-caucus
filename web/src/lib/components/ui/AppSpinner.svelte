<script lang="ts">
	let {
		label = 'Loading...',
		size = 'lg'
	}: {
		label?: string;
		size?: 'sm' | 'md' | 'lg';
	} = $props();
</script>

<div class="flex justify-center py-12" aria-live="polite" aria-label={label}>
	<span class={`loading loading-spinner loading-${size}`}></span>
</div>
