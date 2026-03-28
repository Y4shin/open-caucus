<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { session } from '$lib/stores/session.svelte.js';
	import { committeeClient } from '$lib/api/index.js';
	import type { CommitteeOverview } from '$lib/gen/conference/committees/v1/committees_pb.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';

	const slug = $derived(page.params.committee);

	let committeeState = $state(createRemoteState<CommitteeOverview>());
	let createName = $state('');
	let createPending = $state(false);
	let createError = $state('');
	let actionError = $state('');
	let localActiveMeetingId = $state<string | null>(null);

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
			committeeState.error = getDisplayError(
				err,
				`Committee "${slug}" not found or access denied.`
			);
		} finally {
			committeeState.loading = false;
		}
	}

	const canManage = $derived(
		committeeState.data?.committee?.isChairperson || committeeState.data?.committee?.isAdmin
	);

	const activeMeetingItem = $derived(
		committeeState.data?.meetings.find((m) => m.canViewLive) ?? null
	);

	async function createMeeting(e: Event) {
		e.preventDefault();
		if (!createName.trim() || createPending) return;
		createPending = true;
		createError = '';
		try {
			const response = await fetch(`/api/committee/${slug}/meetings`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ name: createName.trim(), description: '' })
			});
			if (!response.ok) {
				const payload = (await response.json().catch(() => null)) as { error?: string } | null;
				throw new Error(payload?.error || `create failed (${response.status})`);
			}
			createName = '';
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
			const response = await fetch(`/api/committee/${slug}/meetings/${meetingId}/active`, {
				method: 'POST'
			});
			if (!response.ok) {
				const payload = (await response.json().catch(() => null)) as { error?: string } | null;
				localActiveMeetingId = wasActive ? meetingId : null;
				throw new Error(payload?.error || `toggle failed (${response.status})`);
			}
			await loadCommittee();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to toggle active meeting.');
		}
	}

	async function deleteMeeting(meetingId: string, name: string) {
		if (!window.confirm(`Delete "${name}"?`)) return;
		actionError = '';
		try {
			const response = await fetch(`/api/committee/${slug}/meetings/${meetingId}`, {
				method: 'DELETE'
			});
			if (!response.ok) {
				const payload = (await response.json().catch(() => null)) as { error?: string } | null;
				throw new Error(payload?.error || `delete failed (${response.status})`);
			}
			await loadCommittee();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to delete meeting.');
		}
	}
</script>

<div class="space-y-6">
	{#if committeeState.loading}
		<AppSpinner label="Loading committee overview" />
	{:else if committeeState.error}
		<AppAlert message={committeeState.error} />
	{:else if committeeState.data}
		<div class="space-y-2">
			<h1 class="text-3xl font-bold">{committeeState.data.committee?.name ?? slug}</h1>
			<p class="text-base-content/70">Committee dashboard and current meeting access.</p>
		</div>

		{#if actionError}
			<AppAlert message={actionError} />
		{/if}

		{#if canManage}
			<div id="meeting-list-container" class="space-y-6">
				{#if createError}
					<AppAlert message={createError} />
				{/if}
				<form data-testid="committee-create-form" class="flex gap-2" onsubmit={createMeeting}>
					<input
						class="input input-bordered flex-1"
						name="name"
						bind:value={createName}
						placeholder="Meeting name"
						required
					/>
					<button class="btn btn-primary" type="submit" disabled={createPending}>
						{createPending ? 'Creating...' : 'Create Meeting'}
					</button>
				</form>

				{#if committeeState.data.meetings.length}
					<div class="space-y-3">
						{#each committeeState.data.meetings as item}
							<div
								data-testid="committee-meeting-row"
								class="flex items-center justify-between rounded-box border border-base-300 bg-base-100 p-4"
							>
								<div class="flex items-center gap-3">
									<label class="flex cursor-pointer items-center gap-2">
										<input
											type="checkbox"
											data-testid="committee-toggle-active"
											class="checkbox"
											checked={localActiveMeetingId === item.meeting?.meetingId}
											onclick={(e) => toggleActive(e, item.meeting?.meetingId ?? '')}
										/>
										<span class="font-medium">{item.meeting?.name}</span>
									</label>
									{#if localActiveMeetingId === item.meeting?.meetingId}
										<span class="badge badge-success badge-sm">Active</span>
									{/if}
								</div>

								<div class="flex gap-2">
									{#if item.canModerate}
										<a
											class="btn btn-sm btn-outline"
											href="/committee/{slug}/meeting/{item.meeting?.meetingId}/moderate"
										>
											Moderate
										</a>
									{/if}
									<button
										data-testid="committee-delete-meeting"
										class="btn btn-sm btn-error btn-outline"
										type="button"
										onclick={() =>
											deleteMeeting(
												item.meeting?.meetingId ?? '',
												item.meeting?.name ?? ''
											)}
									>
										Delete
									</button>
								</div>
							</div>
						{/each}
					</div>
				{:else}
					<AppAlert tone="info" message="No meetings have been created yet." />
				{/if}
			</div>
		{:else}
			{#if activeMeetingItem}
				<AppCard>
					<div data-testid="committee-active-meeting-card" class="space-y-3">
						<div>
							<p class="text-sm text-base-content/70">Active Meeting</p>
							<p class="text-xl font-semibold" data-testid="committee-active-meeting-name">
								{activeMeetingItem.meeting?.name}
							</p>
						</div>
						<div class="flex gap-2">
							<a
								data-testid="committee-join-active-meeting"
								class="btn btn-primary btn-sm"
								href="/committee/{slug}/meeting/{activeMeetingItem.meeting?.meetingId}/join"
							>
								Join Meeting
							</a>
							{#if activeMeetingItem.canViewLive}
								<a
									class="btn btn-outline btn-sm"
									href="/committee/{slug}/meeting/{activeMeetingItem.meeting?.meetingId}"
								>
									Live View
								</a>
							{/if}
						</div>
					</div>
				</AppCard>
			{:else}
				<AppAlert tone="info" message="No active meeting at the moment." />
			{/if}
		{/if}
	{/if}
</div>
