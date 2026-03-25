<script lang="ts">
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { committeeClient } from '$lib/api/index.js';
	import { session } from '$lib/stores/session.svelte.js';
	import type { CommitteeListItem } from '$lib/gen/conference/committees/v1/committees_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';

	let committeesState = $state(createRemoteState<CommitteeListItem[]>());

	$effect(() => {
		if (!session.loaded) return;
		if (!session.authenticated) {
			goto('/login');
			return;
		}
		loadCommittees();
	});

	async function loadCommittees() {
		committeesState.loading = true;
		committeesState.error = '';
		try {
			const res = await committeeClient.listMyCommittees({});
			committeesState.data = res.committees;
		} catch (err) {
			committeesState.error = getDisplayError(err, 'Failed to load your committees.');
		} finally {
			committeesState.loading = false;
		}
	}
</script>

<div class="space-y-6">
	<div class="space-y-2">
		<h1 class="text-3xl font-bold">Home</h1>
		<p class="text-base-content/70">
			Choose a committee to continue into its current meeting workflow.
		</p>
	</div>

	{#if committeesState.loading}
		<AppSpinner label="Loading committees" />
	{:else if committeesState.error}
		<AppAlert message={committeesState.error} />
	{:else if committeesState.data?.length}
		<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
			{#each committeesState.data as item}
				<AppCard title={item.committee?.name ?? 'Committee'}>
					<p class="text-sm text-base-content/70">
						{item.meetingCount} meeting{item.meetingCount === 1 ? '' : 's'}
					</p>
					{#if item.hasActiveMeeting}
						<p class="text-sm font-medium text-success">Active meeting available</p>
					{/if}
					<div class="card-actions justify-end">
						<a class="btn btn-primary btn-sm" href="/committee/{item.committee?.slug ?? ''}">
							Open
						</a>
					</div>
				</AppCard>
			{/each}
		</div>
	{:else}
		<AppAlert tone="info" message="No committees are available for this account yet." />
	{/if}
</div>
