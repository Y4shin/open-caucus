<script lang="ts">
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { session } from '$lib/stores/session.svelte.js';
	import { adminClient } from '$lib/api/index.js';
	import type { CommitteeRecord } from '$lib/gen/conference/admin/v1/admin_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';

	interface DashboardData {
		totalCommittees: number;
		totalAccounts: number;
		committees: CommitteeRecord[];
	}

	let dashboardState = $state(createRemoteState<DashboardData>());
	let createCommitteePending = $state(false);
	let createCommitteeError = $state('');
	let deleteCommitteePendingSlug = $state('');
	let newCommitteeName = $state('');
	let newCommitteeSlug = $state('');

	$effect(() => {
		if (!session.loaded) return;
		if (!session.authenticated || !session.isAdmin) {
			goto('/admin/login');
			return;
		}
		loadDashboard();
	});

	async function loadDashboard() {
		dashboardState.loading = true;
		dashboardState.error = '';
		try {
			const [dashboard, committees] = await Promise.all([
				adminClient.getAdminDashboard({}),
				adminClient.listCommittees({ page: 1, pageSize: 100 })
			]);
			dashboardState.data = {
				totalCommittees: Number(dashboard.totalCommittees),
				totalAccounts: Number(dashboard.totalAccounts),
				committees: committees.committees
			};
		} catch (err) {
			dashboardState.error = getDisplayError(err, 'Failed to load admin dashboard.');
		} finally {
			dashboardState.loading = false;
		}
	}

	async function createCommittee() {
		createCommitteePending = true;
		createCommitteeError = '';

		try {
			await adminClient.createCommittee({
				name: newCommitteeName.trim(),
				slug: newCommitteeSlug.trim()
			});
			newCommitteeName = '';
			newCommitteeSlug = '';
			await loadDashboard();
		} catch (err) {
			createCommitteeError = getDisplayError(err, 'Failed to create committee.');
		} finally {
			createCommitteePending = false;
		}
	}

	async function deleteCommittee(slug: string) {
		if (!window.confirm(`Delete committee "${slug}"?`)) {
			return;
		}

		deleteCommitteePendingSlug = slug;
		dashboardState.error = '';

		try {
			await adminClient.deleteCommittee({ slug });
			await loadDashboard();
		} catch (err) {
			dashboardState.error = getDisplayError(err, 'Failed to delete committee.');
		} finally {
			deleteCommitteePendingSlug = '';
		}
	}
</script>

<div class="space-y-6">
	<div class="space-y-2">
		<h1 class="text-3xl font-bold">Admin Dashboard</h1>
		<p class="text-base-content/70">
			Manage committees, committee memberships, and platform-wide accounts.
		</p>
	</div>

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

			<div class="flex flex-wrap gap-2">
				<a href="/admin/accounts" class="btn btn-outline">Manage Accounts</a>
			</div>

		<div class="grid gap-6 xl:grid-cols-[minmax(0,24rem)_minmax(0,1fr)]">
			<AppCard title="Create Committee">
				<form
					id="create-committee-form"
					class="space-y-4"
					onsubmit={async (event) => {
						event.preventDefault();
						await createCommittee();
					}}
				>
					<label class="form-control gap-2">
						<span class="label-text font-medium">Name</span>
						<input
							class="input input-bordered"
							name="name"
							bind:value={newCommitteeName}
							required
							placeholder="General Assembly"
						/>
					</label>

					<label class="form-control gap-2">
						<span class="label-text font-medium">Slug</span>
						<input
							class="input input-bordered"
							name="slug"
							bind:value={newCommitteeSlug}
							required
							pattern="[a-z0-9\\-]+"
							placeholder="general-assembly"
						/>
						<span class="text-xs text-base-content/70">
							Lowercase letters, numbers, and hyphens only.
						</span>
					</label>

					{#if createCommitteeError}
						<AppAlert message={createCommitteeError} />
					{/if}

					<button class="btn btn-primary" type="submit" disabled={createCommitteePending}>
						{createCommitteePending ? 'Creating...' : 'Create Committee'}
					</button>
				</form>
			</AppCard>

			<AppCard title="Committees">
				{#if dashboardState.data.committees.length}
					<div class="overflow-x-auto">
						<table class="table table-zebra">
							<thead>
								<tr>
									<th>Name</th>
									<th>Slug</th>
									<th>Members</th>
									<th class="text-right">Actions</th>
								</tr>
							</thead>
							<tbody>
								{#each dashboardState.data.committees as committee}
									<tr>
										<td class="font-medium">{committee.name}</td>
										<td><code>{committee.slug}</code></td>
										<td>{Number(committee.memberCount)}</td>
										<td>
											<div class="flex justify-end gap-2">
												<a class="btn btn-sm btn-outline" href="/admin/committee/{committee.slug}">
													Assign Accounts
												</a>
												<button
													class="btn btn-sm btn-error btn-outline"
													type="submit"
													disabled={deleteCommitteePendingSlug === committee.slug}
													onclick={() => deleteCommittee(committee.slug)}
												>
													{deleteCommitteePendingSlug === committee.slug
														? 'Deleting...'
														: 'Delete'}
												</button>
											</div>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{:else}
					<AppAlert tone="info" message="No committees have been created yet." />
				{/if}
			</AppCard>
		</div>
	{/if}
</div>
