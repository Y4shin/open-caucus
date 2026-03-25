<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { moderationClient } from '$lib/api/index.js';
	import { session } from '$lib/stores/session.svelte.js';
	import type { ModerationView } from '$lib/gen/conference/moderation/v1/moderation_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { connectEventStream } from '$lib/utils/sse.js';

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);

	let moderationState = $state(createRemoteState<ModerationView>());
	let actionError = $state('');
	let togglingSignup = $state(false);
	let refreshTick = $state(0);

	$effect(() => {
		if (!session.loaded) return;
		if (!session.authenticated) {
			goto('/login');
			return;
		}
		refreshTick;
		loadModerationView();
	});

	$effect(() => {
		const eventsUrl = moderationState.data?.eventsUrl;
		if (!eventsUrl) return;
		return connectEventStream(eventsUrl, () => {
			refreshTick += 1;
		});
	});

	async function loadModerationView() {
		moderationState.loading = true;
		moderationState.error = '';
		try {
			const res = await moderationClient.getModerationView({ committeeSlug: slug, meetingId });
			moderationState.data = res.view ?? null;
		} catch (err) {
			moderationState.error = getDisplayError(err, 'Failed to load the moderation view.');
		} finally {
			moderationState.loading = false;
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
			refreshTick += 1;
		} finally {
			togglingSignup = false;
		}
	}
</script>

<div class="space-y-6">
	{#if moderationState.loading}
		<AppSpinner label="Loading moderation view" />
	{:else if moderationState.error}
		<AppAlert message={moderationState.error} />
	{:else if moderationState.data}
		<div class="space-y-2">
			<h1 class="text-3xl font-bold">{moderationState.data.meeting?.meetingName}</h1>
			<p class="text-base-content/70">
				Moderation workspace for {moderationState.data.meeting?.committeeName}
			</p>
		</div>

		{#if actionError}
			<AppAlert message={actionError} />
		{/if}

		<div class="grid gap-4 xl:grid-cols-3">
			<AppCard title="Signup Control">
				<p class="text-sm text-base-content/70">
					Version {moderationState.data.version.toString()}
				</p>
				<p class="mt-3 font-medium">
					Signup is {moderationState.data.attendees?.signupOpen ? 'open' : 'closed'}.
				</p>
				<button class="btn btn-primary btn-sm mt-4" onclick={toggleSignupOpen} disabled={togglingSignup}>
					{#if togglingSignup}
						<span class="loading loading-spinner loading-xs"></span>
					{/if}
					{moderationState.data.attendees?.signupOpen ? 'Close Signup' : 'Open Signup'}
				</button>
			</AppCard>

			<AppCard title="Attendees">
				<div class="space-y-2 text-sm">
					<p>Total: {moderationState.data.attendees?.totalCount ?? 0}</p>
					<p>Guests: {moderationState.data.attendees?.guestCount ?? 0}</p>
					<p>Chairs: {moderationState.data.attendees?.chairCount ?? 0}</p>
					<p>
						Self-signup visible:
						{moderationState.data.attendees?.showSelfSignup ? 'Yes' : 'No'}
					</p>
				</div>
			</AppCard>

			<AppCard title="Speakers">
				<div class="space-y-2 text-sm">
					<p>Total: {moderationState.data.speakers?.totalCount ?? 0}</p>
					<p>Waiting: {moderationState.data.speakers?.waitingCount ?? 0}</p>
					<p>
						Active speaker:
						{moderationState.data.speakers?.hasActiveSpeaker ? 'Yes' : 'No'}
					</p>
				</div>
			</AppCard>
		</div>

		<AppCard title="Current Agenda Point">
			{#if moderationState.data.activeAgendaPoint}
				<p class="font-medium">
					{moderationState.data.activeAgendaPoint.displayNumber}
					{#if moderationState.data.activeAgendaPoint.title}
						: {moderationState.data.activeAgendaPoint.title}
					{/if}
				</p>
			{:else}
				<p class="text-base-content/70">No agenda point is currently active.</p>
			{/if}
		</AppCard>
	{/if}
</div>
