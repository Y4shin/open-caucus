<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { buildDocsOverlayHref } from '$lib/docs/navigation.js';
	import { onDestroy } from 'svelte';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
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
	import { createRemoteState } from '$lib/utils/remote.svelte.js';

	type AgendaImportState = 'ignore' | 'heading' | 'subheading';
	type AgendaImportLine = {
		lineNo: number;
		text: string;
		state: AgendaImportState;
	};
	type AgendaImportPoint = {
		title: string;
		children: string[];
	};

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
	let agendaActionPending = $state('');
	let voteActionPending = $state('');
	let settingsActionPending = $state('');
	let createVoteDetailsOpen = $state(false);
	let voteAccordionOpen = $state<Record<string, boolean>>({});
	let draftVoteEditorOpen = $state<Record<string, boolean>>({});
	let creatingAgenda = $state(false);
	let agendaEditOpen = $state(false);
	let agendaParentId = $state('');
	let editingAgendaPointId = $state('');
	let editingAgendaPointTitle = $state('');
	let creatingVote = $state(false);
	let agendaTitle = $state('');
	let agendaImportOpen = $state(false);
	let agendaImportSource = $state('');
	let agendaImportLines = $state<AgendaImportLine[]>([]);
	let agendaImportFingerprint = $state('');
	let agendaImportWarning = $state('');
	let agendaImportBusy = $state(false);
	let agendaImportStep = $state<'source' | 'correction' | 'diff'>('source');
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
	let agendaTitleInput = $state<HTMLInputElement | null>(null);
	let voteNameInput = $state<HTMLInputElement | null>(null);
	let nowMs = $state(Date.now());
	let speakingSinceMs = $state<Record<string, number>>({});
	let agendaEditDialogEl = $state<HTMLDialogElement | null>(null);
	let agendaImportDialogEl = $state<HTMLDialogElement | null>(null);

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
			if (agendaEditOpen || agendaImportOpen) return;
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

	async function runAgendaAction(key: string, action: () => Promise<void>) {
		actionError = '';
		agendaActionPending = key;
		try {
			await action();
			loadAgenda();
			return true;
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to update the agenda.');
			loadAgenda();
			return false;
		} finally {
			agendaActionPending = '';
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
				title,
				parentAgendaPointId: agendaParentId
			});
			agendaTitle = '';
			agendaParentId = '';
			loadAgenda();
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
			await loadAgenda();
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
		if (!window.confirm('Delete agenda point?')) return;
		await runAgendaAction(`delete-${agendaPointId}`, async () => {
			await agendaClient.deleteAgendaPoint({
				committeeSlug: slug,
				meetingId,
				agendaPointId
			});
			loadAgenda();
		});
	}

	function startEditAgendaPoint(point: AgendaPointRecord) {
		editingAgendaPointId = point.agendaPointId;
		editingAgendaPointTitle = point.title;
	}

	function cancelEditAgendaPoint() {
		editingAgendaPointId = '';
		editingAgendaPointTitle = '';
	}

	async function saveEditAgendaPoint(agendaPointId: string) {
		const title = editingAgendaPointTitle.trim();
		if (!title) return;
		await runAgendaAction(`edit-${agendaPointId}`, async () => {
			await agendaClient.updateAgendaPoint({ committeeSlug: slug, meetingId, agendaPointId, title });
			loadAgenda();
		});
		editingAgendaPointId = '';
		editingAgendaPointTitle = '';
	}

	function flattenAgenda(points: AgendaPointRecord[]) {
		const rows: Array<{ id: string; parentId: string; position: string; title: string }> = [];
		for (const point of points) {
			rows.push({
				id: point.agendaPointId,
				parentId: point.parentId,
				position: point.position.toString(),
				title: point.title.trim()
			});
			for (const child of point.subPoints) {
				rows.push({
					id: child.agendaPointId,
					parentId: child.parentId,
					position: child.position.toString(),
					title: child.title.trim()
				});
			}
		}
		return rows;
	}

	function agendaPointsFlat(points: AgendaPointRecord[] = agendaState.data ?? []) {
		const rows: AgendaPointRecord[] = [];
		const visit = (items: AgendaPointRecord[]) => {
			for (const item of items) {
				rows.push(item);
				if (item.subPoints.length) {
					visit(item.subPoints);
				}
			}
		};
		visit(points);
		return rows;
	}

	function agendaSiblings(point: AgendaPointRecord) {
		return agendaPointsFlat().filter((candidate) => candidate.parentId === point.parentId);
	}

	function agendaPointCanMoveUp(point: AgendaPointRecord) {
		return agendaSiblings(point).findIndex((candidate) => candidate.agendaPointId === point.agendaPointId) > 0;
	}

	function agendaPointCanMoveDown(point: AgendaPointRecord) {
		const siblings = agendaSiblings(point);
		const index = siblings.findIndex((candidate) => candidate.agendaPointId === point.agendaPointId);
		return index !== -1 && index < siblings.length - 1;
	}

	function legacyAgendaDisplayNumber(point: AgendaPointRecord) {
		if (point.displayNumber.startsWith('TOP')) return point.displayNumber;
		return `TOP ${point.displayNumber}`;
	}

	function currentAgendaFingerprint() {
		return JSON.stringify(flattenAgenda(agendaState.data ?? []));
	}

	async function fetchAgendaFingerprint() {
		const res = await agendaClient.listAgendaPoints({ committeeSlug: slug, meetingId });
		return JSON.stringify(flattenAgenda(res.agendaPoints));
	}

	function openAgendaImportDialog() {
		agendaImportOpen = true;
		agendaImportStep = 'source';
		agendaImportWarning = '';
		agendaImportLines = [];
		agendaImportFingerprint = currentAgendaFingerprint();
		agendaImportDialogEl?.showModal();
	}

	function closeAgendaImportDialog() {
		agendaImportOpen = false;
		agendaImportStep = 'source';
		agendaImportWarning = '';
		agendaImportDialogEl?.close();
	}

	function openAgendaEditDialog() {
		agendaEditOpen = true;
		agendaEditDialogEl?.showModal();
	}

	function closeAgendaEditDialog() {
		agendaEditOpen = false;
		agendaEditDialogEl?.close();
	}

	function agendaImportCurrentStep() {
		if (agendaImportStep === 'correction') return 2;
		if (agendaImportStep === 'diff') return 3;
		return 1;
	}

	function nextImportState(state: AgendaImportState): AgendaImportState {
		switch (state) {
			case 'ignore':
				return 'heading';
			case 'heading':
				return 'subheading';
			default:
				return 'ignore';
		}
	}

	function importPrefix(lines: AgendaImportLine[], index: number) {
		let top = 0;
		let sub = 0;
		for (let i = 0; i <= index; i += 1) {
			const line = lines[i];
			if (line.state === 'heading') {
				top += 1;
				sub = 0;
			} else if (line.state === 'subheading' && top > 0) {
				sub += 1;
			}
		}
		const line = lines[index];
		if (line.state === 'heading' && top > 0) return `TOP ${top}`;
		if (line.state === 'subheading' && top > 0 && sub > 0) return `TOP ${top}.${sub}`;
		return '';
	}

	function parseAgendaImportSource(source: string) {
		return source
			.split('\n')
			.map((line, index) => ({ raw: line.trim(), lineNo: index + 1 }))
			.filter((line) => line.raw.length > 0)
			.map(({ raw, lineNo }) => {
				const match = raw.match(/^(?:TOP\s*)?(\d+(?:\.\d+)?)[:.) -]*\s*(.+)$/i);
				if (match) {
					return {
						lineNo,
						text: match[2].trim(),
						state: match[1].includes('.') ? 'subheading' : 'heading'
					} satisfies AgendaImportLine;
				}
				return {
					lineNo,
					text: raw,
					state: 'ignore'
				} satisfies AgendaImportLine;
			});
	}

	function buildImportedAgenda(lines: AgendaImportLine[]) {
		const points: AgendaImportPoint[] = [];
		let currentTop: AgendaImportPoint | null = null;
		for (const line of lines) {
			if (line.state === 'heading') {
				currentTop = { title: line.text, children: [] };
				points.push(currentTop);
				continue;
			}
			if (line.state === 'subheading' && currentTop) {
				currentTop.children.push(line.text);
			}
		}
		return points;
	}

	async function extractAgendaImport() {
		const parsed = parseAgendaImportSource(agendaImportSource.trim());
		if (parsed.length === 0) {
			agendaImportWarning = 'No agenda lines were detected.';
			return;
		}
		agendaImportWarning = '';
		agendaImportLines = parsed;
		agendaImportStep = 'correction';
	}

	function toggleAgendaImportLine(index: number) {
		agendaImportLines = agendaImportLines.map((line, currentIndex) =>
			currentIndex === index ? { ...line, state: nextImportState(line.state) } : line
		);
	}

	function generateAgendaDiff() {
		if (buildImportedAgenda(agendaImportLines).length === 0) {
			agendaImportWarning = 'No agenda headings are selected for import.';
			return;
		}
		agendaImportWarning = '';
		agendaImportFingerprint = currentAgendaFingerprint();
		agendaImportStep = 'diff';
	}

	async function applyAgendaImport() {
		if (agendaImportBusy) return;

		const imported = buildImportedAgenda(agendaImportLines);
		if (imported.length === 0) {
			agendaImportWarning = 'No agenda headings are selected for import.';
			return;
		}

		agendaImportBusy = true;
		agendaImportWarning = '';
		try {
			const latestFingerprint = await fetchAgendaFingerprint();
			if (agendaImportFingerprint !== latestFingerprint) {
				agendaImportWarning = 'Agenda changed while you reviewed this diff';
				return;
			}

			for (const point of agendaState.data ?? []) {
				await agendaClient.deleteAgendaPoint({
					committeeSlug: slug,
					meetingId,
					agendaPointId: point.agendaPointId
				});
			}

			for (const point of imported) {
				const top = await agendaClient.createAgendaPoint({
					committeeSlug: slug,
					meetingId,
					title: point.title
				});
				const parentId = top.agendaPoint?.agendaPointId;
				if (!parentId) continue;
				for (const child of point.children) {
					await agendaClient.createAgendaPoint({
						committeeSlug: slug,
						meetingId,
						title: child,
						parentAgendaPointId: parentId
					});
				}
			}
			closeAgendaImportDialog();
			loadAgenda();
		} catch (err) {
			agendaImportWarning = getDisplayError(err, 'Failed to apply the agenda import.');
		} finally {
			agendaImportBusy = false;
		}
	}

	function handleAgendaImportFile(event: Event) {
		const input = event.currentTarget as HTMLInputElement | null;
		const file = input?.files?.[0];
		if (!file) return;
		file.text().then((content) => {
			agendaImportSource = content;
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

	function voteStateBadgeClass(state: string) {
		switch (state) {
			case 'draft':
				return 'badge badge-neutral badge-sm';
			case 'open':
				return 'badge badge-success badge-sm';
			case 'counting':
				return 'badge badge-warning badge-sm';
			case 'closed':
				return 'badge badge-info badge-sm';
			case 'archived':
				return 'badge badge-ghost badge-sm';
			default:
				return 'badge badge-sm';
		}
	}

	function voteVisibilityBadgeClass(visibility: string) {
		return visibility === 'secret' ? 'badge badge-warning badge-outline badge-sm' : 'badge badge-primary badge-outline badge-sm';
	}

	function voteStateLabel(state: string) {
		switch (state) {
			case 'draft':
				return 'Draft';
			case 'open':
				return 'Open';
			case 'counting':
				return 'Counting';
			case 'closed':
				return 'Closed';
			case 'archived':
				return 'Archived';
			default:
				return state;
		}
	}

	function voteVisibilityLabel(visibility: string) {
		return visibility === 'secret' ? 'Secret' : 'Open';
	}

	function voteBoundsLabel(vote: VoteDefinitionRecord) {
		if (vote.minSelections === vote.maxSelections) {
			return `select exactly ${vote.minSelections.toString()}`;
		}
		return `select between ${vote.minSelections.toString()} and ${vote.maxSelections.toString()}`;
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

	function voteLabelsForEdit(vote: VoteDefinitionRecord) {
		const labels = vote.options.map((option) => option.label);
		if (labels.length < 2) {
			labels.push('Yes', 'No');
		}
		labels.push('');
		return labels;
	}

	function emptyVoteStats() {
		return { eligibleCount: 0n, castCount: 0n, ballotCount: 0n };
	}

	function voteStatsFor(vote: VoteDefinitionRecord) {
		return vote.stats ?? emptyVoteStats();
	}

	function voteTalliesFor(vote: VoteDefinitionRecord) {
		return vote.tally ?? [];
	}

	function voteOutstandingCount(vote: VoteDefinitionRecord) {
		const stats = voteStatsFor(vote);
		const outstanding = stats.castCount - stats.ballotCount;
		return outstanding > 0n ? outstanding : 0n;
	}

	function voteShouldShowTallies(vote: VoteDefinitionRecord) {
		return vote.state === 'closed' || vote.state === 'archived';
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
				Moderation workspace for {moderationState.data.meeting?.committeeName}
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
							Agenda
						</button>
						<button
							type="button"
							class={moderateLeftTab === 'tools' ? 'tab tab-active min-w-0 justify-center truncate text-[0.72rem] sm:text-sm' : 'tab min-w-0 justify-center truncate text-[0.72rem] sm:text-sm'}
							data-moderate-left-tab="tools"
							onclick={() => selectModerateLeftTab('tools')}
						>
							Tools
						</button>
						<button
							type="button"
							class={moderateLeftTab === 'attendees' ? 'tab tab-active min-w-0 justify-center truncate text-[0.72rem] sm:text-sm' : 'tab min-w-0 justify-center truncate text-[0.72rem] sm:text-sm'}
							data-moderate-left-tab="attendees"
							onclick={() => selectModerateLeftTab('attendees')}
						>
							Attendees
						</button>
						<button
							type="button"
							class={moderateLeftTab === 'settings' ? 'tab tab-active min-w-0 justify-center truncate text-[0.72rem] sm:text-sm' : 'tab min-w-0 justify-center truncate text-[0.72rem] sm:text-sm'}
							data-moderate-left-tab="settings"
							onclick={() => selectModerateLeftTab('settings')}
						>
							Settings
						</button>
					</div>
					<div class="mt-3 min-h-0 flex-1">
						<div id="moderate-left-panel-agenda" data-moderate-left-panel="agenda" class={moderateLeftTab === 'agenda' ? 'h-full min-h-0 overflow-y-auto pr-1' : 'hidden'}>
							<section class="min-h-0" data-testid="manage-agenda-card">
								<div class="mb-3 flex items-center justify-between gap-2">
									<h2 class="text-lg font-semibold">Agenda Points</h2>
									<div class="flex items-center gap-2">
										<button type="button" class="btn btn-sm btn-square btn-ghost" title="Open agenda help" aria-label="Open agenda help" onclick={() => openDocsWithHeading('03-chairperson/03-agenda-management-and-import', 'agenda-routes')}><LegacyIcon name="help" class="h-4 w-4" /></button>
										<button type="button" class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Edit agenda" data-manage-dialog-open aria-controls="moderate-agenda-edit-dialog" title="Edit agenda" aria-label="Edit agenda" onclick={openAgendaEditDialog}><LegacyIcon name="settings" class="h-4 w-4" /></button>
									</div>
								</div>
								<div id="moderate-agenda-compact" class="space-y-2">
									{#if agendaPointsFlat().length === 0}
										<p class="text-sm text-base-content/70">No agenda points have been created yet.</p>
									{:else}
										<ul class="list rounded-box border border-base-300 bg-base-100">
											{#each agendaPointsFlat() as point}
												<li class={point.isActive ? 'list-row items-center gap-3 bg-primary/10' : point.parentId ? 'list-row items-center gap-3 pl-8' : 'list-row items-center gap-3'}>
													<span class="badge badge-outline">{legacyAgendaDisplayNumber(point)}</span>
													<span class={point.isActive ? 'flex-1 truncate font-semibold' : 'flex-1 truncate'}>{point.title}</span>
													{#if point.isActive}
														<span class="badge badge-success badge-sm">Active</span>
													{:else}
														<form class="inline" onsubmit={async (event) => { event.preventDefault(); await activateAgendaPoint(point.agendaPointId, point.isActive); }}>
															<button type="submit" class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Activate agenda point" title="Activate agenda point" aria-label="Activate agenda point" disabled={isAgendaBusy(`activate-${point.agendaPointId}`)}><LegacyIcon name="check-circle" class="h-4 w-4" /></button>
														</form>
													{/if}
												</li>
											{/each}
										</ul>
									{/if}
								</div>
								<dialog
									id="moderate-agenda-edit-dialog"
									class="modal"
									data-manage-dialog
									bind:this={agendaEditDialogEl}
									onclose={() => {
										agendaEditOpen = false;
									}}
								>
									<div class="modal-box w-11/12 max-w-5xl">
										<div class="mb-4 flex items-center justify-between gap-2">
											<h3 class="text-lg font-semibold">Edit Agenda</h3>
											<button type="button" class="btn btn-sm btn-ghost" data-manage-dialog-close onclick={closeAgendaEditDialog}>Close</button>
										</div>
										<div id="agenda-point-list-container" class="space-y-3" data-import-open={agendaImportOpen ? 'true' : 'false'} data-import-top-prefix="TOP">
											<div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
												<div class="rounded-box border border-base-300 bg-base-100 p-3 lg:col-span-1">
													<form class="space-y-3" data-testid="manage-agenda-add-form" onsubmit={async (event) => { event.preventDefault(); await createAgendaPoint(); }}>
														<fieldset class="fieldset rounded-box border border-base-300 p-3">
															<legend class="fieldset-legend px-1 text-sm font-semibold">Add Agenda Point</legend>
															<label class="label p-0 text-sm font-medium" for="ap_title">Title</label>
															<input class="input input-bordered input-sm w-full" type="text" id="ap_title" name="title" required placeholder="Agenda point title" bind:value={agendaTitle} bind:this={agendaTitleInput} onkeydown={handleAgendaTitleKeydown} />
															<label class="label mt-2 p-0 text-sm font-medium" for="ap_parent_id">Parent (optional)</label>
															<select class="select select-bordered select-sm w-full" id="ap_parent_id" name="parent_id" bind:value={agendaParentId}>
																<option value="">-- top-level --</option>
																{#each agendaPointsFlat() as point}
																	<option value={point.agendaPointId}>{point.title}</option>
																{/each}
															</select>
															<button type="submit" class="btn btn-sm mt-3 w-full"><LegacyIcon name="arrow-forward" class="h-4 w-4" />Add</button>
															<button type="button" class="btn btn-sm btn-outline mt-2 w-full" data-manage-dialog-open aria-controls="moderate-agenda-import-dialog" onclick={openAgendaImportDialog}>Import</button>
														</fieldset>
													</form>
												</div>
												<div class="rounded-box border border-base-300 bg-base-100 p-3 lg:col-span-2">
													{#if agendaPointsFlat().length === 0}
														<p class="text-sm text-base-content/70">No agenda points have been created yet.</p>
													{:else}
														<div class="grid gap-3">
															{#each agendaPointsFlat() as point}
																<div id={`agenda-point-card-${point.agendaPointId}`} class={point.isActive ? `card rounded-box border border-base-300 bg-base-100 p-3 shadow-sm bg-primary/5 border-primary/40${point.parentId ? ' ml-5' : ''}` : point.parentId ? 'card rounded-box border border-base-300 bg-base-100 p-3 shadow-sm ml-5' : 'card rounded-box border border-base-300 bg-base-100 p-3 shadow-sm'} data-testid="manage-agenda-point-card">
																	{#if editingAgendaPointId === point.agendaPointId}
																		<form class="flex items-center gap-2" data-testid="manage-agenda-point-edit-form" onsubmit={async (event) => { event.preventDefault(); await saveEditAgendaPoint(point.agendaPointId); }}>
																			<input class="input input-bordered input-sm flex-1" type="text" name="title" bind:value={editingAgendaPointTitle} required disabled={isAgendaBusy(`edit-${point.agendaPointId}`)} data-testid="manage-agenda-point-edit-input" />
																			<button type="submit" class="btn btn-sm btn-primary" disabled={isAgendaBusy(`edit-${point.agendaPointId}`)}>Save</button>
																			<button type="button" class="btn btn-sm btn-ghost" onclick={cancelEditAgendaPoint}>Cancel</button>
																		</form>
																	{:else}
																		<div class="flex items-start gap-3">
																			<span class="badge badge-outline shrink-0">{legacyAgendaDisplayNumber(point)}</span>
																			<div class="min-w-0 flex-1">
																				<div class="truncate font-semibold">{point.title}</div>
																				<div class="mt-1 flex flex-wrap gap-1">
																					{#if point.parentId}
																						<span class="badge badge-outline">Child</span>
																					{/if}
																					{#if point.isActive}
																						<span class="badge badge-outline badge-success" data-testid="manage-agenda-active-badge">Active</span>
																					{/if}
																				</div>
																			</div>
																		</div>
																		<div class="mt-2 flex items-center gap-2">
																			<form class="inline-flex" onsubmit={async (event) => { event.preventDefault(); await moveAgendaPoint(point.agendaPointId, 'up'); }}>
																				<button type="submit" class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Move up" title="Move up" aria-label="Move up" disabled={!agendaPointCanMoveUp(point) || isAgendaBusy(`move-${point.agendaPointId}-up`)}><LegacyIcon name="left" class="h-4 w-4 rotate-90" /></button>
																			</form>
																			<form class="inline-flex" onsubmit={async (event) => { event.preventDefault(); await moveAgendaPoint(point.agendaPointId, 'down'); }}>
																				<button type="submit" class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Move down" title="Move down" aria-label="Move down" disabled={!agendaPointCanMoveDown(point) || isAgendaBusy(`move-${point.agendaPointId}-down`)}><LegacyIcon name="right" class="h-4 w-4 rotate-90" /></button>
																			</form>
																			{#if !point.isActive}
																				<form class="inline-flex" onsubmit={async (event) => { event.preventDefault(); await activateAgendaPoint(point.agendaPointId, point.isActive); }}>
																					<button type="submit" class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Activate agenda point" title="Activate agenda point" aria-label="Activate agenda point" disabled={isAgendaBusy(`activate-${point.agendaPointId}`)}><LegacyIcon name="check-circle" class="h-4 w-4" /></button>
																				</form>
																			{/if}
																			<button type="button" class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Edit agenda point" title="Edit agenda point" aria-label="Edit agenda point" data-testid="manage-agenda-point-edit-btn" onclick={() => startEditAgendaPoint(point)}><LegacyIcon name="edit" class="h-4 w-4" /></button>
																			<a href={`/committee/${slug}/meeting/${meetingId}/agenda-point/${point.agendaPointId}/tools`} class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Open tools" title="Open tools" aria-label="Open tools"><LegacyIcon name="settings" class="h-4 w-4" /></a>
																			<form class="inline-flex" onsubmit={async (event) => { event.preventDefault(); await deleteAgendaPoint(point.agendaPointId); }}>
																				<button type="submit" class="btn btn-sm btn-square btn-error tooltip tooltip-left" data-tip="Delete agenda point" title="Delete agenda point" aria-label="Delete agenda point" disabled={isAgendaBusy(`delete-${point.agendaPointId}`)}><LegacyIcon name="trash" class="h-4 w-4" /></button>
																			</form>
																		</div>
																	{/if}
																</div>
															{/each}
														</div>
													{/if}
												</div>
											</div>
											<dialog
												id="moderate-agenda-import-dialog"
												class="modal"
												data-manage-dialog
												bind:this={agendaImportDialogEl}
												onclose={() => {
													agendaImportOpen = false;
													agendaImportStep = 'source';
													agendaImportWarning = '';
												}}
											>
												<div class="modal-box w-11/12 max-w-5xl">
													<div class="mb-4 flex items-center justify-between gap-2">
														<h3 class="text-lg font-semibold">Import Agenda</h3>
														<button type="button" class="btn btn-sm btn-ghost" data-manage-dialog-close onclick={closeAgendaImportDialog}>Close</button>
													</div>
													<div class="space-y-4">
														{#if agendaImportWarning}
															<div class="alert alert-warning text-sm">{agendaImportWarning}</div>
														{/if}
														<div class="space-y-4" data-agenda-import-flow data-agenda-import-step={agendaImportCurrentStep().toString()}>
															<ul class="steps steps-horizontal w-full">
																<li class={agendaImportCurrentStep() >= 1 ? 'step step-primary' : 'step'} data-agenda-import-step-item="1">Source</li>
																<li class={agendaImportCurrentStep() >= 2 ? 'step step-primary' : 'step'} data-agenda-import-step-item="2">Correction</li>
																<li class={agendaImportCurrentStep() >= 3 ? 'step step-primary' : 'step'} data-agenda-import-step-item="3">Diff</li>
															</ul>
															{#if agendaImportCurrentStep() === 1}
																<div data-agenda-import-panel="1" class="">
																	<form class="space-y-3" onsubmit={async (event) => { event.preventDefault(); await extractAgendaImport(); }}>
																		<label class="label p-0 text-sm font-medium" for="agenda-import-source">Source text</label>
																		<textarea id="agenda-import-source" name="source_text" class="textarea textarea-bordered min-h-40 w-full" placeholder="Paste markdown or plaintext agenda here." bind:value={agendaImportSource}></textarea>
																		<div class="flex flex-wrap items-center gap-2">
																			<input type="file" class="file-input file-input-bordered file-input-sm max-w-full" accept=".txt,.md,text/plain,text/markdown" data-agenda-import-file data-target="agenda-import-source" onchange={handleAgendaImportFile} />
																			<button type="submit" class="btn btn-sm btn-outline">Extract Agenda</button>
																		</div>
																	</form>
																</div>
															{:else}
																<div data-agenda-import-panel="1" class="hidden">
																	<form class="space-y-3" onsubmit={async (event) => { event.preventDefault(); await extractAgendaImport(); }}>
																		<label class="label p-0 text-sm font-medium" for="agenda-import-source">Source text</label>
																		<textarea id="agenda-import-source" name="source_text" class="textarea textarea-bordered min-h-40 w-full" placeholder="Paste markdown or plaintext agenda here." bind:value={agendaImportSource}></textarea>
																		<div class="flex flex-wrap items-center gap-2">
																			<input type="file" class="file-input file-input-bordered file-input-sm max-w-full" accept=".txt,.md,text/plain,text/markdown" data-agenda-import-file data-target="agenda-import-source" onchange={handleAgendaImportFile} />
																			<button type="submit" class="btn btn-sm btn-outline">Extract Agenda</button>
																		</div>
																	</form>
																</div>
															{/if}
															{#if agendaImportLines.length > 0}
																<div data-agenda-import-panel="2" class={agendaImportCurrentStep() === 2 ? '' : 'hidden'}>
																	<form id="agenda-import-correction-form" class="space-y-3" onsubmit={(event) => { event.preventDefault(); generateAgendaDiff(); }}>
																		<input type="hidden" name="source_text" value={agendaImportSource} />
																		<h4 class="text-base font-semibold">Correct detected agenda structure</h4>
																		<p class="text-sm text-base-content/70">Click each line to cycle between ignored, heading, and subheading.</p>
																		<div class="max-h-80 overflow-y-auto rounded-box border border-base-300 bg-base-100 p-2">
																			<ul class="space-y-2" data-agenda-import-lines>
																				{#each agendaImportLines as line, index}
																					<li>
																						<input type="hidden" name="line_no" value={line.lineNo.toString()} />
																						<input type="hidden" name="line_text" value={line.text} />
																						<input type="hidden" name="line_detected_state" value={line.state} />
																						<input type="hidden" name="line_state" value={line.state} data-import-line-state />
																						<button type="button" class={line.state === 'heading' ? 'flex w-full items-center gap-3 rounded-box border px-3 py-2 text-left transition-colors border-primary/30 bg-primary/10' : line.state === 'subheading' ? 'flex w-full items-center gap-3 rounded-box border px-3 py-2 text-left transition-colors border-info/30 bg-info/10' : 'flex w-full items-center gap-3 rounded-box border px-3 py-2 text-left transition-colors border-base-300 bg-base-100 opacity-60'} data-import-line-row data-state={line.state} onclick={() => toggleAgendaImportLine(index)}>
																							<span class="w-24 shrink-0 text-xs font-medium tabular-nums text-base-content/60" data-import-line-prefix>{importPrefix(agendaImportLines, index)}</span>
																							<span class="min-w-0 flex-1 truncate text-sm" title={line.text}>{line.text}</span>
																						</button>
																					</li>
																				{/each}
																			</ul>
																		</div>
																		<div class="flex flex-wrap gap-2">
																			<button type="button" class="btn btn-sm btn-ghost" data-agenda-import-back="1" onclick={() => (agendaImportStep = 'source')}>Back</button>
																			<button type="submit" class="btn btn-sm">Generate Diff</button>
																		</div>
																	</form>
																</div>
															{/if}
															{#if agendaImportStep === 'diff' && buildImportedAgenda(agendaImportLines).length > 0}
																<div data-agenda-import-panel="3" class={agendaImportCurrentStep() === 3 ? '' : 'hidden'}>
																	<div class="space-y-3">
																		<h4 class="text-base font-semibold">Agenda Diff</h4>
																		<div id="agenda-import-diff-grid" class="rounded-box border border-base-300 bg-base-100 p-2">
																			<style>
																				#agenda-import-diff-grid [data-diff-cell] {
																					transition: filter 120ms ease, box-shadow 120ms ease;
																				}
																				#agenda-import-diff-grid [data-diff-cell][data-hovered="true"] {
																					filter: brightness(0.95);
																					box-shadow: inset 0 0 0 1px color-mix(in srgb, currentColor 20%, transparent);
																				}
																			</style>
																			<div class="mb-2 hidden grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,2fr)] gap-2 px-1 text-xs font-semibold uppercase tracking-wide text-base-content/60 md:grid">
																				<div>Current</div>
																				<div class="text-center">Change</div>
																				<div>Imported</div>
																			</div>
																			<ul class="space-y-2">
																				{#each buildImportedAgenda(agendaImportLines) as point}
																					<li class="grid grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,2fr)] gap-2 text-sm">
																						<div class="min-h-12 rounded-box border border-dashed border-base-300/70 bg-base-200/20"></div>
																						<div class="flex min-h-12 items-center justify-center"><span class="tooltip tooltip-top text-success" data-tip="Add"><LegacyIcon name="arrow-forward" class="h-5 w-5" /></span></div>
																						<div class="min-h-12 rounded-box border border-success/30 bg-success/10 p-2" data-diff-cell data-diff-hover-key={point.title}>
																							<div class="mb-1 text-xs font-medium uppercase tracking-wide text-base-content/60 md:hidden">Imported</div>
																							<div class="flex items-start justify-between gap-2">
																								<div class="min-w-0 flex-1">
																									<div class="text-xs text-base-content/70">TOP</div>
																									<div class="truncate">{point.title}</div>
																								</div>
																							</div>
																						</div>
																					</li>
																				{/each}
																			</ul>
																		</div>
																		<div class="alert alert-warning text-sm">Applying this import replaces the existing agenda.</div>
																		<div class="flex flex-wrap gap-2">
																			<button type="button" class="btn btn-sm btn-ghost" data-agenda-import-back="2" onclick={() => (agendaImportStep = 'correction')}>Back</button>
																			<form class="inline-flex" onsubmit={(event) => { event.preventDefault(); void applyAgendaImport(); }}>
																				<button type="submit" class="btn btn-sm btn-primary" disabled={agendaImportBusy}>Accept</button>
																			</form>
																			<button type="button" class="btn btn-sm btn-ghost" data-manage-dialog-close onclick={closeAgendaImportDialog}>Deny</button>
																		</div>
																	</div>
																</div>
															{/if}
														</div>
													</div>
												</div>
												<form method="dialog" class="modal-backdrop">
													<button aria-label="Close">Close</button>
												</form>
											</dialog>
										</div>
									</div>
									<form method="dialog" class="modal-backdrop">
										<button aria-label="Close">Close</button>
									</form>
								</dialog>
							</section>
						</div>
						<div id="moderate-left-panel-tools" data-moderate-left-panel="tools" class={moderateLeftTab === 'tools' ? 'h-full min-h-0 overflow-hidden' : 'hidden h-full min-h-0 overflow-hidden'}>
							<section class="flex h-full min-h-0 flex-col overflow-hidden">
								<div class="flex items-center justify-between gap-2">
									<h2 class="text-lg font-semibold">Tools</h2>
								</div>
								<div class="mt-3 min-h-0 flex-1 overflow-y-auto pr-1">
									{#if !moderationState.data.activeAgendaPoint}
										<p class="text-sm text-base-content/70">No active agenda point.</p>
									{:else}
											<div class="space-y-5">
											<div class="space-y-2">
												<div class="flex items-center justify-between gap-2">
													<strong class="text-base">Files</strong>
												</div>
												<p class="text-sm text-base-content/70">No files have been uploaded yet.</p>
											</div>
											<div id="moderate-votes-panel-host">
												<div id="moderate-votes-panel" class="space-y-4" data-choice-label="Choice">
													<div class="flex items-center justify-between gap-2">
														<h3 class="text-base font-semibold">Votes</h3>
														<button type="button" class="btn btn-xs btn-outline" onclick={(event) => { event.preventDefault(); void loadVotes(); }}>Refresh</button>
													</div>
													{#if votesState.loading && !votesState.data}
														<AppSpinner label="Loading votes" />
													{:else if votesState.error && !votesState.data}
														<AppAlert message={votesState.error} />
													{:else if !votesState.data?.hasActiveAgendaPoint}
														<p class="text-sm text-base-content/70">No active agenda point.</p>
													{:else}
														<details class="collapse collapse-arrow border border-base-300 bg-base-100" open={createVoteDetailsOpen} ontoggle={(event) => { createVoteDetailsOpen = (event.currentTarget as HTMLDetailsElement).open; }}>
															<summary class="collapse-title text-sm font-semibold">Create Vote</summary>
															<div class="collapse-content">
																<form class="grid gap-2 md:grid-cols-2" onsubmit={submitCreateVoteForm}>
																	<input class="input input-bordered input-sm md:col-span-2" name="name" placeholder="Vote name" required />
																	<select class="select select-bordered select-sm" name="visibility">
																		<option value="open">Open</option>
																		<option value="secret">Secret</option>
																	</select>
																	<div class="join">
																		<input class="input input-bordered input-sm join-item w-24" type="number" min="0" name="min_selections" value="1" required />
																		<input class="input input-bordered input-sm join-item w-24" type="number" min="1" name="max_selections" value="1" required />
																	</div>
																	<div class="md:col-span-2 space-y-1" data-vote-option-list>
																		<div class="text-xs font-semibold uppercase text-base-content/70">Choices</div>
																		{#each voteDefaultOptionLabels() as label, index}
																			<div data-vote-option-row>
																				<input class="input input-bordered input-sm w-full" name="option_label" value={label} data-vote-option-input placeholder={`Choice ${index + 1}`} autocomplete="off" />
																			</div>
																		{/each}
																		<div class="text-xs text-base-content/70" data-vote-option-hint>Add one choice per field. A new field appears once the last one is filled.</div>
																	</div>
																	<div class="md:col-span-2">
																		<button type="submit" class="btn btn-sm btn-primary">Create Draft Vote</button>
																	</div>
																</form>
															</div>
														</details>

														{#if (votesState.data?.votes ?? []).length === 0}
															<p class="text-sm text-base-content/70">No votes for {votesState.data?.activeAgendaPointTitle}.</p>
														{:else}
															<div class="space-y-3">
																{#each votesState.data?.votes ?? [] as vote}
																	<details
																		class="collapse collapse-arrow border border-base-300 bg-base-100"
																		open={voteAccordionOpen[vote.voteId] ?? voteAccordionDefaultOpen(vote)}
																		ontoggle={(event) => {
																			setVoteAccordionOpen(
																				vote.voteId,
																				(event.currentTarget as HTMLDetailsElement).open
																			);
																		}}
																		data-vote-accordion={vote.voteId}
																	>
																		<summary class="collapse-title py-3 pr-10">
																			<div class="flex flex-wrap items-center gap-2">
																				<h4 class="font-semibold">{vote.name}</h4>
																				<span class={voteStateBadgeClass(vote.state)}>{voteStateLabel(vote.state)}</span>
																				<span class={voteVisibilityBadgeClass(vote.visibility)}>{voteVisibilityLabel(vote.visibility)}</span>
																				<span class="text-xs text-base-content/70">{voteBoundsLabel(vote)}</span>
																			</div>
																		</summary>
																		<div class="collapse-content space-y-3">
																			{#if vote.options.length > 0}
																				<ul class="list rounded-box border border-base-300 bg-base-200/40">
																					{#each vote.options as option}
																						<li class="list-row py-1">
																							<span class="badge badge-outline badge-sm">{option.position.toString()}</span>
																							<span class="flex-1 truncate">{option.label}</span>
																						</li>
																					{/each}
																				</ul>
																			{/if}

																			<div class="rounded-box border border-base-300 bg-base-200/30 p-2">
																					<div class="mb-1 text-xs font-semibold uppercase text-base-content/70">Live Submission Tally</div>
																				<div class="grid gap-1 text-sm sm:grid-cols-2">
																					<div class="flex items-center justify-between gap-2">
																						<span>Eligible</span>
																						<span class="badge badge-outline badge-sm">{voteStatsFor(vote).eligibleCount.toString()}</span>
																					</div>
																					<div class="flex items-center justify-between gap-2">
																						<span>Casts</span>
																						<span class="badge badge-outline badge-sm">{voteStatsFor(vote).castCount.toString()}</span>
																					</div>
																					<div class="flex items-center justify-between gap-2">
																						<span>Counted Ballots</span>
																						<span class="badge badge-outline badge-sm">{voteStatsFor(vote).ballotCount.toString()}</span>
																					</div>
																					<div class="flex items-center justify-between gap-2">
																						<span>Outstanding</span>
																						<span class={voteOutstandingCount(vote) > 0n ? 'badge badge-sm badge-warning' : 'badge badge-sm badge-success'}>{voteOutstandingCount(vote).toString()}</span>
																					</div>
																				</div>
																			</div>

																			{#if vote.state === 'draft'}
																				<div class="flex flex-wrap gap-2">
																					<button type="button" class="btn btn-sm btn-success" onclick={async (event) => { event.preventDefault(); event.stopPropagation(); await openVote(vote.voteId); }}>Open Vote</button>
																				</div>
																				<details
																					class="collapse collapse-arrow border border-base-300 bg-base-200/30"
																					open={draftVoteEditorOpen[vote.voteId] ?? false}
																					ontoggle={(event) => {
																						setDraftVoteEditorOpen(
																							vote.voteId,
																							(event.currentTarget as HTMLDetailsElement).open
																						);
																					}}
																					data-vote-draft-editor={vote.voteId}
																				>
																					<summary class="collapse-title text-sm">Edit Draft</summary>
																					<div class="collapse-content">
																						<form class="grid gap-2 md:grid-cols-2" onsubmit={async (event) => await submitUpdateDraftVoteForm(event, vote.voteId)}>
																							<input class="input input-bordered input-sm md:col-span-2" name="name" value={vote.name} required />
																							<select class="select select-bordered select-sm" name="visibility">
																								<option value="open" selected={vote.visibility === 'open'}>Open</option>
																								<option value="secret" selected={vote.visibility === 'secret'}>Secret</option>
																							</select>
																							<div class="join">
																								<input class="input input-bordered input-sm join-item w-24" type="number" min="0" name="min_selections" value={vote.minSelections.toString()} required />
																								<input class="input input-bordered input-sm join-item w-24" type="number" min="1" name="max_selections" value={vote.maxSelections.toString()} required />
																							</div>
																							<div class="md:col-span-2 space-y-1" data-vote-option-list>
																								<div class="text-xs font-semibold uppercase text-base-content/70">Choices</div>
																								{#each voteLabelsForEdit(vote) as label, index}
																									<div data-vote-option-row>
																										<input class="input input-bordered input-sm w-full" name="option_label" value={label} data-vote-option-input placeholder={`Choice ${index + 1}`} autocomplete="off" />
																									</div>
																								{/each}
																								<div class="text-xs text-base-content/70" data-vote-option-hint>Leave an empty field at the end to add another choice.</div>
																							</div>
																							<div class="md:col-span-2 flex flex-wrap gap-2">
																								<button type="submit" class="btn btn-sm btn-primary">Save Draft</button>
																							</div>
																						</form>
																					</div>
																				</details>
																			{/if}

																			{#if vote.state === 'open' || vote.state === 'counting'}
																				<div class="flex flex-wrap gap-2">
																					<button type="button" class="btn btn-sm btn-warning" onclick={async (event) => { event.preventDefault(); event.stopPropagation(); await closeVote(vote.voteId); }}>Close Vote</button>
																				</div>
																			{/if}

																			<div class="rounded-box border border-base-300 bg-base-200/20 p-2 space-y-2">
																				<div class="text-xs font-semibold uppercase text-base-content/70">Manual Submission</div>
																				{#if vote.visibility === 'open'}
																					{#if vote.state === 'open'}
																						<div class="space-y-2" id={`open-ballot-form-${vote.voteId}`} data-testid="manage-vote-open-ballot-form">
																							<div class="text-xs font-semibold uppercase text-base-content/70">Open Ballot Entry</div>
																							<div class={vote.maxSelections === 1n ? 'grid gap-1 text-sm grid-cols-1' : 'grid gap-1 text-sm grid-cols-2'}>
																								{#each vote.options as option}
																									<label class="label cursor-pointer justify-start gap-2 rounded border border-base-300 px-2 py-1">
																										{#if vote.maxSelections === 1n}
																											<input class="radio radio-sm" type="radio" name={`open-option-${vote.voteId}`} value={option.optionId} />
																										{:else}
																											<input class="checkbox checkbox-sm" type="checkbox" name={`open-option-${vote.voteId}`} value={option.optionId} />
																										{/if}
																										<span>{option.label}</span>
																									</label>
																								{/each}
																							</div>
																							{#if attendeeRows().length === 0}
																								<p class="text-xs text-warning">No attendees available for manual entry.</p>
																							{:else}
																								<div class="join w-full">
																									<input id={`open-ballot-attendee-${vote.voteId}`} class="input input-bordered input-sm join-item w-full" list={`vote-manual-open-attendee-list-${vote.voteId}`} placeholder="Search attendee" required data-testid="open-ballot-attendee-query" />
																									<button type="button" class="btn btn-sm btn-primary join-item" data-testid="open-ballot-submit" onclick={async () => {
																										const container = document.getElementById(`open-ballot-form-${vote.voteId}`);
																										const attendeeInput = document.getElementById(`open-ballot-attendee-${vote.voteId}`) as HTMLInputElement | null;
																										const attendeeQuery = attendeeInput?.value ?? '';
																										const checked = [...(container?.querySelectorAll(`[name="open-option-${vote.voteId}"]:checked`) ?? [])].map((el) => (el as HTMLInputElement).value);
																										await countOpenBallot(vote.voteId, attendeeQuery, checked);
																										if (attendeeInput) attendeeInput.value = '';
																										container?.querySelectorAll(`[name="open-option-${vote.voteId}"]`).forEach((el) => { (el as HTMLInputElement).checked = false; });
																									}}>Submit Ballot</button>
																								</div>
																								<datalist id={`vote-manual-open-attendee-list-${vote.voteId}`}>
																									{#each attendeeRows() as attendee}
																										<option value={`${attendee.attendeeNumber.toString()} ${attendee.fullName}`}></option>
																									{/each}
																								</datalist>
																								<p class="text-xs text-base-content/70">Quick-cast uses attendee number followed by the attendee name.</p>
																							{/if}
																						</div>
																					{:else}
																						<p class="text-xs text-base-content/70">Manual open-ballot entry is available while the vote is open.</p>
																					{/if}
																				{:else}
																					<p class="text-xs text-base-content/70">Secret ballots can be registered first and counted later.</p>
																					{#if vote.state === 'open' || vote.state === 'counting'}
																						<div class={vote.state === 'open' ? 'grid gap-2 md:grid-cols-2' : 'grid gap-2 md:grid-cols-1'}>
																							{#if vote.state === 'open'}
																								<div class="rounded-box border border-base-300 p-2 space-y-2" id={`register-cast-form-${vote.voteId}`} data-testid="manage-vote-register-cast-form">
																									<div class="text-xs font-semibold uppercase text-base-content/70">1. Register Cast</div>
																									{#if attendeeRows().length === 0}
																										<p class="text-xs text-warning">No attendees are available for cast registration.</p>
																									{:else}
																										<div class="join w-full">
																											<input id={`register-cast-attendee-${vote.voteId}`} class="input input-bordered input-sm join-item w-full" list={`vote-manual-secret-attendee-list-${vote.voteId}`} placeholder="Search attendee" required data-testid="register-cast-attendee-query" />
																											<button type="button" class="btn btn-sm join-item" data-testid="register-cast-submit" onclick={async () => {
																												const attendeeInput = document.getElementById(`register-cast-attendee-${vote.voteId}`) as HTMLInputElement | null;
																												const attendeeQuery = attendeeInput?.value ?? '';
																												await registerCast(vote.voteId, attendeeQuery);
																												if (attendeeInput) attendeeInput.value = '';
																											}}>Register Cast</button>
																										</div>
																										<datalist id={`vote-manual-secret-attendee-list-${vote.voteId}`}>
																											{#each attendeeRows() as attendee}
																												<option value={`${attendee.attendeeNumber.toString()} ${attendee.fullName}`}></option>
																											{/each}
																										</datalist>
																										<p class="text-xs text-base-content/70">Quick registration uses attendee number followed by the attendee name.</p>
																									{/if}
																								</div>
																							{/if}
																							<div class="rounded-box border border-base-300 p-2 space-y-2" id={`count-secret-form-${vote.voteId}`} data-testid="manage-vote-count-secret-form">
																								<div class="text-xs font-semibold uppercase text-base-content/70">2. Count Secret Ballot</div>
																								<input id={`secret-receipt-${vote.voteId}`} class="input input-bordered input-sm w-full" placeholder="Receipt token (optional)" data-testid="count-secret-receipt-token" />
																								<div class="grid grid-cols-2 gap-1 text-sm">
																									{#each vote.options as option}
																										<label class="label cursor-pointer justify-start gap-2 rounded border border-base-300 px-2 py-1">
																											{#if vote.maxSelections === 1n}
																												<input class="radio radio-sm" type="radio" name={`secret-option-${vote.voteId}`} value={option.optionId} />
																											{:else}
																												<input class="checkbox checkbox-sm" type="checkbox" name={`secret-option-${vote.voteId}`} value={option.optionId} />
																											{/if}
																											<span>{option.label}</span>
																										</label>
																									{/each}
																								</div>
																								<button type="button" class="btn btn-sm btn-primary" data-testid="count-secret-submit" onclick={async () => {
																									const container = document.getElementById(`count-secret-form-${vote.voteId}`);
																									const receiptInput = document.getElementById(`secret-receipt-${vote.voteId}`) as HTMLInputElement | null;
																									const receiptToken = receiptInput?.value ?? '';
																									const checked = [...(container?.querySelectorAll(`[name="secret-option-${vote.voteId}"]:checked`) ?? [])].map((el) => (el as HTMLInputElement).value);
																									await countSecretBallot(vote.voteId, receiptToken, checked);
																									if (receiptInput) receiptInput.value = '';
																									container?.querySelectorAll(`[name="secret-option-${vote.voteId}"]`).forEach((el) => { (el as HTMLInputElement).checked = false; });
																								}}>Count Ballot</button>
																							</div>
																						</div>
																					{:else}
																						<p class="text-xs text-base-content/70">Manual secret actions are available while vote is open or counting.</p>
																					{/if}
																				{/if}
																			</div>

																			{#if vote.state === 'closed'}
																				<div class="flex flex-wrap gap-2">
																					<button type="button" class="btn btn-sm btn-outline" onclick={async (event) => { event.preventDefault(); event.stopPropagation(); await archiveVote(vote.voteId); }}>Archive Vote</button>
																				</div>
																			{/if}

																			{#if vote.state === 'counting'}
																				<p class="text-sm text-warning">Results are blocked while vote is in counting state.</p>
																			{:else if voteShouldShowTallies(vote)}
																				<div class="rounded-box border border-base-300 bg-base-200/30 p-2">
																					<div class="mb-1 text-xs font-semibold uppercase text-base-content/70">Final Tallies</div>
																					<ul class="space-y-1 text-sm">
																						{#each voteTalliesFor(vote) as row}
																							<li class="flex items-center justify-between gap-2">
																								<span class="truncate">{row.label}</span>
																								<span class="badge badge-outline badge-sm">{row.count.toString()}</span>
																							</li>
																						{/each}
																					</ul>
																					<div class="mt-2 text-xs text-base-content/70">eligible={voteStatsFor(vote).eligibleCount.toString()} casts={voteStatsFor(vote).castCount.toString()} ballots={voteStatsFor(vote).ballotCount.toString()}</div>
																				</div>
																			{/if}
																		</div>
																	</details>
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
									<h2 class="text-lg font-semibold">Attendees</h2>
									<div class="flex min-w-0 flex-wrap items-center justify-end gap-2">
										<form class="inline-flex order-last basis-full justify-center sm:order-none sm:basis-auto sm:justify-start" title={moderationState.data.attendees?.signupOpen ? 'Guest signup is open' : 'Guest signup is closed'}>
											<label class="label cursor-pointer justify-start gap-3" for="manage_signup_open">
												{#if moderationState.data.attendees?.signupOpen}
													<input checked class="toggle toggle-primary toggle-sm" id="manage_signup_open" name="signup_open" type="checkbox" value="true" disabled={togglingSignup || attendeeActionPending !== ''} onchange={toggleSignupOpen} />
												{:else}
													<input class="toggle toggle-primary toggle-sm" id="manage_signup_open" name="signup_open" type="checkbox" value="true" disabled={togglingSignup || attendeeActionPending !== ''} onchange={toggleSignupOpen} />
												{/if}
												<span>Guest signup</span>
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
												<label class="label p-0 text-sm font-medium" for="full_name">Add guest</label>
												<input class="input input-bordered input-sm w-full" type="text" id="full_name" name="full_name" placeholder="Display name" required disabled={attendeeActionPending !== ''} />
											</div>
											<label class="label cursor-pointer justify-start gap-2 p-0 sm:mb-1 whitespace-nowrap" for="manage_guest_gender_quoted">
												<span class="text-sm font-medium">FLINTA*</span>
												<input class="checkbox checkbox-sm" type="checkbox" id="manage_guest_gender_quoted" name="gender_quoted" value="true" disabled={attendeeActionPending !== ''} />
											</label>
											<button class="btn btn-sm sm:mb-1" type="submit" disabled={attendeeActionPending !== ''}>Add</button>
										</form>
									</div>
									{#if attendeeRows().length === 0}
										<p class="text-sm text-base-content/70">No attendees are registered yet.</p>
									{:else}
										<ul class="list rounded-box border border-base-300 bg-base-100" data-testid="manage-attendee-grid">
											{#each attendeeRows() as attendee}
												<li class="list-row grid-cols-1 items-center gap-3" data-testid="manage-attendee-card">
													<div class="col-span-full w-full min-w-0 space-y-2">
														<div class="flex min-w-0 items-center gap-2">
															<div class="w-12 shrink-0 text-base-content/70">#{attendee.attendeeNumber.toString()}</div>
															<div class="min-w-0 flex-1 overflow-x-hidden">
																<div class="truncate overflow-x-hidden font-semibold">{attendee.fullName}</div>
																{#if attendee.isGuest || attendee.isChair || attendee.quoted}
																	<div class="mt-1 hidden flex-wrap items-center gap-1 sm:flex">
																		{#if attendee.isGuest}
																			<span class="badge badge-neutral badge-sm">Guest</span>
																		{/if}
																		{#if attendee.isChair}
																			<span class="tooltip tooltip-right" data-tip="Chairperson">
																				<span class="badge badge-success badge-sm"><LegacyIcon name="crown" class="h-3.5 w-3.5" /></span>
																			</span>
																		{/if}
																		{#if attendee.quoted}
																			<span class="tooltip tooltip-right" data-tip="FLINTA*">
																				<span class="badge badge-info badge-sm" data-testid="manage-attendee-quoted-badge"><LegacyIcon name="transgender" class="h-3.5 w-3.5" /></span>
																			</span>
																		{/if}
																	</div>
																{/if}
															</div>
															<div class="ml-auto shrink-0 self-center">
																<div class="join sm:join-vertical">
																	{#if attendee.isGuest}
																		<a href={attendeeRecoveryURL(attendee.attendeeId)} class="join-item btn btn-sm btn-square tooltip tooltip-left" data-tip="Recovery link" title="Recovery link" aria-label="Recovery link"><LegacyIcon name="history" class="h-4 w-4" /></a>
																	{/if}
																	<form class="inline-flex" onsubmit={async (event) => { event.preventDefault(); await removeAttendee(attendee.attendeeId, attendee.fullName); }}>
																		<button type="submit" class="join-item btn btn-sm btn-square btn-error tooltip tooltip-left" data-tip="Remove attendee" title="Remove attendee" aria-label="Remove attendee" disabled={attendeeActionPending !== ''}><LegacyIcon name="trash" class="h-4 w-4" /></button>
																	</form>
																</div>
															</div>
														</div>
														<div class="flex items-center justify-between gap-3">
															<form class="inline-flex">
																<label class="label cursor-pointer justify-start gap-2 p-0">
																	<input class={attendee.isChair ? 'toggle toggle-sm toggle-primary' : 'toggle toggle-sm'} type="checkbox" checked={attendee.isChair} title="Chairperson" aria-label="Chairperson" disabled={attendeeActionPending !== ''} onchange={async (event) => { event.preventDefault(); event.stopPropagation(); await toggleAttendeeChair(attendee); }} />
																	<span class="text-xs leading-none">Chairperson</span>
																</label>
															</form>
															{#if attendee.isGuest}
																<form class="inline-flex">
																	<label class="label cursor-pointer justify-start gap-2 p-0">
																		<input class={attendee.quoted ? 'toggle toggle-sm toggle-info' : 'toggle toggle-sm'} type="checkbox" checked={attendee.quoted} title="FLINTA*" aria-label="FLINTA*" disabled={attendeeActionPending !== ''} onchange={async (event) => { event.preventDefault(); event.stopPropagation(); await toggleAttendeeQuoted(attendee); }} />
																		<span class="text-xs leading-none">FLINTA*</span>
																	</label>
																</form>
															{:else}
																<div class="inline-flex items-center text-xs leading-none text-base-content/50">FLINTA* unavailable</div>
															{/if}
														</div>
													</div>
												</li>
											{/each}
										</ul>
									{/if}
								</div>
							</section>
						</div>
						<div id="moderate-left-panel-settings" data-moderate-left-panel="settings" class={moderateLeftTab === 'settings' ? 'h-full min-h-0 overflow-y-auto pr-1' : 'hidden h-full min-h-0 overflow-y-auto pr-1'}>
							<section class="min-h-0 space-y-3" data-testid="manage-settings-card">
								<div class="flex flex-wrap items-end justify-between gap-2">
									<h2 class="text-lg font-semibold">Settings</h2>
									<p class="text-sm text-base-content/70">Configure meeting defaults and agenda-point overrides.</p>
								</div>
								<div id="moderate-settings-shell" class="rounded-box border border-base-300 bg-base-200/30 p-3" data-active-tab={moderateSettingsTab} data-settings-tabs-wired="true">
									<div role="tablist" class="tabs tabs-box tabs-sm w-full bg-base-100">
										<button type="button" class={moderateSettingsTab === 'meeting' ? 'tab tab-active flex-1 justify-center' : 'tab flex-1 justify-center'} data-moderate-settings-tab="meeting" onclick={() => (moderateSettingsTab = 'meeting')}>Meeting</button>
										<button type="button" class={moderateSettingsTab === 'agenda' ? 'tab tab-active flex-1 justify-center' : 'tab flex-1 justify-center'} data-moderate-settings-tab="agenda" onclick={() => (moderateSettingsTab = 'agenda')}>Agenda Point</button>
									</div>
									<div class="mt-3 min-h-0">
										<div data-moderate-settings-panel="meeting" class={moderateSettingsTab === 'meeting' ? '' : 'hidden'}>
											<div class="mb-2 text-xs font-semibold uppercase tracking-wide text-base-content/60">Meeting Defaults</div>
											<div id="meeting-settings-container">
												<div class="space-y-3">
													<div class="rounded-box border border-base-300 bg-base-100 p-3">
														<h3 class="mb-2 text-sm font-semibold">Quotation Settings</h3>
														<form class="grid gap-3 md:grid-cols-2">
															<div class="space-y-1">
																<label for="gender_quotation_enabled">FLINTA* quotation</label>
																<select class="select select-bordered select-sm" id="gender_quotation_enabled" name="gender_quotation_enabled" disabled={settingsActionPending !== ''} onchange={(event) => { if (moderationState.data?.settings) { moderationState.data.settings.genderQuotationEnabled = (event.currentTarget as HTMLSelectElement).value === 'true'; } void updateMeetingQuotation(); }}>
																	{#if moderationState.data.settings?.genderQuotationEnabled ?? true}
																		<option selected value="true">Enabled</option>
																		<option value="false">Disabled</option>
																	{:else}
																		<option value="true">Enabled</option>
																		<option selected value="false">Disabled</option>
																	{/if}
																</select>
															</div>
															<div class="space-y-1">
																<label for="first_speaker_quotation_enabled">First-speaker bonus</label>
																<select class="select select-bordered select-sm" id="first_speaker_quotation_enabled" name="first_speaker_quotation_enabled" disabled={settingsActionPending !== ''} onchange={(event) => { if (moderationState.data?.settings) { moderationState.data.settings.firstSpeakerQuotationEnabled = (event.currentTarget as HTMLSelectElement).value === 'true'; } void updateMeetingQuotation(); }}>
																	{#if moderationState.data.settings?.firstSpeakerQuotationEnabled ?? true}
																		<option selected value="true">Enabled</option>
																		<option value="false">Disabled</option>
																	{:else}
																		<option value="true">Enabled</option>
																		<option selected value="false">Disabled</option>
																	{/if}
																</select>
															</div>
														</form>
													</div>
													<div class="rounded-box border border-base-300 bg-base-100 p-3">
														<h3 class="mb-2 text-sm font-semibold">Moderator</h3>
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
											<div class="mb-2 text-xs font-semibold uppercase tracking-wide text-base-content/60">Agenda Point Overrides</div>
											<div id="moderate-speaker-settings-container">
												<p class="text-sm text-base-content/70">No active agenda point. Activate one above to configure agenda-point settings.</p>
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
								<h2 class="text-lg font-semibold">Speakers Queue</h2>
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
									<p class="text-sm text-base-content/70">No active agenda point.</p>
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
														<LegacyIcon name="arrow-forward" class="ui-icon--left" />Start Next Speaker
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
											<LegacyIcon name="history" class="ui-icon--left" />Scroll To Active Position
										</button>
									</div>
									<ul class="list rounded-box border border-base-300 bg-base-100 mt-2 flex-1 overflow-y-auto pr-1 live-speaker-list" data-initial-scroll-top="0" data-manage-speakers-viewport data-testid="manage-speakers-viewport">
										{#each speakerState.data.speakers as speaker, i}
											{@const prevSpeaker = speakerState.data.speakers[i - 1]}
											{#if speaker.state !== 'DONE' && prevSpeaker?.state === 'DONE'}
												<li class="list-row py-0">
													<div class="divider my-0 text-xs text-base-content/40 col-span-full">Upcoming</div>
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
																{#if speaker.speakerType === 'ropm'}
																	<span class="tooltip tooltip-right" data-tip="Point of order">
																		<span class="badge badge-warning badge-sm" data-testid="live-speaker-ropm-badge"><LegacyIcon name="scale" class="h-3.5 w-3.5" /></span>
																	</span>
																{/if}
																{#if speaker.quoted}
																	<span class="tooltip tooltip-right" data-tip="FLINTA*">
																		<span class="badge badge-info badge-sm" data-testid="live-speaker-quoted-badge"><LegacyIcon name="transgender" class="h-3.5 w-3.5" /></span>
																	</span>
																{/if}
																{#if speaker.firstSpeaker}
																	<span class="tooltip tooltip-right" data-tip="First Time">
																		<span class="badge badge-success badge-sm" data-testid="live-speaker-first-badge"><LegacyIcon name="person-raised" class="h-3.5 w-3.5" /></span>
																	</span>
																{/if}
																{#if speaker.priority}
																	<span class="tooltip tooltip-right" data-tip="Priority">
																		<span class="badge badge-warning badge-sm badge-outline" data-testid="live-speaker-priority-icon-badge"><LegacyIcon name="star" class="h-3.5 w-3.5" /></span>
																	</span>
																	<span class="badge badge-warning badge-sm" data-testid="live-speaker-priority-label-badge">Priority</span>
																{/if}
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
													End Speech
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
													Start Next
												</button>
											{/if}
										</div>
									</div>
									<p class="text-sm text-base-content/70">No speakers are queued right now.</p>
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
								<h2 class="text-lg font-semibold">Add Speaker</h2>
								<div class="join min-w-0 w-full max-w-full sm:w-auto sm:min-w-[24rem]">
									<input
										class="input input-bordered input-sm join-item min-w-0 flex-1"
										type="text"
										id="speaker-add-search-input"
										name="q"
										placeholder="Search name or number"
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
													{#if attendee.isGuest} • Guest{/if}
													{#if attendee.quoted} • Quoted{/if}
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
										<p class="text-sm text-base-content/70">No matching attendees found.</p>
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
