<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);

	$effect(() => {
		if (!slug || !meetingId) return;
		goto(`/committee/${slug}/meeting/${meetingId}`);
	});
</script>

<div class="flex min-h-[60vh] items-center justify-center">
	<span class="loading loading-spinner loading-lg"></span>
</div>
