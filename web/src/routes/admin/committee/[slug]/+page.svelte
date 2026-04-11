<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSelect from '$lib/components/ui/AppSelect.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import DataTable from '$lib/components/ui/DataTable.svelte';
	import PaginationNav from '$lib/components/ui/PaginationNav.svelte';
	import { adminClient } from '$lib/api/index.js';
	import type { AccountRecord, CommitteeRecord, CommitteeUserRecord, OAuthRuleRecord } from '$lib/gen/conference/admin/v1/admin_pb.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import * as m from '$lib/paraglide/messages';

	interface CommitteeAdminData {
		committee: CommitteeRecord;
		users: CommitteeUserRecord[];
		oauthRules: OAuthRuleRecord[];
		oauthGroupPrefix: string;
		assignableAccounts: AccountRecord[];
	}

	const slug = $derived(page.params.slug);

	let committeeState = $state(createRemoteState<CommitteeAdminData>());
	let assignAccountPending = $state(false);
	let saveMembershipPendingId = $state('');
	let deleteMembershipPendingId = $state('');
	let createRulePending = $state(false);
	let deleteRulePendingId = $state('');

	let membershipDrafts = $state<Record<string, { role: string; quoted: boolean }>>({});
	let selectedAccountId = $state('');
	let selectedAccountRole = $state('member');
	let selectedAccountQuoted = $state(false);
	let oauthGroupName = $state('');
	let oauthRole = $state('member');

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
		loadCommitteeAdmin();
	});

	function setMembershipDrafts(users: CommitteeUserRecord[]) {
		membershipDrafts = Object.fromEntries(users.map((user) => [user.userId, { role: user.role, quoted: user.quoted }]));
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
			const assignedUsernames = new Set(users.map((user) => user.username));
			const assignableAccounts = accounts.accounts.filter((account) => !assignedUsernames.has(account.username));
			setMembershipDrafts(users);
			committeeState.data = {
				committee: committeeAdmin.committee!,
				users,
				oauthRules: committeeAdmin.oauthRules,
				oauthGroupPrefix: committeeAdmin.oauthGroupPrefix,
				assignableAccounts
			};
		} catch (err) {
			committeeState.error = getDisplayError(err, `Failed to load admin settings for committee "${slug}".`);
		} finally {
			committeeState.loading = false;
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
		if (!window.confirm(m.admin_committee_users_delete_confirm())) {
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
		if (!window.confirm(m.admin_committee_users_delete_confirm())) {
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

<div id="committee-users-container">
	{#if committeeState.loading}
		<AppSpinner label="Loading committee admin" />
	{:else}
		{#if committeeState.error}
			<AppAlert message={committeeState.error} />
		{/if}
		{#if committeeState.data}
			<div class="space-y-4">
			<AppCard class="bg-base-100 shadow-sm">
				<h2 class="text-base font-semibold mb-3">{m.admin_committee_users_assign_heading()}</h2>
				{#if committeeState.data.assignableAccounts.length === 0}
					<p class="text-sm text-base-content/70">{m.admin_committee_users_no_assignable_accounts()}</p>
				{:else}
					<form
						class="grid gap-3 sm:grid-cols-[1fr_auto_auto_auto] sm:items-end"
						onsubmit={(event) => {
							event.preventDefault();
							assignExistingAccount();
						}}
					>
						<div>
							<label class="label text-sm font-medium" for="account_id">{m.admin_committee_users_account_label()}</label>
							<AppSelect
								bind:value={selectedAccountId}
								id="account_id"
								placeholder={m.admin_committee_users_account_placeholder()}
								items={committeeState.data.assignableAccounts.map((a) => ({ value: a.accountId, label: `${a.fullName} (${a.username})` }))}
							/>
						</div>
						<div>
							<label class="label text-sm font-medium" for="role">{m.admin_committee_users_role_label()}</label>
							<AppSelect
								bind:value={selectedAccountRole}
								id="role"
								items={[
									{ value: 'member', label: m.admin_committee_users_role_member() },
									{ value: 'chairperson', label: m.admin_committee_users_role_chairperson() }
								]}
							/>
						</div>
						<label class="label cursor-pointer justify-start gap-2 p-0 sm:mb-1">
							<input class="checkbox checkbox-sm" type="checkbox" name="quoted" value="true" bind:checked={selectedAccountQuoted} />
							<span class="text-sm">{m.admin_committee_users_quoted_label()}</span>
						</label>
						<button class="btn btn-sm btn-primary" type="submit" disabled={assignAccountPending}>{m.admin_committee_users_assign_button()}</button>
					</form>
				{/if}
			</AppCard>
			<AppCard class="bg-base-100 shadow-sm">
				<h2 class="text-base font-semibold mb-3">{m.admin_committee_users_existing_heading()}</h2>
				{#if committeeState.data.users.length === 0}
					<p class="text-sm text-base-content/70">{m.admin_committee_users_empty_state()}</p>
				{:else}
					<DataTable>
						{#snippet header()}
							<tr>
								<th>{m.admin_committee_users_col_username()}</th>
								<th>{m.admin_committee_users_col_fullname()}</th>
								<th>{m.admin_committee_users_col_role()}</th>
								<th>{m.admin_committee_users_col_quoted()}</th>
								<th class="text-right">{m.admin_committee_users_col_actions()}</th>
							</tr>
						{/snippet}
						{#snippet body()}
							{#each committeeState.data?.users ?? [] as user}
								<tr>
									<td>{user.username}</td>
									<td>{user.fullName}</td>
									<td>
										<AppSelect
											value={membershipDrafts[user.userId]?.role ?? user.role}
											disabled={user.isOauthManaged}
											size="xs"
											items={[
												{ value: 'member', label: m.admin_committee_users_role_member() },
												{ value: 'chairperson', label: m.admin_committee_users_role_chairperson() }
											]}
											onValueChange={(next) => {
												membershipDrafts[user.userId] = { ...(membershipDrafts[user.userId] ?? { role: user.role, quoted: user.quoted }), role: next };
											}}
										/>
										{#if user.isOauthManaged}
											<p class="text-[0.65rem] text-base-content/50 mt-0.5">{m.admin_committee_users_oauth_managed_role_hint()}</p>
										{/if}
									</td>
									<td>
										<input
											class="checkbox checkbox-sm"
											type="checkbox"
											name="quoted"
											checked={membershipDrafts[user.userId]?.quoted ?? user.quoted}
											onchange={(event) => {
												const next = (event.currentTarget as HTMLInputElement).checked;
												membershipDrafts[user.userId] = { ...(membershipDrafts[user.userId] ?? { role: user.role, quoted: user.quoted }), quoted: next };
											}}
										/>
									</td>
									<td class="text-right">
										<div class="flex items-center justify-end gap-1">
											<button class="btn btn-xs btn-primary btn-outline" type="button" disabled={saveMembershipPendingId === user.userId} onclick={() => saveMembership(user.userId)}>{m.admin_committee_users_update_button()}</button>
											<button class="btn btn-xs btn-error btn-outline" type="button" disabled={deleteMembershipPendingId === user.userId} onclick={() => deleteMembership(user)}>{m.admin_committee_users_delete_button()}</button>
										</div>
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
			{#if session.oauthEnabled}
				<AppCard class="bg-base-100 shadow-sm">
					<h2 class="text-base font-semibold mb-3">{m.admin_committee_users_oauth_rules_heading()}</h2>
					<form
						class="grid gap-3 sm:grid-cols-[1fr_auto_auto] sm:items-end mb-3"
						onsubmit={(event) => {
							event.preventDefault();
							createOAuthRule();
						}}
					>
						<div>
							<label class="label text-sm font-medium" for="group_name">{m.admin_committee_users_oauth_group_label()}</label>
							<input class="input input-bordered input-sm w-full" type="text" id="group_name" name="group_name" bind:value={oauthGroupName} placeholder={committeeState.data?.oauthGroupPrefix || ''} required />
						</div>
						<div>
							<label class="label text-sm font-medium" for="oauth_rule_role">{m.admin_committee_users_role_label()}</label>
							<AppSelect
								bind:value={oauthRole}
								id="oauth_rule_role"
								items={[
									{ value: 'member', label: m.admin_committee_users_role_member() },
									{ value: 'chairperson', label: m.admin_committee_users_role_chairperson() }
								]}
							/>
						</div>
						<button class="btn btn-sm btn-primary" type="submit" disabled={createRulePending}>{m.admin_committee_users_oauth_rule_add_button()}</button>
					</form>
					{#if (committeeState.data?.oauthRules.length ?? 0) === 0}
						<p class="text-sm text-base-content/70">{m.admin_committee_users_oauth_rules_empty()}</p>
					{:else}
						<DataTable>
							{#snippet header()}
								<tr>
									<th>{m.admin_committee_users_oauth_group_label()}</th>
									<th>{m.admin_committee_users_col_role()}</th>
									<th class="text-right">{m.admin_committee_users_col_actions()}</th>
								</tr>
							{/snippet}
							{#snippet body()}
								{#each committeeState.data?.oauthRules ?? [] as rule}
									<tr>
										<td>{rule.groupName}</td>
										<td>{rule.role}</td>
										<td class="text-right">
											<button class="btn btn-xs btn-error btn-outline" type="button" disabled={deleteRulePendingId === rule.ruleId} onclick={() => deleteOAuthRule(rule)}>{m.admin_dashboard_delete_button()}</button>
										</td>
									</tr>
								{/each}
							{/snippet}
						</DataTable>
					{/if}
				</AppCard>
			{/if}
			</div>
		{/if}
	{/if}
</div>
