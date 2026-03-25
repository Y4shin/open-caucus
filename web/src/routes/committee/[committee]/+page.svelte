<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { session } from '$lib/stores/session.svelte.js';
	import { committeeClient } from '$lib/api/index.js';
	import type { CommitteeOverview } from '$lib/gen/conference/committees/v1/committees_pb.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';

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
</script>

<div class="space-y-6">
	{#if committeeState.loading}
		<AppSpinner label="Loading committee overview" />
	{:else if committeeState.error}
		<AppAlert message={committeeState.error} />
	{:else if committeeState.data}
		<div class="space-y-2">
			<h1 class="text-3xl font-bold">{committeeState.data.committee?.name ?? slug}</h1>
			<p class="text-base-content/70">Committee dashboard and current meeting access.</p>
		</div>

		{#if committeeState.data.meetings.length}
			<div class="grid gap-4 xl:grid-cols-2">
				{#each committeeState.data.meetings as item}
					<AppCard title={item.meeting?.name ?? 'Meeting'}>
						<p class="text-sm text-base-content/70">
							Signup {item.meeting?.signupOpen ? 'is open' : 'is closed'}
						</p>
						<div class="card-actions justify-end gap-2">
							{#if item.canViewLive}
								<a
									class="btn btn-outline btn-sm"
									href="/committee/{slug}/meeting/{item.meeting?.meetingId}"
								>
									Live
								</a>
							{/if}
							{#if item.canJoin}
								<a
									class="btn btn-outline btn-sm"
									href="/committee/{slug}/meeting/{item.meeting?.meetingId}/join"
								>
									Join
								</a>
							{/if}
							{#if item.canModerate}
								<a
									class="btn btn-primary btn-sm"
									href="/committee/{slug}/meeting/{item.meeting?.meetingId}/moderate"
								>
									Moderate
								</a>
							{/if}
						</div>
					</AppCard>
				{/each}
			</div>
		{:else}
			<AppAlert tone="info" message="No meetings are available for this committee yet." />
		{/if}
	{/if}
</div>
