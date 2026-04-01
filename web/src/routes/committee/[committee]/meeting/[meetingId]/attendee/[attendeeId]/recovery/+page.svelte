<script lang="ts">
	import { page } from '$app/state';
	import { attendeeClient } from '$lib/api/index.js';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import type { AttendeeRecoveryView } from '$lib/gen/conference/attendees/v1/attendees_pb.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { onDestroy } from 'svelte';

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);
	const attendeeId = $derived(page.params.attendeeId);
	const baseUrl = $derived(page.url.origin);
	const moderateHref = $derived(`/committee/${slug}/meeting/${meetingId}/moderate`);

	let recoveryState = $state(createRemoteState<AttendeeRecoveryView>());

	onDestroy(() => {
		pageActions.clear();
	});

	$effect(() => {
		if (!session.loaded) return;
		void loadRecovery();
	});

	$effect(() => {
		if (!recoveryState.data) return;
		pageActions.set([], {
			backHref: moderateHref,
			title: 'Guest Recovery',
			subtitle: `${recoveryState.data.attendeeName} - ${recoveryState.data.meetingName}`
		});
	});

	async function loadRecovery() {
		recoveryState.loading = true;
		recoveryState.error = '';
		try {
			const res = await attendeeClient.getAttendeeRecovery({
				committeeSlug: slug,
				meetingId,
				attendeeId,
				baseUrl
			});
			recoveryState.data = res.view ?? null;
		} catch (err) {
			recoveryState.error = getDisplayError(err, 'Failed to load the attendee recovery page.');
		} finally {
			recoveryState.loading = false;
		}
	}
</script>

<div class="space-y-6">
	{#if recoveryState.loading}
		<AppSpinner label="Loading attendee recovery" />
	{:else if recoveryState.error}
		<AppAlert message={recoveryState.error} />
	{:else if recoveryState.data}
		<div class="space-y-2">
			<h1 class="text-3xl font-bold">Guest Recovery</h1>
			<p class="text-base-content/70">{recoveryState.data.attendeeName} - {recoveryState.data.meetingName}</p>
		</div>

		<p>Use this recovery link or QR code to log this guest back into the meeting directly.</p>
		<p>
			<a id="attendee-recovery-link" href={recoveryState.data.loginUrl}>{recoveryState.data.loginUrl}</a>
		</p>
		<img id="attendee-recovery-qr" src={recoveryState.data.qrCodeDataUrl} alt="Guest Recovery QR Code" />
	{/if}
</div>
