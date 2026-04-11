<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onDestroy, onMount } from 'svelte';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppTooltip from '$lib/components/ui/AppTooltip.svelte';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import SpeakerBadges from '$lib/components/ui/SpeakerBadges.svelte';
	import { agendaClient, meetingClient, moderationClient, speakerClient, voteClient } from '$lib/api/index.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import type { AgendaPointRecord } from '$lib/gen/conference/agenda/v1/agenda_pb.js';
	import type { LiveMeetingView } from '$lib/gen/conference/meetings/v1/meetings_pb.js';
	import { MeetingEventKind } from '$lib/gen/conference/meetings/v1/meetings_pb.js';
	import type { SpeakerQueueView } from '$lib/gen/conference/speakers/v1/speakers_pb.js';
	import type { LiveVoteCardView, LiveVotePanelView } from '$lib/gen/conference/votes/v1/votes_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { voteStateBadgeClass, voteVisibilityBadgeClass } from '$lib/utils/votes.js';
	import { listReceipts, saveReceipt, verifyReceipt, type StoredReceipt } from '$lib/utils/receipts.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import * as m from '$lib/paraglide/messages';

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);

	let liveState = $state(createRemoteState<LiveMeetingView>());
	let speakerState = $state(createRemoteState<SpeakerQueueView>());
	let voteState = $state(createRemoteState<LiveVotePanelView>());
	let agendaState = $state(createRemoteState<AgendaPointRecord[]>());
	let actionError = $state('');
	let addingRegular = $state(false);
	let addingRopm = $state(false);
	let submittingVote = $state(false);
	let selectedOptionIds = $state<string[]>([]);
	let nowMs = $state(Date.now());

	// Receipts dialog state
	let receiptsDialogEl = $state<HTMLDialogElement | null>(null);
	let meetingReceipts = $state<StoredReceipt[]>([]);
	let receiptVerifyingId = $state('');
	let receiptVerifyResults = $state<Record<string, string>>({});

	function openReceiptsDialog() {
		const allReceipts = listReceipts();
		const voteIds = new Set(
			(voteState.data?.closedVotes ?? [])
				.concat(voteState.data?.activeVote ? [voteState.data.activeVote] : [])
				.map((v) => v.vote?.voteId ?? '')
				.filter(Boolean)
		);
		meetingReceipts = allReceipts.filter((r) => voteIds.has(r.voteId));
		receiptVerifyResults = {};
		receiptsDialogEl?.showModal();
	}

	async function verifyMeetingReceipt(receipt: StoredReceipt) {
		receiptVerifyingId = receipt.id;
		try {
			const payload = await verifyReceipt(receipt);
			const labels =
				payload && 'choiceLabels' in payload && Array.isArray(payload.choiceLabels)
					? payload.choiceLabels.join(', ')
					: '';
			receiptVerifyResults[receipt.id] = `OK: ${labels || 'no choices'}`;
		} catch (err) {
			receiptVerifyResults[receipt.id] = `Error: ${err instanceof Error ? err.message : 'Verification failed.'}`;
		} finally {
			receiptVerifyingId = '';
		}
	}
	let speakingSinceMs = $state<Record<string, number>>({});
	let canModerate = $state(false);

	onDestroy(() => {
		pageActions.clear();
	});

	onMount(() => {
		let cancelled = false;
		let refreshInterval = 0;
		let clockInterval = 0;

		const waitForSession = async () => {
			while (!cancelled && !session.loaded) {
				await new Promise((resolve) => window.setTimeout(resolve, 25));
			}
		};

		const subscribeToMeetingEvents = async () => {
			try {
				const stream = meetingClient.subscribeMeetingEvents({
					committeeSlug: slug,
					meetingId
				});
				for await (const event of stream) {
					if (cancelled) break;
					switch (event.kind) {
						case MeetingEventKind.SPEAKERS_UPDATED:
							void loadSpeakers();
							void loadLiveMeeting();
							break;
						case MeetingEventKind.VOTES_UPDATED:
							void loadVotes();
							break;
						case MeetingEventKind.AGENDA_UPDATED:
						case MeetingEventKind.MEETING_UPDATED:
						case MeetingEventKind.ATTENDEES_UPDATED:
							void loadLiveMeeting();
							void loadAgenda();
							break;
					}
				}
			} catch {
				// Stream closed or server went away — ignore; periodic refresh will recover.
			}
		};

		clockInterval = window.setInterval(() => {
			nowMs = Date.now();
		}, 1000);

		void (async () => {
			await waitForSession();
			if (cancelled) return;
			if (!session.authenticated) {
				await goto(`/committee/${slug}/meeting/${meetingId}/join`);
				return;
			}
			await loadMeeting();
			if (cancelled) return;
			refreshInterval = window.setInterval(() => {
				void loadLiveMeeting();
				void loadSpeakers();
				void loadVotes();
				void loadAgenda();
			}, 2000);
			void subscribeToMeetingEvents();
		})();

		return () => {
			cancelled = true;
			if (refreshInterval) window.clearInterval(refreshInterval);
			if (clockInterval) window.clearInterval(clockInterval);
		};
	});

	async function loadMeeting() {
		liveState.loading = true;
		speakerState.loading = true;
		voteState.loading = true;
		agendaState.loading = true;
		liveState.error = '';
		speakerState.error = '';
		voteState.error = '';
		agendaState.error = '';
		try {
			const [meetingRes, speakerRes, voteRes, agendaRes] = await Promise.all([
				meetingClient.getLiveMeeting({ committeeSlug: slug, meetingId }),
				speakerClient.listSpeakers({ committeeSlug: slug, meetingId }),
				voteClient.getLiveVotePanel({ committeeSlug: slug, meetingId }),
				agendaClient.listAgendaPoints({ committeeSlug: slug, meetingId })
			]);
			liveState.data = meetingRes.meeting ?? null;
			speakerState.data = speakerRes.view ?? null;
			syncSpeakingSince(speakerState.data?.speakers ?? []);
			voteState.data = voteRes.view ?? null;
			syncSelectedOptionIds();
			agendaState.data = agendaRes.agendaPoints;
			void refreshModerationCapability();
		} catch (err) {
			liveState.error = getDisplayError(err, 'Failed to load the live meeting view.');
			speakerState.error = liveState.error;
			voteState.error = liveState.error;
			agendaState.error = liveState.error;
		} finally {
			liveState.loading = false;
			speakerState.loading = false;
			voteState.loading = false;
			agendaState.loading = false;
		}
	}

	async function refreshModerationCapability() {
		const title = liveState.data?.meetingName ?? '';
		const subtitle = liveState.data?.committeeName ?? '';
		try {
			await moderationClient.getModerationView({ committeeSlug: slug, meetingId });
			canModerate = true;
			pageActions.set(
				[{ label: 'Moderate', href: `/committee/${slug}/meeting/${meetingId}/moderate` }],
				{ backHref: `/committee/${slug}`, title, subtitle }
			);
		} catch {
			canModerate = false;
			pageActions.set([], { backHref: `/committee/${slug}`, title, subtitle });
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
			syncSpeakingSince(speakerState.data?.speakers ?? []);
		} catch {
			// Silent refresh
		}
	}

	async function loadVotes() {
		try {
			const res = await voteClient.getLiveVotePanel({ committeeSlug: slug, meetingId });
			voteState.data = res.view ?? null;
			syncSelectedOptionIds();
		} catch {
			// Silent refresh
		}
	}

	function syncSelectedOptionIds() {
		const activeVote = voteState.data?.activeVote;
		if (!activeVote) {
			if (selectedOptionIds.length > 0) {
				selectedOptionIds = [];
			}
			return;
		}
		const validOptionIds = new Set(activeVote.options.map((option) => option.optionId));
		const nextSelectedOptionIds = selectedOptionIds.filter((optionId) => validOptionIds.has(optionId));
		if (
			nextSelectedOptionIds.length !== selectedOptionIds.length ||
			nextSelectedOptionIds.some((optionId, index) => optionId !== selectedOptionIds[index])
		) {
			selectedOptionIds = nextSelectedOptionIds;
		}
	}

	function visibleLiveVotes() {
		return (voteState.data?.votes ?? []).filter((vote) => {
			if (!vote.hasTimedResults) return true;
			return vote.resultsUntilUnix > 0n && Number(vote.resultsUntilUnix) * 1000 > nowMs;
		});
	}

	function activeLiveVoteID() {
		return voteState.data?.activeVote?.voteId ?? '';
	}

	function isVoteSelectionActive(vote: LiveVoteCardView, optionId: string) {
		return vote.vote?.voteId === activeLiveVoteID() && selectedOptionIds.includes(optionId);
	}


	function voteStateLabel(state: string) {
		switch (state) {
			case 'draft':
				return m.votes_state_draft();
			case 'open':
				return m.votes_state_open();
			case 'counting':
				return m.votes_state_counting();
			case 'closed':
				return m.votes_state_closed();
			case 'archived':
				return m.votes_state_archived();
			default:
				return state;
		}
	}

	function voteVisibilityLabel(visibility: string) {
		return visibility === 'secret' ? m.votes_visibility_secret() : m.votes_visibility_open();
	}

	function voteBoundsLabel(vote: LiveVoteCardView) {
		const minSelections = Number(vote.vote?.minSelections ?? 0n);
		const maxSelections = Number(vote.vote?.maxSelections ?? 0n);
		if (minSelections === maxSelections) {
			return m.votes_select_exactly({ count: minSelections });
		}
		return m.votes_select_between({ min: minSelections, max: maxSelections });
	}

	function voteResultsRemaining(vote: LiveVoteCardView) {
		if (!vote.resultsUntilUnix) return 0;
		return Math.max(0, Math.ceil((Number(vote.resultsUntilUnix) * 1000 - nowMs) / 1000));
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

	function activeSpeakers() {
		return visibleSpeakers().filter((speaker) => speaker.state === 'WAITING' || speaker.state === 'SPEAKING');
	}

	function syncSpeakingSince(speakers: SpeakerQueueView['speakers']) {
		const next = { ...speakingSinceMs };
		const activeIds = new Set(speakers.map((speaker) => speaker.speakerId));
		for (const speaker of speakers) {
			if (speaker.state === 'SPEAKING' && next[speaker.speakerId] == null) {
				next[speaker.speakerId] = Date.now();
			}
		}
		for (const speakerId of Object.keys(next)) {
			if (!activeIds.has(speakerId)) delete next[speakerId];
		}
		speakingSinceMs = next;
	}

	function waitingDisplayNumber(speakerId: string) {
		let position = 0;
		for (const speaker of activeSpeakers()) {
			if (speaker.state === 'WAITING') {
				position++;
				if (speaker.speakerId === speakerId) return position;
			}
		}
		return 0;
	}

	function formatElapsed(totalMs: number) {
		const totalSeconds = Math.max(0, Math.floor(totalMs / 1000));
		const mins = Math.floor(totalSeconds / 60);
		const secs = totalSeconds % 60;
		return `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
	}

	function speakingTimerLabel(speakerId: string) {
		const since = speakingSinceMs[speakerId];
		if (since == null) return '00:00';
		return formatElapsed(nowMs - since);
	}

	function speakerHasBadges(speaker: SpeakerQueueView['speakers'][number]) {
		return speaker.speakerType === 'ropm' || speaker.quoted || speaker.firstSpeaker || speaker.priority || speaker.mine;
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

	async function yieldCurrentSpeaker() {
		const currentSpeaker = (speakerState.data?.speakers ?? []).find(
			(speaker) => speaker.mine && speaker.state === 'SPEAKING'
		);
		if (!currentSpeaker) return;

		actionError = '';
		try {
			await speakerClient.removeSpeaker({
				committeeSlug: slug,
				meetingId,
				speakerId: currentSpeaker.speakerId
			});
			await loadSpeakers();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to yield your speech.');
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

	function isSelectionCountValid(vote: LiveVoteCardView): boolean {
		const min = Number(vote.vote?.minSelections ?? 0n);
		const max = Number(vote.vote?.maxSelections ?? 0n);
		const count = selectedOptionIds.length;
		return count >= min && count <= max;
	}

	async function submitBallot(vote: LiveVoteCardView) {
		const voteRecord = vote.vote;
		if (!voteRecord || submittingVote || !isSelectionCountValid(vote)) return;

		submittingVote = true;
		actionError = '';
		try {
			const res = await voteClient.submitBallot({
				committeeSlug: slug,
				meetingId,
				voteId: voteRecord.voteId,
				selectedOptionIds
			});
			saveReceipt({
				id: `${voteRecord.visibility}:${voteRecord.voteId}:${res.receiptToken}`,
				kind: voteRecord.visibility as 'open' | 'secret',
				voteId: voteRecord.voteId,
				voteName: voteRecord.name,
				receiptToken: res.receiptToken,
				receipt: `${voteRecord.voteId}:${res.receiptToken}`
			});
			selectedOptionIds = [];
			await loadVotes();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to submit your ballot.');
		} finally {
			submittingVote = false;
		}
	}

	async function loadAgenda() {
		try {
			const res = await agendaClient.listAgendaPoints({ committeeSlug: slug, meetingId });
			agendaState.data = res.agendaPoints;
		} catch {
			// Silent refresh
		}
	}

	function flattenAgenda(agendaPoints: AgendaPointRecord[]): AgendaPointRecord[] {
		return agendaPoints.flatMap((agendaPoint) => [agendaPoint, ...flattenAgenda(agendaPoint.subPoints)]);
	}

	function agendaRows() {
		return flattenAgenda(agendaState.data ?? []);
	}

	function currentAgendaPoint() {
		return agendaRows().find((agendaPoint) => agendaPoint.isActive) ?? null;
	}

	function nextAgendaPoint() {
		const rows = agendaRows();
		const activeIndex = rows.findIndex((agendaPoint) => agendaPoint.isActive);
		if (activeIndex === -1 || activeIndex + 1 >= rows.length) return null;
		return rows[activeIndex + 1];
	}

	function liveAgendaRowClass(agendaPoint: AgendaPointRecord) {
		if (agendaPoint.isActive) return 'list-row items-center gap-3 bg-primary/10';
		if (agendaPoint.parentId) return 'list-row items-center gap-3 pl-8';
		return 'list-row items-center gap-3';
	}

	function liveAgendaTitleClass(agendaPoint: AgendaPointRecord) {
		if (agendaPoint.isActive) return 'flex-1 truncate font-semibold';
		return 'flex-1 truncate';
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

		<div id="live-sse-root" class="live-grid grid min-h-0 flex-1 gap-4 [grid-template-rows:auto_minmax(0,1fr)] lg:grid-cols-2 lg:[grid-template-rows:minmax(0,1fr)_auto] lg:[grid-auto-rows:minmax(0,1fr)]">
			<section class="card relative min-h-0 border border-base-300 bg-base-100 shadow-sm">
				<div class="flex h-full min-h-0 flex-col p-4">
					<div class="mb-3 flex items-center justify-between gap-2">
						<h2 class="text-lg font-semibold">{m.meeting_live_agenda_heading()}</h2>
						{#if liveState.data.currentDocument}
							<AppTooltip text="Open document" side="left">
								<button type="button" class="btn btn-sm btn-outline btn-square lg:hidden" data-live-dialog-open aria-controls="live-doc-modal" aria-label="Open document">
									<LegacyIcon name="eye" class="live-agenda-dialog-icon" />
								</button>
							</AppTooltip>
						{/if}
					</div>
					<div id="live-agenda-main-stack" class="min-h-0 flex flex-1 flex-col">
						<div class="flex min-h-0 flex-1 overflow-hidden">
							<div id="live-agenda-panel-body" class="min-h-0 flex-1">
								<div class="flex h-full min-h-0 flex-col gap-3">
									<div class="live-agenda-preview-block lg:hidden">
										<div class="live-agenda-preview-row">
											<span class="live-agenda-preview-label">{m.meeting_live_current_item()}</span>
											<span class="live-agenda-preview-value">
												{#if currentAgendaPoint()}
													{currentAgendaPoint()?.title}
												{:else}
													{m.meeting_live_no_active_agenda()}
												{/if}
											</span>
										</div>
										<div class="live-agenda-preview-row">
											<span class="live-agenda-preview-label">{m.meeting_live_next_item()}</span>
											<span class="live-agenda-preview-value">
												{#if nextAgendaPoint()}
													{nextAgendaPoint()?.title}
												{:else}
													{m.meeting_live_none_label()}
												{/if}
											</span>
										</div>
									</div>
									<div class="hidden min-h-0 flex-1 flex-col lg:flex">
										{#if agendaRows().length === 0}
											<p class="live-empty-state text-base-content/70">{m.meeting_live_no_agenda_points()}</p>
										{:else}
											<ul class="list rounded-box border border-base-300 bg-base-100 min-h-0 flex-1 overflow-y-auto">
												{#each agendaRows() as agendaPoint}
													<li class={liveAgendaRowClass(agendaPoint)}>
														<span class="badge badge-outline">{agendaPoint.displayNumber}</span>
														<span class={liveAgendaTitleClass(agendaPoint)}>{agendaPoint.title}</span>
													</li>
												{/each}
											</ul>
										{/if}
									</div>
								</div>
							</div>
						</div>
						{#if liveState.data.currentDocument}
							<div class="mt-3 hidden gap-2 lg:flex">
								<AppTooltip text="Open document" side="left">
									<button type="button" class="btn btn-sm btn-outline btn-square" data-testid="live-doc-open-desktop" data-live-dialog-open aria-controls="live-doc-modal" aria-label="Open document">
										<LegacyIcon name="eye" />
									</button>
								</AppTooltip>
								<AppTooltip text="Download document" side="left">
									<a href={liveState.data.currentDocument.downloadUrl} download data-testid="live-doc-download-desktop" class="btn btn-sm btn-outline btn-square" aria-label="Download document">
										<LegacyIcon name="download" />
									</a>
								</AppTooltip>
							</div>
						{/if}
					</div>
				</div>
			</section>

			<section class="card min-h-0 border border-base-300 bg-base-100 shadow-sm">
				<div class="flex h-full min-h-0 flex-col p-4">
					<div class="mb-3 flex items-center justify-between gap-2">
						<h2 class="text-lg font-semibold">{m.meeting_live_speakers_heading()}</h2>
						<AppTooltip text="Speakers" side="left">
							<button
								type="button"
								class="btn btn-sm btn-outline btn-square lg:hidden"
								data-live-dialog-open
								aria-controls="speakers-full-dialog"
								aria-label="Speakers"
							>
								<LegacyIcon name="eye" class="live-speaker-history-icon" />
							</button>
						</AppTooltip>
					</div>
					<div id="live-speakers-panel-meta" class="mb-2"></div>
					<div class="live-speakers-sse min-h-0 flex-1">
						<div id="attendee-speakers-list" class="flex h-full min-h-0 flex-col">
							<div class="contents">
								{#if liveState.data.activeAgendaPoint}
									<div class="flex h-full min-h-0 flex-1 flex-col gap-3">
										<div class="min-h-0 flex-1 overflow-y-auto overflow-x-hidden" data-testid="live-speakers-active-viewport">
											{#if actionError}
												<AppAlert message={actionError} />
											{/if}
											{#if activeSpeakers().length === 0}
												<p class="live-empty-state text-base-content/70">{m.meeting_live_no_speakers()}</p>
											{:else}
												<ul class="list w-full min-w-0 rounded-box border border-base-300 bg-base-100 live-speaker-list" data-testid="live-speakers-active-list">
													{#each activeSpeakers() as speaker}
														<li
															class="list-row min-w-0 items-center gap-3"
															data-testid="live-speaker-item"
															data-speaker-state={speaker.state.toLowerCase()}
															data-speaker-mine={speaker.mine ? 'true' : 'false'}
															data-manage-scroll-anchor="false"
														>
															<div class="w-16 shrink-0 text-center font-semibold text-base-content/70">
																{#if speaker.state === 'SPEAKING'}
																	<span class="font-mono text-xs whitespace-nowrap text-base-content/70" data-speaking-since={String(speakingSinceMs[speaker.speakerId] ?? '')}>{speakingTimerLabel(speaker.speakerId)}</span>
																{:else if speaker.state === 'WAITING'}
																	{waitingDisplayNumber(speaker.speakerId)}
																{:else if speaker.speakerType === 'ropm'}
																	<span class="inline-flex items-center" aria-hidden="true"><LegacyIcon name="scale" /></span>
																{:else}
																	&nbsp;
																{/if}
															</div>
															<div class="list-col-grow min-w-0">
																<div class="flex min-w-0 items-center gap-2">
																	<div class="truncate font-semibold" data-testid="live-speaker-name">{speaker.fullName}</div>
																	{#if speakerHasBadges(speaker)}
																		<div class="flex shrink-0 flex-wrap items-center gap-1">
																			<SpeakerBadges speakerType={speaker.speakerType} quoted={speaker.quoted} firstSpeaker={speaker.firstSpeaker} priority={speaker.priority} mine={speaker.mine} />
																		</div>
																	{/if}
																</div>
															</div>
															{#if speaker.state === 'SPEAKING'}
																<div class="shrink-0 self-center">
																	<span class="inline-flex h-9 w-9 items-center justify-center text-info/80" data-testid="live-speaker-speaking-indicator" aria-hidden="true">
																		<LegacyIcon name="mic" class="h-5 w-5" />
																	</span>
																</div>
															{/if}
														</li>
													{/each}
												</ul>
											{/if}
										</div>
										<div class="live-self-add-row mt-auto shrink-0">
											{#if speakerState.data?.canAddSelf}
												{#if speakerState.data?.speakers?.some((speaker) => speaker.mine && speaker.state === 'SPEAKING')}
													<form
														class="w-full"
														onsubmit={(event) => {
															event.preventDefault();
															void yieldCurrentSpeaker();
														}}
													>
														<button
															type="submit"
															class="btn btn-sm btn-error w-full"
															data-testid="live-self-yield"
														>
															<LegacyIcon name="mic" class="live-self-add-icon" />
															<span>{m.meeting_live_yield_speech()}</span>
														</button>
													</form>
												{:else}
													<form
														class="w-full"
														onsubmit={(event) => {
															event.preventDefault();
															const submitter = event.submitter as HTMLButtonElement | null;
															void addSelfSpeaker(submitter?.value === 'ropm' ? 'ropm' : 'regular');
														}}
													>
														<div class="join flex w-full">
															<button
																type="submit"
																name="type"
																value="regular"
																class="join-item btn btn-sm w-2/3"
																data-testid="live-add-self-regular"
																aria-label="Add Myself"
																title="Add Myself"
																disabled={addingRegular || hasWaitingEntry('regular')}
															>
																{#if addingRegular}
																	<span class="loading loading-spinner loading-xs"></span>
																{:else}
																	<LegacyIcon name="person-raised" class="live-self-add-icon" />
																{/if}
															</button>
															<button
																type="submit"
																name="type"
																value="ropm"
																class="join-item btn btn-sm btn-warning w-1/3"
																data-testid="live-add-self-ropm"
																aria-label="Add Myself (Point of Order (PO))"
																title="Add Myself (Point of Order (PO))"
																disabled={addingRopm || hasWaitingEntry('ropm')}
															>
																{#if addingRopm}
																	<span class="loading loading-spinner loading-xs"></span>
																{:else}
																	<LegacyIcon name="scale" class="live-self-add-icon" />
																{/if}
															</button>
														</div>
													</form>
												{/if}
											{/if}
										</div>
										<dialog id="speakers-full-dialog" class="modal" data-live-dialog>
											<div class="modal-box max-w-4xl live-speaker-history-dialog">
												<div class="mb-3 flex items-center justify-between gap-3">
													<h3 class="text-lg font-semibold">{m.meeting_live_speaker_history_title()}</h3>
													<button type="button" class="btn btn-sm btn-ghost shrink-0" data-live-dialog-close>{m.meeting_live_speaker_history_close()}</button>
												</div>
												{#if activeSpeakers().length === 0}
													<p class="live-empty-state text-base-content/70">{m.meeting_live_no_speakers()}</p>
												{:else}
													<ul class="list w-full min-w-0 rounded-box border border-base-300 bg-base-100 live-speaker-list">
														{#each activeSpeakers() as speaker}
															<li
																class="list-row min-w-0 items-center gap-3"
																data-testid="live-speaker-item"
																data-speaker-state={speaker.state.toLowerCase()}
																data-speaker-mine={speaker.mine ? 'true' : 'false'}
																data-manage-scroll-anchor="false"
															>
																<div class="w-16 shrink-0 text-center font-semibold text-base-content/70">
																	{#if speaker.state === 'SPEAKING'}
																		<span class="font-mono text-xs whitespace-nowrap text-base-content/70" data-speaking-since={String(speakingSinceMs[speaker.speakerId] ?? '')}>{speakingTimerLabel(speaker.speakerId)}</span>
																	{:else if speaker.state === 'WAITING'}
																		{waitingDisplayNumber(speaker.speakerId)}
																	{:else if speaker.speakerType === 'ropm'}
																		<span class="inline-flex items-center" aria-hidden="true"><LegacyIcon name="scale" /></span>
																	{:else}
																		&nbsp;
																	{/if}
																</div>
																<div class="list-col-grow min-w-0">
																	<div class="flex min-w-0 items-center gap-2">
																		<div class="truncate font-semibold" data-testid="live-speaker-name">{speaker.fullName}</div>
																		{#if speakerHasBadges(speaker)}
																			<div class="flex shrink-0 flex-wrap items-center gap-1">
																				<SpeakerBadges speakerType={speaker.speakerType} quoted={speaker.quoted} firstSpeaker={speaker.firstSpeaker} priority={speaker.priority} mine={speaker.mine} />
																			</div>
																		{/if}
																	</div>
																</div>
																{#if speaker.state === 'SPEAKING'}
																	<div class="shrink-0 self-center">
																		<span class="inline-flex h-9 w-9 items-center justify-center text-info/80" data-testid="live-speaker-speaking-indicator" aria-hidden="true">
																			<LegacyIcon name="mic" class="h-5 w-5" />
																		</span>
																	</div>
																{/if}
															</li>
														{/each}
													</ul>
												{/if}
											</div>
										</dialog>
									</div>
								{:else}
									<p>{m.meeting_live_no_active_agenda()}</p>
								{/if}
							</div>
						</div>
					</div>
				</div>
			</section>

			<section class="card min-h-0 border border-base-300 bg-base-100 shadow-sm lg:col-span-2">
				<div class="p-4">
					<div id="live-votes-panel" class="space-y-3">
						<div class="flex items-center justify-between gap-2">
							<h2 class="text-lg font-semibold">{m.votes_votes()}</h2>
							<div class="flex gap-1">
								<button type="button" class="btn btn-xs btn-outline" onclick={openReceiptsDialog}>{m.votes_my_receipts()}</button>
								<button
									type="button"
									class="btn btn-xs btn-outline"
									onclick={() => {
										void loadVotes();
									}}
								>
									{m.common_refresh()}
							</button>
							</div>
						</div>
						{#if !voteState.data?.hasActiveAgenda}
							<p class="text-sm text-base-content/70">{m.meeting_live_no_active_agenda()}</p>
						{:else if visibleLiveVotes().length === 0}
							<p class="text-sm text-base-content/70">{m.votes_no_live_votes()}</p>
						{:else}
							<div class="space-y-3">
								{#each visibleLiveVotes() as vote}
									<article
										class="rounded-box border border-base-300 bg-base-100 p-3 space-y-2"
										data-vote-card
										data-vote-id={vote.vote?.voteId ?? ''}
										data-vote-state={vote.vote?.state ?? ''}
										data-vote-results-until={String(vote.resultsUntilUnix ?? 0n)}
									>
										<div class="flex flex-wrap items-center gap-2">
											<h3 class="font-semibold">{vote.vote?.name}</h3>
											<span class={voteVisibilityBadgeClass(vote.vote?.visibility ?? '')}>{voteVisibilityLabel(vote.vote?.visibility ?? '')}</span>
											<span class={voteStateBadgeClass(vote.vote?.state ?? '')}>{voteStateLabel(vote.vote?.state ?? '')}</span>
											<span class="text-xs text-base-content/70">{voteBoundsLabel(vote)}</span>
											{#if !vote.isEligible && vote.vote?.state === 'open'}
												<span class="badge badge-error badge-outline badge-sm">{m.votes_not_eligible()}</span>
											{/if}
										</div>

										{#if vote.hasTimedResults}
											<div class="rounded-box border border-base-300 bg-base-200/40 p-2 space-y-2">
												<div class="text-xs font-semibold uppercase text-base-content/70">{m.votes_final_results_visible_30s()}</div>
												<ul class="space-y-1 text-sm">
													{#each vote.timedResults as row}
														<li class="flex items-center justify-between gap-2">
															<span class="truncate">{row.label}</span>
															<span class="badge badge-outline badge-sm">{String(row.count)}</span>
														</li>
													{/each}
												</ul>
												<div class="text-xs text-base-content/70">
													eligible={String(vote.stats?.eligibleCount ?? 0n)} casts={String(vote.stats?.castCount ?? 0n)} ballots={String(vote.stats?.ballotCount ?? 0n)}
												</div>
												<div class="text-xs text-base-content/70">
													{m.votes_hiding_in()} <span data-vote-results-countdown>{voteResultsRemaining(vote)}</span>s
												</div>
											</div>
										{:else if vote.resultsBlockedCounting}
											<div class="alert alert-warning text-sm">
												<span>{m.votes_counting_phase_message()}</span>
											</div>
										{:else if vote.vote?.state === 'open'}
											{#if vote.alreadyVoted}
												<div class="alert alert-success text-sm" data-vote-submitted-screen>
													<span>{m.votes_vote_submitted_message()}</span>
												</div>
											{:else}
												<div data-vote-inputs class="space-y-2">
													<div class={Number(vote.vote?.maxSelections ?? 0n) === 1 ? 'grid gap-1 grid-cols-1' : 'grid gap-1 grid-cols-2'}>
														{#each vote.vote?.options ?? [] as option}
															<label class="label cursor-pointer justify-start gap-2 rounded border border-base-300 px-2 py-1">
																<input
																	type={Number(vote.vote?.maxSelections ?? 0n) === 1 ? 'radio' : 'checkbox'}
																	class={Number(vote.vote?.maxSelections ?? 0n) === 1 ? 'radio radio-sm' : 'checkbox checkbox-sm'}
																	name="option_id"
																	value={option.optionId}
																	checked={isVoteSelectionActive(vote, option.optionId)}
																	disabled={!vote.isEligible || submittingVote}
																	onchange={() => chooseVoteOption(option.optionId, Number(vote.vote?.maxSelections ?? 0n) > 1)}
																/>
																<span>{option.label}</span>
															</label>
														{/each}
													</div>
													{#if Number(vote.vote?.minSelections ?? 0n) !== Number(vote.vote?.maxSelections ?? 0n)}
														<p class="text-xs text-base-content/60">{m.votes_select_between({ min: Number(vote.vote?.minSelections ?? 0n), max: Number(vote.vote?.maxSelections ?? 0n) })}</p>
													{:else if Number(vote.vote?.minSelections ?? 0n) > 1}
														<p class="text-xs text-base-content/60">{m.votes_select_exactly({ count: Number(vote.vote?.minSelections ?? 0n) })}</p>
													{/if}
													<button
														type="button"
														class="btn btn-sm btn-primary"
														disabled={!vote.isEligible || submittingVote || !isSelectionCountValid(vote)}
														onclick={() => {
															void submitBallot(vote);
														}}
													>
														{vote.vote?.visibility === 'secret' ? m.votes_submit_secret_ballot() : m.votes_submit_open_ballot()}
													</button>
												</div>
											{/if}
										{/if}
									</article>
								{/each}
							</div>
						{/if}
						<div id="live-vote-last-receipt" class="hidden" data-receipt-b64=""></div>
					</div>
				</div>
			</section>
		</div>
	{/if}
</div>

<dialog id="meeting-receipts-dialog" class="modal" bind:this={receiptsDialogEl}>
	<div class="modal-box max-w-lg">
		<div class="mb-4 flex items-center justify-between">
			<h3 class="text-lg font-semibold">{m.votes_my_receipts()}</h3>
			<button type="button" class="btn btn-sm btn-ghost" onclick={() => receiptsDialogEl?.close()}>{m.common_close()}</button>
		</div>
		{#if meetingReceipts.length === 0}
			<p class="text-sm text-base-content/70">{m.votes_no_stored_receipts()}</p>
		{:else}
			<div class="space-y-2">
				{#each meetingReceipts as receipt}
					<div class="rounded-box border border-base-300 bg-base-200/30 p-3 space-y-2">
						<div class="flex flex-wrap items-center gap-2">
							<span class="badge badge-outline badge-sm">{receipt.kind}</span>
							<span class="font-semibold">{receipt.voteName}</span>
						</div>
						<div class="text-xs text-base-content/70 break-all">{receipt.receiptToken}</div>
						<div class="flex flex-wrap items-center gap-2">
							<button
								class="btn btn-xs btn-primary"
								type="button"
								disabled={receiptVerifyingId === receipt.id}
								onclick={() => verifyMeetingReceipt(receipt)}
							>
								{receiptVerifyingId === receipt.id ? m.votes_verifying() : m.votes_verify()}
							</button>
							{#if receiptVerifyResults[receipt.id]}
								<span class={receiptVerifyResults[receipt.id].startsWith('Error') ? 'text-xs text-error' : 'text-xs text-success'}>
									{receiptVerifyResults[receipt.id]}
								</span>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
	<form method="dialog" class="modal-backdrop"><button aria-label="Close">Close</button></form>
</dialog>
