<script lang="ts">
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { adminClient } from '$lib/api/index.js';
	import type { AccountRecord } from '$lib/gen/conference/admin/v1/admin_pb.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';

	let accountsState = $state(createRemoteState<AccountRecord[]>());
	let createAccountPending = $state(false);
	let updatePendingId = $state('');
	let createAccountError = $state('');
	let newUsername = $state('');
	let newFullName = $state('');
	let newPassword = $state('');

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
			accountsState.data = res.accounts;
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

	async function toggleAdmin(account: AccountRecord) {
		updatePendingId = account.accountId;
		accountsState.error = '';
		try {
			await adminClient.setAccountAdmin({
				accountId: account.accountId,
				isAdmin: !account.isAdmin
			});
			await loadAccounts();
		} catch (err) {
			accountsState.error = getDisplayError(err, 'Failed to update account permissions.');
		} finally {
			updatePendingId = '';
		}
	}
</script>

<div class="space-y-6">
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div class="space-y-2">
			<h1 class="text-3xl font-bold">Admin Accounts</h1>
			<p class="text-base-content/70">
				Review platform accounts and grant or revoke administrator access.
			</p>
		</div>
		<a href="/admin" class="btn btn-outline">Back to Admin</a>
	</div>

	{#if accountsState.loading}
		<AppSpinner label="Loading accounts" />
	{:else}
		{#if accountsState.error}
			<AppAlert message={accountsState.error} />
		{/if}

		<div class="grid gap-6 xl:grid-cols-[minmax(0,24rem)_minmax(0,1fr)]">
			<AppCard title="Add New Account">
				<form
					id="create-account-form"
					class="space-y-4"
					onsubmit={async (event) => {
						event.preventDefault();
						await createAccount();
					}}
				>
					<label class="form-control gap-2">
						<span class="label-text font-medium">Username</span>
						<input
							class="input input-bordered"
							name="username"
							bind:value={newUsername}
							required
						/>
					</label>
					<label class="form-control gap-2">
						<span class="label-text font-medium">Full Name</span>
						<input
							class="input input-bordered"
							name="full_name"
							bind:value={newFullName}
							required
						/>
					</label>
					<label class="form-control gap-2">
						<span class="label-text font-medium">Password</span>
						<input
							class="input input-bordered"
							type="password"
							name="password"
							bind:value={newPassword}
							required
						/>
					</label>

					{#if createAccountError}
						<AppAlert message={createAccountError} />
					{/if}

					<button class="btn btn-primary" type="submit" disabled={createAccountPending}>
						{createAccountPending ? 'Creating...' : 'Create Account'}
					</button>
				</form>
			</AppCard>

			<AppCard title="Accounts">
				{#if accountsState.data?.length}
					<div class="overflow-x-auto">
						<table class="table table-zebra">
							<thead>
								<tr>
									<th>Username</th>
									<th>Full Name</th>
									<th>Admin</th>
									<th class="text-right">Actions</th>
								</tr>
							</thead>
							<tbody>
								{#each accountsState.data as account}
									<tr>
										<td class="font-medium">{account.username}</td>
										<td>{account.fullName}</td>
										<td>
											<span class={`badge ${account.isAdmin ? 'badge-success' : 'badge-ghost'}`}>
												{account.isAdmin ? 'Admin' : 'Standard'}
											</span>
										</td>
										<td>
											<div class="flex justify-end">
												<button
													class="btn btn-sm btn-outline"
													type="button"
													disabled={updatePendingId === account.accountId}
													onclick={() => toggleAdmin(account)}
												>
													{#if updatePendingId === account.accountId}
														Saving...
													{:else if account.isAdmin}
														Revoke Admin
													{:else}
														Make Admin
													{/if}
												</button>
											</div>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{:else}
					<AppAlert tone="info" message="No accounts are available yet." />
				{/if}
			</AppCard>
		</div>
	{/if}
</div>
