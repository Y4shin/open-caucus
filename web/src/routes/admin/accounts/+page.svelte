<script lang="ts">
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { adminClient } from '$lib/api/index.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';

	type AccountRow = {
		accountId: string;
		username: string;
		fullName: string;
		isAdmin: boolean;
	};

	let accountsState = $state(createRemoteState<AccountRow[]>());
	let createAccountPending = $state(false);
	let createAccountError = $state('');
	let newUsername = $state('');
	let newFullName = $state('');
	let newPassword = $state('');

	$effect(() => {
		pageActions.set([], { backHref: '/admin' });
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
		loadAccounts();
	});

	async function loadAccounts() {
		accountsState.loading = true;
		accountsState.error = '';
		try {
			const res = await adminClient.listAccounts({ page: 1, pageSize: 200 });
			accountsState.data = res.accounts.map((account) => ({
				accountId: account.accountId,
				username: account.username,
				fullName: account.fullName,
				isAdmin: account.isAdmin
			}));
		} catch (err) {
			accountsState.error = getDisplayError(err, 'Failed to load accounts.');
		} finally {
			accountsState.loading = false;
		}
	}

	async function createAccount() {
		createAccountPending = true;
		createAccountError = '';
		try {
			await adminClient.createAccount({
				username: newUsername.trim(),
				fullName: newFullName.trim(),
				password: newPassword
			});
			newUsername = '';
			newFullName = '';
			newPassword = '';
			await loadAccounts();
		} catch (err) {
			createAccountError = getDisplayError(err, 'Failed to create account.');
		} finally {
			createAccountPending = false;
		}
	}
</script>

{#if accountsState.loading}
	<AppSpinner label="Loading accounts" />
{:else}
	{#if accountsState.error}
		<AppAlert message={accountsState.error} />
	{/if}

	<div id="account-list-container">
		{#if createAccountError}
			<AppAlert message={createAccountError} />
		{/if}
		<section class="panel card bg-base-100 border border-base-300 shadow-sm rounded-box p-4 mb-4">
			<h2>Add New Account</h2>
			<form
				id="create-account-form"
				onsubmit={(event) => {
					event.preventDefault();
					createAccount();
				}}
			>
				<div>
					<label for="username">Username:</label>
					<input
						class="input input-bordered input-sm"
						type="text"
						id="username"
						name="username"
						bind:value={newUsername}
						oninput={(event) => {
							newUsername = (event.currentTarget as HTMLInputElement).value;
						}}
						required
					/>
				</div>
				<div>
					<label for="full_name">Full Name:</label>
					<input
						class="input input-bordered input-sm"
						type="text"
						id="full_name"
						name="full_name"
						bind:value={newFullName}
						oninput={(event) => {
							newFullName = (event.currentTarget as HTMLInputElement).value;
						}}
						required
					/>
				</div>
				{#if session.passwordEnabled}
					<div>
						<label for="password">Password:</label>
						<input
							class="input input-bordered input-sm"
							type="password"
							id="password"
							name="password"
							bind:value={newPassword}
							oninput={(event) => {
								newPassword = (event.currentTarget as HTMLInputElement).value;
							}}
							required
						/>
					</div>
				{/if}
				<button class="btn btn-sm" type="submit" disabled={createAccountPending}>Create Account</button>
			</form>
		</section>
		<section class="panel card bg-base-100 border border-base-300 shadow-sm rounded-box p-4 mb-4">
			<h2>Existing Accounts</h2>
			{#if accountsState.data?.length === 0}
				<p>No accounts yet.</p>
			{:else}
				<table class="data-table table table-zebra w-full">
					<thead>
						<tr>
							<th>Username</th>
							<th>Full Name</th>
							<th>Admin</th>
						</tr>
					</thead>
					<tbody>
						{#each accountsState.data ?? [] as account}
							<tr>
								<td>{account.username}</td>
								<td>{account.fullName}</td>
								<td>{account.isAdmin ? 'Yes' : 'No'}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}
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
		</section>
	</div>
{/if}
