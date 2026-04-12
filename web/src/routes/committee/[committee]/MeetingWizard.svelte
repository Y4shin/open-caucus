<script lang="ts">
	import { Dialog } from 'bits-ui';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import AppSwitch from '$lib/components/ui/AppSwitch.svelte';
	import DateRangeTimePicker from '$lib/components/ui/DateRangeTimePicker.svelte';
	import { agendaClient, committeeClient, moderationClient } from '$lib/api/index.js';
	import type { MemberRecord } from '$lib/gen/conference/committees/v1/committees_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import * as m from '$lib/paraglide/messages';
	import AgendaImportPreview from './meeting/[meetingId]/moderate/AgendaImportPreview.svelte';
	import type { AgendaImportState } from './meeting/[meetingId]/moderate/agenda-import.js';
	import { parseAgendaImportSource, buildImportedAgenda } from './meeting/[meetingId]/moderate/agenda-import.js';
	import { type CalendarDateTime } from '@internationalized/date';
	import { calendarDateTimeToUTC, formatCalendarDateTime } from '$lib/utils/datetime.js';

	let {
		slug,
		onCreated,
		emailEnabled = false
	}: {
		slug: string;
		onCreated: () => void;
		emailEnabled?: boolean;
	} = $props();

	let dialogOpen = $state(false);
	let step = $state<1 | 2 | 3 | 4>(1);
	let submitting = $state(false);
	let error = $state('');

	// Step 1: Basic data
	let meetingName = $state('');
	let meetingDescription = $state('');
	let signupOpen = $state(false);
	let meetingStartAt = $state<CalendarDateTime | undefined>(undefined);
	let meetingEndAt = $state<CalendarDateTime | undefined>(undefined);

	// Step 2: Agenda — reuses the inline editor from the import dialog
	let agendaRawText = $state('');
	let agendaParsedText = $state('');
	let agendaFormat = $state<'markdown' | 'plaintext'>('plaintext');
	let agendaFormatManuallySet = $state(false);
	let agendaLineStates = $state(new Map<number, AgendaImportState>());

	let agendaLines = $derived(
		parseAgendaImportSource(agendaParsedText, agendaFormat).map((line) => ({
			...line,
			state: agendaLineStates.get(line.lineNo) ?? line.state
		}))
	);
	let parsedAgenda = $derived(buildImportedAgenda(agendaLines));

	function setAgendaFromSource(source: string) {
		const trimmed = source.trim();
		if (!trimmed) return;
		if (!agendaFormatManuallySet) {
			agendaFormat =
				/^#{1,6}\s/m.test(trimmed) || /^[-*+]\s+\S/m.test(trimmed) ? 'markdown' : 'plaintext';
		}
		agendaLineStates = new Map();
		agendaRawText = trimmed;
		agendaParsedText = trimmed;
	}

	function toggleAgendaLine(index: number) {
		const line = agendaLines[index];
		if (!line) return;
		const nextState: AgendaImportState =
			line.state === 'ignore' ? 'heading' : line.state === 'heading' ? 'subheading' : 'ignore';
		agendaLineStates = new Map(agendaLineStates).set(line.lineNo, nextState);
	}

	function setAgendaFormat(fmt: 'markdown' | 'plaintext') {
		agendaFormat = fmt;
		agendaFormatManuallySet = true;
		agendaLineStates = new Map();
	}

	// Sync parsed text when raw text changes (user edits the textarea directly)
	$effect(() => {
		if (agendaRawText !== agendaParsedText) {
			agendaParsedText = agendaRawText;
		}
	});

	// Step 3: Invites (member checklist)
	let membersList = $state<MemberRecord[]>([]);
	let selectedMemberIds = $state<Set<string>>(new Set());
	let membersLoading = $state(false);
	let sendInvites = $state(false);

	export function open() {
		step = 1;
		meetingName = '';
		meetingDescription = '';
		signupOpen = false;
		meetingStartAt = undefined;
		meetingEndAt = undefined;
		agendaRawText = '';
		agendaParsedText = '';
		agendaFormat = 'plaintext';
		agendaFormatManuallySet = false;
		agendaLineStates = new Map();
		membersList = [];
		selectedMemberIds = new Set();
		sendInvites = emailEnabled;
		error = '';
		submitting = false;
		dialogOpen = true;
		loadMembersForInvites();
	}

	async function loadMembersForInvites() {
		membersLoading = true;
		try {
			const res = await committeeClient.listCommitteeMembers({ committeeSlug: slug });
			membersList = res.members;
			selectedMemberIds = new Set(
				membersList
					.filter((member) => !!(member.email || member.username))
					.map((member) => member.userId)
			);
		} catch {
			membersList = [];
			selectedMemberIds = new Set();
		} finally {
			membersLoading = false;
		}
	}

	function toggleMember(userId: string) {
		const next = new Set(selectedMemberIds);
		if (next.has(userId)) {
			next.delete(userId);
		} else {
			next.add(userId);
		}
		selectedMemberIds = next;
	}

	function selectAllMembers() {
		selectedMemberIds = new Set(
			membersList
				.filter((member) => !!(member.email || member.username))
				.map((member) => member.userId)
		);
	}

	function deselectAllMembers() {
		selectedMemberIds = new Set();
	}

	function canProceed(): boolean {
		if (step === 1) return meetingName.trim().length > 0;
		return true;
	}

	function nextStep() {
		if (step < 4) step = (step + 1) as 1 | 2 | 3 | 4;
	}

	function prevStep() {
		if (step > 1) step = (step - 1) as 1 | 2 | 3 | 4;
	}

	async function submit() {
		submitting = true;
		error = '';
		try {
			const meetingRes = await committeeClient.createMeeting({
				committeeSlug: slug,
				name: meetingName.trim(),
				description: meetingDescription.trim(),
				startAt: meetingStartAt ? calendarDateTimeToUTC(meetingStartAt) : undefined,
				endAt: meetingEndAt ? calendarDateTimeToUTC(meetingEndAt) : undefined
			});
			const meetingId = meetingRes.meeting?.meetingId ?? '';
			if (!meetingId) throw new Error('Meeting creation returned no ID');

			for (const point of parsedAgenda) {
				const pointRes = await agendaClient.createAgendaPoint({
					committeeSlug: slug,
					meetingId,
					title: point.title
				});
				const parentId = pointRes.agendaPoint?.agendaPointId ?? '';
				for (const child of point.children) {
					await agendaClient.createAgendaPoint({
						committeeSlug: slug,
						meetingId,
						title: child,
						parentAgendaPointId: parentId
					});
				}
			}

			if (sendInvites && emailEnabled && selectedMemberIds.size > 0) {
				await committeeClient.sendInviteEmails({
					committeeSlug: slug,
					meetingId,
					baseUrl: window.location.origin,
					memberIds: [...selectedMemberIds]
				});
			}

			if (signupOpen) {
				await moderationClient.toggleSignupOpen({
					committeeSlug: slug,
					meetingId,
					desiredOpen: true,
					expectedVersion: 0n
				});
			}

			dialogOpen = false;
			onCreated();
		} catch (err) {
			error = getDisplayError(err, 'Failed to create meeting.');
		} finally {
			submitting = false;
		}
	}
</script>

<Dialog.Root bind:open={dialogOpen}>
	<Dialog.Portal>
	<Dialog.Overlay class="fixed inset-0 z-40 bg-black/50" />
	<Dialog.Content class="fixed left-1/2 top-1/2 z-50 -translate-x-1/2 -translate-y-1/2 w-11/12 max-w-3xl rounded-box bg-base-100 p-6 shadow-xl max-h-[90vh] overflow-y-auto">
		<div class="mb-4 flex items-center justify-between">
			<Dialog.Title class="text-lg font-semibold">{m.committee_create_meeting_heading()}</Dialog.Title>
			<div class="flex items-center gap-2">
				<ul class="steps steps-horizontal text-xs">
					<li class={step >= 1 ? 'step step-primary' : 'step'}>{m.wizard_step_basics()}</li>
					<li class={step >= 2 ? 'step step-primary' : 'step'}>{m.wizard_step_agenda()}</li>
					<li class={step >= 3 ? 'step step-primary' : 'step'}>{m.wizard_step_invites()}</li>
					<li class={step >= 4 ? 'step step-primary' : 'step'}>{m.wizard_step_review()}</li>
				</ul>
				<button type="button" class="btn btn-sm btn-ghost" onclick={() => dialogOpen = false}>{m.common_close()}</button>
			</div>
		</div>

		{#if error}
			<AppAlert message={error} />
		{/if}

		{#if step === 1}
			<div class="space-y-3">
				<div>
					<label class="label text-sm font-medium" for="wizard-name">{m.committee_name_label()}</label>
					<input class="input input-bordered input-sm w-full" type="text" id="wizard-name" bind:value={meetingName} required />
				</div>
				<div>
					<label class="label text-sm font-medium" for="wizard-desc">{m.committee_description_label()}</label>
					<input class="input input-bordered input-sm w-full" id="wizard-desc" bind:value={meetingDescription} />
				</div>
				<AppSwitch bind:checked={signupOpen} label={m.committee_signup_label()} />
				<DateRangeTimePicker
					bind:startValue={meetingStartAt}
					bind:endValue={meetingEndAt}
					startLabel={m.wizard_start_at_label()}
					endLabel={m.wizard_end_at_label()}
				/>
			</div>
		{:else if step === 2}
			<div class="space-y-3">
				<div class="flex items-center justify-between gap-2">
					<p class="text-sm text-base-content/70">{agendaFormat === 'markdown' ? m.wizard_agenda_description_markdown() : m.wizard_agenda_description_plaintext()}</p>
					<div class="join">
						<button type="button" class={agendaFormat === 'plaintext' ? 'join-item btn btn-xs btn-primary' : 'join-item btn btn-xs btn-ghost'} onclick={() => setAgendaFormat('plaintext')}>Plaintext</button>
						<button type="button" class={agendaFormat === 'markdown' ? 'join-item btn btn-xs btn-primary' : 'join-item btn btn-xs btn-ghost'} onclick={() => setAgendaFormat('markdown')}>Markdown</button>
					</div>
				</div>
				<AgendaImportPreview
					bind:rawText={agendaRawText}
					lines={agendaLines}
					onToggle={toggleAgendaLine}
					onPasteText={setAgendaFromSource}
				/>
			</div>
		{:else if step === 3}
			<div class="space-y-3">
				<p class="text-sm text-base-content/70">{m.wizard_invites_description()}</p>
				{#if !emailEnabled}
					<div class="alert alert-warning text-sm">{m.email_not_configured_hint()}</div>
				{:else}
					<AppSwitch
						checked={sendInvites}
						label={m.wizard_send_invites_toggle()}
						onCheckedChange={(v) => { sendInvites = v; }}
					/>
				{/if}
				{#if membersLoading}
					<AppSpinner label="Loading members" />
				{:else if membersList.length === 0}
					<p class="text-sm text-base-content/70">{m.wizard_invites_description()}</p>
				{:else}
					<div class="flex gap-2 mb-2">
						<button type="button" class="btn btn-xs btn-ghost" disabled={!sendInvites || !emailEnabled} onclick={selectAllMembers}>Select all</button>
						<button type="button" class="btn btn-xs btn-ghost" disabled={!sendInvites || !emailEnabled} onclick={deselectAllMembers}>Deselect all</button>
					</div>
					<ul class="space-y-1 max-h-64 overflow-y-auto rounded-box border border-base-300 bg-base-200/30 p-3 {!sendInvites || !emailEnabled ? 'opacity-50' : ''}">
						{#each membersList as member}
							{@const hasContact = !!(member.email || member.username)}
							<li class="flex items-center gap-2 {hasContact ? '' : 'opacity-50'}">
								<input
									class="checkbox checkbox-sm"
									type="checkbox"
									checked={selectedMemberIds.has(member.userId)}
									disabled={!hasContact || !sendInvites || !emailEnabled}
									onchange={() => toggleMember(member.userId)}
								/>
								<span class="text-sm">{member.fullName}</span>
								{#if member.email}
									<span class="badge badge-outline badge-xs">{member.email}</span>
								{:else if member.username}
									<span class="badge badge-outline badge-xs">{member.username}</span>
								{:else}
									<span class="badge badge-ghost badge-xs">{m.members_no_contact()}</span>
								{/if}
								<span class="badge badge-sm {member.role === 'chairperson' ? 'badge-primary' : 'badge-neutral'}">{member.role === 'chairperson' ? m.admin_committee_users_role_chairperson() : m.admin_committee_users_role_member()}</span>
							</li>
						{/each}
					</ul>
				{/if}
			</div>
		{:else if step === 4}
			<div class="space-y-4">
				<div class="rounded-box border border-base-300 bg-base-200/30 p-3 space-y-2">
					<h4 class="text-sm font-semibold">{m.wizard_step_basics()}</h4>
					<p class="text-sm"><strong>{meetingName}</strong></p>
					{#if meetingDescription}<p class="text-sm text-base-content/70">{meetingDescription}</p>{/if}
					{#if signupOpen}<span class="badge badge-outline badge-sm">{m.committee_signup_label()}</span>{/if}
					{#if meetingStartAt || meetingEndAt}
						<p class="text-sm text-base-content/70">
							{#if meetingStartAt}{m.wizard_start_at_label()} {formatCalendarDateTime(meetingStartAt)}{/if}
							{#if meetingStartAt && meetingEndAt} &mdash; {/if}
							{#if meetingEndAt}{m.wizard_end_at_label()} {formatCalendarDateTime(meetingEndAt)}{/if}
						</p>
					{/if}
				</div>
				{#if parsedAgenda.length > 0}
					<div class="rounded-box border border-base-300 bg-base-200/30 p-3 space-y-2">
						<h4 class="text-sm font-semibold">{m.wizard_step_agenda()} ({parsedAgenda.length})</h4>
						<ul class="text-sm space-y-0.5">
							{#each parsedAgenda as point, i}
								<li>{i + 1}. {point.title}{#if point.children.length > 0} <span class="text-base-content/50">(+{point.children.length} sub)</span>{/if}</li>
							{/each}
						</ul>
					</div>
				{/if}
				<div class="rounded-box border border-base-300 bg-base-200/30 p-3 space-y-2">
					<h4 class="text-sm font-semibold">{m.wizard_step_invites()}</h4>
					{#if sendInvites && emailEnabled}
						<p class="text-sm text-base-content/70">{m.wizard_invites_count({ count: selectedMemberIds.size })}</p>
					{:else}
						<p class="text-sm {!emailEnabled ? 'text-warning' : 'text-base-content/70'}">{m.wizard_invites_will_not_send()}</p>
					{/if}
				</div>
			</div>
		{/if}

		<div class="modal-action">
			{#if step > 1}
				<button type="button" class="btn btn-sm btn-ghost" onclick={prevStep} disabled={submitting}>{m.common_back()}</button>
			{/if}
			<div class="flex-1"></div>
			{#if step < 4}
				<button type="button" class="btn btn-sm btn-primary" onclick={nextStep} disabled={!canProceed()}>{m.wizard_next()}</button>
			{:else}
				{#if submitting}
					<AppSpinner label="Creating meeting..." />
				{/if}
				<button type="button" class="btn btn-sm btn-primary" onclick={submit} disabled={submitting || !canProceed()}>{m.wizard_create()}</button>
			{/if}
		</div>
	</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
