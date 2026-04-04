<script lang="ts">
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { agendaClient, attendeeClient, committeeClient } from '$lib/api/index.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import * as m from '$lib/paraglide/messages';

	let {
		slug,
		onCreated
	}: {
		slug: string;
		onCreated: () => void;
	} = $props();

	let dialogEl = $state<HTMLDialogElement | null>(null);
	let step = $state<1 | 2 | 3 | 4>(1);
	let submitting = $state(false);
	let error = $state('');

	// Step 1: Basic data
	let meetingName = $state('');
	let meetingDescription = $state('');
	let signupOpen = $state(false);

	// Step 2: Agenda (text-based)
	let agendaText = $state('');

	// Step 3: Participants (text-based)
	let participantsText = $state('');

	// Parsed state
	interface ParsedAgendaPoint {
		title: string;
		children: string[];
	}

	interface ParsedParticipant {
		name: string;
		isChair: boolean;
		isQuoted: boolean;
	}

	function parseAgenda(text: string): ParsedAgendaPoint[] {
		const lines = text.split('\n').map((l) => l.trimEnd()).filter((l) => l.trim().length > 0);
		const points: ParsedAgendaPoint[] = [];
		let current: ParsedAgendaPoint | null = null;

		for (const line of lines) {
			const indent = line.length - line.trimStart().length;
			const trimmed = line.trim().replace(/^(?:TOP\s*)?(\d+(?:\.\d+)*)[\s:.)\-]+/i, '').trim();
			if (!trimmed) continue;

			if (indent >= 2 || /^\d+\.\d+/.test(line.trim())) {
				// Subpoint
				if (current) {
					current.children.push(trimmed);
				} else {
					current = { title: trimmed, children: [] };
					points.push(current);
				}
			} else {
				current = { title: trimmed, children: [] };
				points.push(current);
			}
		}
		return points;
	}

	function parseParticipants(text: string): ParsedParticipant[] {
		const lines = text.split('\n').filter((l) => l.trim().length > 0);
		return lines.map((line) => {
			let name = line.trim();
			let isChair = false;
			let isQuoted = false;

			// Detect flags in brackets or suffixes
			const chairPattern = /\[(?:chair|vorsitz|chairperson)\]/i;
			const quotedPattern = /\[(?:flinta\*?|f|quoted)\]/i;

			if (chairPattern.test(name)) {
				isChair = true;
				name = name.replace(chairPattern, '').trim();
			}
			if (quotedPattern.test(name)) {
				isQuoted = true;
				name = name.replace(quotedPattern, '').trim();
			}

			// Also detect trailing markers like * for FLINTA or ^ for chair
			if (name.endsWith('*')) {
				isQuoted = true;
				name = name.slice(0, -1).trim();
			}
			if (name.endsWith('^')) {
				isChair = true;
				name = name.slice(0, -1).trim();
			}

			return { name, isChair, isQuoted };
		});
	}

	let parsedAgenda = $derived(parseAgenda(agendaText));
	let parsedParticipants = $derived(parseParticipants(participantsText));

	export function open() {
		step = 1;
		meetingName = '';
		meetingDescription = '';
		signupOpen = false;
		agendaText = '';
		participantsText = '';
		error = '';
		submitting = false;
		dialogEl?.showModal();
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
			// Step 1: Create meeting
			const meetingRes = await committeeClient.createMeeting({
				committeeSlug: slug,
				name: meetingName.trim(),
				description: meetingDescription.trim()
			});
			const meetingId = meetingRes.meeting?.meetingId ?? '';
			if (!meetingId) throw new Error('Meeting creation returned no ID');

			// Step 2: Create agenda points
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

			// Step 3: Create participants
			for (const participant of parsedParticipants) {
				const attendeeRes = await attendeeClient.createAttendee({
					committeeSlug: slug,
					meetingId,
					fullName: participant.name,
					genderQuoted: participant.isQuoted
				});
				if (participant.isChair && attendeeRes.attendee?.attendeeId) {
					await attendeeClient.setChairperson({
						committeeSlug: slug,
						meetingId,
						attendeeId: attendeeRes.attendee.attendeeId,
						isChair: true
					});
				}
			}

			// Enable signup if requested
			if (signupOpen) {
				await committeeClient.toggleSignupOpen({
					committeeSlug: slug,
					meetingId,
					open: true
				});
			}

			dialogEl?.close();
			onCreated();
		} catch (err) {
			error = getDisplayError(err, 'Failed to create meeting.');
		} finally {
			submitting = false;
		}
	}
</script>

<dialog class="modal" bind:this={dialogEl}>
	<div class="modal-box w-11/12 max-w-3xl">
		<div class="mb-4 flex items-center justify-between">
			<h3 class="text-lg font-semibold">{m.committee_create_meeting_heading()}</h3>
			<div class="flex items-center gap-2">
				<ul class="steps steps-horizontal text-xs">
					<li class={step >= 1 ? 'step step-primary' : 'step'}>{m.wizard_step_basics()}</li>
					<li class={step >= 2 ? 'step step-primary' : 'step'}>{m.wizard_step_agenda()}</li>
					<li class={step >= 3 ? 'step step-primary' : 'step'}>{m.wizard_step_participants()}</li>
					<li class={step >= 4 ? 'step step-primary' : 'step'}>{m.wizard_step_review()}</li>
				</ul>
				<button type="button" class="btn btn-sm btn-ghost" onclick={() => dialogEl?.close()}>{m.common_close()}</button>
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
				<label class="label cursor-pointer justify-start gap-3">
					<input type="checkbox" class="toggle toggle-primary toggle-sm" bind:checked={signupOpen} />
					<span class="text-sm">{m.committee_signup_label()}</span>
				</label>
			</div>
		{:else if step === 2}
			<div class="space-y-3">
				<p class="text-sm text-base-content/70">{m.wizard_agenda_description()}</p>
				<textarea
					class="textarea textarea-bordered w-full font-mono text-sm"
					rows="12"
					placeholder={m.wizard_agenda_placeholder()}
					bind:value={agendaText}
				></textarea>
				{#if parsedAgenda.length > 0}
					<div class="rounded-box border border-base-300 bg-base-200/30 p-3">
						<h4 class="mb-2 text-sm font-semibold">{m.wizard_preview()} ({parsedAgenda.length})</h4>
						<ul class="space-y-1 text-sm">
							{#each parsedAgenda as point, i}
								<li>
									<span class="font-medium">{i + 1}. {point.title}</span>
									{#if point.children.length > 0}
										<ul class="ml-4 text-base-content/70">
											{#each point.children as child, j}
												<li>{i + 1}.{j + 1} {child}</li>
											{/each}
										</ul>
									{/if}
								</li>
							{/each}
						</ul>
					</div>
				{/if}
			</div>
		{:else if step === 3}
			<div class="space-y-3">
				<p class="text-sm text-base-content/70">{m.wizard_participants_description()}</p>
				<textarea
					class="textarea textarea-bordered w-full font-mono text-sm"
					rows="12"
					placeholder={m.wizard_participants_placeholder()}
					bind:value={participantsText}
				></textarea>
				{#if parsedParticipants.length > 0}
					<div class="rounded-box border border-base-300 bg-base-200/30 p-3">
						<h4 class="mb-2 text-sm font-semibold">{m.wizard_preview()} ({parsedParticipants.length})</h4>
						<ul class="space-y-1 text-sm">
							{#each parsedParticipants as p}
								<li class="flex items-center gap-2">
									<span>{p.name}</span>
									{#if p.isChair}<span class="badge badge-success badge-xs">Chair</span>{/if}
									{#if p.isQuoted}<span class="badge badge-info badge-xs">FLINTA*</span>{/if}
								</li>
							{/each}
						</ul>
					</div>
				{/if}
			</div>
		{:else if step === 4}
			<div class="space-y-4">
				<div class="rounded-box border border-base-300 bg-base-200/30 p-3 space-y-2">
					<h4 class="text-sm font-semibold">{m.wizard_step_basics()}</h4>
					<p class="text-sm"><strong>{meetingName}</strong></p>
					{#if meetingDescription}<p class="text-sm text-base-content/70">{meetingDescription}</p>{/if}
					{#if signupOpen}<span class="badge badge-outline badge-sm">{m.committee_signup_label()}</span>{/if}
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
				{#if parsedParticipants.length > 0}
					<div class="rounded-box border border-base-300 bg-base-200/30 p-3 space-y-2">
						<h4 class="text-sm font-semibold">{m.wizard_step_participants()} ({parsedParticipants.length})</h4>
						<ul class="text-sm space-y-0.5">
							{#each parsedParticipants as p}
								<li>
									{p.name}
									{#if p.isChair} <span class="badge badge-success badge-xs">Chair</span>{/if}
									{#if p.isQuoted} <span class="badge badge-info badge-xs">FLINTA*</span>{/if}
								</li>
							{/each}
						</ul>
					</div>
				{/if}
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
	</div>
	<form method="dialog" class="modal-backdrop"><button aria-label="Close">Close</button></form>
</dialog>
