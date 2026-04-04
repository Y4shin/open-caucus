<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { buildDocsOverlayHref } from '$lib/docs/navigation.js';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import SpeakerBadges from '$lib/components/ui/SpeakerBadges.svelte';
	import { speakerClient } from '$lib/api/index.js';
	import type { AttendeeRecord } from '$lib/gen/conference/attendees/v1/attendees_pb.js';
	import type { SpeakerQueueView } from '$lib/gen/conference/speakers/v1/speakers_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import * as m from '$lib/paraglide/messages';

	let {
		speakers,
		attendees,
		hasActivePoint,
		slug,
		meetingId,
		onError
	}: {
		speakers: SpeakerQueueView['speakers'] | undefined;
		attendees: AttendeeRecord[];
		hasActivePoint: boolean;
		slug: string;
		meetingId: string;
		onError: (msg: string) => void;
	} = $props();

	let speakerActionPending = $state('');
	let speakerSearch = $state('');
	let searchInput = $state<HTMLInputElement | null>(null);
	let speakingSinceMs = $state<Record<string, number>>({});
	let nowMs = $state(Date.now());

	$effect(() => {
		const interval = window.setInterval(() => {
			nowMs = Date.now();
		}, 1000);
		return () => window.clearInterval(interval);
	});

	$effect(() => {
		if (speakers) syncSpeakingSince(speakers);
	});

	function activeSpeaker() {
		return speakers?.find((speaker) => speaker.state === 'SPEAKING') ?? null;
	}

	function nextWaitingSpeaker() {
		return speakers?.find((speaker) => speaker.state === 'WAITING') ?? null;
	}

	function syncSpeakingSince(spkrs: SpeakerQueueView['speakers']) {
		const next = { ...speakingSinceMs };
		const activeIds = new Set(spkrs.map((s) => s.speakerId));
		for (const s of spkrs) {
			if (s.state === 'SPEAKING' && next[s.speakerId] == null) {
				next[s.speakerId] = Date.now();
			}
		}
		for (const speakerId of Object.keys(next)) {
			if (!activeIds.has(speakerId)) delete next[speakerId];
		}
		speakingSinceMs = next;
	}

	function waitingDisplayNumber(speakerId: string) {
		const spkrs = speakers ?? [];
		const doneCount = spkrs.filter((s) => s.state === 'DONE').length;
		const speakingCount = spkrs.filter((s) => s.state === 'SPEAKING').length;
		let waitingPosition = 0;
		for (const s of spkrs) {
			if (s.state === 'WAITING') {
				waitingPosition++;
				if (s.speakerId === speakerId) return doneCount + speakingCount + waitingPosition;
			}
		}
		return 0;
	}

	function doneDisplayNumber(speakerId: string) {
		let position = 0;
		for (const s of speakers ?? []) {
			if (s.state === 'DONE') {
				position++;
				if (s.speakerId === speakerId) return position;
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
		return speaker.speakerType === 'ropm' || speaker.quoted || speaker.firstSpeaker || speaker.priority;
	}

	function scrollToInitialSpeaker() {
		const target = document.querySelector('#speakers-list-container [data-manage-scroll-anchor="true"]');
		if (target instanceof HTMLElement) {
			target.scrollIntoView({ block: 'center', behavior: 'smooth' });
		}
	}

	function isInitialScrollSpeaker(speakerId: string) {
		const active = activeSpeaker();
		if (active) return active.speakerId === speakerId;
		const next = nextWaitingSpeaker();
		return next?.speakerId === speakerId;
	}

	function normalized(value: string) {
		return value.trim().toLowerCase();
	}

	function hasOpenSpeaker(attendeeId: string, speakerType: string) {
		return (speakers ?? []).some(
			(s) =>
				s.attendeeId === attendeeId &&
				s.speakerType === speakerType &&
				(s.state === 'WAITING' || s.state === 'SPEAKING')
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
		return [...attendees]
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

	async function runSpeakerAction(
		key: string,
		action: () => Promise<{ view?: SpeakerQueueView }>
	) {
		onError('');
		speakerActionPending = key;
		try {
			await action();
			return true;
		} catch (err) {
			onError(getDisplayError(err, 'Failed to update the speakers queue.'));
			return false;
		} finally {
			speakerActionPending = '';
		}
	}

	async function endCurrentSpeaker() {
		const current = activeSpeaker();
		if (!current) {
			onError('No active speaker is available.');
			return;
		}

		speakerActionPending = 'end-current';
		try {
			await speakerClient.setSpeakerDone({
				committeeSlug: slug,
				meetingId,
				speakerId: current.speakerId
			});
		} catch (err) {
			onError(getDisplayError(err, 'Failed to update the speakers queue.'));
		} finally {
			speakerActionPending = '';
		}
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
		if (!topCandidate || hasOpenSpeaker(topCandidate.attendeeId, 'regular')) return;
		await addCandidate(topCandidate.attendeeId, 'regular');
	}
</script>

<div id="moderate-right-resizable-stack" data-meeting-id={meetingId} class="flex min-h-0 flex-1 flex-col gap-4">
	<section
		id="moderate-speakers-card"
		class="card h-[50dvh] min-h-0 min-w-0 border border-base-300 bg-base-100 shadow-sm lg:h-auto lg:flex-1"
		data-testid="manage-speakers-card"
	>
		<div class="card-body flex min-h-0 flex-col p-4">
			<div class="mb-3 flex items-center justify-between gap-2">
				<h2 class="text-lg font-semibold">{m.meeting_manage_speakers_list()}</h2>
				<button
					type="button"
					class="btn btn-ghost btn-xs"
					title="Open speakers help"
					aria-label="Open speakers help"
					onclick={() => goto(buildDocsOverlayHref('03-chairperson/05-speakers-moderator-and-quotation', page.url))}
				>
					<LegacyIcon name="help" />
				</button>
			</div>
			<div id="speakers-list-container" class="flex min-h-0 flex-1 flex-col">
				{#if !hasActivePoint}
					<p class="text-sm text-base-content/70">{m.meeting_manage_no_active_agenda_for_speakers()}</p>
				{:else if speakers?.length}
					<div class="mb-2 flex flex-wrap items-center justify-between gap-2" data-testid="manage-speakers-quick-controls">
						<div class="flex flex-wrap items-center gap-2">
							{#if activeSpeaker()}
								<form
									class="inline"
									onsubmit={(event) => {
										event.preventDefault();
										void endCurrentSpeaker();
									}}
								>
									<button
										type="submit"
										class="btn btn-sm btn-success whitespace-nowrap"
										data-testid="manage-end-current-speaker"
										data-testid-group="manage-speakers-quick-button"
										title="End current speech"
										aria-label="End current speech"
										disabled={speakerActionPending !== '' || !activeSpeaker()}
									>
										<LegacyIcon name="check-circle" class="ui-icon--left" />End Speech
									</button>
								</form>
							{:else if nextWaitingSpeaker()}
								<form
									class="inline"
									onsubmit={(event) => {
										event.preventDefault();
										void runSpeakerAction('start-next', async () => {
											const next = nextWaitingSpeaker();
											if (!next) throw new Error('No waiting speaker is available.');
											return await speakerClient.setSpeakerSpeaking({
												committeeSlug: slug,
												meetingId,
												speakerId: next.speakerId
											});
										});
									}}
								>
									<button
										type="submit"
										class="btn btn-sm btn-primary whitespace-nowrap"
										data-testid="manage-start-next-speaker"
										data-testid-group="manage-speakers-quick-button"
										title="Start next speaker"
										aria-label="Start next speaker"
										disabled={speakerActionPending !== '' || !nextWaitingSpeaker()}
									>
										<LegacyIcon name="arrow-forward" class="ui-icon--left" />{m.meeting_moderate_start_next_speaker()}
									</button>
								</form>
							{/if}
						</div>
						<button
							type="button"
							class="btn btn-sm btn-ghost whitespace-nowrap"
							data-manage-speakers-reset-scroll
							data-testid="manage-speakers-reset-scroll"
							title="Scroll to active position"
							aria-label="Scroll to active position"
							onclick={scrollToInitialSpeaker}
						>
							<LegacyIcon name="history" class="ui-icon--left" />{m.meeting_moderate_scroll_to_active()}
						</button>
					</div>
					<ul class="list rounded-box border border-base-300 bg-base-100 mt-2 flex-1 overflow-y-auto pr-1 live-speaker-list" data-initial-scroll-top="0" data-manage-speakers-viewport data-testid="manage-speakers-viewport">
						{#each speakers as speaker, i}
							{@const prevSpeaker = speakers[i - 1]}
							{#if speaker.state !== 'DONE' && prevSpeaker?.state === 'DONE'}
								<li class="list-row py-0">
									<div class="divider my-0 text-xs text-base-content/40 col-span-full">{m.meeting_moderate_upcoming_divider()}</div>
								</li>
							{/if}
							<li
								class="list-row min-w-0 items-center gap-3"
								data-testid="live-speaker-item"
								data-speaker-state={speaker.state.toLowerCase()}
								data-speaker-mine="false"
								data-manage-scroll-anchor={isInitialScrollSpeaker(speaker.speakerId) ? 'true' : 'false'}
							>
								<div class="w-16 shrink-0 text-center font-semibold text-base-content/70">
									{#if speaker.state === 'SPEAKING'}
										<span class="font-mono text-xs whitespace-nowrap text-base-content/70" data-speaking-since={String(speakingSinceMs[speaker.speakerId] ?? '')}>{speakingTimerLabel(speaker.speakerId)}</span>
									{:else if speaker.state === 'WAITING'}
										{waitingDisplayNumber(speaker.speakerId)}
									{:else if speaker.state === 'DONE'}
										{doneDisplayNumber(speaker.speakerId)}
									{/if}
								</div>
								<div class="list-col-grow min-w-0">
									<div class="flex min-w-0 items-center gap-2">
										<div class="truncate font-semibold" data-testid="live-speaker-name">{speaker.fullName}</div>
										{#if speakerHasBadges(speaker)}
											<div class="flex shrink-0 flex-wrap items-center gap-1">
												<SpeakerBadges speakerType={speaker.speakerType} quoted={speaker.quoted} firstSpeaker={speaker.firstSpeaker} priority={speaker.priority} />
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
								{:else if speaker.state === 'WAITING'}
									<div class="shrink-0 self-center">
										<div class="join join-horizontal">
											<button
												type="button"
												class="join-item btn btn-sm btn-square tooltip tooltip-left"
												title={speaker.priority ? 'Remove Priority' : 'Give Priority'}
												aria-label={speaker.priority ? 'Remove Priority' : 'Give Priority'}
												data-tip={speaker.priority ? 'Remove Priority' : 'Give Priority'}
												onclick={() =>
													void runSpeakerAction(`priority-${speaker.speakerId}`, async () =>
														await speakerClient.setSpeakerPriority({
															committeeSlug: slug,
															meetingId,
															speakerId: speaker.speakerId,
															priority: !speaker.priority
														})
													)}
												disabled={speakerActionPending !== ''}
											>
												<LegacyIcon name={speaker.priority ? 'star' : 'star-outline'} />
											</button>
											<button
												type="button"
												class="join-item btn btn-sm btn-square tooltip tooltip-left"
												title="Start"
												aria-label="Start"
												data-tip="Start"
												onclick={() =>
													void runSpeakerAction(`start-${speaker.speakerId}`, async () =>
														await speakerClient.setSpeakerSpeaking({
															committeeSlug: slug,
															meetingId,
															speakerId: speaker.speakerId
														})
													)}
												disabled={speakerActionPending !== ''}
											>
												<LegacyIcon name="arrow-forward" />
											</button>
											<button
												type="button"
												class="join-item btn btn-sm btn-square btn-error tooltip tooltip-left"
												title="Remove"
												aria-label="Remove"
												data-tip="Remove"
												onclick={() =>
													void runSpeakerAction(`remove-${speaker.speakerId}`, async () =>
														await speakerClient.removeSpeaker({
															committeeSlug: slug,
															meetingId,
															speakerId: speaker.speakerId
														})
													)}
												disabled={speakerActionPending !== ''}
											>
												<LegacyIcon name="trash" />
											</button>
										</div>
									</div>
								{/if}
							</li>
						{/each}
					</ul>
				{:else}
					<div class="mb-2 flex flex-wrap items-center justify-between gap-2" data-testid="manage-speakers-quick-controls">
						<div class="flex flex-wrap items-center gap-2">
							{#if activeSpeaker()}
								<button
									class="btn btn-sm btn-success whitespace-nowrap"
									data-testid="manage-end-current-speaker"
									data-testid-group="manage-speakers-quick-button"
									title="End current speech"
									aria-label="End current speech"
									onclick={endCurrentSpeaker}
									disabled={speakerActionPending !== '' || !activeSpeaker()}
								>
									<LegacyIcon name="check-circle" class="ui-icon--left" />
									{m.meeting_live_yield_speech()}
								</button>
							{:else if nextWaitingSpeaker()}
								<button
									class="btn btn-sm btn-primary whitespace-nowrap"
									data-testid="manage-start-next-speaker"
									data-testid-group="manage-speakers-quick-button"
									title="Start next speaker"
									aria-label="Start next speaker"
									onclick={() =>
										runSpeakerAction('start-next', async () => {
											const next = nextWaitingSpeaker();
											if (!next) throw new Error('No waiting speaker is available.');
											return await speakerClient.setSpeakerSpeaking({
												committeeSlug: slug,
												meetingId,
												speakerId: next.speakerId
											});
										})}
									disabled={speakerActionPending !== '' || !nextWaitingSpeaker()}
								>
									<LegacyIcon name="arrow-forward" class="ui-icon--left" />
									{m.meeting_moderate_start_next()}
								</button>
							{/if}
						</div>
					</div>
					<p class="text-sm text-base-content/70">{m.meeting_live_no_speakers()}</p>
				{/if}
			</div>
		</div>
	</section>

	<section
		id="moderate-attendees-card"
		class="card h-[50dvh] min-h-0 min-w-0 border border-base-300 bg-base-100 shadow-sm lg:h-auto lg:flex-1"
	>
		<div class="card-body flex min-h-0 flex-col gap-3 overflow-hidden p-4">
			<div class="flex min-w-0 flex-wrap items-center justify-between gap-3">
				<h2 class="text-lg font-semibold">{m.meeting_manage_add_speaker()}</h2>
				<div class="join min-w-0 w-full max-w-full sm:w-auto sm:min-w-[24rem]">
					<input
						class="input input-bordered input-sm join-item min-w-0 flex-1"
						type="text"
						id="speaker-add-search-input"
						name="q"
						placeholder={m.meeting_moderate_search_attendee_placeholder()}
						bind:value={speakerSearch}
						bind:this={searchInput}
						onkeydown={handleSpeakerSearchEnter}
					/>
					<button
						type="button"
						class="btn btn-sm btn-square btn-ghost join-item border-base-300"
						title="Open add-speaker help"
						aria-label="Open add-speaker help"
						onclick={() => goto(buildDocsOverlayHref('03-chairperson/05-speakers-moderator-and-quotation', page.url))}
					>
						<LegacyIcon name="help" />
					</button>
				</div>
			</div>
			<div class="min-h-0 flex-1 overflow-x-hidden overflow-y-auto pr-1">
				<div id="speaker-add-candidates-container" class="space-y-2">
					{#each sortedCandidates().slice(0, 8) as attendee}
						<div class="flex flex-wrap items-center justify-between gap-3 rounded-box border border-base-300 bg-base-100 px-3 py-3" data-testid="manage-speaker-candidate-card">
							<div>
								<div class="font-medium">{attendee.fullName}</div>
								<div class="text-sm text-base-content/70">
									#{attendee.attendeeNumber.toString()}
									{#if attendee.isGuest} • {m.meeting_moderate_guest_badge()}{/if}
									{#if attendee.quoted} • {m.meeting_join_quoted_label()}{/if}
								</div>
							</div>
							<div class="join join-horizontal">
								<button
									class="join-item btn btn-sm btn-square tooltip tooltip-left"
									title="Add regular speech"
									aria-label="Add regular speech"
									data-tip="Add regular speech"
									onclick={() => addCandidate(attendee.attendeeId, 'regular')}
									disabled={speakerActionPending !== '' || hasOpenSpeaker(attendee.attendeeId, 'regular')}
								>
									<LegacyIcon name="person-raised" />
								</button>
								<button
									class="join-item btn btn-sm btn-square btn-warning tooltip tooltip-left"
									title="Add Point of Order (PO) speech"
									aria-label="Add Point of Order (PO) speech"
									data-tip="Add Point of Order (PO) speech"
									onclick={() => addCandidate(attendee.attendeeId, 'ropm')}
									disabled={speakerActionPending !== '' || hasOpenSpeaker(attendee.attendeeId, 'ropm')}
								>
									<LegacyIcon name="scale" />
								</button>
							</div>
						</div>
					{/each}
					{#if sortedCandidates().length === 0}
						<p class="text-sm text-base-content/70">{m.meeting_moderate_no_matching_attendees()}</p>
					{/if}
				</div>
			</div>
		</div>
	</section>
</div>
