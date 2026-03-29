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

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);
	const liveHref = $derived(`/committee/${slug}/meeting/${meetingId}`);
	const joinHref = $derived(`/committee/${slug}/meeting/${meetingId}/join`);
	const secretFromURL = $derived(page.url.searchParams.get('secret') ?? '');

	let meetingState = $state(createRemoteState<JoinMeetingView>());
	let attendeeSecret = $state('');
	let submitting = $state(false);
	let actionError = $state('');

	onDestroy(() => {
		pageActions.clear();
	});

	$effect(() => {
		if (!session.loaded) return;
		loadMeeting();
	});

	$effect(() => {
		if (!meetingState.data) return;
		pageActions.set([], {
			title: meetingState.data.meetingName,
			subtitle: meetingState.data.committeeName
		});
	});

	$effect(() => {
		if (!meetingState.data || !session.authenticated) return;
		if (meetingState.data.capabilities?.alreadyJoined) {
			goto(liveHref);
		}
	});

	$effect(() => {
		if (!meetingState.data || submitting || session.authenticated || !secretFromURL) return;
		attendeeSecret = secretFromURL;
		void loginWithSecret(secretFromURL, 'Failed to log in with that recovery link.');
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
		await loginWithSecret(attendeeSecret, 'Failed to log in with that access code.');
	}

	async function loginWithSecret(secret: string, fallbackMessage: string) {
		actionError = '';
		submitting = true;
		try {
			await attendeeClient.attendeeLogin({
				committeeSlug: slug,
				meetingId,
				attendeeSecret: secret
			});
			await session.load();
			await goto(liveHref);
		} catch (err) {
			actionError = getDisplayError(err, fallbackMessage);
		} finally {
			submitting = false;
		}
	}
</script>

<div class="flex h-full min-h-0 w-full flex-1 justify-center">
	<div class="h-full min-h-0 w-[90%] md:w-2/3 space-y-6">
		{#if meetingState.loading}
			<AppSpinner label="Loading attendee login" />
		{:else if meetingState.error}
			<AppAlert message={meetingState.error} />
		{:else if meetingState.data}
			<div class="space-y-2">
				<h1 class="text-3xl font-bold">{meetingState.data.meetingName}</h1>
				<p class="text-base-content/70">{meetingState.data.committeeName}</p>
			</div>

			<h3>Enter Your Access Code</h3>
			<div id="app-notification-target">
				{#if actionError}
					<AppAlert message={actionError} />
				{/if}
			</div>

			<form action={joinHref.replace('/join', '/attendee-login')} method="POST" onsubmit={handleSubmit}>
				<div>
					<label for="secret">Access Code:</label>
					<input
						id="secret"
						class="input input-bordered input-sm"
						name="secret"
						bind:value={attendeeSecret}
						autocomplete="off"
						type="text"
						required
					/>
				</div>

				<button class="btn btn-sm" type="submit" disabled={submitting}
					>{#if submitting}<span class="loading loading-spinner loading-xs"></span>{/if}Enter Meeting</button
				>
			</form>
		{/if}
	</div>
</div>
