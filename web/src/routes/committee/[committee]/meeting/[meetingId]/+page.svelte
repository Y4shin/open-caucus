<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { meetingClient, speakerClient, voteClient } from '$lib/api/index.js';
	import { session } from '$lib/stores/session.svelte.js';
	import type { LiveMeetingView } from '$lib/gen/conference/meetings/v1/meetings_pb.js';
	import { MeetingEventKind } from '$lib/gen/conference/meetings/v1/meetings_pb.js';
	import type { SpeakerQueueView } from '$lib/gen/conference/speakers/v1/speakers_pb.js';
	import type { LiveVotePanelView } from '$lib/gen/conference/votes/v1/votes_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { saveReceipt } from '$lib/utils/receipts.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);

	let liveState = $state(createRemoteState<LiveMeetingView>());
	let speakerState = $state(createRemoteState<SpeakerQueueView>());
	let voteState = $state(createRemoteState<LiveVotePanelView>());
	let actionError = $state('');
	let addingRegular = $state(false);
	let addingRopm = $state(false);
	let submittingVote = $state(false);
	let selectedOptionIds = $state<string[]>([]);
	let voteReceipt = $state('');

	$effect(() => {
		const activeVote = voteState.data?.activeVote;
		if (!activeVote) {
			selectedOptionIds = [];
			return;
		}
		const validOptionIds = new Set(activeVote.options.map((option) => option.optionId));
		selectedOptionIds = selectedOptionIds.filter((optionId) => validOptionIds.has(optionId));
	});

	$effect(() => {
		if (!session.loaded) return;
		if (!session.authenticated) {
			goto(`/committee/${slug}/meeting/${meetingId}/join`);
			return;
		}
		loadMeeting();
	});

	// Subscribe to the typed Connect stream and selectively refetch only the view that changed.
	$effect(() => {
		if (!session.loaded || !session.authenticated) return;
		const currentSlug = slug;
		const currentMeetingId = meetingId;
		let cancelled = false;
		(async () => {
			try {
				const stream = meetingClient.subscribeMeetingEvents({
					committeeSlug: currentSlug,
					meetingId: currentMeetingId
				});
				for await (const event of stream) {
					if (cancelled) break;
					switch (event.kind) {
						case MeetingEventKind.SPEAKERS_UPDATED:
							loadSpeakers();
							break;
						case MeetingEventKind.VOTES_UPDATED:
							loadVotes();
							break;
						case MeetingEventKind.AGENDA_UPDATED:
						case MeetingEventKind.MEETING_UPDATED:
						case MeetingEventKind.ATTENDEES_UPDATED:
							loadLiveMeeting();
							break;
					}
				}
			} catch {
				// Stream closed or server went away — ignore; the initial load already has the data.
			}
		})();
		return () => {
			cancelled = true;
		};
	});

	async function loadMeeting() {
		liveState.loading = true;
		speakerState.loading = true;
		voteState.loading = true;
		liveState.error = '';
		speakerState.error = '';
		voteState.error = '';
		try {
			const [meetingRes, speakerRes, voteRes] = await Promise.all([
				meetingClient.getLiveMeeting({ committeeSlug: slug, meetingId }),
				speakerClient.listSpeakers({ committeeSlug: slug, meetingId }),
				voteClient.getLiveVotePanel({ committeeSlug: slug, meetingId })
			]);
			liveState.data = meetingRes.meeting ?? null;
			speakerState.data = speakerRes.view ?? null;
			voteState.data = voteRes.view ?? null;
		} catch (err) {
			liveState.error = getDisplayError(err, 'Failed to load the live meeting view.');
			speakerState.error = liveState.error;
			voteState.error = liveState.error;
		} finally {
			liveState.loading = false;
			speakerState.loading = false;
			voteState.loading = false;
		}
	}

	async function loadLiveMeeting() {
		try {
			const res = await meetingClient.getLiveMeeting({ committeeSlug: slug, meetingId });
			liveState.data = res.meeting ?? null;
		} catch {
			// Silent refresh — don't clobber existing data on transient errors
		}
	}

	async function loadSpeakers() {
		try {
			const res = await speakerClient.listSpeakers({ committeeSlug: slug, meetingId });
			speakerState.data = res.view ?? null;
		} catch {
			// Silent refresh
		}
	}

	async function loadVotes() {
		try {
			const res = await voteClient.getLiveVotePanel({ committeeSlug: slug, meetingId });
			voteState.data = res.view ?? null;
		} catch {
			// Silent refresh
		}
	}

	function hasWaitingEntry(type: string) {
		return (speakerState.data?.speakers ?? []).some(
			(speaker) => speaker.mine && speaker.state === 'WAITING' && speaker.speakerType === type
		);
	}

	function visibleSpeakers() {
		return (speakerState.data?.speakers ?? liveState.data?.speakers ?? []).filter(
			(speaker) => speaker.state !== 'DONE' && speaker.state !== 'WITHDRAWN'
		);
	}

	async function addSelfSpeaker(speakerType: string) {
		if (speakerType === 'regular') {
			addingRegular = true;
		} else {
			addingRopm = true;
		}
		actionError = '';
		try {
			const res = await speakerClient.addSpeaker({
				committeeSlug: slug,
				meetingId,
				speakerType
			});
			speakerState.data = res.view ?? speakerState.data;
			loadSpeakers();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to add you to the speakers list.');
		} finally {
			if (speakerType === 'regular') {
				addingRegular = false;
			} else {
				addingRopm = false;
			}
		}
	}

	function chooseVoteOption(optionId: string, multiSelect: boolean) {
		if (!multiSelect) {
			selectedOptionIds = [optionId];
			return;
		}
		if (selectedOptionIds.includes(optionId)) {
			selectedOptionIds = selectedOptionIds.filter((id) => id !== optionId);
			return;
		}
		selectedOptionIds = [...selectedOptionIds, optionId];
	}

	async function submitBallot() {
		const activeVote = voteState.data?.activeVote;
		if (!activeVote || selectedOptionIds.length === 0 || submittingVote) return;

		submittingVote = true;
		actionError = '';
		voteReceipt = '';
		try {
			const res = await voteClient.submitBallot({
				committeeSlug: slug,
				meetingId,
				voteId: activeVote.voteId,
				selectedOptionIds
			});
			voteReceipt = res.receiptToken;
			saveReceipt({
				id: `${activeVote.visibility}:${activeVote.voteId}:${res.receiptToken}`,
				kind: activeVote.visibility as 'open' | 'secret',
				voteId: activeVote.voteId,
				voteName: activeVote.name,
				receiptToken: res.receiptToken,
				receipt: `${activeVote.voteId}:${res.receiptToken}`
			});
			selectedOptionIds = [];
			loadVotes();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to submit your ballot.');
		} finally {
			submittingVote = false;
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

		{#if actionError}
			<AppAlert message={actionError} />
		{/if}

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

				{#if speakerState.data?.canAddSelf}
					<div class="mt-4 flex flex-wrap gap-2">
						<button
							class="btn btn-secondary btn-sm"
							data-testid="live-add-self-regular"
							onclick={() => addSelfSpeaker('regular')}
							disabled={addingRegular || hasWaitingEntry('regular')}
						>
							{#if addingRegular}
								<span class="loading loading-spinner loading-xs"></span>
							{/if}
							Add Myself (Regular)
						</button>
						<button
							class="btn btn-outline btn-sm"
							data-testid="live-add-self-ropm"
							onclick={() => addSelfSpeaker('ropm')}
							disabled={addingRopm || hasWaitingEntry('ropm')}
						>
							{#if addingRopm}
								<span class="loading loading-spinner loading-xs"></span>
							{/if}
							Add Myself (PO)
						</button>
					</div>
				{/if}
			</AppCard>
		</div>

		<AppCard title="Speakers">
			<div id="attendee-speakers-list">
				{#if visibleSpeakers().length}
					<div class="space-y-2" data-testid="live-speakers-active-viewport">
						{#each visibleSpeakers() as speaker}
							<div
								class="flex items-center justify-between rounded-box border border-base-300 px-3 py-2"
								data-testid="live-speaker-item"
								data-speaker-state={speaker.state.toLowerCase()}
								data-speaker-mine={speaker.mine ? 'true' : 'false'}
							>
							<div>
								<div class="font-medium" data-testid="live-speaker-name">{speaker.fullName}</div>
								<div class="text-sm text-base-content/70">
									{speaker.speakerType} • {speaker.state}
								</div>
							</div>
							<div class="flex items-center gap-2">
								{#if speaker.quoted}
									<span class="badge badge-outline badge-sm" data-testid="live-speaker-quoted-badge">Q</span>
								{/if}
								{#if speaker.mine}
									<span class="badge badge-primary">You</span>
								{/if}
							</div>
							</div>
						{/each}
					</div>
				{:else}
					<p class="text-base-content/70">No speakers are queued right now.</p>
				{/if}
			</div>
		</AppCard>

		<AppCard title="Vote">
			<div id="live-votes-panel">
				{#if voteState.loading}
					<AppSpinner label="Loading vote" />
				{:else if voteState.error}
					<AppAlert message={voteState.error} />
				{:else if voteState.data?.hasActiveVote && voteState.data.activeVote}
					<div class="space-y-4" data-vote-card>
						<div>
							<div class="flex flex-wrap items-center gap-2">
								<h3 class="text-xl font-semibold">{voteState.data.activeVote.name}</h3>
								<span class="badge badge-outline">{voteState.data.activeVote.visibility}</span>
							</div>
							<p class="text-sm text-base-content/70">
								Choose between {voteState.data.activeVote.minSelections.toString()} and
								{voteState.data.activeVote.maxSelections.toString()} options.
							</p>
						</div>

						{#if voteReceipt}
							<div class="rounded-box border border-success/30 bg-success/10 px-3 py-2 text-sm">
								Ballot received. Receipt token: <span class="font-mono">{voteReceipt}</span>
								<a class="link link-primary ml-2" href="/receipts">Open vault</a>
							</div>
						{/if}

						{#if voteState.data.alreadyVoted}
							<div class="rounded-box border border-base-300 bg-base-200/50 px-3 py-2 text-sm">
								You have already voted in this round.
							</div>
						{:else if !voteState.data.isEligible}
							<div class="rounded-box border border-warning/30 bg-warning/10 px-3 py-2 text-sm">
								You are not eligible to vote in this round.
							</div>
						{:else}
							<div class="space-y-3">
								{#each voteState.data.activeVote.options as option}
									<label class="flex items-center gap-3 rounded-box border border-base-300 px-3 py-3">
										<input
											type={voteState.data.activeVote.maxSelections > 1n ? 'checkbox' : 'radio'}
											name="live-vote-option"
											value={option.optionId}
											checked={selectedOptionIds.includes(option.optionId)}
											onchange={() =>
												chooseVoteOption(
													option.optionId,
													voteState.data?.activeVote?.maxSelections !== undefined &&
														voteState.data.activeVote.maxSelections > 1n
												)}
										/>
										<span>{option.label}</span>
									</label>
								{/each}

								<button
									class="btn btn-primary"
									onclick={submitBallot}
									disabled={submittingVote || selectedOptionIds.length === 0}
								>
									{#if submittingVote}
										<span class="loading loading-spinner loading-xs"></span>
									{/if}
									{voteState.data.activeVote.visibility === 'secret'
										? 'Submit Secret Ballot'
										: 'Submit Open Ballot'}
								</button>
							</div>
						{/if}
					</div>
				{:else}
					<p class="text-base-content/70">No vote is currently open.</p>
				{/if}
			</div>
		</AppCard>
	{/if}
</div>
