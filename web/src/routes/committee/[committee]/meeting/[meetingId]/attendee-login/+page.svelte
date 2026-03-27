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
	const joinHref = $derived(`/committee/${slug}/meeting/${meetingId}/join`);

	let meetingState = $state(createRemoteState<JoinMeetingView>());
	let attendeeSecret = $state('');
	let submitting = $state(false);
	let actionError = $state('');

	$effect(() => {
		if (!session.loaded) return;
		loadMeeting();
	});

	$effect(() => {
		if (!meetingState.data || !session.authenticated) return;
		if (meetingState.data.capabilities?.alreadyJoined) {
			goto(liveHref);
		}
	});

	async function loadMeeting() {
		meetingState.loading = true;
		meetingState.error = '';
		try {
			const res = await meetingClient.getJoinMeeting({ committeeSlug: slug, meetingId });
			meetingState.data = res.meeting ?? null;
		} catch (err) {
			meetingState.error = getDisplayError(err, 'Failed to load the attendee login flow.');
		} finally {
			meetingState.loading = false;
		}
	}

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		actionError = '';
		submitting = true;
		try {
			await attendeeClient.attendeeLogin({
				committeeSlug: slug,
				meetingId,
				attendeeSecret
			});
			await session.load();
			goto(liveHref);
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to log in with that access code.');
		} finally {
			submitting = false;
		}
	}
</script>

<div class="flex justify-center">
	<div class="w-full max-w-xl space-y-6">
		{#if meetingState.loading}
			<AppSpinner label="Loading attendee login" />
		{:else if meetingState.error}
			<AppAlert message={meetingState.error} />
		{:else if meetingState.data}
			<div class="space-y-2 text-center">
				<h1 class="text-3xl font-bold">{meetingState.data.meetingName}</h1>
				<p class="text-base-content/70">
					Enter your attendee access code to join {meetingState.data.committeeName}.
				</p>
			</div>

			<AppCard title="Attendee Login">
				{#if actionError}
					<div class="mb-4">
						<AppAlert message={actionError} />
					</div>
				{/if}

				<form class="space-y-4" onsubmit={handleSubmit}>
					<label class="form-control">
						<div class="label"><span class="label-text">Access Code</span></div>
						<input
							class="input input-bordered"
							name="secret"
							bind:value={attendeeSecret}
							autocomplete="one-time-code"
							required
						/>
					</label>

					<div class="flex flex-wrap gap-2">
						<button class="btn btn-primary" type="submit" disabled={submitting}>
							{#if submitting}
								<span class="loading loading-spinner loading-xs"></span>
							{/if}
							Enter Meeting
						</button>
						<a class="btn btn-outline" href={joinHref}>Back to Join</a>
					</div>
				</form>
			</AppCard>
		{/if}
	</div>
</div>
