<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onDestroy, onMount } from 'svelte';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { agendaClient, meetingClient, moderationClient, speakerClient, voteClient } from '$lib/api/index.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import type { AgendaPointRecord } from '$lib/gen/conference/agenda/v1/agenda_pb.js';
	import type { LiveMeetingView } from '$lib/gen/conference/meetings/v1/meetings_pb.js';
	import { MeetingEventKind } from '$lib/gen/conference/meetings/v1/meetings_pb.js';
	import type { SpeakerQueueView } from '$lib/gen/conference/speakers/v1/speakers_pb.js';
	import type { LiveVotePanelView } from '$lib/gen/conference/votes/v1/votes_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { saveReceipt } from '$lib/utils/receipts.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';

	const liveVotesLegacyScript = String.raw`
		(function() {
			if (!window.voteReceiptVault) {
				window.voteReceiptVault = {
					open: function() {
						return new Promise(function(resolve, reject) {
							var req = indexedDB.open("conference_tool_receipts", 1);
							req.onupgradeneeded = function(event) {
								var db = event.target.result;
								if (!db.objectStoreNames.contains("receipts")) {
									var store = db.createObjectStore("receipts", { keyPath: "id" });
									store.createIndex("vote_id", "vote_id", { unique: false });
									store.createIndex("kind", "kind", { unique: false });
								}
							};
							req.onsuccess = function() { resolve(req.result); };
							req.onerror = function() { reject(req.error); };
						});
					},
					put: function(payload) {
						if (!payload || !payload.id) { return Promise.resolve(); }
						return this.open().then(function(db) {
							return new Promise(function(resolve, reject) {
								var tx = db.transaction("receipts", "readwrite");
								tx.objectStore("receipts").put(payload);
								tx.oncomplete = function() { resolve(); };
								tx.onerror = function() { reject(tx.error); };
							});
						});
					}
				};
			}

			function randomToken() {
				if (window.crypto && window.crypto.randomUUID) {
					return window.crypto.randomUUID().replace(/-/g, "");
				}
				return String(Date.now()) + String(Math.random()).replace("0.", "");
			}

			function encodePayload(value) {
				try {
					return btoa(unescape(encodeURIComponent(value)));
				} catch (error) {
					return "";
				}
			}

			function decodeB64JSON(raw) {
				if (!raw) { return null; }
				try {
					return JSON.parse(decodeURIComponent(escape(atob(raw))));
				} catch (error) {
					return null;
				}
			}

			var submissionStorageKey = "conference_tool_live_vote_submissions";
			var countdownIntervalID = 0;
			var refreshTimeoutID = 0;

			function readSubmissionState() {
				try {
					var raw = window.sessionStorage.getItem(submissionStorageKey);
					if (!raw) { return {}; }
					var parsed = JSON.parse(raw);
					if (parsed && typeof parsed === "object") {
						return parsed;
					}
				} catch (error) {
				}
				return {};
			}

			function writeSubmissionState(state) {
				try {
					window.sessionStorage.setItem(submissionStorageKey, JSON.stringify(state || {}));
				} catch (error) {
				}
			}

			function pruneSubmissionState() {
				var state = readSubmissionState();
				var now = Date.now();
				var maxAgeMs = 24 * 60 * 60 * 1000;
				var changed = false;
				for (var key in state) {
					if (!Object.prototype.hasOwnProperty.call(state, key)) { continue; }
					var ts = Number(state[key]);
					if (!Number.isFinite(ts) || now-ts > maxAgeMs) {
						delete state[key];
						changed = true;
					}
				}
				if (changed) {
					writeSubmissionState(state);
				}
				return state;
			}

			function markVoteSubmitted(voteID) {
				var numeric = Number(voteID);
				if (!Number.isFinite(numeric) || numeric <= 0) { return; }
				var state = pruneSubmissionState();
				state[String(Math.floor(numeric))] = Date.now();
				writeSubmissionState(state);
			}

			function applySubmittedScreens() {
				var panel = document.getElementById("live-votes-panel");
				if (!panel) { return; }
				var state = pruneSubmissionState();
				var cards = panel.querySelectorAll("[data-vote-card]");
				for (var i = 0; i < cards.length; i++) {
					var card = cards[i];
					var voteID = card.getAttribute("data-vote-id") || "";
					var voteState = card.getAttribute("data-vote-state") || "";
					var submittedScreen = card.querySelector("[data-vote-submitted-screen]");
					var voteInputs = card.querySelector("[data-vote-inputs]");
					if (!submittedScreen || !voteInputs) {
						continue;
					}
					var showSubmitted = voteState === "open" && !!state[voteID];
					submittedScreen.classList.toggle("hidden", !showSubmitted);
					voteInputs.classList.toggle("hidden", showSubmitted);
				}
			}

			function applyResultsCountdown() {
				var panel = document.getElementById("live-votes-panel");
				if (!panel) { return; }
				var cards = panel.querySelectorAll("[data-vote-card]");
				var nowMs = Date.now();
				var soonestExpiryMs = 0;
				for (var i = 0; i < cards.length; i++) {
					var card = cards[i];
					var untilRaw = card.getAttribute("data-vote-results-until") || "";
					var untilSec = Number.parseInt(untilRaw, 10);
					if (!Number.isFinite(untilSec) || untilSec <= 0) {
						continue;
					}
					var untilMs = untilSec * 1000;
					var remainingSec = Math.max(0, Math.ceil((untilMs - nowMs) / 1000));
					var countdown = card.querySelector("[data-vote-results-countdown]");
					if (countdown) {
						countdown.textContent = String(remainingSec);
					}
					if (untilMs > nowMs && (soonestExpiryMs === 0 || untilMs < soonestExpiryMs)) {
						soonestExpiryMs = untilMs;
					}
				}
				if (refreshTimeoutID) {
					window.clearTimeout(refreshTimeoutID);
					refreshTimeoutID = 0;
				}
				if (soonestExpiryMs > 0) {
					var delay = Math.max(150, soonestExpiryMs - nowMs + 250);
					refreshTimeoutID = window.setTimeout(function() {
						var currentPanel = document.getElementById("live-votes-panel");
						if (currentPanel && window.htmx) {
							window.htmx.trigger(currentPanel, "reload");
						}
					}, delay);
				}
			}

			function applyLiveVoteUIState() {
				applySubmittedScreens();
				applyResultsCountdown();
			}

			function persistLatestReceipt() {
				var node = document.getElementById("live-vote-last-receipt");
				if (!node) { return; }
				var raw = node.getAttribute("data-receipt-b64") || "";
				if (!raw) { return; }
				var payload = decodeB64JSON(raw);
				if (!payload) { return; }
				window.voteReceiptVault.put(payload);
				markVoteSubmitted(payload.vote_id);
				node.setAttribute("data-receipt-b64", "");
			}

			if (!window.__voteBallotFormsWired) {
				document.addEventListener("submit", function(event) {
					var form = event.target;
					if (!(form instanceof HTMLFormElement)) { return; }
					if (!form.matches("[data-vote-open-form], [data-vote-secret-form]")) { return; }

					var receiptTokenInput = form.querySelector("input[name='receipt_token']");
					if (receiptTokenInput && !receiptTokenInput.value) {
						receiptTokenInput.value = randomToken();
					}

					if (form.matches("[data-vote-secret-form]")) {
						var nonceInput = form.querySelector("input[name='nonce']");
						var payloadInput = form.querySelector("input[name='encrypted_commitment_b64']");
						if (!nonceInput || !payloadInput) { return; }
						if (!nonceInput.value) {
							nonceInput.value = randomToken();
						}
						var selected = [];
						var options = form.querySelectorAll("input[name='option_id']");
						for (var i = 0; i < options.length; i++) {
							if (options[i].checked) {
								selected.push(options[i].value);
							}
						}
						var attendeeID = form.getAttribute("data-attendee-id") || "";
						payloadInput.value = encodePayload(attendeeID + ":" + selected.join(",") + ":" + nonceInput.value);
					}
				});
				document.addEventListener("htmx:afterSwap", function(event) {
					if (!event.target) { return; }
					if (event.target.id === "live-votes-panel" || (event.target.closest && event.target.closest("#live-votes-panel"))) {
						persistLatestReceipt();
						applyLiveVoteUIState();
					}
				});
				window.__voteBallotFormsWired = true;
			}

			persistLatestReceipt();
			applyLiveVoteUIState();
			if (!window.__voteResultsCountdownIntervalID) {
				window.__voteResultsCountdownIntervalID = window.setInterval(applyResultsCountdown, 1000);
			}
		})();`;

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);

	let liveState = $state(createRemoteState<LiveMeetingView>());
	let speakerState = $state(createRemoteState<SpeakerQueueView>());
	let voteState = $state(createRemoteState<LiveVotePanelView>());
	let agendaState = $state(createRemoteState<AgendaPointRecord[]>());
	let legacyLiveVotesPanelHTML = $state('');
	let actionError = $state('');
	let addingRegular = $state(false);
	let addingRopm = $state(false);
	let submittingVote = $state(false);
	let selectedOptionIds = $state<string[]>([]);
	let voteReceipt = $state('');
	let nowMs = $state(Date.now());
	let speakingSinceMs = $state<Record<string, number>>({});
	let canModerate = $state(false);

	onDestroy(() => {
		pageActions.clear();
	});

	onMount(() => {
		let cancelled = false;
		let refreshInterval = 0;
		let clockInterval = 0;
		let legacyVotesInterval = 0;

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
							void loadLegacyLiveVotesPanel(true);
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

		const refreshLegacyVotesPanel = () => {
			if (!document.getElementById('live-votes-panel-host')) return;
			void loadLegacyLiveVotesPanel(true);
		};

		clockInterval = window.setInterval(() => {
			nowMs = Date.now();
		}, 1000);
		refreshLegacyVotesPanel();
		legacyVotesInterval = window.setInterval(refreshLegacyVotesPanel, 1000);

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
				void loadLegacyLiveVotesPanel(true);
			}, 2000);
			void subscribeToMeetingEvents();
		})();

		return () => {
			cancelled = true;
			if (refreshInterval) window.clearInterval(refreshInterval);
			if (clockInterval) window.clearInterval(clockInterval);
			if (legacyVotesInterval) window.clearInterval(legacyVotesInterval);
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
			void loadLegacyLiveVotesPanel(true);
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
			void loadLegacyLiveVotesPanel(true);
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

	async function loadLegacyLiveVotesPanel(force = false) {
		if (legacyLiveVotesPanelHTML && !force) {
			return;
		}
		try {
			const response = await fetch(`/committee/${slug}/meeting/${meetingId}/votes/live/partial`, {
				headers: {
					'HX-Request': 'true'
				},
				credentials: 'same-origin'
			});
			if (!response.ok) {
				return;
			}
			const html = await response.text();
			legacyLiveVotesPanelHTML = html;
			requestAnimationFrame(() => {
				const host = document.getElementById('live-votes-panel-host');
				const htmx = (window as typeof window & { htmx?: { process?: (node: Element) => void } }).htmx;
				if (host) {
					host.innerHTML = html;
					if (typeof htmx?.process === 'function') {
						htmx.process(host);
					}
				}
			});
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

	function liveSpeakerSelfAddURL() {
		return `/committee/${slug}/meeting/${meetingId}/speaker/self-add`;
	}

	function liveSpeakerSelfYieldURL() {
		return `/committee/${slug}/meeting/${meetingId}/speaker/self-yield`;
	}

	function liveVotesRefreshURL() {
		return `/committee/${slug}/meeting/${meetingId}/votes/live/partial`;
	}

	function escapeHTML(value: string) {
		return value
			.replaceAll('&', '&amp;')
			.replaceAll('<', '&lt;')
			.replaceAll('>', '&gt;')
			.replaceAll('"', '&quot;')
			.replaceAll("'", '&#39;');
	}

	function liveVotesTemplateHTML() {
		const refreshURL = escapeHTML(liveVotesRefreshURL());
		const noVotesMarkup = `<div id="live-votes-panel" class="space-y-3" sse-swap="votes-updated" hx-get="${refreshURL}" hx-trigger="reload" hx-swap="outerHTML" hx-swap-oob="outerHTML"><div class="flex items-center justify-between gap-2"><h2 class="text-lg font-semibold">Votes</h2><button type="button" class="btn btn-xs btn-outline" hx-get="${refreshURL}" hx-target="#live-votes-panel" hx-swap="outerHTML">Refresh</button></div><p class="text-sm text-base-content/70">No open or recently closed votes right now.</p><div id="live-vote-last-receipt" class="hidden" data-receipt-b64=""></div></div>`;
		return `${noVotesMarkup}<script>${liveVotesLegacyScript}\n\t<\/script>`;
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
						<h2 class="text-lg font-semibold">Agenda</h2>
						{#if liveState.data.currentDocument}
							<button type="button" class="btn btn-sm btn-outline btn-square tooltip tooltip-left lg:hidden" data-live-dialog-open aria-controls="live-doc-modal" title="Open document" aria-label="Open document" data-tip="Open document">
								<LegacyIcon name="eye" class="live-agenda-dialog-icon" />
							</button>
						{/if}
					</div>
					<div id="live-agenda-main-stack" class="min-h-0 flex flex-1 flex-col">
						<div class="flex min-h-0 flex-1 overflow-hidden">
							<div id="live-agenda-panel-body" class="min-h-0 flex-1">
								<div class="flex h-full min-h-0 flex-col gap-3">
									<div class="live-agenda-preview-block lg:hidden">
										<div class="live-agenda-preview-row">
											<span class="live-agenda-preview-label">Current</span>
											<span class="live-agenda-preview-value">
												{#if currentAgendaPoint()}
													{currentAgendaPoint()?.title}
												{:else}
													No active agenda point.
												{/if}
											</span>
										</div>
										<div class="live-agenda-preview-row">
											<span class="live-agenda-preview-label">Next</span>
											<span class="live-agenda-preview-value">
												{#if nextAgendaPoint()}
													{nextAgendaPoint()?.title}
												{:else}
													None
												{/if}
											</span>
										</div>
									</div>
									<div class="hidden min-h-0 flex-1 flex-col lg:flex">
										{#if agendaRows().length === 0}
											<p class="live-empty-state text-base-content/70">No agenda points are available.</p>
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
								<button type="button" class="btn btn-sm btn-outline btn-square tooltip tooltip-left" data-testid="live-doc-open-desktop" data-live-dialog-open aria-controls="live-doc-modal" title="Open document" aria-label="Open document" data-tip="Open document">
									<LegacyIcon name="eye" />
								</button>
								<a href={liveState.data.currentDocument.downloadUrl} download data-testid="live-doc-download-desktop" class="btn btn-sm btn-outline btn-square tooltip tooltip-left" title="Download document" aria-label="Download document" data-tip="Download document">
									<LegacyIcon name="download" />
								</a>
							</div>
						{/if}
					</div>
				</div>
			</section>

			<section class="card min-h-0 border border-base-300 bg-base-100 shadow-sm">
				<div class="flex h-full min-h-0 flex-col p-4">
					<div class="mb-3 flex items-center justify-between gap-2">
						<h2 class="text-lg font-semibold">Speakers</h2>
						<button
							type="button"
							class="btn btn-sm btn-outline btn-square tooltip tooltip-left lg:hidden"
							data-live-dialog-open
							aria-controls="speakers-full-dialog"
							aria-label="Speakers"
							title="Speakers"
							data-tip="Speakers"
						>
							<LegacyIcon name="eye" class="live-speaker-history-icon" />
						</button>
					</div>
					<div id="live-speakers-panel-meta" class="mb-2"></div>
					<div class="live-speakers-sse min-h-0 flex-1">
						<div id="attendee-speakers-list" sse-swap="speakers-updated" class="flex h-full min-h-0 flex-col">
							<div class="contents">
								{#if liveState.data.activeAgendaPoint}
									<div class="flex h-full min-h-0 flex-1 flex-col gap-3">
										<div class="min-h-0 flex-1 overflow-y-auto overflow-x-hidden" data-testid="live-speakers-active-viewport">
											{#if actionError}
												<AppAlert message={actionError} />
											{/if}
											{#if activeSpeakers().length === 0}
												<p class="live-empty-state text-base-content/70">No speakers in the queue yet.</p>
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
																		{#if speakerHasBadges(speaker)}
																			<div class="flex shrink-0 flex-wrap items-center gap-1">
																				{#if speaker.speakerType === 'ropm'}
																					<span class="tooltip tooltip-right" data-tip="Point of order">
																						<span class="badge badge-warning badge-sm"><LegacyIcon name="scale" class="h-3.5 w-3.5" /></span>
																					</span>
																				{/if}
																				{#if speaker.quoted}
																					<span class="tooltip tooltip-right" data-tip="Quoted" data-testid="live-speaker-quoted-badge">
																						<span class="badge badge-info badge-sm"><LegacyIcon name="quoted" class="h-3.5 w-3.5" /></span>
																					</span>
																				{/if}
																				{#if speaker.firstSpeaker}
																					<span class="tooltip tooltip-right" data-tip="First speaker">
																						<span class="badge badge-success badge-sm" data-testid="live-speaker-first-badge"><LegacyIcon name="person-raised" class="h-3.5 w-3.5" /></span>
																					</span>
																				{/if}
																				{#if speaker.priority}
																					<span class="tooltip tooltip-right" data-tip="Priority">
																						<span class="badge badge-warning badge-sm badge-outline" data-testid="live-speaker-priority-icon-badge"><LegacyIcon name="star" class="h-3.5 w-3.5" /></span>
																					</span>
																					<span class="badge badge-warning badge-sm" data-testid="live-speaker-priority-label-badge">Priority</span>
																				{/if}
																				{#if speaker.mine}
																					<span class="badge badge-primary badge-sm">You</span>
																				{/if}
																			</div>
																		{/if}
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
															<span>Yield Speech</span>
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
														hx-post={liveSpeakerSelfAddURL()}
														hx-target="#attendee-speakers-list"
														hx-swap="innerHTML"
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
													<h3 class="text-lg font-semibold">Speakers List</h3>
													<button type="button" class="btn btn-sm btn-ghost shrink-0" data-live-dialog-close>Close</button>
												</div>
												{#if activeSpeakers().length === 0}
													<p class="live-empty-state text-base-content/70">No speakers in the queue yet.</p>
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
																				{#if speaker.speakerType === 'ropm'}
																					<span class="tooltip tooltip-right" data-tip="Point of order">
																						<span class="badge badge-warning badge-sm"><LegacyIcon name="scale" class="h-3.5 w-3.5" /></span>
																					</span>
																				{/if}
																				{#if speaker.quoted}
																					<span class="tooltip tooltip-right" data-tip="Quoted" data-testid="live-speaker-quoted-badge">
																						<span class="badge badge-info badge-sm"><LegacyIcon name="quoted" class="h-3.5 w-3.5" /></span>
																					</span>
																				{/if}
																				{#if speaker.firstSpeaker}
																					<span class="tooltip tooltip-right" data-tip="First speaker">
																						<span class="badge badge-success badge-sm" data-testid="live-speaker-first-badge"><LegacyIcon name="person-raised" class="h-3.5 w-3.5" /></span>
																					</span>
																				{/if}
																				{#if speaker.priority}
																					<span class="tooltip tooltip-right" data-tip="Priority">
																						<span class="badge badge-warning badge-sm badge-outline" data-testid="live-speaker-priority-icon-badge"><LegacyIcon name="star" class="h-3.5 w-3.5" /></span>
																					</span>
																					<span class="badge badge-warning badge-sm" data-testid="live-speaker-priority-label-badge">Priority</span>
																				{/if}
																				{#if speaker.mine}
																					<span class="badge badge-primary badge-sm">You</span>
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
																{/if}
															</li>
														{/each}
													</ul>
												{/if}
											</div>
										</dialog>
									</div>
								{:else}
									<p>No active agenda point.</p>
								{/if}
								<div hidden>
									<div id="live-agenda-main-stack" class="min-h-0 flex flex-1 flex-col" hx-swap-oob="outerHTML">
										<div class="flex min-h-0 flex-1 overflow-hidden">
											<div id="live-agenda-panel-body" class="min-h-0 flex-1">
												<div class="flex h-full min-h-0 flex-col gap-3">
													<div class="live-agenda-preview-block lg:hidden">
														<div class="live-agenda-preview-row">
															<span class="live-agenda-preview-label">Current</span>
															<span class="live-agenda-preview-value">
																{#if currentAgendaPoint()}
																	{currentAgendaPoint()?.title}
																{:else}
																	No active agenda point.
																{/if}
															</span>
														</div>
														<div class="live-agenda-preview-row">
															<span class="live-agenda-preview-label">Next</span>
															<span class="live-agenda-preview-value">
																{#if nextAgendaPoint()}
																	{nextAgendaPoint()?.title}
																{:else}
																	None
																{/if}
															</span>
														</div>
													</div>
													<div class="hidden min-h-0 flex-1 flex-col lg:flex">
														{#if agendaRows().length === 0}
															<p class="live-empty-state text-base-content/70">No agenda points are available.</p>
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
									</div>
								</div>
								<div id="live-speakers-panel-meta" hx-swap-oob="innerHTML" hidden></div>
								<template>{@html liveVotesTemplateHTML()}</template>
								<div id="live-doc-fab-oob" hx-swap-oob="innerHTML:#live-doc-fab-wrapper" hidden></div>
							</div>
						</div>
					</div>
				</div>
			</section>

			<section class="card min-h-0 border border-base-300 bg-base-100 shadow-sm lg:col-span-2">
				<div class="p-4">
					<div id="live-votes-panel-host">
						{#if legacyLiveVotesPanelHTML}
							{@html legacyLiveVotesPanelHTML}
						{:else}
							{@html liveVotesTemplateHTML()}
						{/if}
					</div>
				</div>
			</section>
		</div>
	{/if}
</div>
