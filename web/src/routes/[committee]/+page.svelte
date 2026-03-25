<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { session } from '$lib/stores/session.svelte.js';
	import { committeeClient } from '$lib/api/index.js';
	import type { CommitteeOverview } from '$lib/gen/conference/committees/v1/committees_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';

	const slug = $derived(page.params.committee);

	let committeeState = $state(createRemoteState<CommitteeOverview>());

	$effect(() => {
		if (!session.loaded) return;
		if (!session.authenticated) {
			goto('/login');
			return;
		}
		loadCommittee();
	});

	async function loadCommittee() {
		committeeState.loading = true;
		committeeState.error = '';
		try {
			const res = await committeeClient.getCommitteeOverview({ committeeSlug: slug });
			committeeState.data = res.overview ?? null;
		} catch (err) {
			committeeState.error = getDisplayError(
				err,
				`Committee "${slug}" not found or access denied.`
			);
		} finally {
			committeeState.loading = false;
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
	{#if committeeState.loading}
		<AppSpinner label="Loading committee overview" />
	{:else if committeeState.error}
		<AppAlert message={committeeState.error} />
	{:else if committeeState.data}
		<div class="flex items-center justify-between">
			<h1 class="text-3xl font-bold">{committeeState.data.committee?.name ?? slug}</h1>
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
