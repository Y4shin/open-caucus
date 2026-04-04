<script lang="ts">
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { adminClient } from '$lib/api/index.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import DataTable from '$lib/components/ui/DataTable.svelte';
	import PaginationNav from '$lib/components/ui/PaginationNav.svelte';
	import * as m from '$lib/paraglide/messages';

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
		<AppCard class="bg-base-100 shadow-sm mb-4">
			<h2>{m.admin_accounts_add_heading()}</h2>
			<form
				id="create-account-form"
				onsubmit={(event) => {
					event.preventDefault();
					createAccount();
				}}
			>
				<div>
					<label for="username">{m.admin_accounts_username_label()}</label>
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
					<label for="full_name">{m.admin_accounts_fullname_label()}</label>
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
						<label for="password">{m.admin_accounts_password_label()}</label>
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
				<button class="btn btn-sm" type="submit" disabled={createAccountPending}>{m.admin_accounts_create_button()}</button>
			</form>
		</AppCard>
		<AppCard class="bg-base-100 shadow-sm mb-4">
			<h2>{m.admin_accounts_existing_heading()}</h2>
			{#if accountsState.data?.length === 0}
				<p>{m.admin_accounts_empty_state()}</p>
			{:else}
				<DataTable>
					{#snippet header()}
						<tr>
							<th>{m.admin_accounts_col_username()}</th>
							<th>{m.admin_accounts_col_fullname()}</th>
							<th>{m.admin_accounts_col_admin()}</th>
						</tr>
					{/snippet}
					{#snippet body()}
						{#each accountsState.data ?? [] as account}
							<tr>
								<td>{account.username}</td>
								<td>{account.fullName}</td>
								<td>{account.isAdmin ? m.common_yes() : m.common_no()}</td>
							</tr>
						{/each}
					{/snippet}
				</DataTable>
			{/if}
			<PaginationNav />
		</AppCard>
	</div>
{/if}
