<script lang="ts">
	import { goto } from '$app/navigation';
	import { session } from '$lib/stores/session.svelte.js';
	import { adminClient } from '$lib/api/index.js';

	let dashboardData = $state<{
		totalCommittees: number;
		totalAccounts: number;
	} | null>(null);
	let loading = $state(true);
	let errorMsg = $state('');

	$effect(() => {
		if (!session.loaded) return;
		if (!session.authenticated || !session.isAdmin) {
			goto('/login');
			return;
		}
		loadDashboard();
	});

	async function loadDashboard() {
		try {
			const res = await adminClient.getAdminDashboard({});
			dashboardData = {
				totalCommittees: res.totalCommittees,
				totalAccounts: res.totalAccounts
			};
		} catch {
			errorMsg = 'Failed to load admin dashboard.';
		} finally {
			loading = false;
		}
	}
</script>

<div class="space-y-6">
	<h1 class="text-3xl font-bold">Admin Dashboard</h1>

	{#if loading}
		<div class="flex justify-center py-12">
			<span class="loading loading-spinner loading-lg"></span>
		</div>
	{:else if errorMsg}
		<div role="alert" class="alert alert-error">
			<span>{errorMsg}</span>
		</div>
	{:else if dashboardData}
		<div class="stats shadow">
			<div class="stat">
				<div class="stat-title">Committees</div>
				<div class="stat-value">{dashboardData.totalCommittees}</div>
			</div>
			<div class="stat">
				<div class="stat-title">Accounts</div>
				<div class="stat-value">{dashboardData.totalAccounts}</div>
			</div>
		</div>

		<div class="flex gap-2">
			<a href="/admin/committees" class="btn btn-outline">Manage Committees</a>
			<a href="/admin/accounts" class="btn btn-outline">Manage Accounts</a>
		</div>
	{/if}
</div>
