<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { session } from '$lib/stores/session.svelte.js';
	import { committeeClient } from '$lib/api/index.js';
	import type { Committee } from '$lib/gen/conference/committees/v1/committees_pb.js';

	const slug = $derived(page.params.committee);

	let committee = $state<Committee | undefined>(undefined);
	let loading = $state(true);
	let errorMsg = $state('');

	$effect(() => {
		if (!session.loaded) return;
		if (!session.authenticated) {
			goto('/login');
			return;
		}
		loadCommittee();
	});

	async function loadCommittee() {
		loading = true;
		errorMsg = '';
		try {
			const res = await committeeClient.getCommittee({ slug });
			committee = res.committee;
		} catch {
			errorMsg = `Committee "${slug}" not found or access denied.`;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		// Reload when slug changes (navigating between committees)
		if (slug && session.loaded && session.authenticated) {
			loadCommittee();
		}
	});
</script>

<div class="space-y-6">
	{#if loading}
		<div class="flex justify-center py-12">
			<span class="loading loading-spinner loading-lg"></span>
		</div>
	{:else if errorMsg}
		<div role="alert" class="alert alert-error">
			<span>{errorMsg}</span>
		</div>
	{:else if committee}
		<div class="flex items-center justify-between">
			<h1 class="text-3xl font-bold">{committee.name}</h1>
		</div>

		<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
			<div class="card bg-base-200">
				<div class="card-body">
					<h2 class="card-title">Meetings</h2>
					<p class="text-base-content/60">View and manage committee meetings.</p>
					<div class="card-actions">
						<a href="/{slug}/meetings" class="btn btn-primary btn-sm">Open</a>
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
