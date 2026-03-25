<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { meetingClient } from '$lib/api/index.js';
	import { session } from '$lib/stores/session.svelte.js';
	import type { LiveMeetingView } from '$lib/gen/conference/meetings/v1/meetings_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { connectEventStream } from '$lib/utils/sse.js';

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);

	let liveState = $state(createRemoteState<LiveMeetingView>());
	let refreshTick = $state(0);

	$effect(() => {
		if (!session.loaded) return;
		if (!session.authenticated) {
			goto(`/committee/${slug}/meeting/${meetingId}/join`);
			return;
		}
		refreshTick;
		loadMeeting();
	});

	$effect(() => {
		const eventsUrl = liveState.data?.eventsUrl;
		if (!eventsUrl) return;
		return connectEventStream(eventsUrl, () => {
			refreshTick += 1;
		});
	});

	async function loadMeeting() {
		liveState.loading = true;
		liveState.error = '';
		try {
			const res = await meetingClient.getLiveMeeting({ committeeSlug: slug, meetingId });
			liveState.data = res.meeting ?? null;
		} catch (err) {
			liveState.error = getDisplayError(err, 'Failed to load the live meeting view.');
		} finally {
			liveState.loading = false;
		}
	}
</script>

<div class="space-y-6">
	{#if liveState.loading}
		<AppSpinner label="Loading live meeting" />
	{:else if liveState.error}
		<AppAlert message={liveState.error} />
	{:else if liveState.data}
		<div class="space-y-2">
			<h1 class="text-3xl font-bold">{liveState.data.meetingName}</h1>
			<p class="text-base-content/70">{liveState.data.committeeName}</p>
		</div>

		<div class="grid gap-4 xl:grid-cols-3">
			<AppCard title="Current Agenda Point">
				{#if liveState.data.activeAgendaPoint}
					<p class="font-medium">
						{liveState.data.activeAgendaPoint.displayNumber}
						{#if liveState.data.activeAgendaPoint.title}
							: {liveState.data.activeAgendaPoint.title}
						{/if}
					</p>
				{:else}
					<p class="text-base-content/70">No agenda point is active.</p>
				{/if}
			</AppCard>

			<AppCard title="Current Document">
				{#if liveState.data.currentDocument}
					<p class="font-medium">
						{liveState.data.currentDocument.label || liveState.data.currentDocument.filename}
					</p>
					<a
						class="btn btn-outline btn-sm mt-3"
						href={liveState.data.currentDocument.downloadUrl}
						target="_blank"
						rel="noreferrer"
					>
						Open document
					</a>
				{:else}
					<p class="text-base-content/70">No document is currently published.</p>
				{/if}
			</AppCard>

			<AppCard title="Actions">
				<div class="flex flex-wrap gap-2">
					{#if liveState.data.capabilities?.canSelfSignup}
						<a class="btn btn-outline btn-sm" href="/committee/{slug}/meeting/{meetingId}/join">
							Join Meeting
						</a>
					{/if}
					<a class="btn btn-outline btn-sm" href="/committee/{slug}">
						Back to Committee
					</a>
					<a class="btn btn-primary btn-sm" href="/committee/{slug}/meeting/{meetingId}/moderate">
						Moderate
					</a>
				</div>
			</AppCard>
		</div>

		<AppCard title="Speakers">
			{#if liveState.data.speakers.length}
				<div class="space-y-2">
					{#each liveState.data.speakers as speaker}
						<div class="flex items-center justify-between rounded-box border border-base-300 px-3 py-2">
							<div>
								<div class="font-medium">{speaker.fullName}</div>
								<div class="text-sm text-base-content/70">
									{speaker.speakerType} • {speaker.state}
								</div>
							</div>
							{#if speaker.mine}
								<span class="badge badge-primary">You</span>
							{/if}
						</div>
					{/each}
				</div>
			{:else}
				<p class="text-base-content/70">No speakers are queued right now.</p>
			{/if}
		</AppCard>
	{/if}
</div>
