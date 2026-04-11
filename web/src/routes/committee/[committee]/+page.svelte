<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSwitch from '$lib/components/ui/AppSwitch.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import PaginationNav from '$lib/components/ui/PaginationNav.svelte';
	import MeetingWizard from './MeetingWizard.svelte';
	import MembersPanel from './MembersPanel.svelte';
	import { session } from '$lib/stores/session.svelte.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import { committeeClient, moderationClient } from '$lib/api/index.js';
	import type { CommitteeOverview } from '$lib/gen/conference/committees/v1/committees_pb.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import * as m from '$lib/paraglide/messages';

	const slug = $derived(page.params.committee);

	let committeeState = $state(createRemoteState<CommitteeOverview>());
	let actionError = $state('');
	let localActiveMeetingId = $state<string | null>(null);
	let signupTogglePendingMeetingId = $state('');
	let wizardRef = $state<ReturnType<typeof MeetingWizard> | null>(null);

	$effect(() => {
		pageActions.set([], { backHref: '/home' });
		return () => { pageActions.clear(); };
	});

	$effect(() => {
		if (!session.loaded) return;
		if (!session.authenticated) {
			goto('/login');
			return;
		}
		loadCommittee();
	});

	$effect(() => {
		if (committeeState.data) {
			const active = committeeState.data.meetings.find((m) => m.canViewLive);
			localActiveMeetingId = active?.meeting?.meetingId ?? null;
		}
	});

	async function loadCommittee() {
		committeeState.loading = true;
		committeeState.error = '';
		try {
			const res = await committeeClient.getCommitteeOverview({ committeeSlug: slug });
			committeeState.data = res.overview ?? null;
		} catch (err) {
			committeeState.error = getDisplayError(err, `Committee "${slug}" not found or access denied.`);
		} finally {
			committeeState.loading = false;
		}
	}

	const canManage = $derived(committeeState.data?.committee?.isChairperson || committeeState.data?.committee?.isAdmin);
	const activeMeetingItem = $derived(committeeState.data?.meetings.find((m) => m.canViewLive) ?? null);

	async function toggleActive(e: Event, meetingId: string) {
		e.preventDefault();
		actionError = '';
		const wasActive = localActiveMeetingId === meetingId;
		localActiveMeetingId = wasActive ? null : meetingId;
		try {
			await committeeClient.toggleMeetingActive({
				committeeSlug: slug,
				meetingId
			});
			await loadCommittee();
		} catch (err) {
			localActiveMeetingId = wasActive ? meetingId : null;
			actionError = getDisplayError(err, 'Failed to toggle active meeting.');
		}
	}

	async function deleteMeeting(meetingId: string) {
		if (!window.confirm(m.committee_delete_confirm())) return;
		actionError = '';
		try {
			await committeeClient.deleteMeeting({
				committeeSlug: slug,
				meetingId
			});
			await loadCommittee();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to delete meeting.');
		}
	}

	async function toggleSignupOpen(meetingId: string, currentOpen: boolean) {
		if (signupTogglePendingMeetingId) return;
		signupTogglePendingMeetingId = meetingId;
		actionError = '';
		try {
			const moderation = await moderationClient.getModerationView({ committeeSlug: slug, meetingId });
			const expectedVersion = moderation.view?.version ?? 0n;
			await moderationClient.toggleSignupOpen({
				committeeSlug: slug,
				meetingId,
				desiredOpen: !currentOpen,
				expectedVersion
			});
			await loadCommittee();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to toggle meeting signup.');
		} finally {
			signupTogglePendingMeetingId = '';
		}
	}
</script>

{#if committeeState.loading}
	<AppSpinner label="Loading committee overview" />
{:else if committeeState.error}
	<AppAlert message={committeeState.error} />
{:else if committeeState.data}
	<p>{m.committee_welcome()}</p>

	{#if actionError}
		<AppAlert message={actionError} />
	{/if}
	{#if canManage}
		<div id="meeting-list-container" class="space-y-4">

			<section class="min-w-0 rounded-box border border-base-300 bg-base-200 p-4">
				<div class="mb-3 flex items-center justify-between gap-2">
					<h3 class="text-lg font-semibold">{m.committee_meetings_heading()}</h3>
					<button type="button" class="btn btn-sm btn-primary" data-testid="committee-create-form" onclick={() => wizardRef?.open()}>{m.committee_create_button()}</button>
				</div>
				{#if committeeState.data.meetings.length === 0}
					<p class="text-base-content/70">{m.committee_empty_state()}</p>
				{:else}
					<ul class="list rounded-box border border-base-300 bg-base-100" data-testid="committee-meeting-list">
						{#each committeeState.data.meetings as item}
							<li class="list-row items-center gap-3" data-testid="committee-meeting-row">
								<div class="list-col-grow">
									<div class="truncate font-medium">{item.meeting?.name}</div>
									{#if item.meeting?.description}
										<div class="truncate text-sm text-base-content/70">{item.meeting.description}</div>
									{:else}
										<div class="truncate text-sm text-base-content/70">{m.committee_no_description()}</div>
									{/if}
								</div>
								<div class="flex flex-col items-start justify-center gap-1 pr-2">
									<AppSwitch
										checked={localActiveMeetingId === item.meeting?.meetingId}
										id={"meeting-active-toggle-" + (item.meeting?.meetingId ?? '')}
										label={m.committee_col_active()}
										onCheckedChange={() => toggleActive(new Event('change'), item.meeting?.meetingId ?? '')}
									/>
									<AppSwitch
										checked={item.meeting?.signupOpen ?? false}
										id={"meeting-signup-toggle-" + (item.meeting?.meetingId ?? '')}
										disabled={signupTogglePendingMeetingId === (item.meeting?.meetingId ?? '')}
										label={m.committee_col_signup()}
										onCheckedChange={() => toggleSignupOpen(item.meeting?.meetingId ?? '', item.meeting?.signupOpen ?? false)}
									/>
								</div>
								<div class="flex items-center justify-end gap-1">
									<a
										href={"/committee/" + slug + "/meeting/" + (item.meeting?.meetingId ?? '')}
										class="btn btn-sm btn-square"
										title="View"
										aria-label="View"
									>
										<LegacyIcon name="eye" class="h-4 w-4" />
									</a>
									<a
										href={"/committee/" + slug + "/meeting/" + (item.meeting?.meetingId ?? '') + "/moderate"}
										class="btn btn-sm btn-square"
										title="Manage"
										aria-label="Manage"
									>
										<LegacyIcon name="settings" class="h-4 w-4" />
									</a>
									<form
										class="inline"
										onsubmit={(event) => {
											event.preventDefault();
											deleteMeeting(item.meeting?.meetingId ?? '');
										}}
									>
										<button
											class="btn btn-sm btn-square btn-error"
											type="submit"
											title="Delete"
											aria-label="Delete"
										>
											<LegacyIcon name="trash" class="h-4 w-4" />
										</button>
									</form>
								</div>
							</li>
						{/each}
					</ul>
					<div class="mt-3 flex justify-center">
						<PaginationNav />
					</div>
				{/if}
			</section>
		</div>

			<MembersPanel slug={slug ?? ''} />
	{:else}
		<section class="card border border-base-300 bg-base-200 p-4 mt-4" data-testid="committee-active-meeting-card">
			<h3 class="text-lg font-semibold">{m.committee_active_meeting_heading()}</h3>
			{#if activeMeetingItem?.meeting}
				<div class="mt-2">
					<p class="font-medium" data-testid="committee-active-meeting-name">{activeMeetingItem.meeting.name}</p>
					{#if activeMeetingItem.meeting.description}
						<p class="text-sm text-base-content/70">{activeMeetingItem.meeting.description}</p>
					{/if}
				</div>
				<a
					class="btn btn-primary btn-sm mt-3"
					data-testid="committee-join-active-meeting"
					href={"/committee/" + slug + "/meeting/" + activeMeetingItem.meeting.meetingId + "/join"}
				>
					{m.committee_join_active_meeting()}
				</a>
			{:else}
				<p class="text-base-content/70">{m.committee_no_active_meeting()}</p>
			{/if}
		</section>
	{/if}
{/if}

{#if canManage}
	<MeetingWizard bind:this={wizardRef} slug={slug ?? ''} onCreated={loadCommittee} />
{/if}
