<script lang="ts">
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import { session } from '$lib/stores/session.svelte.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import { adminClient } from '$lib/api/index.js';
	import type { CommitteeRecord } from '$lib/gen/conference/admin/v1/admin_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';

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
		if (!window.confirm('Are you sure you want to delete this committee?')) {
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

<h2>Committees</h2>

{#if dashboardState.loading}
	<AppSpinner label="Loading admin dashboard" />
{:else if dashboardState.error}
	<AppAlert message={dashboardState.error} />
{:else}
	<section class="panel card bg-base-100 border border-base-300 shadow-sm rounded-box p-4 mb-4">
		<h3>Add New Committee</h3>
		<form
			id="create-committee-form"
			onsubmit={(event) => {
				event.preventDefault();
				createCommittee();
			}}
		>
			<div>
				<label for="name">Committee Name:</label>
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
				<label for="slug">Slug (URL-friendly identifier):</label>
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
				<small>Only lowercase letters, numbers, and hyphens</small>
			</div>
			{#if createCommitteeError}
				<AppAlert message={createCommitteeError} />
			{/if}
			<button class="btn btn-sm" type="submit" disabled={createCommitteePending}>Create Committee</button>
		</form>
	</section>

	<section class="panel card bg-base-100 border border-base-300 shadow-sm rounded-box p-4 mb-4">
		<h3>Existing Committees</h3>
		<div id="committee-list-container">
			{#if dashboardState.data?.length === 0}
				<p>No committees have been created yet.</p>
			{:else}
				<div id="committee-list">
					<table class="data-table table table-zebra w-full">
						<thead>
							<tr>
								<th>Name</th>
								<th>Slug</th>
								<th>Actions</th>
							</tr>
						</thead>
						<tbody>
							{#each dashboardState.data ?? [] as committee}
								<tr>
									<td>{committee.name}</td>
									<td>{committee.slug}</td>
									<td>
										<a href={"/admin/committee/" + committee.slug}>Assign Accounts</a>{' |'}<form
											class="inline-form inline"
											onsubmit={(event) => {
												event.preventDefault();
												deleteCommittee(committee.slug);
											}}
										>
											<button class="btn btn-sm" type="submit" disabled={deleteCommitteePendingSlug === committee.slug}>Delete</button>
										</form>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>
		<div class="centered-pagination-wrap flex justify-center">
			<nav class="pagination-nav join">
				<button type="button" disabled class="ui-icon-label btn btn-sm">
					<LegacyIcon name="left" class="ui-icon--left" />
					<span class="ui-icon-text">Previous</span>
				</button>
				<button class="btn btn-sm" type="button" disabled>1</button>
				<button type="button" disabled class="ui-icon-label btn btn-sm">
					<span class="ui-icon-text">Next</span>
					<LegacyIcon name="right" class="ui-icon--right" />
				</button>
			</nav>
		</div>
	</section>
{/if}
