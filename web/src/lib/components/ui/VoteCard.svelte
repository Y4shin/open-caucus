<script lang="ts">
	import { Collapsible } from 'bits-ui';
	import AppSelect from '$lib/components/ui/AppSelect.svelte';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import type { AttendeeRecord } from '$lib/gen/conference/attendees/v1/attendees_pb.js';
	import type { VoteDefinitionRecord } from '$lib/gen/conference/votes/v1/votes_pb.js';
	import { voteStateBadgeClass, voteVisibilityBadgeClass } from '$lib/utils/votes.js';
	import * as m from '$lib/paraglide/messages';

	let {
		vote,
		open,
		draftEditorOpen,
		attendees,
		onToggle,
		onDraftEditorToggle,
		onOpenVote,
		onCloseVote,
		onArchiveVote,
		onUpdateDraft,
		onCountOpenBallot,
		onRegisterCast,
		onCountSecretBallot
	}: {
		vote: VoteDefinitionRecord;
		open: boolean;
		draftEditorOpen: boolean;
		attendees: AttendeeRecord[];
		onToggle: (open: boolean) => void;
		onDraftEditorToggle: (open: boolean) => void;
		onOpenVote: () => Promise<void>;
		onCloseVote: () => Promise<void>;
		onArchiveVote: () => Promise<void>;
		onUpdateDraft: (event: SubmitEvent) => Promise<void>;
		onCountOpenBallot: (attendeeQuery: string, optionIds: string[]) => Promise<void>;
		onRegisterCast: (attendeeQuery: string) => Promise<void>;
		onCountSecretBallot: (receiptToken: string, optionIds: string[]) => Promise<void>;
	} = $props();

	function voteStateLabel(state: string) {
		switch (state) {
			case 'draft': return m.votes_state_draft();
			case 'open': return m.votes_state_open();
			case 'counting': return m.votes_state_counting();
			case 'closed': return m.votes_state_closed();
			case 'archived': return m.votes_state_archived();
			default: return state;
		}
	}

	function voteVisibilityLabel(visibility: string) {
		return visibility === 'secret' ? m.votes_visibility_secret() : m.votes_visibility_open();
	}

	function voteBoundsLabel() {
		if (vote.minSelections === vote.maxSelections) {
			return m.votes_select_exactly({ count: Number(vote.minSelections) });
		}
		return m.votes_select_between({ min: Number(vote.minSelections), max: Number(vote.maxSelections) });
	}

	let draftVisibilityOverride = $state<string | null>(null);
	let draftVisibility = $derived(draftVisibilityOverride ?? vote.visibility);

	function voteLabelsForEdit() {
		const labels = vote.options.map((option) => option.label);
		if (labels.length < 2) {
			labels.push('Yes', 'No');
		}
		labels.push('');
		return labels;
	}

	function voteStats() {
		return vote.stats ?? { eligibleCount: 0n, castCount: 0n, ballotCount: 0n };
	}

	function voteOutstandingCount() {
		const stats = voteStats();
		const outstanding = stats.castCount - stats.ballotCount;
		return outstanding > 0n ? outstanding : 0n;
	}

	function voteShouldShowTallies() {
		return vote.state === 'closed' || vote.state === 'archived';
	}
</script>

<Collapsible.Root {open} onOpenChange={onToggle} data-vote-accordion={vote.voteId} class="rounded-box border border-base-300 bg-base-100">
	<Collapsible.Trigger class="flex w-full cursor-pointer items-center justify-between px-4 py-3">
		<div class="flex flex-wrap items-center gap-2">
			<h4 class="font-semibold">{vote.name}</h4>
			<span class={voteStateBadgeClass(vote.state)}>{voteStateLabel(vote.state)}</span>
			<span class={voteVisibilityBadgeClass(vote.visibility)}>{voteVisibilityLabel(vote.visibility)}</span>
			<span class="text-xs text-base-content/70">{voteBoundsLabel()}</span>
		</div>
		<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4 shrink-0 transition-transform {open ? 'rotate-180' : ''}">
			<path fill-rule="evenodd" d="M5.22 8.22a.75.75 0 0 1 1.06 0L10 11.94l3.72-3.72a.75.75 0 1 1 1.06 1.06l-4.25 4.25a.75.75 0 0 1-1.06 0l-4.25-4.25a.75.75 0 0 1 0-1.06Z" clip-rule="evenodd" />
		</svg>
	</Collapsible.Trigger>
	<Collapsible.Content class="space-y-3 px-4 pb-4">
		{#if vote.options.length > 0}
			<ul class="list rounded-box border border-base-300 bg-base-200/40">
				{#each vote.options as option}
					<li class="list-row py-1">
						<span class="badge badge-outline badge-sm">{option.position.toString()}</span>
						<span class="flex-1 truncate">{option.label}</span>
					</li>
				{/each}
			</ul>
		{/if}

		<div class="rounded-box border border-base-300 bg-base-200/30 p-2">
			<div class="mb-1 text-xs font-semibold uppercase text-base-content/70">{m.votes_live_submission_tally()}</div>
			<div class="grid gap-1 text-sm sm:grid-cols-2">
				<div class="flex items-center justify-between gap-2">
					<span>{m.votes_eligible()}</span>
					<span class="badge badge-outline badge-sm">{voteStats().eligibleCount.toString()}</span>
				</div>
				<div class="flex items-center justify-between gap-2">
					<span>{m.votes_casts()}</span>
					<span class="badge badge-outline badge-sm">{voteStats().castCount.toString()}</span>
				</div>
				<div class="flex items-center justify-between gap-2">
					<span>{m.votes_counted_ballots()}</span>
					<span class="badge badge-outline badge-sm">{voteStats().ballotCount.toString()}</span>
				</div>
				<div class="flex items-center justify-between gap-2">
					<span>{m.votes_outstanding()}</span>
					<span class={voteOutstandingCount() > 0n ? 'badge badge-sm badge-warning' : 'badge badge-sm badge-success'}>{voteOutstandingCount().toString()}</span>
				</div>
			</div>
		</div>

		{#if vote.state === 'draft'}
			<div class="flex flex-wrap gap-2">
				<button type="button" class="btn btn-sm btn-success" onclick={async (event) => { event.preventDefault(); event.stopPropagation(); await onOpenVote(); }}>{m.votes_open_vote()}</button>
			</div>
			<Collapsible.Root open={draftEditorOpen} onOpenChange={onDraftEditorToggle} data-vote-draft-editor={vote.voteId} class="rounded-box border border-base-300 bg-base-200/30">
				<Collapsible.Trigger class="flex w-full cursor-pointer items-center justify-between px-4 py-2 text-sm">
					{m.votes_edit_draft()}
					<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4 shrink-0 transition-transform {draftEditorOpen ? 'rotate-180' : ''}">
						<path fill-rule="evenodd" d="M5.22 8.22a.75.75 0 0 1 1.06 0L10 11.94l3.72-3.72a.75.75 0 1 1 1.06 1.06l-4.25 4.25a.75.75 0 0 1-1.06 0l-4.25-4.25a.75.75 0 0 1 0-1.06Z" clip-rule="evenodd" />
					</svg>
				</Collapsible.Trigger>
				<Collapsible.Content class="px-4 pb-4">
					<form class="grid gap-2 md:grid-cols-2" onsubmit={onUpdateDraft}>
						<input class="input input-bordered input-sm md:col-span-2" name="name" value={vote.name} required />
						<AppSelect
							value={draftVisibility}
							items={[
								{ value: 'open', label: m.votes_visibility_open() },
								{ value: 'secret', label: m.votes_visibility_secret() }
							]}
							onValueChange={(v) => { draftVisibilityOverride = v; }}
						/>
						<input type="hidden" name="visibility" value={draftVisibility} />
						<div class="join">
							<input class="input input-bordered input-sm join-item w-24" type="number" min="0" name="min_selections" value={vote.minSelections.toString()} required />
							<input class="input input-bordered input-sm join-item w-24" type="number" min="1" name="max_selections" value={vote.maxSelections.toString()} required />
						</div>
						<div class="md:col-span-2 space-y-1" data-vote-option-list>
							<div class="text-xs font-semibold uppercase text-base-content/70">{m.votes_choices_label()}</div>
							{#each voteLabelsForEdit() as label, index}
								<div data-vote-option-row>
									<input class="input input-bordered input-sm w-full" name="option_label" value={label} data-vote-option-input placeholder={m.votes_choice_n_placeholder({ n: index + 1 })} autocomplete="off" />
								</div>
							{/each}
							<div class="text-xs text-base-content/70" data-vote-option-hint>{m.votes_edit_draft_hint()}</div>
						</div>
						<div class="md:col-span-2 flex flex-wrap gap-2">
							<button type="submit" class="btn btn-sm btn-primary">{m.votes_save_draft()}</button>
						</div>
					</form>
				</Collapsible.Content>
			</Collapsible.Root>
		{/if}

		{#if vote.state === 'open' || vote.state === 'counting'}
			<div class="flex flex-wrap gap-2">
				<button type="button" class="btn btn-sm btn-warning" onclick={async (event) => { event.preventDefault(); event.stopPropagation(); await onCloseVote(); }}>{m.votes_close_vote()}</button>
			</div>
		{/if}

		<div class="rounded-box border border-base-300 bg-base-200/20 p-2 space-y-2">
			<div class="text-xs font-semibold uppercase text-base-content/70">{m.votes_manual_submission()}</div>
			{#if vote.visibility === 'open'}
				{#if vote.state === 'open'}
					<div class="space-y-2" id={`open-ballot-form-${vote.voteId}`} data-testid="manage-vote-open-ballot-form">
						<div class="text-xs font-semibold uppercase text-base-content/70">{m.votes_open_ballot_entry()}</div>
						<div class={vote.maxSelections === 1n ? 'grid gap-1 text-sm grid-cols-1' : 'grid gap-1 text-sm grid-cols-2'}>
							{#each vote.options as option}
								<label class="label cursor-pointer justify-start gap-2 rounded border border-base-300 px-2 py-1">
									{#if vote.maxSelections === 1n}
										<input class="radio radio-sm" type="radio" name={`open-option-${vote.voteId}`} value={option.optionId} />
									{:else}
										<input class="checkbox checkbox-sm" type="checkbox" name={`open-option-${vote.voteId}`} value={option.optionId} />
									{/if}
									<span>{option.label}</span>
								</label>
							{/each}
						</div>
						{#if attendees.length === 0}
							<p class="text-xs text-warning">{m.votes_no_attendees_for_entry()}</p>
						{:else}
							<div class="join w-full">
								<input id={`open-ballot-attendee-${vote.voteId}`} class="input input-bordered input-sm join-item w-full" list={`vote-manual-open-attendee-list-${vote.voteId}`} placeholder={m.votes_search_attendee()} required data-testid="open-ballot-attendee-query" />
								<button type="button" class="btn btn-sm btn-primary join-item" data-testid="open-ballot-submit" onclick={async () => {
									const container = document.getElementById(`open-ballot-form-${vote.voteId}`);
									const attendeeInput = document.getElementById(`open-ballot-attendee-${vote.voteId}`) as HTMLInputElement | null;
									const attendeeQuery = attendeeInput?.value ?? '';
									const checked = [...(container?.querySelectorAll(`[name="open-option-${vote.voteId}"]:checked`) ?? [])].map((el) => (el as HTMLInputElement).value);
									await onCountOpenBallot(attendeeQuery, checked);
									if (attendeeInput) attendeeInput.value = '';
									container?.querySelectorAll(`[name="open-option-${vote.voteId}"]`).forEach((el) => { (el as HTMLInputElement).checked = false; });
								}}>{m.votes_submit_ballot()}</button>
							</div>
							<datalist id={`vote-manual-open-attendee-list-${vote.voteId}`}>
								{#each attendees as attendee}
									<option value={`${attendee.attendeeNumber.toString()} ${attendee.fullName}`}></option>
								{/each}
							</datalist>
							<p class="text-xs text-base-content/70">{m.votes_quick_cast_hint()}</p>
						{/if}
					</div>
				{:else}
					<p class="text-xs text-base-content/70">{m.votes_open_ballot_hint()}</p>
				{/if}
			{:else}
				<p class="text-xs text-base-content/70">{m.votes_secret_ballot_hint()}</p>
				{#if vote.state === 'open' || vote.state === 'counting'}
					<div class={vote.state === 'open' ? 'grid gap-2 md:grid-cols-2' : 'grid gap-2 md:grid-cols-1'}>
						{#if vote.state === 'open'}
							<div class="rounded-box border border-base-300 p-2 space-y-2" id={`register-cast-form-${vote.voteId}`} data-testid="manage-vote-register-cast-form">
								<div class="text-xs font-semibold uppercase text-base-content/70">{m.votes_register_cast_step()}</div>
								{#if attendees.length === 0}
									<p class="text-xs text-warning">{m.votes_no_attendees_for_cast()}</p>
								{:else}
									<div class="join w-full">
										<input id={`register-cast-attendee-${vote.voteId}`} class="input input-bordered input-sm join-item w-full" list={`vote-manual-secret-attendee-list-${vote.voteId}`} placeholder={m.votes_search_attendee()} required data-testid="register-cast-attendee-query" />
										<button type="button" class="btn btn-sm join-item" data-testid="register-cast-submit" onclick={async () => {
											const attendeeInput = document.getElementById(`register-cast-attendee-${vote.voteId}`) as HTMLInputElement | null;
											const attendeeQuery = attendeeInput?.value ?? '';
											await onRegisterCast(attendeeQuery);
											if (attendeeInput) attendeeInput.value = '';
										}}>{m.votes_register_cast()}</button>
									</div>
									<datalist id={`vote-manual-secret-attendee-list-${vote.voteId}`}>
										{#each attendees as attendee}
											<option value={`${attendee.attendeeNumber.toString()} ${attendee.fullName}`}></option>
										{/each}
									</datalist>
									<p class="text-xs text-base-content/70">{m.votes_quick_registration_hint()}</p>
								{/if}
							</div>
						{/if}
						<div class="rounded-box border border-base-300 p-2 space-y-2" id={`count-secret-form-${vote.voteId}`} data-testid="manage-vote-count-secret-form">
							<div class="text-xs font-semibold uppercase text-base-content/70">{m.votes_count_secret_step()}</div>
							<input id={`secret-receipt-${vote.voteId}`} class="input input-bordered input-sm w-full" placeholder={m.votes_receipt_token_optional()} data-testid="count-secret-receipt-token" />
							<div class="grid grid-cols-2 gap-1 text-sm">
								{#each vote.options as option}
									<label class="label cursor-pointer justify-start gap-2 rounded border border-base-300 px-2 py-1">
										{#if vote.maxSelections === 1n}
											<input class="radio radio-sm" type="radio" name={`secret-option-${vote.voteId}`} value={option.optionId} />
										{:else}
											<input class="checkbox checkbox-sm" type="checkbox" name={`secret-option-${vote.voteId}`} value={option.optionId} />
										{/if}
										<span>{option.label}</span>
									</label>
								{/each}
							</div>
							<button type="button" class="btn btn-sm btn-primary" data-testid="count-secret-submit" onclick={async () => {
								const container = document.getElementById(`count-secret-form-${vote.voteId}`);
								const receiptInput = document.getElementById(`secret-receipt-${vote.voteId}`) as HTMLInputElement | null;
								const receiptToken = receiptInput?.value ?? '';
								const checked = [...(container?.querySelectorAll(`[name="secret-option-${vote.voteId}"]:checked`) ?? [])].map((el) => (el as HTMLInputElement).value);
								await onCountSecretBallot(receiptToken, checked);
								if (receiptInput) receiptInput.value = '';
								container?.querySelectorAll(`[name="secret-option-${vote.voteId}"]`).forEach((el) => { (el as HTMLInputElement).checked = false; });
							}}>{m.votes_count_ballot()}</button>
						</div>
					</div>
				{:else}
					<p class="text-xs text-base-content/70">{m.votes_secret_actions_hint()}</p>
				{/if}
			{/if}
		</div>

		{#if vote.state === 'closed'}
			<div class="flex flex-wrap gap-2">
				<button type="button" class="btn btn-sm btn-outline" onclick={async (event) => { event.preventDefault(); event.stopPropagation(); await onArchiveVote(); }}>{m.votes_archive_vote()}</button>
			</div>
		{/if}

		{#if vote.state === 'counting'}
			<p class="text-sm text-warning">{m.votes_results_blocked_counting()}</p>
		{:else if voteShouldShowTallies()}
			<div class="rounded-box border border-base-300 bg-base-200/30 p-2">
				<div class="mb-1 text-xs font-semibold uppercase text-base-content/70">{m.votes_final_tallies()}</div>
				<ul class="space-y-1 text-sm">
					{#each vote.tally ?? [] as row}
						<li class="flex items-center justify-between gap-2">
							<span class="truncate">{row.label}</span>
							<span class="badge badge-outline badge-sm">{row.count.toString()}</span>
						</li>
					{/each}
				</ul>
				<div class="mt-2 text-xs text-base-content/70">{m.votes_summary_counts({ eligible: voteStats().eligibleCount.toString(), casts: voteStats().castCount.toString(), ballots: voteStats().ballotCount.toString() })}</div>
			</div>
		{/if}
	</Collapsible.Content>
</Collapsible.Root>
