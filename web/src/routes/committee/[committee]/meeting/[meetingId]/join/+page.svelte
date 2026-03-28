<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { attendeeClient, meetingClient } from '$lib/api/index.js';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import type { JoinMeetingView } from '$lib/gen/conference/meetings/v1/meetings_pb.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);
	const liveHref = $derived(`/committee/${slug}/meeting/${meetingId}`);
	const attendeeLoginHref = $derived(`/committee/${slug}/meeting/${meetingId}/attendee-login`);
	const prefilledMeetingSecret = $derived(page.url.searchParams.get('meeting_secret') ?? '');

	let joinState = $state(createRemoteState<JoinMeetingView>());
	let actionError = $state('');
	let fullName = $state('');
	let meetingSecret = $state('');
	let genderQuoted = $state(false);
	let submitting = $state(false);

	$effect(() => {
		if (!session.loaded) return;
		loadJoinView();
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
			<p class="text-base-content/70">Join {joinState.data.committeeName} and enter the live meeting.</p>
		</div>

		<div id="app-notification-target">
			{#if actionError}
				<AppAlert message={actionError} />
			{/if}
		</div>

		<div class="grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(18rem,1fr)]">
			<AppCard title="Meeting Entry">
				{#if joinState.data.capabilities?.alreadyJoined}
					<div class="space-y-4">
						<p class="text-base-content/80">
							You are already signed up as
							<span class="font-medium">{joinState.data.currentAttendee?.fullName}</span>.
						</p>
						{#if joinState.data.currentAttendee?.attendeeNumber}
							<p class="text-sm text-base-content/70">
								Attendee #{joinState.data.currentAttendee.attendeeNumber.toString()}
							</p>
						{/if}
						<div class="flex flex-wrap gap-2">
							<a class="btn btn-primary" href={liveHref}>Enter Meeting</a>
							<a class="btn btn-outline" href={attendeeLoginHref}>Use Access Code</a>
						</div>
					</div>
				{:else if joinState.data.capabilities?.canSelfSignup}
					<div class="space-y-4">
						<p class="text-base-content/80">
							Join with your current committee account and continue directly to the live meeting.
						</p>
						<button class="btn btn-primary" onclick={handleSelfSignup} disabled={submitting}>
							{#if submitting}
								<span class="loading loading-spinner loading-xs"></span>
							{/if}
							Sign Up
						</button>
					</div>
				{:else if joinState.data.capabilities?.canGuestJoin}
					<form class="space-y-4" onsubmit={handleGuestSignup}>
						<label class="form-control">
							<div class="label"><span class="label-text">Full Name</span></div>
							<input class="input input-bordered" name="full_name" bind:value={fullName} required />
						</label>

						{#if prefilledMeetingSecret}
							<div class="rounded-box border border-base-300 bg-base-200/60 px-4 py-3 text-sm text-base-content/70">
								The meeting secret was provided by your join link.
							</div>
						{:else}
							<label class="form-control">
								<div class="label"><span class="label-text">Meeting Secret</span></div>
								<input
									class="input input-bordered"
									type="password"
									name="meeting_secret"
									bind:value={meetingSecret}
									required
								/>
							</label>
						{/if}

						<label class="label cursor-pointer justify-start gap-3">
							<input class="checkbox" type="checkbox" bind:checked={genderQuoted} />
							<span class="label-text">Quoted / feminine forms apply to me</span>
						</label>

						<button class="btn btn-primary" type="submit" disabled={submitting}>
							{#if submitting}
								<span class="loading loading-spinner loading-xs"></span>
							{/if}
							Sign Up as Guest
						</button>
					</form>
				{:else}
					<div class="space-y-4">
						<p class="text-base-content/80">
							Meeting signup is currently closed for new attendees.
						</p>
						<a class="btn btn-outline" href={attendeeLoginHref}>I Already Have an Access Code</a>
					</div>
				{/if}
			</AppCard>

			<AppCard title="Existing Attendee?">
				<div class="space-y-3 text-sm text-base-content/75">
					<p>Use your attendee access code if you have already joined this meeting before.</p>
					<a class="btn btn-outline w-full" href={attendeeLoginHref}>Attendee Login</a>
					<a class="btn btn-ghost w-full" href={`/committee/${slug}`}>Back to Committee</a>
				</div>
			</AppCard>
		</div>
	{/if}
</div>
