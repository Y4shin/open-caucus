<script lang="ts">
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import { committeeClient } from '$lib/api/index.js';
	import { session } from '$lib/stores/session.svelte.js';
	import type { CommitteeListItem } from '$lib/gen/conference/committees/v1/committees_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import * as m from '$lib/paraglide/messages';

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

{#if committeesState.loading}
	<AppSpinner label="Loading committees" />
{:else if committeesState.error}
	<AppAlert message={committeesState.error} />
{:else}
	<div class="rounded-box border-base-300 bg-base-200 shadow-sm p-4">
		{#if committeesState.data?.length}
			<ul class="list">
				<li class="p-4 pb-2 text-xs opacity-60 tracking-wide">{m.home_title()}</li>
				{#each committeesState.data as item}
					<li class="list-row">
						<div class="list-col-grow">{item.committee?.name ?? 'Committee'}</div>
						<a
							class="btn btn-ghost btn-small btn-square"
							href="/committee/{item.committee?.slug ?? ''}"
							aria-label={m.home_goto_committee({ name: item.committee?.name ?? 'Committee' })}
						><LegacyIcon name="right" class="h-4 w-4" /></a>
					</li>
				{/each}
			</ul>
		{:else}
			<p>{m.home_no_committees()}</p>
		{/if}
	</div>
{/if}
