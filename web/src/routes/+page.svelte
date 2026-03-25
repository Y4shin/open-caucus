<script lang="ts">
	import { goto } from '$app/navigation';
	import { session } from '$lib/stores/session.svelte.js';

	$effect(() => {
		if (!session.loaded) return;

		if (!session.authenticated) {
			goto('/login');
			return;
		}
		if (session.isAdmin && session.availableCommittees.length === 0) {
			goto('/admin');
			return;
		}
		if (session.availableCommittees.length > 0) {
			goto(`/${session.availableCommittees[0].slug}`);
		}
	});
</script>

<div class="flex min-h-[60vh] items-center justify-center">
	<span class="loading loading-spinner loading-lg"></span>
</div>
