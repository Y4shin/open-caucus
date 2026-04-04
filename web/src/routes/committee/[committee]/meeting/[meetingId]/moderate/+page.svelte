<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onDestroy } from 'svelte';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import AgendaSection from './AgendaSection.svelte';
	import VotesPanelSection from './VotesPanelSection.svelte';
	import SpeakersSection from './SpeakersSection.svelte';
	import AttendeeRow from '$lib/components/ui/AttendeeRow.svelte';
	import { agendaClient, attendeeClient, meetingClient, moderationClient, speakerClient, voteClient } from '$lib/api/index.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import { session } from '$lib/stores/session.svelte.js';
	import type { AgendaPointRecord } from '$lib/gen/conference/agenda/v1/agenda_pb.js';
	import type { AttendeeRecord } from '$lib/gen/conference/attendees/v1/attendees_pb.js';
	import { MeetingEventKind } from '$lib/gen/conference/meetings/v1/meetings_pb.js';
	import type { ModerationView } from '$lib/gen/conference/moderation/v1/moderation_pb.js';
	import type { SpeakerQueueView } from '$lib/gen/conference/speakers/v1/speakers_pb.js';
	import type { VotesPanelView } from '$lib/gen/conference/votes/v1/votes_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
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
	let settingsActionPending = $state('');
	let agendaAnyDialogOpen = $state(false);
	let moderateSettingsTab = $state<'meeting' | 'agenda'>('meeting');
	let moderateLeftTab = $state<'agenda' | 'tools' | 'attendees' | 'settings'>('agenda');

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

	function selectModerateLeftTab(tab: 'agenda' | 'tools' | 'attendees' | 'settings') {
		moderateLeftTab = tab;
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

	function settingsAttendees() {
		return attendeeRows();
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
													<VotesPanelSection
														votesPanel={votesState.data}
														votesLoading={votesState.loading}
														votesError={votesState.error}
														attendees={attendeeState.data ?? []}
														{slug}
														{meetingId}
														onError={(msg) => (actionError = msg)}
														onNotice={(msg) => (actionNotice = msg)}
													/>
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
				<SpeakersSection
					speakers={speakerState.data?.speakers}
					attendees={attendeeState.data ?? []}
					hasActivePoint={!!moderationState.data?.activeAgendaPoint}
					{slug}
					{meetingId}
					onError={(msg) => (actionError = msg)}
				/>
			</div>
		</div>

	{/if}
</div>
