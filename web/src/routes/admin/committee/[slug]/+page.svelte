<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSelect from '$lib/components/ui/AppSelect.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import DataTable from '$lib/components/ui/DataTable.svelte';
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
	let createRulePending = $state(false);
	let deleteRulePendingId = $state('');

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

	async function loadCommitteeAdmin() {
		committeeState.loading = true;
		committeeState.error = '';
		try {
			const committeeAdmin = await adminClient.getCommitteeAdmin({ slug });
			committeeState.data = {
				committee: committeeAdmin.committee!,
				users: committeeAdmin.users,
				oauthRules: committeeAdmin.oauthRules,
				oauthGroupPrefix: committeeAdmin.oauthGroupPrefix,
				assignableAccounts: []
			};
		} catch (err) {
			committeeState.error = getDisplayError(err, `Failed to load admin settings for committee "${slug}".`);
		} finally {
			committeeState.loading = false;
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
				<h2 class="text-base font-semibold mb-3">{m.members_heading()}</h2>
				<p class="text-sm text-base-content/70 mb-2">Member management has moved to the committee view.</p>
				<a href={`/committee/${page.params.slug}`} class="btn btn-sm btn-primary">Manage Members</a>
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
