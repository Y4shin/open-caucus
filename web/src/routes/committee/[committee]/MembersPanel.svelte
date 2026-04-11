<script lang="ts">
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSelect from '$lib/components/ui/AppSelect.svelte';
	import AppSwitch from '$lib/components/ui/AppSwitch.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import DataTable from '$lib/components/ui/DataTable.svelte';
	import { committeeClient } from '$lib/api/index.js';
	import type { MemberRecord, AssignableAccount } from '$lib/gen/conference/committees/v1/committees_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import * as m from '$lib/paraglide/messages';

	let { slug }: { slug: string } = $props();

	let members = $state<MemberRecord[]>([]);
	let assignableAccounts = $state<AssignableAccount[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Add member form state
	let addMode = $state<'email' | 'account'>('email');
	let addEmail = $state('');
	let addName = $state('');
	let addRole = $state('member');
	let addQuoted = $state(false);
	let addAccountId = $state('');
	let addAccountRole = $state('member');
	let addAccountQuoted = $state(false);
	let addPending = $state(false);

	// Inline edit drafts
	let editDrafts = $state<Record<string, { role: string; quoted: boolean }>>({});
	let savePendingId = $state('');
	let removePendingId = $state('');

	$effect(() => {
		loadMembers();
	});

	async function loadMembers() {
		loading = true;
		error = '';
		try {
			const [membersRes, accountsRes] = await Promise.all([
				committeeClient.listCommitteeMembers({ committeeSlug: slug }),
				committeeClient.listAssignableAccounts({ committeeSlug: slug })
			]);
			members = membersRes.members;
			assignableAccounts = accountsRes.accounts;
			editDrafts = Object.fromEntries(
				members.map((member) => [member.userId, { role: member.role, quoted: member.quoted }])
			);
		} catch (err) {
			error = getDisplayError(err, 'Failed to load members.');
		} finally {
			loading = false;
		}
	}

	async function addByEmail() {
		addPending = true;
		error = '';
		try {
			await committeeClient.addMemberByEmail({
				committeeSlug: slug,
				email: addEmail.trim(),
				fullName: addName.trim(),
				role: addRole,
				quoted: addQuoted
			});
			addEmail = '';
			addName = '';
			addRole = 'member';
			addQuoted = false;
			await loadMembers();
		} catch (err) {
			error = getDisplayError(err, 'Failed to add member by email.');
		} finally {
			addPending = false;
		}
	}

	async function assignAccount() {
		addPending = true;
		error = '';
		try {
			await committeeClient.assignAccountToCommittee({
				committeeSlug: slug,
				accountId: addAccountId,
				role: addAccountRole,
				quoted: addAccountQuoted
			});
			addAccountId = '';
			addAccountRole = 'member';
			addAccountQuoted = false;
			await loadMembers();
		} catch (err) {
			error = getDisplayError(err, 'Failed to assign account.');
		} finally {
			addPending = false;
		}
	}

	async function saveMember(userId: string) {
		const draft = editDrafts[userId];
		if (!draft) return;
		savePendingId = userId;
		error = '';
		try {
			await committeeClient.updateMember({
				committeeSlug: slug,
				userId,
				role: draft.role,
				quoted: draft.quoted
			});
			await loadMembers();
		} catch (err) {
			error = getDisplayError(err, 'Failed to update member.');
		} finally {
			savePendingId = '';
		}
	}

	async function removeMember(userId: string) {
		if (!window.confirm(m.members_remove_confirm())) return;
		removePendingId = userId;
		error = '';
		try {
			await committeeClient.removeMember({ committeeSlug: slug, userId });
			await loadMembers();
		} catch (err) {
			error = getDisplayError(err, 'Failed to remove member.');
		} finally {
			removePendingId = '';
		}
	}

	function contactDisplay(member: MemberRecord): string | null {
		if (member.email) return member.email;
		if (member.username) return member.username;
		return null;
	}

	const roleItems = [
		{ value: 'member', label: m.admin_committee_users_role_member() },
		{ value: 'chairperson', label: m.admin_committee_users_role_chairperson() }
	];
</script>

<section class="min-w-0 rounded-box border border-base-300 bg-base-200 p-4 mt-4">
	<h3 class="text-lg font-semibold mb-3">{m.members_heading()}</h3>

	{#if error}
		<AppAlert message={error} />
	{/if}

	{#if loading}
		<AppSpinner label="Loading members" />
	{:else}
		<!-- Add member section -->
		<div class="rounded-box border border-base-300 bg-base-100 p-4 mb-4">
			<h4 class="text-sm font-semibold mb-2">{m.members_add_heading()}</h4>
			<div class="tabs tabs-bordered mb-3">
				<button
					type="button"
					class="tab {addMode === 'email' ? 'tab-active' : ''}"
					onclick={() => addMode = 'email'}
				>{m.members_add_by_email()}</button>
				<button
					type="button"
					class="tab {addMode === 'account' ? 'tab-active' : ''}"
					onclick={() => addMode = 'account'}
				>{m.members_add_assign_account()}</button>
			</div>

			{#if addMode === 'email'}
				<form
					class="grid gap-3 sm:grid-cols-[1fr_1fr_auto_auto_auto] sm:items-end"
					onsubmit={(event) => { event.preventDefault(); addByEmail(); }}
				>
					<div>
						<label class="label text-sm font-medium" for="member-email">{m.members_email_label()}</label>
						<input class="input input-bordered input-sm w-full" type="email" id="member-email" bind:value={addEmail} required />
					</div>
					<div>
						<label class="label text-sm font-medium" for="member-name">{m.members_name_label()}</label>
						<input class="input input-bordered input-sm w-full" type="text" id="member-name" bind:value={addName} required />
					</div>
					<div>
						<label class="label text-sm font-medium" for="member-role">{m.members_role_label()}</label>
						<AppSelect bind:value={addRole} id="member-role" items={roleItems} />
					</div>
					<AppSwitch bind:checked={addQuoted} label={m.members_col_quoted()} />
					<button class="btn btn-sm btn-primary" type="submit" disabled={addPending}>{m.members_add_button()}</button>
				</form>
			{:else}
				<form
					class="grid gap-3 sm:grid-cols-[1fr_auto_auto_auto] sm:items-end"
					onsubmit={(event) => { event.preventDefault(); assignAccount(); }}
				>
					<div>
						<label class="label text-sm font-medium" for="member-account">{m.members_add_assign_account()}</label>
						<AppSelect
							bind:value={addAccountId}
							id="member-account"
							placeholder="Select account..."
							items={assignableAccounts.map((a) => ({ value: a.accountId, label: `${a.fullName} (${a.username})` }))}
						/>
					</div>
					<div>
						<label class="label text-sm font-medium" for="member-account-role">{m.members_role_label()}</label>
						<AppSelect bind:value={addAccountRole} id="member-account-role" items={roleItems} />
					</div>
					<AppSwitch bind:checked={addAccountQuoted} label={m.members_col_quoted()} />
					<button class="btn btn-sm btn-primary" type="submit" disabled={addPending || !addAccountId}>{m.members_add_button()}</button>
				</form>
			{/if}
		</div>

		<!-- Members table -->
		{#if members.length === 0}
			<p class="text-base-content/70">{m.members_empty()}</p>
		{:else}
			<DataTable>
				{#snippet header()}
					<tr>
						<th>{m.members_col_name()}</th>
						<th>{m.members_col_contact()}</th>
						<th>{m.members_col_role()}</th>
						<th>{m.members_col_quoted()}</th>
						<th class="text-right">{m.members_col_actions()}</th>
					</tr>
				{/snippet}
				{#snippet body()}
					{#each members as member}
						<tr>
							<td>{member.fullName}</td>
							<td>
								{#if contactDisplay(member)}
									{contactDisplay(member)}
								{:else}
									<span class="badge badge-ghost badge-sm">{m.members_no_contact()}</span>
								{/if}
							</td>
							<td>
								<AppSelect
									value={editDrafts[member.userId]?.role ?? member.role}
									size="xs"
									items={roleItems}
									onValueChange={(next) => {
										editDrafts[member.userId] = {
											...(editDrafts[member.userId] ?? { role: member.role, quoted: member.quoted }),
											role: next
										};
									}}
								/>
							</td>
							<td>
								<AppSwitch
									checked={editDrafts[member.userId]?.quoted ?? member.quoted}
									size="xs"
									onCheckedChange={(next) => {
										editDrafts[member.userId] = {
											...(editDrafts[member.userId] ?? { role: member.role, quoted: member.quoted }),
											quoted: next
										};
									}}
								/>
							</td>
							<td class="text-right">
								<div class="flex items-center justify-end gap-1">
									<button
										class="btn btn-xs btn-primary btn-outline"
										type="button"
										disabled={savePendingId === member.userId}
										onclick={() => saveMember(member.userId)}
									>Save</button>
									<button
										class="btn btn-xs btn-error btn-outline"
										type="button"
										disabled={removePendingId === member.userId}
										onclick={() => removeMember(member.userId)}
									>Remove</button>
								</div>
							</td>
						</tr>
					{/each}
				{/snippet}
			</DataTable>
		{/if}
	{/if}
</section>
