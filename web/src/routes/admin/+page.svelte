<script lang="ts">
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import DataTable from '$lib/components/ui/DataTable.svelte';
	import PaginationNav from '$lib/components/ui/PaginationNav.svelte';
	import { session } from '$lib/stores/session.svelte.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import { adminClient } from '$lib/api/index.js';
	import type { CommitteeRecord } from '$lib/gen/conference/admin/v1/admin_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import * as m from '$lib/paraglide/messages';

	let dashboardState = $state(createRemoteState<CommitteeRecord[]>());
	let createCommitteePending = $state(false);
	let createCommitteeError = $state('');
	let deleteCommitteePendingSlug = $state('');
	let newCommitteeName = $state('');
	let newCommitteeSlug = $state('');

	$effect(() => {
		pageActions.set([{ label: 'Manage Accounts', href: '/admin/accounts', kind: 'ghost' }], { backHref: '/home' });
		return () => {
			pageActions.clear();
		};
	});

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
			const committees = await adminClient.listCommittees({ page: 1, pageSize: 100 });
			dashboardState.data = committees.committees;
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
		if (!window.confirm(m.admin_dashboard_delete_confirm())) {
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

<h2>{m.admin_dashboard_committees_heading()}</h2>

{#if dashboardState.loading}
	<AppSpinner label="Loading admin dashboard" />
{:else if dashboardState.error}
	<AppAlert message={dashboardState.error} />
{:else}
	<AppCard class="bg-base-100 shadow-sm mb-4">
		<h3>{m.admin_dashboard_add_committee_heading()}</h3>
		<form
			id="create-committee-form"
			onsubmit={(event) => {
				event.preventDefault();
				createCommittee();
			}}
		>
			<div>
				<label for="name">{m.admin_dashboard_name_label()}</label>
				<input
					class="input input-bordered input-sm"
					type="text"
					id="name"
					name="name"
					bind:value={newCommitteeName}
					oninput={(event) => {
						newCommitteeName = (event.currentTarget as HTMLInputElement).value;
					}}
					required
				/>
			</div>
			<div>
				<label for="slug">{m.admin_dashboard_slug_label()}</label>
				<input
					class="input input-bordered input-sm"
					type="text"
					id="slug"
					name="slug"
					bind:value={newCommitteeSlug}
					oninput={(event) => {
						newCommitteeSlug = (event.currentTarget as HTMLInputElement).value;
					}}
					required
					pattern="[a-z0-9\-]+"
				/>
				<small>{m.admin_dashboard_slug_help()}</small>
			</div>
			{#if createCommitteeError}
				<AppAlert message={createCommitteeError} />
			{/if}
			<button class="btn btn-sm" type="submit" disabled={createCommitteePending}>{m.admin_dashboard_create_button()}</button>
		</form>
	</AppCard>

	<AppCard class="bg-base-100 shadow-sm mb-4">
		<h3>{m.admin_dashboard_existing_heading()}</h3>
		<div id="committee-list-container">
			{#if dashboardState.data?.length === 0}
				<p>{m.admin_dashboard_empty_state()}</p>
			{:else}
				<div id="committee-list">
					<DataTable>
						{#snippet header()}
							<tr>
								<th>{m.admin_dashboard_col_name()}</th>
								<th>{m.admin_dashboard_col_slug()}</th>
								<th>{m.admin_dashboard_col_actions()}</th>
							</tr>
						{/snippet}
						{#snippet body()}
							{#each dashboardState.data ?? [] as committee}
								<tr>
									<td>{committee.name}</td>
									<td>{committee.slug}</td>
									<td>
										<a href={"/admin/committee/" + committee.slug}>{m.admin_dashboard_manage_users_link()}</a>{' |'}<form
											class="inline-form inline"
											onsubmit={(event) => {
												event.preventDefault();
												deleteCommittee(committee.slug);
											}}
										>
											<button class="btn btn-sm" type="submit" disabled={deleteCommitteePendingSlug === committee.slug}>{m.admin_dashboard_delete_button()}</button>
										</form>
									</td>
								</tr>
							{/each}
						{/snippet}
					</DataTable>
				</div>
			{/if}
		</div>
		<div class="centered-pagination-wrap flex justify-center">
			<PaginationNav />
		</div>
	</AppCard>
{/if}
