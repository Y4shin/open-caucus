<script lang="ts">
	let {
		message,
		tone = 'error'
	}: {
		message: string;
		tone?: 'error' | 'info' | 'success' | 'warning';
	} = $props();
</script>

<div role="alert" class={`alert alert-${tone}`}>
	<span>{message}</span>
</div>
