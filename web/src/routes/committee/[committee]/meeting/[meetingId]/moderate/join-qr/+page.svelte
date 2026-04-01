<script lang="ts">
	import { page } from '$app/state';
	import { meetingClient } from '$lib/api/index.js';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import type { MeetingJoinQrView } from '$lib/gen/conference/meetings/v1/meetings_pb.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { onDestroy } from 'svelte';

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);
	const baseUrl = $derived(page.url.origin);
	const moderateHref = $derived(`/committee/${slug}/meeting/${meetingId}/moderate`);

	let qrState = $state(createRemoteState<MeetingJoinQrView>());

	onDestroy(() => {
		pageActions.clear();
	});

	$effect(() => {
		if (!session.loaded) return;
		void loadJoinQr();
	});

	$effect(() => {
		if (!qrState.data) return;
		pageActions.set([], {
			backHref: moderateHref,
			title: 'Guest Join QR',
			subtitle: `${qrState.data.meetingName} - ${qrState.data.committeeName}`
		});
	});

	async function loadJoinQr() {
		qrState.loading = true;
		qrState.error = '';
		try {
			const res = await meetingClient.getMeetingJoinQr({
				committeeSlug: slug,
				meetingId,
				baseUrl
			});
			qrState.data = res.view ?? null;
		} catch (err) {
			qrState.error = getDisplayError(err, 'Failed to load the join QR code.');
		} finally {
			qrState.loading = false;
		}
	}
</script>

<div class="space-y-6">
	{#if qrState.loading}
		<AppSpinner label="Loading join QR" />
	{:else if qrState.error}
		<AppAlert message={qrState.error} />
	{:else if qrState.data}
		<div class="space-y-2">
			<h1 class="text-3xl font-bold">Guest Join QR</h1>
			<p class="text-base-content/70">{qrState.data.meetingName} - {qrState.data.committeeName}</p>
		</div>

		<p>Guests can scan this QR code to open the join page with the meeting secret prefilled.</p>
		<p>
			<a class="plain-text-link link link-hover" href={qrState.data.joinUrl}>{qrState.data.joinUrl}</a>
		</p>
		<img id="join-qr-code" src={qrState.data.qrCodeDataUrl} alt="Guest Join QR Code" />
	{/if}
</div>
