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

	<div id="account-list-container" class="space-y-4">
		<AppCard class="bg-base-100 shadow-sm">
			<h2 class="text-base font-semibold mb-3">{m.admin_accounts_add_heading()}</h2>
			<form
				id="create-account-form"
				class="grid gap-3 sm:grid-cols-[1fr_1fr_1fr_auto] sm:items-end"
				onsubmit={(event) => {
					event.preventDefault();
					createAccount();
				}}
			>
				<div>
					<label class="label text-sm font-medium" for="username">{m.admin_accounts_username_label()}</label>
					<input
						class="input input-bordered input-sm w-full"
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
					<label class="label text-sm font-medium" for="full_name">{m.admin_accounts_fullname_label()}</label>
					<input
						class="input input-bordered input-sm w-full"
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
						<label class="label text-sm font-medium" for="password">{m.admin_accounts_password_label()}</label>
						<input
							class="input input-bordered input-sm w-full"
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
				<button class="btn btn-sm btn-primary" type="submit" disabled={createAccountPending}>{m.admin_accounts_create_button()}</button>
			</form>
			{#if createAccountError}
				<div class="mt-3"><AppAlert message={createAccountError} /></div>
			{/if}
		</AppCard>
		<AppCard class="bg-base-100 shadow-sm">
			<h2 class="text-base font-semibold mb-3">{m.admin_accounts_existing_heading()}</h2>
			{#if accountsState.data?.length === 0}
				<p class="text-sm text-base-content/70">{m.admin_accounts_empty_state()}</p>
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
								<td>
									{#if account.isAdmin}
										<span class="badge badge-success badge-sm">{m.common_yes()}</span>
									{:else}
										<span class="badge badge-ghost badge-sm">{m.common_no()}</span>
									{/if}
								</td>
							</tr>
						{/each}
					{/snippet}
				</DataTable>
			{/if}
			<div class="mt-3 flex justify-center">
				<PaginationNav />
			</div>
		</AppCard>
	</div>
{/if}
