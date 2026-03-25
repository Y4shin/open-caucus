<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { agendaClient, attendeeClient, moderationClient, speakerClient, voteClient } from '$lib/api/index.js';
	import { session } from '$lib/stores/session.svelte.js';
	import type { AgendaPointRecord } from '$lib/gen/conference/agenda/v1/agenda_pb.js';
	import type { AttendeeRecord } from '$lib/gen/conference/attendees/v1/attendees_pb.js';
	import type { ModerationView } from '$lib/gen/conference/moderation/v1/moderation_pb.js';
	import type { SpeakerQueueView } from '$lib/gen/conference/speakers/v1/speakers_pb.js';
	import type {
		VoteDefinitionRecord,
		VoteTallyEntry,
		VotesPanelView
	} from '$lib/gen/conference/votes/v1/votes_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { connectEventStream } from '$lib/utils/sse.js';

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);

	let moderationState = $state(createRemoteState<ModerationView>());
	let speakerState = $state(createRemoteState<SpeakerQueueView>());
	let attendeeState = $state(createRemoteState<AttendeeRecord[]>());
	let agendaState = $state(createRemoteState<AgendaPointRecord[]>());
	let votesState = $state(createRemoteState<VotesPanelView>());
	let actionError = $state('');
	let togglingSignup = $state(false);
	let speakerActionPending = $state('');
	let agendaActionPending = $state('');
	let voteActionPending = $state('');
	let creatingAgenda = $state(false);
	let creatingVote = $state(false);
	let agendaTitle = $state('');
	let voteName = $state('');
	let voteVisibility = $state<'open' | 'secret'>('open');
	let voteMinSelections = $state('1');
	let voteMaxSelections = $state('1');
	let voteOptionsText = $state('Yes\nNo');
	let lastClosedVote = $state<{
		vote: VoteDefinitionRecord;
		tally: VoteTallyEntry[];
		outcome: string;
	} | null>(null);
	let speakerSearch = $state('');
	let searchInput = $state<HTMLInputElement | null>(null);
	let agendaTitleInput = $state<HTMLInputElement | null>(null);
	let voteNameInput = $state<HTMLInputElement | null>(null);
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
		attendeeState.loading = true;
		agendaState.loading = true;
		votesState.loading = true;
		moderationState.error = '';
		speakerState.error = '';
		attendeeState.error = '';
		agendaState.error = '';
		votesState.error = '';
		try {
			const [moderationRes, speakerRes, attendeeRes, agendaRes, votesRes] = await Promise.all([
				moderationClient.getModerationView({ committeeSlug: slug, meetingId }),
				speakerClient.listSpeakers({ committeeSlug: slug, meetingId }),
				attendeeClient.listAttendees({ committeeSlug: slug, meetingId }),
				agendaClient.listAgendaPoints({ committeeSlug: slug, meetingId }),
				voteClient.getVotesPanel({ committeeSlug: slug, meetingId })
			]);
			moderationState.data = moderationRes.view ?? null;
			speakerState.data = speakerRes.view ?? null;
			attendeeState.data = attendeeRes.attendees;
			agendaState.data = agendaRes.agendaPoints;
			votesState.data = votesRes.view ?? null;
		} catch (err) {
			moderationState.error = getDisplayError(err, 'Failed to load the moderation view.');
			speakerState.error = moderationState.error;
			attendeeState.error = moderationState.error;
			agendaState.error = moderationState.error;
			votesState.error = moderationState.error;
		} finally {
			moderationState.loading = false;
			speakerState.loading = false;
			attendeeState.loading = false;
			agendaState.loading = false;
			votesState.loading = false;
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
			return true;
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to update the speakers queue.');
			refreshTick += 1;
			return false;
		} finally {
			speakerActionPending = '';
		}
	}

	async function runAgendaAction(key: string, action: () => Promise<void>) {
		actionError = '';
		agendaActionPending = key;
		try {
			await action();
			refreshTick += 1;
			return true;
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to update the agenda.');
			refreshTick += 1;
			return false;
		} finally {
			agendaActionPending = '';
		}
	}

	async function runVoteAction(key: string, action: () => Promise<void>) {
		actionError = '';
		voteActionPending = key;
		try {
			await action();
			return true;
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to update the votes panel.');
			refreshTick += 1;
			return false;
		} finally {
			voteActionPending = '';
		}
	}

	function normalized(value: string) {
		return value.trim().toLowerCase();
	}

	function hasOpenSpeaker(attendeeId: string, speakerType: string) {
		return (speakerState.data?.speakers ?? []).some(
			(speaker) => speaker.attendeeId === attendeeId && speaker.speakerType === speakerType
		);
	}

	function candidateRank(attendee: AttendeeRecord, query: string) {
		const trimmed = normalized(query);
		if (!trimmed) return 1000 + Number(attendee.attendeeNumber);

		const name = normalized(attendee.fullName);
		const number = attendee.attendeeNumber.toString();
		const words = name.split(/\s+/);

		if (number === trimmed) return 0;
		if (name === trimmed) return 10;
		if (words.some((word) => word === trimmed)) return 20;
		if (name.startsWith(trimmed)) return 30;
		if (words.some((word) => word.startsWith(trimmed))) return 40;
		if (name.includes(trimmed)) return 50;
		if (number.includes(trimmed)) return 60;
		return Number.POSITIVE_INFINITY;
	}

	function sortedCandidates() {
		return [...(attendeeState.data ?? [])]
			.filter((attendee) => candidateRank(attendee, speakerSearch) < Number.POSITIVE_INFINITY)
			.sort((left, right) => {
				const rankDiff = candidateRank(left, speakerSearch) - candidateRank(right, speakerSearch);
				if (rankDiff !== 0) return rankDiff;

				const lengthDiff = left.fullName.length - right.fullName.length;
				if (lengthDiff !== 0) return lengthDiff;

				const nameDiff = left.fullName.localeCompare(right.fullName);
				if (nameDiff !== 0) return nameDiff;

				return Number(left.attendeeNumber - right.attendeeNumber);
			});
	}

	async function addCandidate(attendeeId: string, speakerType: string) {
		const didAdd = await runSpeakerAction(`add-${attendeeId}-${speakerType}`, async () => {
			return await speakerClient.addSpeaker({
				committeeSlug: slug,
				meetingId,
				attendeeId,
				speakerType
			});
		});
		if (didAdd) {
			speakerSearch = '';
			searchInput?.focus();
		}
	}

	async function handleSpeakerSearchEnter(event: KeyboardEvent) {
		if (event.key !== 'Enter') return;
		event.preventDefault();

		const topCandidate = sortedCandidates()[0];
		if (!topCandidate || hasOpenSpeaker(topCandidate.attendeeId, 'regular')) {
			return;
		}

		await addCandidate(topCandidate.attendeeId, 'regular');
	}

	function isAgendaBusy(key: string) {
		return agendaActionPending !== '' && agendaActionPending !== key;
	}

	async function createAgendaPoint() {
		const title = agendaTitle.trim();
		if (!title || creatingAgenda) return;

		actionError = '';
		creatingAgenda = true;
		try {
			await agendaClient.createAgendaPoint({
				committeeSlug: slug,
				meetingId,
				title
			});
			agendaTitle = '';
			refreshTick += 1;
			agendaTitleInput?.focus();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to create the agenda point.');
		} finally {
			creatingAgenda = false;
		}
	}

	async function handleAgendaTitleKeydown(event: KeyboardEvent) {
		if (event.key !== 'Enter') return;
		event.preventDefault();
		await createAgendaPoint();
	}

	async function activateAgendaPoint(agendaPointId: string, active: boolean) {
		await runAgendaAction(`activate-${agendaPointId}`, async () => {
			const res = await agendaClient.activateAgendaPoint({
				committeeSlug: slug,
				meetingId,
				agendaPointId: active ? '' : agendaPointId
			});

			if (moderationState.data) {
				moderationState.data.activeAgendaPoint = res.activeAgendaPoint;
			}
		});
	}

	async function moveAgendaPoint(agendaPointId: string, direction: 'up' | 'down') {
		await runAgendaAction(`move-${agendaPointId}-${direction}`, async () => {
			const res = await agendaClient.moveAgendaPoint({
				committeeSlug: slug,
				meetingId,
				agendaPointId,
				direction
			});
			agendaState.data = res.agendaPoints;
		});
	}

	async function deleteAgendaPoint(agendaPointId: string) {
		await runAgendaAction(`delete-${agendaPointId}`, async () => {
			await agendaClient.deleteAgendaPoint({
				committeeSlug: slug,
				meetingId,
				agendaPointId
			});
		});
	}

	function parsedVoteOptions() {
		return voteOptionsText
			.split('\n')
			.map((line) => line.trim())
			.filter(Boolean);
	}

	function bigintFromInput(value: string) {
		const parsed = Number.parseInt(value, 10);
		return Number.isFinite(parsed) ? BigInt(parsed) : 0n;
	}

	function canCreateVote() {
		return voteName.trim().length > 0 && parsedVoteOptions().length >= 2;
	}

	async function createVote() {
		if (creatingVote || !canCreateVote()) return;

		actionError = '';
		creatingVote = true;
		try {
			const res = await voteClient.createVote({
				committeeSlug: slug,
				meetingId,
				name: voteName.trim(),
				visibility: voteVisibility,
				minSelections: bigintFromInput(voteMinSelections),
				maxSelections: bigintFromInput(voteMaxSelections),
				optionLabels: parsedVoteOptions()
			});

			lastClosedVote = null;
			if (votesState.data && res.vote) {
				votesState.data.votes = [...votesState.data.votes, res.vote];
			}
			voteName = '';
			voteVisibility = 'open';
			voteMinSelections = '1';
			voteMaxSelections = '1';
			voteOptionsText = 'Yes\nNo';
			refreshTick += 1;
			voteNameInput?.focus();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to create the vote.');
		} finally {
			creatingVote = false;
		}
	}

	async function openVote(voteId: string) {
		await runVoteAction(`open-${voteId}`, async () => {
			const res = await voteClient.openVote({
				committeeSlug: slug,
				meetingId,
				voteId
			});
			lastClosedVote = null;
			if (votesState.data) {
				votesState.data.activeVote = res.vote;
				votesState.data.activeVoteStats = res.stats;
			}
			refreshTick += 1;
		});
	}

	async function closeVote(voteId: string) {
		await runVoteAction(`close-${voteId}`, async () => {
			const res = await voteClient.closeVote({
				committeeSlug: slug,
				meetingId,
				voteId
			});
			lastClosedVote = res.vote
				? {
						vote: res.vote,
						tally: res.tally,
						outcome: res.outcome
					}
				: null;
			if (votesState.data) {
				votesState.data.activeVote = undefined;
				votesState.data.activeVoteStats = undefined;
				votesState.data.activeVoteTally = [];
			}
			refreshTick += 1;
		});
	}

	async function archiveVote(voteId: string) {
		await runVoteAction(`archive-${voteId}`, async () => {
			await voteClient.archiveVote({
				committeeSlug: slug,
				meetingId,
				voteId
			});
			if (lastClosedVote?.vote.voteId === voteId) {
				lastClosedVote = null;
			}
			refreshTick += 1;
		});
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

			{#if moderationState.data.activeAgendaPoint}
				<div class="mb-4 rounded-box border border-base-300 bg-base-200/40 p-4">
					<div class="mb-3">
						<label class="label" for="speaker-add-search-input">
							<span class="label-text font-medium">Add Speaker</span>
						</label>
						<input
							id="speaker-add-search-input"
							class="input input-bordered w-full"
							bind:value={speakerSearch}
							bind:this={searchInput}
							onkeydown={handleSpeakerSearchEnter}
							placeholder="Search by attendee name or number"
						/>
					</div>

					<div id="speaker-add-candidates-container" class="space-y-2">
						{#each sortedCandidates().slice(0, 8) as attendee}
							<div
								class="flex flex-wrap items-center justify-between gap-3 rounded-box border border-base-300 bg-base-100 px-3 py-3"
								data-testid="manage-speaker-candidate-card"
							>
								<div>
									<div class="font-medium">{attendee.fullName}</div>
									<div class="text-sm text-base-content/70">
										#{attendee.attendeeNumber.toString()}
										{#if attendee.isGuest}
											• Guest
										{/if}
										{#if attendee.quoted}
											• Quoted
										{/if}
									</div>
								</div>
								<div class="flex flex-wrap gap-2">
									<button
										class="btn btn-primary btn-xs"
										title="Add regular speech"
										onclick={() => addCandidate(attendee.attendeeId, 'regular')}
										disabled={
											speakerActionPending !== '' || hasOpenSpeaker(attendee.attendeeId, 'regular')
										}
									>
										Regular
									</button>
									<button
										class="btn btn-outline btn-xs"
										title="Add Point of Order (PO) speech"
										onclick={() => addCandidate(attendee.attendeeId, 'ropm')}
										disabled={
											speakerActionPending !== '' || hasOpenSpeaker(attendee.attendeeId, 'ropm')
										}
									>
										PO
									</button>
								</div>
							</div>
						{/each}
						{#if sortedCandidates().length === 0}
							<p class="text-sm text-base-content/70">No matching attendees found.</p>
						{/if}
					</div>
				</div>
			{/if}

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

		<AppCard title="Agenda Management">
			<div class="space-y-4">
				<div class="flex flex-col gap-3 md:flex-row">
					<input
						class="input input-bordered flex-1"
						placeholder="Add a top-level agenda point"
						bind:value={agendaTitle}
						bind:this={agendaTitleInput}
						onkeydown={handleAgendaTitleKeydown}
					/>
					<button
						class="btn btn-primary"
						onclick={createAgendaPoint}
						disabled={creatingAgenda || agendaTitle.trim().length === 0}
					>
						{#if creatingAgenda}
							<span class="loading loading-spinner loading-xs"></span>
						{/if}
						Add Agenda Point
					</button>
				</div>

				{#if agendaState.loading}
					<AppSpinner label="Loading agenda" />
				{:else if agendaState.error}
					<AppAlert message={agendaState.error} />
				{:else if agendaState.data?.length}
					<div class="space-y-2" id="manage-agenda-list">
						{#snippet agendaRows(points: AgendaPointRecord[], depth: number)}
							{#each points as point, index}
								<div
									class="rounded-box border border-base-300 bg-base-100 px-3 py-3"
									data-testid="manage-agenda-item"
									data-agenda-active={point.isActive ? 'true' : 'false'}
								>
									<div class="flex flex-wrap items-start justify-between gap-3">
										<div class="min-w-0" style={`padding-left: ${depth * 1.25}rem`}>
											<div class="flex flex-wrap items-center gap-2">
												<span class="font-medium">{point.displayNumber}</span>
												{#if point.title}
													<span>{point.title}</span>
												{:else}
													<span class="text-base-content/60">Untitled</span>
												{/if}
												{#if point.isActive}
													<span class="badge badge-primary badge-sm">Active</span>
												{/if}
											</div>
											<div class="mt-1 text-sm text-base-content/70">
												Position {point.position.toString()}
												{#if point.genderQuotation}
													• Gender quotation
												{/if}
												{#if point.firstSpeakerQuotation}
													• First speaker quotation
												{/if}
											</div>
										</div>

										<div class="flex flex-wrap gap-2">
											<button
												class="btn btn-ghost btn-xs"
												title={point.isActive ? 'Deactivate' : 'Activate'}
												onclick={() => activateAgendaPoint(point.agendaPointId, point.isActive)}
												disabled={isAgendaBusy(`activate-${point.agendaPointId}`)}
											>
												{point.isActive ? 'Deactivate' : 'Activate'}
											</button>
											{#if !point.parentId}
												<button
													class="btn btn-ghost btn-xs"
													title="Move up"
													onclick={() => moveAgendaPoint(point.agendaPointId, 'up')}
													disabled={index === 0 || isAgendaBusy(`move-${point.agendaPointId}-up`)}
												>
													Up
												</button>
												<button
													class="btn btn-ghost btn-xs"
													title="Move down"
													onclick={() => moveAgendaPoint(point.agendaPointId, 'down')}
													disabled={
														index === points.length - 1 || isAgendaBusy(`move-${point.agendaPointId}-down`)
													}
												>
													Down
												</button>
											{/if}
											<button
												class="btn btn-ghost btn-xs text-error"
												title="Delete"
												onclick={() => deleteAgendaPoint(point.agendaPointId)}
												disabled={isAgendaBusy(`delete-${point.agendaPointId}`)}
											>
												Delete
											</button>
										</div>
									</div>
								</div>

								{#if point.subPoints.length}
									{@render agendaRows(point.subPoints, depth + 1)}
								{/if}
							{/each}
						{/snippet}

						{@render agendaRows(agendaState.data, 0)}
					</div>
				{:else}
					<p class="text-base-content/70">No agenda points have been created yet.</p>
				{/if}
			</div>
		</AppCard>

		<AppCard title="Votes">
			{#if votesState.loading}
				<AppSpinner label="Loading votes" />
			{:else if votesState.error}
				<AppAlert message={votesState.error} />
			{:else if votesState.data}
				<div class="space-y-4" id="moderate-votes-panel">
					{#if !votesState.data.hasActiveAgendaPoint}
						<p class="text-base-content/70">No active agenda point.</p>
					{:else}
						<p class="text-sm text-base-content/70">
							Active agenda point: {votesState.data.activeAgendaPointTitle || 'Current item'}
						</p>

						<div class="grid gap-3 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,0.9fr)]">
							<div class="rounded-box border border-base-300 bg-base-100 p-4">
								<div class="mb-3">
									<h3 class="font-semibold">Create Draft Vote</h3>
									<p class="text-sm text-base-content/70">
										Create a draft, then open it when the chair is ready.
									</p>
								</div>

								<div class="space-y-3">
									<input
										class="input input-bordered w-full"
										placeholder="Vote name"
										bind:value={voteName}
										bind:this={voteNameInput}
									/>

									<div class="grid gap-3 sm:grid-cols-3">
										<select class="select select-bordered" bind:value={voteVisibility}>
											<option value="open">Open</option>
											<option value="secret">Secret</option>
										</select>
										<input
											class="input input-bordered"
											type="number"
											min="1"
											bind:value={voteMinSelections}
											placeholder="Min"
										/>
										<input
											class="input input-bordered"
											type="number"
											min="1"
											bind:value={voteMaxSelections}
											placeholder="Max"
										/>
									</div>

									<textarea
										class="textarea textarea-bordered min-h-32 w-full"
										bind:value={voteOptionsText}
										placeholder="One option per line"
									></textarea>

									<button
										class="btn btn-primary"
										onclick={createVote}
										disabled={creatingVote || !canCreateVote()}
									>
										{#if creatingVote}
											<span class="loading loading-spinner loading-xs"></span>
										{/if}
										Create Draft Vote
									</button>
								</div>
							</div>

							<div class="rounded-box border border-base-300 bg-base-100 p-4">
								<h3 class="mb-3 font-semibold">Vote Status</h3>
								{#if votesState.data.activeVote}
									<div class="space-y-2">
										<div class="flex flex-wrap items-center gap-2">
											<span class="font-medium">{votesState.data.activeVote.name}</span>
											<span class="badge badge-primary">{votesState.data.activeVote.state}</span>
										</div>
										{#if votesState.data.activeVoteStats}
											<p class="text-sm text-base-content/70">
												{votesState.data.activeVoteStats.castCount.toString()} of
												{votesState.data.activeVoteStats.eligibleCount.toString()} eligible voters
												have cast ballots.
											</p>
										{/if}
										<button
											class="btn btn-secondary btn-sm"
											onclick={() => closeVote(votesState.data?.activeVote?.voteId ?? '')}
											disabled={
												!votesState.data.activeVote ||
												voteActionPending !== '' ||
												votesState.data.activeVote.voteId.length === 0
											}
										>
											Close Vote
										</button>
									</div>
								{:else if lastClosedVote}
									<div class="space-y-3">
										<div class="flex flex-wrap items-center gap-2">
											<span class="font-medium">{lastClosedVote.vote.name}</span>
											<span class="badge badge-outline">{lastClosedVote.outcome}</span>
										</div>
										{#if lastClosedVote.tally.length}
											<div class="space-y-2">
												<p class="text-sm font-medium">Final Tallies</p>
												{#each lastClosedVote.tally as entry}
													<div class="flex items-center justify-between text-sm">
														<span>{entry.label}</span>
														<span>{entry.count.toString()}</span>
													</div>
												{/each}
											</div>
										{/if}
										<button
											class="btn btn-outline btn-sm"
											onclick={() => lastClosedVote && archiveVote(lastClosedVote.vote.voteId)}
											disabled={voteActionPending !== ''}
										>
											Archive Vote
										</button>
									</div>
								{:else}
									<p class="text-sm text-base-content/70">
										No vote is currently open for the active agenda point.
									</p>
								{/if}
							</div>
						</div>

						<div class="space-y-3">
							<h3 class="font-semibold">Vote Definitions</h3>
							{#if votesState.data.votes.length}
								<div class="space-y-3">
									{#each votesState.data.votes as vote}
										<div class="rounded-box border border-base-300 bg-base-100 p-4">
											<div class="flex flex-wrap items-start justify-between gap-3">
												<div class="space-y-1">
													<div class="flex flex-wrap items-center gap-2">
														<span class="font-medium">{vote.name}</span>
														<span class="badge badge-outline">{vote.state}</span>
														<span class="badge badge-ghost">{vote.visibility}</span>
													</div>
													<p class="text-sm text-base-content/70">
														{vote.minSelections.toString()} to {vote.maxSelections.toString()} selections
													</p>
													<p class="text-sm text-base-content/70">
														{vote.options.map((option) => option.label).join(' • ')}
													</p>
												</div>

												<div class="flex flex-wrap gap-2">
													{#if vote.state === 'draft'}
														<button
															class="btn btn-primary btn-xs"
															onclick={() => openVote(vote.voteId)}
															disabled={voteActionPending !== ''}
														>
															Open Vote
														</button>
													{/if}
													{#if vote.state === 'closed'}
														<button
															class="btn btn-outline btn-xs"
															onclick={() => archiveVote(vote.voteId)}
															disabled={voteActionPending !== ''}
														>
															Archive Vote
														</button>
													{/if}
												</div>
											</div>
										</div>
									{/each}
								</div>
							{:else}
								<p class="text-sm text-base-content/70">
									No votes have been defined for the active agenda point yet.
								</p>
							{/if}
						</div>
					{/if}
				</div>
			{/if}
		</AppCard>
	{/if}
</div>
