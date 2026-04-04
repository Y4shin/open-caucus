<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { buildDocsOverlayHref } from '$lib/docs/navigation.js';
	import { onDestroy } from 'svelte';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import AgendaSection from './AgendaSection.svelte';
	import AttendeeRow from '$lib/components/ui/AttendeeRow.svelte';
	import SpeakerBadges from '$lib/components/ui/SpeakerBadges.svelte';
	import VoteCard from '$lib/components/ui/VoteCard.svelte';
	import { agendaClient, attendeeClient, meetingClient, moderationClient, speakerClient, voteClient } from '$lib/api/index.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import { session } from '$lib/stores/session.svelte.js';
	import type { AgendaPointRecord } from '$lib/gen/conference/agenda/v1/agenda_pb.js';
	import type { AttendeeRecord } from '$lib/gen/conference/attendees/v1/attendees_pb.js';
	import { MeetingEventKind } from '$lib/gen/conference/meetings/v1/meetings_pb.js';
	import type { ModerationView } from '$lib/gen/conference/moderation/v1/moderation_pb.js';
	import type { SpeakerQueueView } from '$lib/gen/conference/speakers/v1/speakers_pb.js';
	import type {
		VoteDefinitionRecord,
		VotesPanelView
	} from '$lib/gen/conference/votes/v1/votes_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { voteStateBadgeClass, voteVisibilityBadgeClass } from '$lib/utils/votes.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import * as m from '$lib/paraglide/messages';

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);

	let moderationState = $state(createRemoteState<ModerationView>());
	let speakerState = $state(createRemoteState<SpeakerQueueView>());
	let attendeeState = $state(createRemoteState<AttendeeRecord[]>());
	let agendaState = $state(createRemoteState<AgendaPointRecord[]>());
	let votesState = $state(createRemoteState<VotesPanelView>());
	let actionError = $state('');
	let actionNotice = $state('');
	let togglingSignup = $state(false);
	let attendeeActionPending = $state('');
	let speakerActionPending = $state('');
	let voteActionPending = $state('');
	let settingsActionPending = $state('');
	let createVoteDetailsOpen = $state(false);
	let voteAccordionOpen = $state<Record<string, boolean>>({});
	let draftVoteEditorOpen = $state<Record<string, boolean>>({});
	let agendaAnyDialogOpen = $state(false);
	let creatingVote = $state(false);
	let moderateSettingsTab = $state<'meeting' | 'agenda'>('meeting');
	let voteName = $state('');
	let voteVisibility = $state<'open' | 'secret'>('open');
	let voteMinSelections = $state('1');
	let voteMaxSelections = $state('1');
	let voteOptionsText = $state('Yes\nNo');
	let draftOptionTexts = $state<Record<string, string>>({});
	let draftMinSelections = $state<Record<string, string>>({});
	let draftMaxSelections = $state<Record<string, string>>({});
	let moderateLeftTab = $state<'agenda' | 'tools' | 'attendees' | 'settings'>('agenda');
	let speakerSearch = $state('');
	let searchInput = $state<HTMLInputElement | null>(null);
	let voteNameInput = $state<HTMLInputElement | null>(null);
	let nowMs = $state(Date.now());
	let speakingSinceMs = $state<Record<string, number>>({});

	onDestroy(() => {
		pageActions.clear();
	});

	$effect(() => {
		if (!moderationState.data?.meeting) return;
		pageActions.set([], {
			backHref: `/committee/${slug}`,
			title: `Moderate ${moderationState.data.meeting.meetingName}`
		});
	});
	$effect(() => {
		if (!session.loaded) return;
		if (!session.authenticated) {
			goto('/login');
			return;
		}
		loadModerationView();
	});

	$effect(() => {
		const interval = window.setInterval(() => {
			nowMs = Date.now();
		}, 1000);
		return () => window.clearInterval(interval);
	});

	$effect(() => {
		if (!session.loaded || !session.authenticated) return;
		const interval = window.setInterval(() => {
			if (agendaAnyDialogOpen) return;
			loadModeration();
			loadSpeakers();
			loadAttendees();
			loadAgenda();
			loadVotes();
		}, 2000);
		return () => window.clearInterval(interval);
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
							loadAgenda();
							break;
						case MeetingEventKind.ATTENDEES_UPDATED:
							loadAttendees();
							break;
						case MeetingEventKind.MEETING_UPDATED:
							loadModeration();
							break;
					}
				}
			} catch {
				// Stream closed or server went away — ignore.
			}
		})();
		return () => {
			cancelled = true;
		};
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
			syncSpeakingSince(speakerState.data?.speakers ?? []);
			attendeeState.data = attendeeRes.attendees;
			agendaState.data = agendaRes.agendaPoints;
			votesState.data = votesRes.view ?? null;
			syncVotePanelOpenState(votesRes.view?.votes ?? []);
			for (const vote of votesRes.view?.votes ?? []) {
				if (vote.state !== 'draft') continue;
				if (!(vote.voteId in draftOptionTexts)) {
					draftOptionTexts[vote.voteId] = vote.options.map((option) => option.label).join('\n');
				}
				if (!(vote.voteId in draftMinSelections)) {
					draftMinSelections[vote.voteId] = vote.minSelections.toString();
				}
				if (!(vote.voteId in draftMaxSelections)) {
					draftMaxSelections[vote.voteId] = vote.maxSelections.toString();
				}
			}
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

	async function loadModeration() {
		try {
			const res = await moderationClient.getModerationView({ committeeSlug: slug, meetingId });
			moderationState.data = res.view ?? null;
		} catch {
			// Silent refresh
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

	async function loadAttendees() {
		try {
			const res = await attendeeClient.listAttendees({ committeeSlug: slug, meetingId });
			attendeeState.data = res.attendees;
		} catch {
			// Silent refresh
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

	async function loadVotes() {
		try {
			const res = await voteClient.getVotesPanel({ committeeSlug: slug, meetingId });
			votesState.data = res.view ?? null;
			syncVotePanelOpenState(res.view?.votes ?? []);
		} catch {
			// Silent refresh
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
			loadModeration();
		} finally {
			togglingSignup = false;
		}
	}

	function attendeeRecoveryURL(attendeeId: string) {
		return `/committee/${slug}/meeting/${meetingId}/attendee/${attendeeId}/recovery`;
	}

	function manageJoinQrURL() {
		return `/committee/${slug}/meeting/${meetingId}/moderate/join-qr`;
	}

	async function addGuestAttendee(event: SubmitEvent) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement | null;
		if (!form || attendeeActionPending !== '') return;

		const fullName = String(new FormData(form).get('full_name') ?? '').trim();
		if (!fullName) {
			actionError = 'Name is required.';
			return;
		}

		attendeeActionPending = 'add-guest';
		actionError = '';
		try {
			const genderQuoted = form.querySelector<HTMLInputElement>('input[name="gender_quoted"]')?.checked ?? false;
			await attendeeClient.createAttendee({ committeeSlug: slug, meetingId, fullName, genderQuoted });
			form.reset();
			await Promise.all([loadModeration(), loadAttendees(), loadSpeakers()]);
			searchInput?.focus();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to add the guest attendee.');
		} finally {
			attendeeActionPending = '';
		}
	}

	async function selfSignupAttendee() {
		if (attendeeActionPending !== '') return;
		attendeeActionPending = 'self-signup';
		actionError = '';
		try {
			await attendeeClient.selfSignup({ committeeSlug: slug, meetingId });
			await Promise.all([loadModeration(), loadAttendees(), loadSpeakers()]);
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to sign you up as an attendee.');
		} finally {
			attendeeActionPending = '';
		}
	}

	async function removeAttendee(attendeeId: string, fullName: string) {
		if (!window.confirm('Remove attendee?')) return;
		if (attendeeActionPending !== '') return;

		attendeeActionPending = `remove-${attendeeId}`;
		actionError = '';
		try {
			await attendeeClient.deleteAttendee({ committeeSlug: slug, meetingId, attendeeId });
			await Promise.all([loadModeration(), loadAttendees(), loadSpeakers()]);
		} catch (err) {
			actionError = getDisplayError(err, `Failed to remove ${fullName}.`);
		} finally {
			attendeeActionPending = '';
		}
	}

	async function toggleAttendeeChair(attendee: AttendeeRecord) {
		if (attendeeActionPending !== '') return;
		attendeeActionPending = `chair-${attendee.attendeeId}`;
		actionError = '';
		try {
			await attendeeClient.setChairperson({ committeeSlug: slug, meetingId, attendeeId: attendee.attendeeId, isChair: !attendee.isChair });
			await loadAttendees();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to update chairperson status.');
			loadAttendees();
		} finally {
			attendeeActionPending = '';
		}
	}

	async function toggleAttendeeQuoted(attendee: AttendeeRecord) {
		if (attendeeActionPending !== '') return;
		attendeeActionPending = `quoted-${attendee.attendeeId}`;
		actionError = '';
		try {
			await attendeeClient.setQuoted({ committeeSlug: slug, meetingId, attendeeId: attendee.attendeeId, quoted: !attendee.quoted });
			await loadAttendees();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to update FLINTA* status.');
			loadAttendees();
		} finally {
			attendeeActionPending = '';
		}
	}

	function attendeeRows() {
		return [...(attendeeState.data ?? [])].sort((left, right) => Number(left.attendeeNumber - right.attendeeNumber));
	}

	function activeSpeaker() {
		return speakerState.data?.speakers.find((speaker) => speaker.state === 'SPEAKING') ?? null;
	}

	function nextWaitingSpeaker() {
		return speakerState.data?.speakers.find((speaker) => speaker.state === 'WAITING') ?? null;
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
		const speakers = speakerState.data?.speakers ?? [];
		const doneCount = speakers.filter((s) => s.state === 'DONE').length;
		const speakingCount = speakers.filter((s) => s.state === 'SPEAKING').length;
		let waitingPosition = 0;
		for (const speaker of speakers) {
			if (speaker.state === 'WAITING') {
				waitingPosition++;
				if (speaker.speakerId === speakerId) return doneCount + speakingCount + waitingPosition;
			}
		}
		return 0;
	}

	function doneDisplayNumber(speakerId: string) {
		let position = 0;
		for (const speaker of speakerState.data?.speakers ?? []) {
			if (speaker.state === 'DONE') {
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

	function formatDuration(seconds: bigint | number) {
		const total = Number(seconds);
		if (total <= 0) return '—';
		const mins = Math.floor(total / 60);
		const secs = total % 60;
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

	function selectModerateLeftTab(tab: 'agenda' | 'tools' | 'attendees' | 'settings') {
		moderateLeftTab = tab;
	}

	function openDocs(path: string) {
		goto(buildDocsOverlayHref(path, page.url));
	}

	function openDocsWithHeading(path: string, heading?: string) {
		goto(buildDocsOverlayHref(path, page.url, { heading }));
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
			loadSpeakers();
			return true;
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to update the speakers queue.');
			loadSpeakers();
			return false;
		} finally {
			speakerActionPending = '';
		}
	}

	async function endCurrentSpeaker() {
		const current = activeSpeaker();
		if (!current) {
			actionError = 'No active speaker is available.';
			return;
		}

		actionError = '';
		speakerActionPending = 'end-current';
		try {
			const res = await speakerClient.setSpeakerDone({
				committeeSlug: slug,
				meetingId,
				speakerId: current.speakerId
			});
			speakerState.data = res.view ?? speakerState.data;
			loadSpeakers();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to update the speakers queue.');
			loadSpeakers();
		} finally {
			speakerActionPending = '';
		}
	}

	async function runVoteAction(key: string, action: () => Promise<void>) {
		actionError = '';
		actionNotice = '';
		voteActionPending = key;
		try {
			await action();
			return true;
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to update the votes panel.');
			loadVotes();
			return false;
		} finally {
			voteActionPending = '';
		}
	}

	async function updateMeetingModerator(event: Event) {
		const target = event.currentTarget as HTMLSelectElement | null;
		if (!target || settingsActionPending !== '') return;

		settingsActionPending = 'moderator';
		actionError = '';
		try {
			await moderationClient.setMeetingModerator({
				committeeSlug: slug,
				meetingId,
				moderatorAttendeeId: target.value
			});
			if (moderationState.data?.settings) {
				moderationState.data.settings.moderatorAttendeeId = target.value;
			}
			loadModeration();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to update the meeting moderator.');
			loadModeration();
		} finally {
			settingsActionPending = '';
		}
	}

	async function updateMeetingQuotation() {
		const settings = moderationState.data?.settings;
		if (!settings || settingsActionPending !== '') return;

		settingsActionPending = 'quotation';
		actionError = '';
		try {
			await moderationClient.setMeetingQuotation({
				committeeSlug: slug,
				meetingId,
				genderQuotationEnabled: settings.genderQuotationEnabled,
				firstSpeakerQuotationEnabled: settings.firstSpeakerQuotationEnabled
			});
			loadModeration();
			loadSpeakers();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to update the quotation settings.');
			loadModeration();
			loadSpeakers();
		} finally {
			settingsActionPending = '';
		}
	}

	function normalized(value: string) {
		return value.trim().toLowerCase();
	}

	function hasOpenSpeaker(attendeeId: string, speakerType: string) {
		return (speakerState.data?.speakers ?? []).some(
			(speaker) =>
				speaker.attendeeId === attendeeId &&
				speaker.speakerType === speakerType &&
				(speaker.state === 'WAITING' || speaker.state === 'SPEAKING')
		);
	}

	function settingsAttendees() {
		return attendeeRows();
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

	function getDraftOptionsText(vote: VoteDefinitionRecord) {
		return draftOptionTexts[vote.voteId] ?? '';
	}

	function getDraftMinSelections(vote: VoteDefinitionRecord) {
		return draftMinSelections[vote.voteId] ?? vote.minSelections.toString();
	}

	function getDraftMaxSelections(vote: VoteDefinitionRecord) {
		return draftMaxSelections[vote.voteId] ?? vote.maxSelections.toString();
	}

	async function saveDraftVote(vote: VoteDefinitionRecord) {
		await runVoteAction(`save-draft-${vote.voteId}`, async () => {
			await voteClient.updateVoteDraft({
				committeeSlug: slug,
				meetingId,
				voteId: vote.voteId,
				name: vote.name.trim(),
				visibility: vote.visibility,
				minSelections: bigintFromInput(getDraftMinSelections(vote)),
				maxSelections: bigintFromInput(getDraftMaxSelections(vote)),
				optionLabels: getDraftOptionsText(vote)
					.split('\n')
					.map((line) => line.trim())
					.filter(Boolean)
			});
			loadVotes();
		});
	}

	async function createVote() {
		if (creatingVote || !canCreateVote()) return;

		actionError = '';
		creatingVote = true;
		try {
			await voteClient.createVote({
				committeeSlug: slug,
				meetingId,
				name: voteName.trim(),
				visibility: voteVisibility,
				minSelections: bigintFromInput(voteMinSelections),
				maxSelections: bigintFromInput(voteMaxSelections),
				optionLabels: parsedVoteOptions()
			});

			voteName = '';
			voteVisibility = 'open';
			voteMinSelections = '1';
			voteMaxSelections = '1';
			voteOptionsText = 'Yes\nNo';
			await loadVotes();
			voteNameInput?.focus();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to create the vote.');
		} finally {
			creatingVote = false;
		}
	}

	async function openVote(voteId: string) {
		await runVoteAction(`open-${voteId}`, async () => {
			await voteClient.openVote({
				committeeSlug: slug,
				meetingId,
				voteId
			});
			await loadVotes();
		});
	}

	async function closeVote(voteId: string) {
		await runVoteAction(`close-${voteId}`, async () => {
			await voteClient.closeVote({
				committeeSlug: slug,
				meetingId,
				voteId
			});
			await loadVotes();
		});
	}

	async function archiveVote(voteId: string) {
		await runVoteAction(`archive-${voteId}`, async () => {
			await voteClient.archiveVote({
				committeeSlug: slug,
				meetingId,
				voteId
			});
			actionNotice = 'Vote archived.';
			await loadVotes();
		});
	}


	function voteDefaultOptionLabels() {
		return ['Yes', 'No', 'Abstain', ''];
	}

	function voteAccordionDefaultOpen(vote: VoteDefinitionRecord) {
		return vote.state === 'open' || vote.state === 'counting';
	}

	function syncVotePanelOpenState(votes: VoteDefinitionRecord[]) {
		const nextVoteAccordionOpen: Record<string, boolean> = {};
		const nextDraftVoteEditorOpen: Record<string, boolean> = {};
		for (const vote of votes) {
			nextVoteAccordionOpen[vote.voteId] =
				voteAccordionOpen[vote.voteId] ?? voteAccordionDefaultOpen(vote);
			if (vote.state === 'draft') {
				nextDraftVoteEditorOpen[vote.voteId] = draftVoteEditorOpen[vote.voteId] ?? false;
			}
		}
		voteAccordionOpen = nextVoteAccordionOpen;
		draftVoteEditorOpen = nextDraftVoteEditorOpen;
	}

	function setVoteAccordionOpen(voteId: string, open: boolean) {
		voteAccordionOpen = { ...voteAccordionOpen, [voteId]: open };
	}

	function setDraftVoteEditorOpen(voteId: string, open: boolean) {
		draftVoteEditorOpen = { ...draftVoteEditorOpen, [voteId]: open };
	}

	function resolveAttendeeIdFromQuery(query: string): string | null {
		const trimmed = query.trim();
		const rows = attendeeRows();
		const leadingNum = trimmed.match(/^(\d+)/);
		if (leadingNum) {
			const num = BigInt(leadingNum[1]);
			const found = rows.find((a) => a.attendeeNumber === num);
			if (found) return found.attendeeId;
		}
		const exact = rows.find((a) => a.fullName.toLowerCase() === trimmed.toLowerCase());
		if (exact) return exact.attendeeId;
		const lower = trimmed.toLowerCase();
		const matches = rows.filter((a) =>
			`${a.attendeeNumber} ${a.fullName}`.toLowerCase().includes(lower)
		);
		if (matches.length === 1) return matches[0].attendeeId;
		return null;
	}

	async function registerCast(voteId: string, attendeeQuery: string) {
		const attendeeId = resolveAttendeeIdFromQuery(attendeeQuery);
		if (!attendeeId) throw new Error('Could not resolve attendee from query');
		await runVoteAction(`register-cast-${voteId}`, async () => {
			await voteClient.registerCast({ committeeSlug: slug, meetingId, voteId, attendeeId });
			await loadVotes();
		});
	}

	async function countOpenBallot(voteId: string, attendeeQuery: string, selectedOptionIds: string[]) {
		const attendeeId = resolveAttendeeIdFromQuery(attendeeQuery);
		if (!attendeeId) throw new Error('Could not resolve attendee from query');
		await runVoteAction(`ballot-open-${voteId}`, async () => {
			await voteClient.countOpenBallot({
				committeeSlug: slug,
				meetingId,
				voteId,
				attendeeId,
				selectedOptionIds
			});
			await loadVotes();
		});
	}

	async function countSecretBallot(voteId: string, receiptToken: string, selectedOptionIds: string[]) {
		await runVoteAction(`ballot-secret-${voteId}`, async () => {
			await voteClient.countSecretBallot({
				committeeSlug: slug,
				meetingId,
				voteId,
				receiptToken,
				selectedOptionIds
			});
			await loadVotes();
		});
	}

	async function submitCreateVoteForm(event: SubmitEvent) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement | null;
		if (!form) return;
		const data = new FormData(form);
		await runVoteAction('create-vote', async () => {
			await voteClient.createVote({
				committeeSlug: slug,
				meetingId,
				name: String(data.get('name') ?? '').trim(),
				visibility: String(data.get('visibility') ?? 'open'),
				minSelections: bigintFromInput(String(data.get('min_selections') ?? '1')),
				maxSelections: bigintFromInput(String(data.get('max_selections') ?? '1')),
				optionLabels: data
					.getAll('option_label')
					.map((value) => String(value).trim())
					.filter(Boolean)
			});
			form.reset();
			createVoteDetailsOpen = false;
			await loadVotes();
		});
	}

	async function submitUpdateDraftVoteForm(event: SubmitEvent, voteId: string) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement | null;
		if (!form) return;
		const data = new FormData(form);
		await runVoteAction(`save-draft-${voteId}`, async () => {
			await voteClient.updateVoteDraft({
				committeeSlug: slug,
				meetingId,
				voteId,
				name: String(data.get('name') ?? '').trim(),
				visibility: String(data.get('visibility') ?? 'open'),
				minSelections: bigintFromInput(String(data.get('min_selections') ?? '1')),
				maxSelections: bigintFromInput(String(data.get('max_selections') ?? '1')),
				optionLabels: data
					.getAll('option_label')
					.map((value) => String(value).trim())
					.filter(Boolean)
			});
			await loadVotes();
		});
	}
</script>

<div class="flex min-h-0 flex-1 flex-col gap-4">
	{#if moderationState.loading && !moderationState.data}
		<AppSpinner label="Loading moderation view" />
	{:else if moderationState.error && !moderationState.data}
		<AppAlert message={moderationState.error} />
	{:else if moderationState.data}
		<div class="shrink-0 space-y-2">
			<h1 class="text-3xl font-bold">{moderationState.data.meeting?.meetingName}</h1>
			<p class="text-base-content/70">
				{m.meeting_moderate_workspace_for({ committee: moderationState.data.meeting?.committeeName ?? "" })}
			</p>
		</div>

		{#if actionError}
			<div class="shrink-0"><AppAlert message={actionError} /></div>
		{/if}
		{#if actionNotice}
			<div class="shrink-0"><AppAlert tone="success" message={actionNotice} /></div>
		{/if}

		<div id="moderate-sse-root" class="grid min-h-0 flex-1 gap-4 overflow-y-auto lg:grid-cols-2 lg:grid-rows-1 lg:overflow-hidden">
			<section
				id="moderate-left-controls"
				data-meeting-id={meetingId}
				data-tabs-wired="true"
				class="order-2 card min-h-0 min-w-0 overflow-hidden border border-base-300 bg-base-100 shadow-sm lg:order-1 lg:h-full lg:self-stretch"
			>
				<div class="card-body flex h-full min-h-0 flex-col overflow-hidden p-4">
					<div role="tablist" class="tabs tabs-border tabs-sm grid w-full grid-cols-4 [--tab-p:0.35rem] sm:[--tab-p:0.75rem]">
						<button
							type="button"
							class={moderateLeftTab === 'agenda' ? 'tab tab-active min-w-0 justify-center truncate text-[0.72rem] sm:text-sm' : 'tab min-w-0 justify-center truncate text-[0.72rem] sm:text-sm'}
							data-moderate-left-tab="agenda"
							onclick={() => selectModerateLeftTab('agenda')}
						>
							{m.meeting_moderate_agenda_tab()}
						</button>
						<button
							type="button"
							class={moderateLeftTab === 'tools' ? 'tab tab-active min-w-0 justify-center truncate text-[0.72rem] sm:text-sm' : 'tab min-w-0 justify-center truncate text-[0.72rem] sm:text-sm'}
							data-moderate-left-tab="tools"
							onclick={() => selectModerateLeftTab('tools')}
						>
							{m.meeting_moderate_tools_tab()}
						</button>
						<button
							type="button"
							class={moderateLeftTab === 'attendees' ? 'tab tab-active min-w-0 justify-center truncate text-[0.72rem] sm:text-sm' : 'tab min-w-0 justify-center truncate text-[0.72rem] sm:text-sm'}
							data-moderate-left-tab="attendees"
							onclick={() => selectModerateLeftTab('attendees')}
						>
							{m.meeting_moderate_attendees_tab()}
						</button>
						<button
							type="button"
							class={moderateLeftTab === 'settings' ? 'tab tab-active min-w-0 justify-center truncate text-[0.72rem] sm:text-sm' : 'tab min-w-0 justify-center truncate text-[0.72rem] sm:text-sm'}
							data-moderate-left-tab="settings"
							onclick={() => selectModerateLeftTab('settings')}
						>
							{m.meeting_moderate_settings_tab()}
						</button>
					</div>
					<div class="mt-3 min-h-0 flex-1">
						<div id="moderate-left-panel-agenda" data-moderate-left-panel="agenda" class={moderateLeftTab === 'agenda' ? 'h-full min-h-0 overflow-y-auto pr-1' : 'hidden'}>
							<AgendaSection
								agendaPoints={agendaState.data ?? []}
								{slug}
								{meetingId}
								bind:anyDialogOpen={agendaAnyDialogOpen}
								onError={(msg) => (actionError = msg)}
								onReload={loadAgenda}
								onSetActivePoint={(p) => { if (moderationState.data) moderationState.data.activeAgendaPoint = p; }}
							/>
						</div>
						<div id="moderate-left-panel-tools" data-moderate-left-panel="tools" class={moderateLeftTab === 'tools' ? 'h-full min-h-0 overflow-hidden' : 'hidden h-full min-h-0 overflow-hidden'}>
							<section class="flex h-full min-h-0 flex-col overflow-hidden">
								<div class="flex items-center justify-between gap-2">
									<h2 class="text-lg font-semibold">{m.meeting_moderate_tools_tab()}</h2>
								</div>
								<div class="mt-3 min-h-0 flex-1 overflow-y-auto pr-1">
									{#if !moderationState.data.activeAgendaPoint}
										<p class="text-sm text-base-content/70">{m.meeting_manage_no_active_agenda_for_settings()}</p>
									{:else}
											<div class="space-y-5">
											<div class="space-y-2">
												<div class="flex items-center justify-between gap-2">
													<strong class="text-base">{m.meeting_moderate_files_heading()}</strong>
												</div>
												<p class="text-sm text-base-content/70">{m.meeting_moderate_no_files()}</p>
											</div>
											<div id="moderate-votes-panel-host">
												<div id="moderate-votes-panel" class="space-y-4" data-choice-label="Choice">
													<div class="flex items-center justify-between gap-2">
														<h3 class="text-base font-semibold">{m.votes_votes()}</h3>
														<button type="button" class="btn btn-xs btn-outline" onclick={(event) => { event.preventDefault(); void loadVotes(); }}>{m.common_refresh()}</button>
													</div>
													{#if votesState.loading && !votesState.data}
														<AppSpinner label="Loading votes" />
													{:else if votesState.error && !votesState.data}
														<AppAlert message={votesState.error} />
													{:else if !votesState.data?.hasActiveAgendaPoint}
														<p class="text-sm text-base-content/70">{m.meeting_manage_no_active_agenda_for_speakers()}</p>
													{:else}
														<details class="collapse collapse-arrow border border-base-300 bg-base-100" open={createVoteDetailsOpen} ontoggle={(event) => { createVoteDetailsOpen = (event.currentTarget as HTMLDetailsElement).open; }}>
															<summary class="collapse-title text-sm font-semibold">{m.votes_create_vote()}</summary>
															<div class="collapse-content">
																<form class="grid gap-2 md:grid-cols-2" onsubmit={submitCreateVoteForm}>
																	<input class="input input-bordered input-sm md:col-span-2" name="name" placeholder={m.votes_vote_name_placeholder()} required />
																	<select class="select select-bordered select-sm" name="visibility">
																		<option value="open">{m.votes_visibility_open()}</option>
																		<option value="secret">{m.votes_visibility_secret()}</option>
																	</select>
																	<div class="join">
																		<input class="input input-bordered input-sm join-item w-24" type="number" min="0" name="min_selections" value="1" required />
																		<input class="input input-bordered input-sm join-item w-24" type="number" min="1" name="max_selections" value="1" required />
																	</div>
																	<div class="md:col-span-2 space-y-1" data-vote-option-list>
																		<div class="text-xs font-semibold uppercase text-base-content/70">{m.votes_choices_label()}</div>
																		{#each voteDefaultOptionLabels() as label, index}
																			<div data-vote-option-row>
																				<input class="input input-bordered input-sm w-full" name="option_label" value={label} data-vote-option-input placeholder={m.votes_choice_n_placeholder({ n: index + 1 })} autocomplete="off" />
																			</div>
																		{/each}
																		<div class="text-xs text-base-content/70" data-vote-option-hint>Add one choice per field. A new field appears once the last one is filled.</div>
																	</div>
																	<div class="md:col-span-2">
																		<button type="submit" class="btn btn-sm btn-primary">{m.votes_create_draft_vote()}</button>
																	</div>
																</form>
															</div>
														</details>

														{#if (votesState.data?.votes ?? []).length === 0}
															<p class="text-sm text-base-content/70">{m.votes_no_votes_for_agenda_point({ agendaPoint: votesState.data?.activeAgendaPointTitle ?? "" })}</p>
														{:else}
															<div class="space-y-3">
																{#each votesState.data?.votes ?? [] as vote}
																	<VoteCard
																		{vote}
																		open={voteAccordionOpen[vote.voteId] ?? voteAccordionDefaultOpen(vote)}
																		draftEditorOpen={draftVoteEditorOpen[vote.voteId] ?? false}
																		attendees={attendeeRows()}
																		onToggle={(open) => setVoteAccordionOpen(vote.voteId, open)}
																		onDraftEditorToggle={(open) => setDraftVoteEditorOpen(vote.voteId, open)}
																		onOpenVote={async () => openVote(vote.voteId)}
																		onCloseVote={async () => closeVote(vote.voteId)}
																		onArchiveVote={async () => archiveVote(vote.voteId)}
																		onUpdateDraft={async (event) => submitUpdateDraftVoteForm(event, vote.voteId)}
																		onCountOpenBallot={async (attendeeQuery, optionIds) => countOpenBallot(vote.voteId, attendeeQuery, optionIds)}
																		onRegisterCast={async (attendeeQuery) => registerCast(vote.voteId, attendeeQuery)}
																		onCountSecretBallot={async (receiptToken, optionIds) => countSecretBallot(vote.voteId, receiptToken, optionIds)}
																	/>
																{/each}
															</div>
														{/if}
													{/if}
												</div>
											</div>
										</div>
									{/if}
								</div>
							</section>
						</div>
						<div id="moderate-left-panel-attendees" data-moderate-left-panel="attendees" class={moderateLeftTab === 'attendees' ? 'h-full min-h-0 overflow-y-auto pr-1' : 'hidden h-full min-h-0 overflow-y-auto pr-1'}>
							<section class="min-h-0" data-testid="manage-attendees-card">
								<div class="mb-3 flex items-center justify-between gap-2">
									<h2 class="text-lg font-semibold">{m.meeting_moderate_attendees_tab()}</h2>
									<div class="flex min-w-0 flex-wrap items-center justify-end gap-2">
										<form class="inline-flex order-last basis-full justify-center sm:order-none sm:basis-auto sm:justify-start" title={moderationState.data.attendees?.signupOpen ? 'Guest signup is open' : 'Guest signup is closed'}>
											<label class="label cursor-pointer justify-start gap-3" for="manage_signup_open">
												{#if moderationState.data.attendees?.signupOpen}
													<input checked class="toggle toggle-primary toggle-sm" id="manage_signup_open" name="signup_open" type="checkbox" value="true" disabled={togglingSignup || attendeeActionPending !== ''} onchange={toggleSignupOpen} />
												{:else}
													<input class="toggle toggle-primary toggle-sm" id="manage_signup_open" name="signup_open" type="checkbox" value="true" disabled={togglingSignup || attendeeActionPending !== ''} onchange={toggleSignupOpen} />
												{/if}
												<span>{m.meeting_moderate_guest_signup_label()}</span>
											</label>
										</form>
										<form class="inline-flex" data-testid="manage-self-signup-form" onsubmit={async (event) => { event.preventDefault(); await selfSignupAttendee(); }}>
											<button type="submit" class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Sign yourself up" title="Sign yourself up" aria-label="Sign yourself up" disabled={attendeeActionPending !== ''}><LegacyIcon name="person-raised" class="h-4 w-4" /></button>
										</form>
										<a href={manageJoinQrURL()} class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Show signup QR" title="Show signup QR" aria-label="Show signup QR"><LegacyIcon name="qr-code" class="h-4 w-4" /></a>
									</div>
								</div>
								<div id="attendee-list-container">
									<div class="mb-4">
										<form class="grid w-full gap-2 sm:grid-cols-[minmax(0,1fr)_auto_auto] sm:items-end" data-testid="manage-add-guest-form" onsubmit={addGuestAttendee}>
											<div class="w-full min-w-0 flex-1">
												<label class="label p-0 text-sm font-medium" for="full_name">{m.meeting_moderate_add_guest_label()}</label>
												<input class="input input-bordered input-sm w-full" type="text" id="full_name" name="full_name" placeholder={m.meeting_moderate_display_name_placeholder()} required disabled={attendeeActionPending !== ''} />
											</div>
											<label class="label cursor-pointer justify-start gap-2 p-0 sm:mb-1 whitespace-nowrap" for="manage_guest_gender_quoted">
												<span class="text-sm font-medium">{m.meeting_join_quoted_label()}</span>
												<input class="checkbox checkbox-sm" type="checkbox" id="manage_guest_gender_quoted" name="gender_quoted" value="true" disabled={attendeeActionPending !== ''} />
											</label>
											<button class="btn btn-sm sm:mb-1" type="submit" disabled={attendeeActionPending !== ''}>{m.meeting_moderate_add_guest_button()}</button>
										</form>
									</div>
									{#if attendeeRows().length === 0}
										<p class="text-sm text-base-content/70">{m.meeting_moderate_no_attendees()}</p>
									{:else}
										<ul class="list rounded-box border border-base-300 bg-base-100" data-testid="manage-attendee-grid">
											{#each attendeeRows() as attendee}
												<AttendeeRow
													{attendee}
													{attendeeActionPending}
													onRemove={removeAttendee}
													onToggleChair={toggleAttendeeChair}
													onToggleQuoted={toggleAttendeeQuoted}
													recoveryURL={attendeeRecoveryURL}
												/>
											{/each}
										</ul>
									{/if}
								</div>
							</section>
						</div>
						<div id="moderate-left-panel-settings" data-moderate-left-panel="settings" class={moderateLeftTab === 'settings' ? 'h-full min-h-0 overflow-y-auto pr-1' : 'hidden h-full min-h-0 overflow-y-auto pr-1'}>
							<section class="min-h-0 space-y-3" data-testid="manage-settings-card">
								<div class="flex flex-wrap items-end justify-between gap-2">
									<h2 class="text-lg font-semibold">{m.meeting_moderate_settings_tab()}</h2>
									<p class="text-sm text-base-content/70">{m.meeting_moderate_settings_description()}</p>
								</div>
								<div id="moderate-settings-shell" class="rounded-box border border-base-300 bg-base-200/30 p-3" data-active-tab={moderateSettingsTab} data-settings-tabs-wired="true">
									<div role="tablist" class="tabs tabs-box tabs-sm w-full bg-base-100">
										<button type="button" class={moderateSettingsTab === 'meeting' ? 'tab tab-active flex-1 justify-center' : 'tab flex-1 justify-center'} data-moderate-settings-tab="meeting" onclick={() => (moderateSettingsTab = 'meeting')}>{m.meeting_moderate_settings_meeting_tab()}</button>
										<button type="button" class={moderateSettingsTab === 'agenda' ? 'tab tab-active flex-1 justify-center' : 'tab flex-1 justify-center'} data-moderate-settings-tab="agenda" onclick={() => (moderateSettingsTab = 'agenda')}>{m.meeting_moderate_settings_agenda_tab()}</button>
									</div>
									<div class="mt-3 min-h-0">
										<div data-moderate-settings-panel="meeting" class={moderateSettingsTab === 'meeting' ? '' : 'hidden'}>
											<div class="mb-2 text-xs font-semibold uppercase tracking-wide text-base-content/60">{m.meeting_moderate_meeting_defaults()}</div>
											<div id="meeting-settings-container">
												<div class="space-y-3">
													<div class="rounded-box border border-base-300 bg-base-100 p-3">
														<h3 class="mb-2 text-sm font-semibold">{m.meeting_manage_agenda_point_quotation_settings()}</h3>
														<form class="grid gap-3 md:grid-cols-2">
															<div class="space-y-1">
																<label for="gender_quotation_enabled">{m.meeting_moderate_flinta_quotation_label()}</label>
																<select class="select select-bordered select-sm" id="gender_quotation_enabled" name="gender_quotation_enabled" disabled={settingsActionPending !== ''} onchange={(event) => { if (moderationState.data?.settings) { moderationState.data.settings.genderQuotationEnabled = (event.currentTarget as HTMLSelectElement).value === 'true'; } void updateMeetingQuotation(); }}>
																	{#if moderationState.data.settings?.genderQuotationEnabled ?? true}
																		<option selected value="true">{m.meeting_moderate_enabled()}</option>
																		<option value="false">{m.meeting_moderate_disabled()}</option>
																	{:else}
																		<option value="true">{m.meeting_moderate_enabled()}</option>
																		<option selected value="false">{m.meeting_moderate_disabled()}</option>
																	{/if}
																</select>
															</div>
															<div class="space-y-1">
																<label for="first_speaker_quotation_enabled">{m.meeting_moderate_first_speaker_bonus_label()}</label>
																<select class="select select-bordered select-sm" id="first_speaker_quotation_enabled" name="first_speaker_quotation_enabled" disabled={settingsActionPending !== ''} onchange={(event) => { if (moderationState.data?.settings) { moderationState.data.settings.firstSpeakerQuotationEnabled = (event.currentTarget as HTMLSelectElement).value === 'true'; } void updateMeetingQuotation(); }}>
																	{#if moderationState.data.settings?.firstSpeakerQuotationEnabled ?? true}
																		<option selected value="true">{m.meeting_moderate_enabled()}</option>
																		<option value="false">{m.meeting_moderate_disabled()}</option>
																	{:else}
																		<option value="true">{m.meeting_moderate_enabled()}</option>
																		<option selected value="false">{m.meeting_moderate_disabled()}</option>
																	{/if}
																</select>
															</div>
														</form>
													</div>
													<div class="rounded-box border border-base-300 bg-base-100 p-3">
														<h3 class="mb-2 text-sm font-semibold">{m.meeting_manage_agenda_point_moderator()}</h3>
														<form class="flex flex-wrap items-end gap-3">
															<select class="select select-bordered select-sm" id="meeting_moderator_attendee_id" name="attendee_id" disabled={settingsActionPending !== ''} onchange={updateMeetingModerator}>
																{#if !(moderationState.data.settings?.moderatorAttendeeId ?? '')}
																	<option selected value="">-- none --</option>
																{:else}
																	<option value="">-- none --</option>
																{/if}
																{#each settingsAttendees() as attendee}
																	{#if (moderationState.data.settings?.moderatorAttendeeId ?? '') === attendee.attendeeId}
																		<option selected value={attendee.attendeeId}>{attendee.fullName}</option>
																	{:else}
																		<option value={attendee.attendeeId}>{attendee.fullName}</option>
																	{/if}
																{/each}
															</select>
														</form>
													</div>
												</div>
											</div>
										</div>
										<div data-moderate-settings-panel="agenda" class={moderateSettingsTab === 'agenda' ? '' : 'hidden'}>
											<div class="mb-2 text-xs font-semibold uppercase tracking-wide text-base-content/60">{m.meeting_moderate_agenda_point_overrides()}</div>
											<div id="moderate-speaker-settings-container">
												<p class="text-sm text-base-content/70">{m.meeting_manage_no_active_agenda_for_settings()}</p>
											</div>
										</div>
									</div>
								</div>
							</section>
						</div>
					</div>
				</div>
			</section>

			<div id="moderate-dependent-container" class="order-1 flex min-h-0 min-w-0 flex-col lg:order-2 lg:self-stretch">
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
									onclick={() => openDocs('03-chairperson/05-speakers-moderator-and-quotation')}
								>
									<LegacyIcon name="help" />
								</button>
							</div>
							<div id="speakers-list-container" class="flex min-h-0 flex-1 flex-col">
								{#if !moderationState.data.activeAgendaPoint}
									<p class="text-sm text-base-content/70">{m.meeting_manage_no_active_agenda_for_speakers()}</p>
								{:else if speakerState.data?.speakers?.length}
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
															if (!next) {
																throw new Error('No waiting speaker is available.');
															}
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
										{#each speakerState.data.speakers as speaker, i}
											{@const prevSpeaker = speakerState.data.speakers[i - 1]}
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
															class="join-item btn btn-sm btn-square tooltip tooltip-left join-item btn btn-sm btn-error btn-square tooltip tooltip-left"
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
										onclick={() => openDocs('03-chairperson/05-speakers-moderator-and-quotation')}
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
			</div>
		</div>

	{/if}
</div>
