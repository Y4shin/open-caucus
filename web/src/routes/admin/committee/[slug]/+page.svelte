<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
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
			<section class="panel card bg-base-100 border border-base-300 shadow-sm rounded-box p-4 mb-4">
				<h2>{m.admin_committee_users_assign_heading()}</h2>
				{#if committeeState.data.assignableAccounts.length === 0}
					<p>{m.admin_committee_users_no_assignable_accounts()}</p>
				{:else}
					<form
						onsubmit={(event) => {
							event.preventDefault();
							assignExistingAccount();
						}}
					>
						<div>
							<label for="account_id">{m.admin_committee_users_account_label()}</label>
							<select class="select select-bordered select-sm" id="account_id" name="account_id" bind:value={selectedAccountId} required>
								<option value="">{m.admin_committee_users_account_placeholder()}</option>
								{#each committeeState.data.assignableAccounts as account}
									<option value={account.accountId}>{account.fullName} ({account.username})</option>
								{/each}
							</select>
						</div>
						<div>
							<label for="role">{m.admin_committee_users_role_label()}</label>
							<select class="select select-bordered select-sm" id="role" name="role" bind:value={selectedAccountRole} required>
								<option value="member">{m.admin_committee_users_role_member()}</option>
								<option value="chairperson">{m.admin_committee_users_role_chairperson()}</option>
							</select>
						</div>
						<div>
							<label>
								<input class="checkbox checkbox-sm" type="checkbox" name="quoted" value="true" bind:checked={selectedAccountQuoted} />
								{m.admin_committee_users_quoted_label()}
							</label>
						</div>
						<button class="btn btn-sm" type="submit" disabled={assignAccountPending}>{m.admin_committee_users_assign_button()}</button>
					</form>
				{/if}
			</section>
			<section class="panel card bg-base-100 border border-base-300 shadow-sm rounded-box p-4 mb-4">
				<h2>{m.admin_committee_users_existing_heading()}</h2>
				{#if committeeState.data.users.length === 0}
					<p>{m.admin_committee_users_empty_state()}</p>
				{:else}
					<table class="data-table table table-zebra w-full">
						<thead>
							<tr>
								<th>{m.admin_committee_users_col_username()}</th>
								<th>{m.admin_committee_users_col_fullname()}</th>
								<th>{m.admin_committee_users_col_role()}</th>
								<th>{m.admin_committee_users_col_quoted()}</th>
								<th>{m.admin_committee_users_col_actions()}</th>
							</tr>
						</thead>
						<tbody>
							{#each committeeState.data.users as user}
								<tr>
									<td>{user.username}</td>
									<td>{user.fullName}</td>
									<td>
										<select
											class="select select-bordered select-xs"
											name="role"
											form={"membership-update-" + user.userId}
											disabled={user.isOauthManaged}
											value={membershipDrafts[user.userId]?.role ?? user.role}
											onchange={(event) => {
												const next = (event.currentTarget as HTMLSelectElement).value;
												membershipDrafts[user.userId] = { ...(membershipDrafts[user.userId] ?? { role: user.role, quoted: user.quoted }), role: next };
											}}
										>
											<option value="member" selected={(membershipDrafts[user.userId]?.role ?? user.role) === 'member'}>{m.admin_committee_users_role_member()}</option>
											<option value="chairperson" selected={(membershipDrafts[user.userId]?.role ?? user.role) === 'chairperson'}>{m.admin_committee_users_role_chairperson()}</option>
										</select>
										{#if user.isOauthManaged}
											<div class="text-xs text-base-content/70 mt-1">{m.admin_committee_users_oauth_managed_role_hint()}</div>
										{/if}
									</td>
									<td>
										<input type="hidden" name="quoted" value="false" form={"membership-update-" + user.userId} />
										<input
											class="checkbox checkbox-sm"
											type="checkbox"
											name="quoted"
											value="true"
											checked={membershipDrafts[user.userId]?.quoted ?? user.quoted}
											form={"membership-update-" + user.userId}
											onchange={(event) => {
												const next = (event.currentTarget as HTMLInputElement).checked;
												membershipDrafts[user.userId] = { ...(membershipDrafts[user.userId] ?? { role: user.role, quoted: user.quoted }), quoted: next };
											}}
										/>
									</td>
									<td>
										<form
											id={"membership-update-" + user.userId}
											class="inline-form inline"
											onsubmit={(event) => {
												event.preventDefault();
												saveMembership(user.userId);
											}}
										>
											{#if user.isOauthManaged}
												<input type="hidden" name="role" value={user.role} />
											{/if}
											<button class="btn btn-sm" type="submit" disabled={saveMembershipPendingId === user.userId}>{m.admin_committee_users_update_button()}</button>
										</form>
										<form
											class="inline-form inline ml-1"
											onsubmit={(event) => {
												event.preventDefault();
												deleteMembership(user);
											}}
										>
											<button class="btn btn-sm" type="submit" disabled={deleteMembershipPendingId === user.userId}>{m.admin_committee_users_delete_button()}</button>
										</form>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}
				<nav class="pagination-nav join">
					<button type="button" disabled class="ui-icon-label btn btn-sm">
						<LegacyIcon name="left" class="ui-icon--left" />
						<span class="ui-icon-text">{m.pagination_previous()}</span>
					</button>
					<button class="btn btn-sm" type="button" disabled>1</button>
					<button type="button" disabled class="ui-icon-label btn btn-sm">
						<span class="ui-icon-text">{m.pagination_next()}</span>
						<LegacyIcon name="right" class="ui-icon--right" />
					</button>
				</nav>
			</section>
			{#if session.oauthEnabled}
				<section class="panel card bg-base-100 border border-base-300 shadow-sm rounded-box p-4 mb-4">
					<h2>{m.admin_committee_users_oauth_rules_heading()}</h2>
					<form
						class="flex flex-wrap gap-2 items-end"
						onsubmit={(event) => {
							event.preventDefault();
							createOAuthRule();
						}}
					>
						<div>
							<label for="group_name">{m.admin_committee_users_oauth_group_label()}</label>
							<input class="input input-bordered input-sm" type="text" id="group_name" name="group_name" bind:value={oauthGroupName} required />
						</div>
						<div>
							<label for="oauth_rule_role">{m.admin_committee_users_role_label()}</label>
							<select class="select select-bordered select-sm" id="oauth_rule_role" name="role" bind:value={oauthRole} required>
								<option value="member">{m.admin_committee_users_role_member()}</option>
								<option value="chairperson">{m.admin_committee_users_role_chairperson()}</option>
							</select>
						</div>
						<button class="btn btn-sm" type="submit" disabled={createRulePending}>{m.admin_committee_users_oauth_rule_add_button()}</button>
					</form>
					{#if committeeState.data.oauthRules.length === 0}
						<p class="mt-2">{m.admin_committee_users_oauth_rules_empty()}</p>
					{:else}
						<table class="data-table table table-zebra w-full mt-2">
							<thead>
								<tr>
									<th>{m.admin_committee_users_oauth_group_label()}</th>
									<th>{m.admin_committee_users_col_role()}</th>
									<th>{m.admin_committee_users_col_actions()}</th>
								</tr>
							</thead>
							<tbody>
								{#each committeeState.data.oauthRules as rule}
									<tr>
										<td>{rule.groupName}</td>
										<td>{rule.role}</td>
										<td>
											<form
												class="inline-form inline"
												onsubmit={(event) => {
													event.preventDefault();
													deleteOAuthRule(rule);
												}}
											>
												<button class="btn btn-sm" type="submit" disabled={deleteRulePendingId === rule.ruleId}>{m.admin_dashboard_delete_button()}</button>
											</form>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					{/if}
				</section>
			{/if}
		{/if}
	{/if}
</div>
