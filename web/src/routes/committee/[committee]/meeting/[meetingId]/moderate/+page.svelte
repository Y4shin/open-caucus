<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { moderationClient, speakerClient } from '$lib/api/index.js';
	import { session } from '$lib/stores/session.svelte.js';
	import type { ModerationView } from '$lib/gen/conference/moderation/v1/moderation_pb.js';
	import type { SpeakerQueueView } from '$lib/gen/conference/speakers/v1/speakers_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { connectEventStream } from '$lib/utils/sse.js';

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);

	let moderationState = $state(createRemoteState<ModerationView>());
	let speakerState = $state(createRemoteState<SpeakerQueueView>());
	let actionError = $state('');
	let togglingSignup = $state(false);
	let speakerActionPending = $state('');
	let refreshTick = $state(0);

	$effect(() => {
		if (!session.loaded) return;
		if (!session.authenticated) {
			goto('/login');
			return;
		}
		refreshTick;
		loadModerationView();
	});

	$effect(() => {
		const eventsUrl = moderationState.data?.eventsUrl;
		if (!eventsUrl) return;
		return connectEventStream(eventsUrl, () => {
			refreshTick += 1;
		});
	});

	async function loadModerationView() {
		moderationState.loading = true;
		speakerState.loading = true;
		moderationState.error = '';
		speakerState.error = '';
		try {
			const [moderationRes, speakerRes] = await Promise.all([
				moderationClient.getModerationView({ committeeSlug: slug, meetingId }),
				speakerClient.listSpeakers({ committeeSlug: slug, meetingId })
			]);
			moderationState.data = moderationRes.view ?? null;
			speakerState.data = speakerRes.view ?? null;
		} catch (err) {
			moderationState.error = getDisplayError(err, 'Failed to load the moderation view.');
			speakerState.error = moderationState.error;
		} finally {
			moderationState.loading = false;
			speakerState.loading = false;
		}
	}

	async function toggleSignupOpen() {
		const view = moderationState.data;
		if (!view?.attendees || togglingSignup) return;

		actionError = '';
		togglingSignup = true;

		try {
			const res = await moderationClient.toggleSignupOpen({
				committeeSlug: slug,
				meetingId,
				desiredOpen: !view.attendees.signupOpen,
				expectedVersion: view.version
			});

			view.attendees.signupOpen = res.signupOpen;
			view.version = res.version;
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to update signup state.');
			refreshTick += 1;
		} finally {
			togglingSignup = false;
		}
	}

	function activeSpeaker() {
		return speakerState.data?.speakers.find((speaker) => speaker.state === 'SPEAKING') ?? null;
	}

	function nextWaitingSpeaker() {
		return speakerState.data?.speakers.find((speaker) => speaker.state === 'WAITING') ?? null;
	}

	async function runSpeakerAction(
		key: string,
		action: () => Promise<{ view?: SpeakerQueueView }>
	) {
		actionError = '';
		speakerActionPending = key;
		try {
			const res = await action();
			speakerState.data = res.view ?? speakerState.data;
			refreshTick += 1;
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to update the speakers queue.');
			refreshTick += 1;
		} finally {
			speakerActionPending = '';
		}
	}
</script>

<div class="space-y-6">
	{#if moderationState.loading}
		<AppSpinner label="Loading moderation view" />
	{:else if moderationState.error}
		<AppAlert message={moderationState.error} />
	{:else if moderationState.data}
		<div class="space-y-2">
			<h1 class="text-3xl font-bold">{moderationState.data.meeting?.meetingName}</h1>
			<p class="text-base-content/70">
				Moderation workspace for {moderationState.data.meeting?.committeeName}
			</p>
		</div>

		{#if actionError}
			<AppAlert message={actionError} />
		{/if}

		<div class="grid gap-4 xl:grid-cols-3">
			<AppCard title="Signup Control">
				<p class="text-sm text-base-content/70">
					Version {moderationState.data.version.toString()}
				</p>
				<p class="mt-3 font-medium">
					Signup is {moderationState.data.attendees?.signupOpen ? 'open' : 'closed'}.
				</p>
				<button class="btn btn-primary btn-sm mt-4" onclick={toggleSignupOpen} disabled={togglingSignup}>
					{#if togglingSignup}
						<span class="loading loading-spinner loading-xs"></span>
					{/if}
					{moderationState.data.attendees?.signupOpen ? 'Close Signup' : 'Open Signup'}
				</button>
			</AppCard>

			<AppCard title="Attendees">
				<div class="space-y-2 text-sm">
					<p>Total: {moderationState.data.attendees?.totalCount ?? 0}</p>
					<p>Guests: {moderationState.data.attendees?.guestCount ?? 0}</p>
					<p>Chairs: {moderationState.data.attendees?.chairCount ?? 0}</p>
					<p>
						Self-signup visible:
						{moderationState.data.attendees?.showSelfSignup ? 'Yes' : 'No'}
					</p>
				</div>
			</AppCard>

			<AppCard title="Speakers">
				<div class="space-y-2 text-sm">
					<p>Total: {moderationState.data.speakers?.totalCount ?? 0}</p>
					<p>Waiting: {moderationState.data.speakers?.waitingCount ?? 0}</p>
					<p>
						Active speaker:
						{moderationState.data.speakers?.hasActiveSpeaker ? 'Yes' : 'No'}
					</p>
				</div>
			</AppCard>
		</div>

		<AppCard title="Speakers Queue">
			<div class="mb-4 flex flex-wrap gap-2">
				<button
					class="btn btn-primary btn-sm"
					title="Start next speaker"
					onclick={() =>
						runSpeakerAction('start-next', async () => {
							const next = nextWaitingSpeaker();
							if (!next) {
								throw new Error('No waiting speaker is available.');
							}
							return await speakerClient.setSpeakerSpeaking({
								committeeSlug: slug,
								meetingId,
								speakerId: next.speakerId
							});
						})}
					disabled={speakerActionPending !== '' || !nextWaitingSpeaker()}
				>
					Start Next
				</button>
				<button
					class="btn btn-outline btn-sm"
					data-testid="manage-end-current-speaker"
					title="End current speech"
					onclick={() =>
						runSpeakerAction('end-current', async () => {
							const current = activeSpeaker();
							if (!current) {
								throw new Error('No active speaker is available.');
							}
							return await speakerClient.setSpeakerDone({
								committeeSlug: slug,
								meetingId,
								speakerId: current.speakerId
							});
						})}
					disabled={speakerActionPending !== '' || !activeSpeaker()}
				>
					End Current
				</button>
			</div>

			<div id="speakers-list-container">
				{#if speakerState.data?.speakers?.length}
					<div class="space-y-2" data-testid="manage-speakers-viewport">
						{#each speakerState.data.speakers as speaker}
							<div
								class="rounded-box border border-base-300 px-3 py-3"
								data-testid="live-speaker-item"
								data-speaker-state={speaker.state.toLowerCase()}
							>
								<div class="flex flex-wrap items-start justify-between gap-3">
									<div>
										<div class="font-medium" data-testid="live-speaker-name">{speaker.fullName}</div>
										<div class="text-sm text-base-content/70">
											{speaker.speakerType} • {speaker.state}
										</div>
									</div>
									<div class="flex flex-wrap gap-2">
										{#if speaker.state === 'WAITING'}
											<button
												class="btn btn-ghost btn-xs"
												title={speaker.priority ? 'Remove Priority' : 'Give Priority'}
												onclick={() =>
													runSpeakerAction(`priority-${speaker.speakerId}`, async () => {
														return await speakerClient.setSpeakerPriority({
															committeeSlug: slug,
															meetingId,
															speakerId: speaker.speakerId,
															priority: !speaker.priority
														});
													})}
												disabled={speakerActionPending !== ''}
											>
												{speaker.priority ? 'Priority On' : 'Give Priority'}
											</button>
											<button
												class="btn btn-ghost btn-xs"
												title="Start"
												onclick={() =>
													runSpeakerAction(`start-${speaker.speakerId}`, async () => {
														return await speakerClient.setSpeakerSpeaking({
															committeeSlug: slug,
															meetingId,
															speakerId: speaker.speakerId
														});
													})}
												disabled={speakerActionPending !== ''}
											>
												Start
											</button>
										{/if}
										{#if speaker.state !== 'DONE'}
											<button
												class="btn btn-ghost btn-xs text-error"
												title="Remove"
												onclick={() =>
													runSpeakerAction(`remove-${speaker.speakerId}`, async () => {
														return await speakerClient.removeSpeaker({
															committeeSlug: slug,
															meetingId,
															speakerId: speaker.speakerId
														});
													})}
												disabled={speakerActionPending !== ''}
											>
												Remove
											</button>
										{/if}
									</div>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<p class="text-base-content/70">No speakers are queued for the active agenda point.</p>
				{/if}
			</div>
		</AppCard>

		<AppCard title="Current Agenda Point">
			{#if moderationState.data.activeAgendaPoint}
				<p class="font-medium">
					{moderationState.data.activeAgendaPoint.displayNumber}
					{#if moderationState.data.activeAgendaPoint.title}
						: {moderationState.data.activeAgendaPoint.title}
					{/if}
				</p>
			{:else}
				<p class="text-base-content/70">No agenda point is currently active.</p>
			{/if}
		</AppCard>
	{/if}
</div>
