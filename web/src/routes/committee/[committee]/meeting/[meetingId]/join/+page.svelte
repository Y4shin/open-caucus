<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { attendeeClient, meetingClient } from '$lib/api/index.js';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import type { JoinMeetingView } from '$lib/gen/conference/meetings/v1/meetings_pb.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { onDestroy } from 'svelte';
	import * as m from '$lib/paraglide/messages';

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);
	const liveHref = $derived(`/committee/${slug}/meeting/${meetingId}`);
	const joinHref = $derived(`/committee/${slug}/meeting/${meetingId}/join`);
	const attendeeLoginHref = $derived(`/committee/${slug}/meeting/${meetingId}/attendee-login`);
	const prefilledMeetingSecret = $derived(page.url.searchParams.get('meeting_secret') ?? '');

	let joinState = $state(createRemoteState<JoinMeetingView>());
	let actionError = $state('');
	let fullName = $state('');
	let meetingSecret = $state('');
	let genderQuoted = $state(false);
	let submitting = $state(false);

	onDestroy(() => {
		pageActions.clear();
	});

	$effect(() => {
		if (!session.loaded) return;
		loadJoinView();
	});

	$effect(() => {
		if (!joinState.data) return;
		pageActions.set([], {
			title: joinState.data.meetingName,
			subtitle: joinState.data.committeeName
		});
	});

	async function loadJoinView() {
		joinState.loading = true;
		joinState.error = '';
		try {
			const res = await meetingClient.getJoinMeeting({ committeeSlug: slug, meetingId });
			joinState.data = res.meeting ?? null;
		} catch (err) {
			joinState.error = getDisplayError(err, 'Failed to load the join flow.');
		} finally {
			joinState.loading = false;
		}
	}

	async function handleSelfSignup() {
		actionError = '';
		submitting = true;
		try {
			await attendeeClient.selfSignup({ committeeSlug: slug, meetingId });
			goto(liveHref);
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to join the meeting.');
			await loadJoinView();
		} finally {
			submitting = false;
		}
	}

	async function handleGuestSignup(event: SubmitEvent) {
		event.preventDefault();
		actionError = '';
		submitting = true;
		try {
			const joinRes = await attendeeClient.guestJoin({
				committeeSlug: slug,
				meetingId,
				fullName,
				meetingSecret: prefilledMeetingSecret || meetingSecret,
				genderQuoted
			});
			await attendeeClient.attendeeLogin({
				committeeSlug: slug,
				meetingId,
				attendeeSecret: joinRes.attendeeSecret
			});
			await session.load();
			goto(liveHref);
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to join the meeting as a guest.');
			await loadJoinView();
		} finally {
			submitting = false;
		}
	}
</script>

<div class="space-y-6">
	{#if joinState.loading}
		<AppSpinner label="Loading meeting join flow" />
	{:else if joinState.error}
		<AppAlert message={joinState.error} />
	{:else if joinState.data}
		<div class="space-y-2">
			<h1 class="text-3xl font-bold">{joinState.data.meetingName}</h1>
			<p class="text-base-content/70">{joinState.data.committeeName}</p>
		</div>

		<div id="app-notification-target">
			{#if actionError}
				<AppAlert message={actionError} />
			{/if}
		</div>

		<div class="flex h-full min-h-0 w-full flex-1 justify-center">
			<div class="h-full min-h-0 w-[90%] md:w-2/3">
				<div class="space-y-4">
				{#if joinState.data.capabilities?.alreadyJoined}
					<div class="space-y-4">
						<p>
							{m.meeting_join_already_signed_up()}
							<span class="font-medium">{joinState.data.currentAttendee?.fullName}</span>.
						</p>
						<form
							action={joinHref}
							method="POST"
							onsubmit={(event) => {
								event.preventDefault();
								goto(liveHref);
							}}
						>
							<button class="btn btn-sm" type="submit">{m.meeting_join_enter_button()}</button>
						</form>
					</div>
				{:else if joinState.data.capabilities?.canSelfSignup}
					<div class="space-y-4">
						<h3>{m.meeting_join_signup_heading()}</h3>
						<p>{m.meeting_join_member_logged_in()}</p>
						<form
							action={joinHref}
							method="POST"
							onsubmit={(event) => {
								event.preventDefault();
								void handleSelfSignup();
							}}
						>
							<button class="btn btn-sm" type="submit" disabled={submitting}
								>{#if submitting}<span class="loading loading-spinner loading-xs"></span>{/if}{m.meeting_join_signup_button()}</button
							>
						</form>
					</div>
				{:else if joinState.data.capabilities?.canGuestJoin}
					<div class="space-y-4">
					<h3>{m.meeting_join_guest_heading()}</h3>
					<form class="space-y-4" onsubmit={handleGuestSignup}>
						<div>
							<label for="full_name">{m.meeting_join_name_label()}</label>
							<input id="full_name" class="input input-bordered input-sm" name="full_name" bind:value={fullName} required />
						</div>

						{#if prefilledMeetingSecret}
							<div class="rounded-box border border-base-300 bg-base-200/60 px-4 py-3 text-sm text-base-content/70">
								The meeting secret was provided by your join link.
							</div>
						{:else}
							<div>
								<label for="meeting_secret">{m.meeting_join_meeting_secret_label()}</label>
								<input
									id="meeting_secret"
									class="input input-bordered input-sm"
									type="password"
									name="meeting_secret"
									bind:value={meetingSecret}
									required
								/>
							</div>
						{/if}

						<div>
							<label for="guest_gender_quoted">{m.meeting_join_quoted_label()}</label>
							<input
								id="guest_gender_quoted"
								class="checkbox checkbox-sm"
								type="checkbox"
								name="gender_quoted"
								bind:checked={genderQuoted}
							/>
						</div>

						<button class="btn btn-sm" type="submit" disabled={submitting}>
							{#if submitting}
								<span class="loading loading-spinner loading-xs"></span>
							{/if}
							{m.meeting_join_guest_button()}
						</button>
					</form>
					</div>
				{:else}
					<p>{m.meeting_join_signup_closed()}</p>
				{/if}
				</div>
				<div class="pt-2">
					<p><a href={attendeeLoginHref}>Attendee Login</a></p>
				</div>
			</div>
		</div>
	{/if}
</div>
