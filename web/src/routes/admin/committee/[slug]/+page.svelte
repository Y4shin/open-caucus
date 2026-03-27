<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { adminClient } from '$lib/api/index.js';
	import type {
		AccountRecord,
		CommitteeRecord,
		CommitteeUserRecord,
		OAuthRuleRecord
	} from '$lib/gen/conference/admin/v1/admin_pb.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';

	interface CommitteeAdminData {
		committee: CommitteeRecord;
		users: CommitteeUserRecord[];
		oauthRules: OAuthRuleRecord[];
		assignableAccounts: AccountRecord[];
	}

	const slug = $derived(page.params.slug);

	let committeeState = $state(createRemoteState<CommitteeAdminData>());
	let createUserPending = $state(false);
	let assignAccountPending = $state(false);
	let saveMembershipPendingId = $state('');
	let deleteMembershipPendingId = $state('');
	let createRulePending = $state(false);
	let deleteRulePendingId = $state('');

	let membershipDrafts = $state<Record<string, { role: string; quoted: boolean }>>({});

	let localUsername = $state('');
	let localFullName = $state('');
	let localPassword = $state('');
	let localRole = $state('member');
	let localQuoted = $state(false);

	let selectedAccountId = $state('');
	let selectedAccountRole = $state('member');
	let selectedAccountQuoted = $state(false);

	let oauthGroupName = $state('');
	let oauthRole = $state('member');

	$effect(() => {
		if (!session.loaded) return;
		if (!session.authenticated || !session.isAdmin) {
			goto('/admin/login');
			return;
		}
		loadCommitteeAdmin();
	});

	function setMembershipDrafts(users: CommitteeUserRecord[]) {
		membershipDrafts = Object.fromEntries(
			users.map((user) => [user.userId, { role: user.role, quoted: user.quoted }])
		);
	}

	async function loadCommitteeAdmin() {
		committeeState.loading = true;
		committeeState.error = '';

		try {
			const [committeeAdmin, accounts] = await Promise.all([
				adminClient.getCommitteeAdmin({ slug }),
				adminClient.listAccounts({ page: 1, pageSize: 500 })
			]);

			const users = committeeAdmin.users;
			const assignedIds = new Set(users.map((user) => user.userId));
			const assignableAccounts = accounts.accounts.filter((account) => !assignedIds.has(account.accountId));

			setMembershipDrafts(users);
			committeeState.data = {
				committee: committeeAdmin.committee!,
				users,
				oauthRules: committeeAdmin.oauthRules,
				assignableAccounts
			};
		} catch (err) {
			committeeState.error = getDisplayError(
				err,
				`Failed to load admin settings for committee "${slug}".`
			);
		} finally {
			committeeState.loading = false;
		}
	}

	async function createCommitteeUser() {
		createUserPending = true;
		committeeState.error = '';

		try {
			await adminClient.createCommitteeUser({
				slug,
				username: localUsername.trim(),
				fullName: localFullName.trim(),
				password: localPassword,
				role: localRole,
				quoted: localQuoted
			});
			localUsername = '';
			localFullName = '';
			localPassword = '';
			localRole = 'member';
			localQuoted = false;
			await loadCommitteeAdmin();
		} catch (err) {
			committeeState.error = getDisplayError(err, 'Failed to create committee user.');
		} finally {
			createUserPending = false;
		}
	}

	async function assignExistingAccount() {
		assignAccountPending = true;
		committeeState.error = '';

		try {
			await adminClient.assignAccountToCommittee({
				slug,
				accountId: selectedAccountId,
				role: selectedAccountRole,
				quoted: selectedAccountQuoted
			});
			selectedAccountId = '';
			selectedAccountRole = 'member';
			selectedAccountQuoted = false;
			await loadCommitteeAdmin();
		} catch (err) {
			committeeState.error = getDisplayError(err, 'Failed to assign account to committee.');
		} finally {
			assignAccountPending = false;
		}
	}

	async function saveMembership(userId: string) {
		const draft = membershipDrafts[userId];
		if (!draft) return;

		saveMembershipPendingId = userId;
		committeeState.error = '';

		try {
			await adminClient.updateCommitteeUser({
				slug,
				userId,
				role: draft.role,
				quoted: draft.quoted
			});
			await loadCommitteeAdmin();
		} catch (err) {
			committeeState.error = getDisplayError(err, 'Failed to update committee membership.');
		} finally {
			saveMembershipPendingId = '';
		}
	}

	async function deleteMembership(user: CommitteeUserRecord) {
		if (!window.confirm(`Remove ${user.fullName} from this committee?`)) {
			return;
		}

		deleteMembershipPendingId = user.userId;
		committeeState.error = '';

		try {
			await adminClient.deleteCommitteeUser({ slug, userId: user.userId });
			await loadCommitteeAdmin();
		} catch (err) {
			committeeState.error = getDisplayError(err, 'Failed to remove committee user.');
		} finally {
			deleteMembershipPendingId = '';
		}
	}

	async function createOAuthRule() {
		createRulePending = true;
		committeeState.error = '';

		try {
			await adminClient.createOAuthRule({
				slug,
				groupName: oauthGroupName.trim(),
				role: oauthRole
			});
			oauthGroupName = '';
			oauthRole = 'member';
			await loadCommitteeAdmin();
		} catch (err) {
			committeeState.error = getDisplayError(err, 'Failed to create OAuth rule.');
		} finally {
			createRulePending = false;
		}
	}

	async function deleteOAuthRule(rule: OAuthRuleRecord) {
		if (!window.confirm(`Delete OAuth rule for group "${rule.groupName}"?`)) {
			return;
		}

		deleteRulePendingId = rule.ruleId;
		committeeState.error = '';

		try {
			await adminClient.deleteOAuthRule({ slug, ruleId: rule.ruleId });
			await loadCommitteeAdmin();
		} catch (err) {
			committeeState.error = getDisplayError(err, 'Failed to delete OAuth rule.');
		} finally {
			deleteRulePendingId = '';
		}
	}
</script>

<div class="space-y-6" id="committee-users-container">
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div class="space-y-2">
			<h1 class="text-3xl font-bold">
				{committeeState.data?.committee.name ?? slug}
			</h1>
			<p class="text-base-content/70">Manage committee members, local users, and OAuth rules.</p>
		</div>
		<a href="/admin" class="btn btn-outline">Back to Admin</a>
	</div>

	{#if committeeState.loading}
		<AppSpinner label="Loading committee admin" />
	{:else}
		{#if committeeState.error}
			<AppAlert message={committeeState.error} />
		{/if}

		{#if committeeState.data}
			<div class="stats shadow">
				<div class="stat">
					<div class="stat-title">Committee</div>
					<div class="stat-value text-lg">{committeeState.data.committee.name}</div>
					<div class="stat-desc"><code>{committeeState.data.committee.slug}</code></div>
				</div>
				<div class="stat">
					<div class="stat-title">Members</div>
					<div class="stat-value">{Number(committeeState.data.committee.memberCount)}</div>
				</div>
				<div class="stat">
					<div class="stat-title">OAuth Rules</div>
					<div class="stat-value">{committeeState.data.oauthRules.length}</div>
				</div>
			</div>

			<div class="grid gap-6 xl:grid-cols-2">
				<AppCard title="Create Local User">
					<form
						class="space-y-4"
						onsubmit={async (event) => {
							event.preventDefault();
							await createCommitteeUser();
						}}
					>
						<label class="form-control gap-2">
							<span class="label-text font-medium">Username</span>
								<input class="input input-bordered" bind:value={localUsername} required />
						</label>
						<label class="form-control gap-2">
							<span class="label-text font-medium">Full Name</span>
								<input class="input input-bordered" bind:value={localFullName} required />
						</label>
						<label class="form-control gap-2">
							<span class="label-text font-medium">Password</span>
								<input
									class="input input-bordered"
									type="password"
									bind:value={localPassword}
									name="password"
									required
								/>
						</label>
						<label class="form-control gap-2">
							<span class="label-text font-medium">Role</span>
								<select class="select select-bordered" bind:value={localRole} name="role">
								<option value="member">Member</option>
								<option value="chairperson">Chairperson</option>
							</select>
						</label>
						<label class="label cursor-pointer justify-start gap-3">
							<input class="checkbox" type="checkbox" bind:checked={localQuoted} />
							<span class="label-text">Quoted speaker status</span>
						</label>
						<button class="btn btn-primary" type="submit" disabled={createUserPending}>
							{createUserPending ? 'Creating...' : 'Create User'}
						</button>
					</form>
				</AppCard>

				<AppCard title="Assign Existing Account">
					{#if committeeState.data.assignableAccounts.length}
						<form
							class="space-y-4"
							onsubmit={async (event) => {
								event.preventDefault();
								await assignExistingAccount();
							}}
						>
							<label class="form-control gap-2">
								<span class="label-text font-medium">Account</span>
								<select
									class="select select-bordered"
									bind:value={selectedAccountId}
									name="account_id"
									required
								>
									<option value="" disabled>Select an account</option>
									{#each committeeState.data.assignableAccounts as account}
										<option value={account.accountId}>
											{account.fullName} ({account.username})
										</option>
									{/each}
								</select>
							</label>
							<label class="form-control gap-2">
								<span class="label-text font-medium">Role</span>
								<select class="select select-bordered" bind:value={selectedAccountRole} name="role">
									<option value="member">Member</option>
									<option value="chairperson">Chairperson</option>
								</select>
							</label>
							<label class="label cursor-pointer justify-start gap-3">
								<input
									class="checkbox"
									type="checkbox"
									name="quoted"
									bind:checked={selectedAccountQuoted}
								/>
								<span class="label-text">Quoted speaker status</span>
							</label>
							<button class="btn btn-primary" type="submit" disabled={assignAccountPending}>
								{assignAccountPending ? 'Assigning...' : 'Assign Account'}
							</button>
						</form>
					{:else}
						<AppAlert
							tone="info"
							message="Every known account is already assigned to this committee."
						/>
					{/if}
				</AppCard>
			</div>

			<AppCard title="Committee Members">
				{#if committeeState.data.users.length}
					<div class="overflow-x-auto">
						<table class="table table-zebra">
							<thead>
								<tr>
									<th>Username</th>
									<th>Full Name</th>
									<th>Role</th>
									<th>Quoted</th>
									<th>Managed By</th>
									<th class="text-right">Actions</th>
								</tr>
							</thead>
							<tbody>
								{#each committeeState.data.users as user}
									<tr>
										<td class="font-medium">{user.username}</td>
										<td>{user.fullName}</td>
										<td>
											<select
												class="select select-bordered select-sm"
												bind:value={membershipDrafts[user.userId].role}
												name="role"
												disabled={user.isOauthManaged}
											>
												<option value="member">Member</option>
												<option value="chairperson">Chairperson</option>
											</select>
										</td>
										<td>
											<input
												class="checkbox"
												type="checkbox"
												name="quoted"
												bind:checked={membershipDrafts[user.userId].quoted}
											/>
										</td>
										<td>
											<span class={`badge ${user.isOauthManaged ? 'badge-warning' : 'badge-ghost'}`}>
												{user.isOauthManaged ? 'OAuth' : 'Local'}
											</span>
										</td>
										<td>
											<div class="flex justify-end gap-2">
												<button
													class="btn btn-sm btn-outline"
													type="button"
													disabled={
														saveMembershipPendingId === user.userId ||
														deleteMembershipPendingId === user.userId
													}
													onclick={() => saveMembership(user.userId)}
												>
													{saveMembershipPendingId === user.userId ? 'Saving...' : 'Save'}
												</button>
												<button
													class="btn btn-sm btn-error btn-outline"
													type="button"
													disabled={
														saveMembershipPendingId === user.userId ||
														deleteMembershipPendingId === user.userId
													}
													onclick={() => deleteMembership(user)}
												>
													{deleteMembershipPendingId === user.userId ? 'Removing...' : 'Remove'}
												</button>
											</div>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{:else}
					<AppAlert tone="info" message="This committee has no assigned users yet." />
				{/if}
			</AppCard>

			<AppCard title="OAuth Group Rules">
				<form
					class="mb-6 grid gap-4 md:grid-cols-[minmax(0,1fr)_12rem_auto]"
					onsubmit={async (event) => {
						event.preventDefault();
						await createOAuthRule();
					}}
				>
					<label class="form-control gap-2">
						<span class="label-text font-medium">Group Name</span>
						<input class="input input-bordered" bind:value={oauthGroupName} required />
					</label>
					<label class="form-control gap-2">
						<span class="label-text font-medium">Role</span>
						<select class="select select-bordered" bind:value={oauthRole}>
							<option value="member">Member</option>
							<option value="chairperson">Chairperson</option>
						</select>
					</label>
					<div class="flex items-end">
						<button class="btn btn-primary w-full md:w-auto" type="submit" disabled={createRulePending}>
							{createRulePending ? 'Adding...' : 'Add Rule'}
						</button>
					</div>
				</form>

				{#if committeeState.data.oauthRules.length}
					<div class="overflow-x-auto">
						<table class="table table-zebra">
							<thead>
								<tr>
									<th>Group</th>
									<th>Role</th>
									<th class="text-right">Actions</th>
								</tr>
							</thead>
							<tbody>
								{#each committeeState.data.oauthRules as rule}
									<tr>
										<td class="font-medium">{rule.groupName}</td>
										<td>{rule.role}</td>
										<td>
											<div class="flex justify-end">
												<button
													class="btn btn-sm btn-error btn-outline"
													type="button"
													disabled={deleteRulePendingId === rule.ruleId}
													onclick={() => deleteOAuthRule(rule)}
												>
													{deleteRulePendingId === rule.ruleId ? 'Deleting...' : 'Delete'}
												</button>
											</div>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{:else}
					<AppAlert tone="info" message="No OAuth rules are configured for this committee." />
				{/if}
			</AppCard>
		{/if}
	{/if}
</div>
