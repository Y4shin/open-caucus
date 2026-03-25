<script lang="ts">
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { session } from '$lib/stores/session.svelte.js';
	import { adminClient } from '$lib/api/index.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';

	let dashboardState = $state(createRemoteState<{
		totalCommittees: number;
		totalAccounts: number;
	}>());

	$effect(() => {
		if (!session.loaded) return;
		if (!session.authenticated || !session.isAdmin) {
			goto('/login');
			return;
		}
		loadDashboard();
	});

	async function loadDashboard() {
		dashboardState.loading = true;
		dashboardState.error = '';
		try {
			const res = await adminClient.getAdminDashboard({});
			dashboardState.data = {
				totalCommittees: Number(res.totalCommittees),
				totalAccounts: Number(res.totalAccounts)
			};
		} catch (err) {
			dashboardState.error = getDisplayError(err, 'Failed to load admin dashboard.');
		} finally {
			dashboardState.loading = false;
		}
	}
</script>

<div class="space-y-6">
	<h1 class="text-3xl font-bold">Admin Dashboard</h1>

	{#if dashboardState.loading}
		<AppSpinner label="Loading admin dashboard" />
	{:else if dashboardState.error}
		<AppAlert message={dashboardState.error} />
	{:else if dashboardState.data}
		<div class="stats shadow">
			<div class="stat">
				<div class="stat-title">Committees</div>
				<div class="stat-value">{dashboardState.data.totalCommittees}</div>
			</div>
			<div class="stat">
				<div class="stat-title">Accounts</div>
				<div class="stat-value">{dashboardState.data.totalAccounts}</div>
			</div>
		</div>

		<div class="flex gap-2">
			<a href="/admin/committees" class="btn btn-outline">Manage Committees</a>
			<a href="/admin/accounts" class="btn btn-outline">Manage Accounts</a>
		</div>
	{/if}
</div>
