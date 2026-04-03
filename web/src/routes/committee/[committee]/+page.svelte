<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import { session } from '$lib/stores/session.svelte.js';
	import { committeeClient, moderationClient } from '$lib/api/index.js';
	import type { CommitteeOverview } from '$lib/gen/conference/committees/v1/committees_pb.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import * as m from '$lib/paraglide/messages';

	const slug = $derived(page.params.committee);

	let committeeState = $state(createRemoteState<CommitteeOverview>());
	let createName = $state('');
	let createDescription = $state('');
	let createSignupOpen = $state(false);
	let createPending = $state(false);
	let createError = $state('');
	let actionError = $state('');
	let localActiveMeetingId = $state<string | null>(null);
	let signupTogglePendingMeetingId = $state('');

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

	async function createMeeting(e: Event) {
		e.preventDefault();
		if (!createName.trim() || createPending) return;
		createPending = true;
		createError = '';
		try {
			await committeeClient.createMeeting({
				committeeSlug: slug,
				name: createName.trim(),
				description: createDescription.trim()
			});
			createName = '';
			createDescription = '';
			createSignupOpen = false;
			await loadCommittee();
		} catch (err) {
			createError = getDisplayError(err, 'Failed to create meeting.');
		} finally {
			createPending = false;
		}
	}

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
	{#if createError}
		<AppAlert message={createError} />
	{/if}

	{#if canManage}
		<div id="meeting-list-container" class="grid items-start gap-4 md:grid-cols-[20rem_minmax(0,1fr)]">
			<form
				class="fieldset w-full rounded-box border border-base-300 bg-base-200 p-4"
				data-testid="committee-create-form"
				onsubmit={createMeeting}
			>
				<legend class="fieldset-legend">{m.committee_create_meeting_heading()}</legend>
				<label class="label" for="name">{m.committee_name_label()}</label>
				<input
					class="input input-bordered input-sm w-full"
					type="text"
					id="name"
					name="name"
					placeholder={m.committee_name_label()}
					bind:value={createName}
					required
				/>
				<label class="label" for="description">{m.committee_description_label()}</label>
				<input
					class="input input-bordered input-sm w-full"
					id="description"
					name="description"
					placeholder={m.committee_description_label()}
					bind:value={createDescription}
				/>
				<label class="label cursor-pointer justify-start gap-3">
					<input
						type="checkbox"
						id="signup_open"
						name="signup_open"
						class="toggle toggle-primary toggle-sm"
						bind:checked={createSignupOpen}
					/>
					<span>{m.committee_signup_label()}</span>
				</label>
				<button class="btn btn-primary btn-sm" type="submit" disabled={createPending}>{m.committee_create_button()}</button>
			</form>

			<section class="min-w-0 rounded-box border border-base-300 bg-base-200 p-4">
				<h3 class="mb-3 text-lg font-semibold">{m.committee_meetings_heading()}</h3>
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
									<label class="text-xs font-medium text-base-content/70">
										<input
											class="toggle toggle-primary toggle-sm"
											type="checkbox"
											id={"meeting-active-toggle-" + (item.meeting?.meetingId ?? '')}
											checked={localActiveMeetingId === item.meeting?.meetingId}
											data-testid="committee-toggle-active"
											onchange={(e) => toggleActive(e, item.meeting?.meetingId ?? '')}
										/>
										{m.committee_col_active()}
									</label>
									<label class="text-xs font-medium text-base-content/70">
										<input
											class="toggle toggle-primary toggle-sm"
											type="checkbox"
											id={"meeting-signup-toggle-" + (item.meeting?.meetingId ?? '')}
											checked={item.meeting?.signupOpen ?? false}
											data-testid="committee-toggle-signup-open"
											disabled={signupTogglePendingMeetingId === (item.meeting?.meetingId ?? '')}
											onchange={() => toggleSignupOpen(item.meeting?.meetingId ?? '', item.meeting?.signupOpen ?? false)}
										/>
										{m.committee_col_signup()}
									</label>
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
						<nav class="pagination-nav join">
							<button type="button" disabled class="ui-icon-label btn btn-sm">
								<LegacyIcon name="left" class="ui-icon--left" />
								<span class="ui-icon-text">{m.pagination_previous()}</span>
							</button>
							<button class="btn btn-sm" type="button" disabled>1</button>
							<button type="button" disabled class="ui-icon-label btn btn-sm">
								<span class="ui-icon-text">{m.pagination_next()}</span>
								<LegacyIcon name="right" class="ui-icon--right" />
							</button>
						</nav>
					</div>
				{/if}
			</section>
		</div>
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
