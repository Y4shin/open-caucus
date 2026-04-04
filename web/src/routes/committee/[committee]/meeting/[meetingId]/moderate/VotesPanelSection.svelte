<script lang="ts">
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import VoteCard from '$lib/components/ui/VoteCard.svelte';
	import { voteClient } from '$lib/api/index.js';
	import type { AttendeeRecord } from '$lib/gen/conference/attendees/v1/attendees_pb.js';
	import type { VoteDefinitionRecord, VotesPanelView } from '$lib/gen/conference/votes/v1/votes_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import * as m from '$lib/paraglide/messages';

	let {
		votesPanel,
		votesLoading,
		votesError,
		attendees,
		slug,
		meetingId,
		onError,
		onNotice
	}: {
		votesPanel: VotesPanelView | null;
		votesLoading: boolean;
		votesError: string;
		attendees: AttendeeRecord[];
		slug: string;
		meetingId: string;
		onError: (msg: string) => void;
		onNotice: (msg: string) => void;
	} = $props();

	let voteActionPending = $state('');
	let createVoteDetailsOpen = $state(false);
	let voteAccordionOpen = $state<Record<string, boolean>>({});
	let draftVoteEditorOpen = $state<Record<string, boolean>>({});

	$effect(() => {
		if (votesPanel?.votes) syncVotePanelOpenState(votesPanel.votes);
	});

	async function runVoteAction(key: string, action: () => Promise<void>) {
		onError('');
		onNotice('');
		voteActionPending = key;
		try {
			await action();
			return true;
		} catch (err) {
			onError(getDisplayError(err, 'Failed to update the votes panel.'));
			return false;
		} finally {
			voteActionPending = '';
		}
	}

	function bigintFromInput(value: string) {
		const parsed = Number.parseInt(value, 10);
		return Number.isFinite(parsed) ? BigInt(parsed) : 0n;
	}

	function voteDefaultOptionLabels() {
		return ['Yes', 'No', 'Abstain', ''];
	}

	function voteAccordionDefaultOpen(vote: VoteDefinitionRecord) {
		return vote.state === 'open' || vote.state === 'counting';
	}

	function syncVotePanelOpenState(votes: VoteDefinitionRecord[]) {
		const nextVoteAccordionOpen: Record<string, boolean> = {};
		const nextDraftVoteEditorOpen: Record<string, boolean> = {};
		for (const vote of votes) {
			nextVoteAccordionOpen[vote.voteId] =
				voteAccordionOpen[vote.voteId] ?? voteAccordionDefaultOpen(vote);
			if (vote.state === 'draft') {
				nextDraftVoteEditorOpen[vote.voteId] = draftVoteEditorOpen[vote.voteId] ?? false;
			}
		}
		voteAccordionOpen = nextVoteAccordionOpen;
		draftVoteEditorOpen = nextDraftVoteEditorOpen;
	}

	function setVoteAccordionOpen(voteId: string, open: boolean) {
		voteAccordionOpen = { ...voteAccordionOpen, [voteId]: open };
	}

	function setDraftVoteEditorOpen(voteId: string, open: boolean) {
		draftVoteEditorOpen = { ...draftVoteEditorOpen, [voteId]: open };
	}

	function resolveAttendeeIdFromQuery(query: string): string | null {
		const trimmed = query.trim();
		const leadingNum = trimmed.match(/^(\d+)/);
		if (leadingNum) {
			const num = BigInt(leadingNum[1]);
			const found = attendees.find((a) => a.attendeeNumber === num);
			if (found) return found.attendeeId;
		}
		const exact = attendees.find((a) => a.fullName.toLowerCase() === trimmed.toLowerCase());
		if (exact) return exact.attendeeId;
		const lower = trimmed.toLowerCase();
		const matches = attendees.filter((a) =>
			`${a.attendeeNumber} ${a.fullName}`.toLowerCase().includes(lower)
		);
		if (matches.length === 1) return matches[0].attendeeId;
		return null;
	}

	async function submitCreateVoteForm(event: SubmitEvent) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement | null;
		if (!form) return;
		const data = new FormData(form);
		await runVoteAction('create-vote', async () => {
			await voteClient.createVote({
				committeeSlug: slug,
				meetingId,
				name: String(data.get('name') ?? '').trim(),
				visibility: String(data.get('visibility') ?? 'open'),
				minSelections: bigintFromInput(String(data.get('min_selections') ?? '1')),
				maxSelections: bigintFromInput(String(data.get('max_selections') ?? '1')),
				optionLabels: data
					.getAll('option_label')
					.map((value) => String(value).trim())
					.filter(Boolean)
			});
			form.reset();
			createVoteDetailsOpen = false;
			const res = await voteClient.getVotesPanel({ committeeSlug: slug, meetingId });
			syncVotePanelOpenState(res.view?.votes ?? []);
		});
	}

	async function submitUpdateDraftVoteForm(event: SubmitEvent, voteId: string) {
		event.preventDefault();
		const form = event.currentTarget as HTMLFormElement | null;
		if (!form) return;
		const data = new FormData(form);
		await runVoteAction(`save-draft-${voteId}`, async () => {
			await voteClient.updateVoteDraft({
				committeeSlug: slug,
				meetingId,
				voteId,
				name: String(data.get('name') ?? '').trim(),
				visibility: String(data.get('visibility') ?? 'open'),
				minSelections: bigintFromInput(String(data.get('min_selections') ?? '1')),
				maxSelections: bigintFromInput(String(data.get('max_selections') ?? '1')),
				optionLabels: data
					.getAll('option_label')
					.map((value) => String(value).trim())
					.filter(Boolean)
			});
		});
	}

	async function openVote(voteId: string) {
		await runVoteAction(`open-${voteId}`, async () => {
			await voteClient.openVote({ committeeSlug: slug, meetingId, voteId });
		});
	}

	async function closeVote(voteId: string) {
		await runVoteAction(`close-${voteId}`, async () => {
			await voteClient.closeVote({ committeeSlug: slug, meetingId, voteId });
		});
	}

	async function archiveVote(voteId: string) {
		await runVoteAction(`archive-${voteId}`, async () => {
			await voteClient.archiveVote({ committeeSlug: slug, meetingId, voteId });
			onNotice('Vote archived.');
		});
	}

	async function registerCast(voteId: string, attendeeQuery: string) {
		const attendeeId = resolveAttendeeIdFromQuery(attendeeQuery);
		if (!attendeeId) throw new Error('Could not resolve attendee from query');
		await runVoteAction(`register-cast-${voteId}`, async () => {
			await voteClient.registerCast({ committeeSlug: slug, meetingId, voteId, attendeeId });
		});
	}

	async function countOpenBallot(voteId: string, attendeeQuery: string, selectedOptionIds: string[]) {
		const attendeeId = resolveAttendeeIdFromQuery(attendeeQuery);
		if (!attendeeId) throw new Error('Could not resolve attendee from query');
		await runVoteAction(`ballot-open-${voteId}`, async () => {
			await voteClient.countOpenBallot({
				committeeSlug: slug,
				meetingId,
				voteId,
				attendeeId,
				selectedOptionIds
			});
		});
	}

	async function countSecretBallot(voteId: string, receiptToken: string, selectedOptionIds: string[]) {
		await runVoteAction(`ballot-secret-${voteId}`, async () => {
			await voteClient.countSecretBallot({
				committeeSlug: slug,
				meetingId,
				voteId,
				receiptToken,
				selectedOptionIds
			});
		});
	}
</script>

{#if votesLoading && !votesPanel}
	<AppSpinner label="Loading votes" />
{:else if votesError && !votesPanel}
	<AppAlert message={votesError} />
{:else if !votesPanel?.hasActiveAgendaPoint}
	<p class="text-sm text-base-content/70">{m.meeting_manage_no_active_agenda_for_speakers()}</p>
{:else}
	<details class="collapse collapse-arrow border border-base-300 bg-base-100" open={createVoteDetailsOpen} ontoggle={(event) => { createVoteDetailsOpen = (event.currentTarget as HTMLDetailsElement).open; }}>
		<summary class="collapse-title text-sm font-semibold">{m.votes_create_vote()}</summary>
		<div class="collapse-content">
			<form class="grid gap-2 md:grid-cols-2" onsubmit={submitCreateVoteForm}>
				<input class="input input-bordered input-sm md:col-span-2" name="name" placeholder={m.votes_vote_name_placeholder()} required />
				<select class="select select-bordered select-sm" name="visibility">
					<option value="open">{m.votes_visibility_open()}</option>
					<option value="secret">{m.votes_visibility_secret()}</option>
				</select>
				<div class="join">
					<input class="input input-bordered input-sm join-item w-24" type="number" min="0" name="min_selections" value="1" required />
					<input class="input input-bordered input-sm join-item w-24" type="number" min="1" name="max_selections" value="1" required />
				</div>
				<div class="md:col-span-2 space-y-1" data-vote-option-list>
					<div class="text-xs font-semibold uppercase text-base-content/70">{m.votes_choices_label()}</div>
					{#each voteDefaultOptionLabels() as label, index}
						<div data-vote-option-row>
							<input class="input input-bordered input-sm w-full" name="option_label" value={label} data-vote-option-input placeholder={m.votes_choice_n_placeholder({ n: index + 1 })} autocomplete="off" />
						</div>
					{/each}
					<div class="text-xs text-base-content/70" data-vote-option-hint>Add one choice per field. A new field appears once the last one is filled.</div>
				</div>
				<div class="md:col-span-2">
					<button type="submit" class="btn btn-sm btn-primary">{m.votes_create_draft_vote()}</button>
				</div>
			</form>
		</div>
	</details>

	{#if (votesPanel?.votes ?? []).length === 0}
		<p class="text-sm text-base-content/70">{m.votes_no_votes_for_agenda_point({ agendaPoint: votesPanel?.activeAgendaPointTitle ?? "" })}</p>
	{:else}
		<div class="space-y-3">
			{#each votesPanel?.votes ?? [] as vote}
				<VoteCard
					{vote}
					open={voteAccordionOpen[vote.voteId] ?? voteAccordionDefaultOpen(vote)}
					draftEditorOpen={draftVoteEditorOpen[vote.voteId] ?? false}
					{attendees}
					onToggle={(open) => setVoteAccordionOpen(vote.voteId, open)}
					onDraftEditorToggle={(open) => setDraftVoteEditorOpen(vote.voteId, open)}
					onOpenVote={async () => openVote(vote.voteId)}
					onCloseVote={async () => closeVote(vote.voteId)}
					onArchiveVote={async () => archiveVote(vote.voteId)}
					onUpdateDraft={async (event) => submitUpdateDraftVoteForm(event, vote.voteId)}
					onCountOpenBallot={async (attendeeQuery, optionIds) => countOpenBallot(vote.voteId, attendeeQuery, optionIds)}
					onRegisterCast={async (attendeeQuery) => registerCast(vote.voteId, attendeeQuery)}
					onCountSecretBallot={async (receiptToken, optionIds) => countSecretBallot(vote.voteId, receiptToken, optionIds)}
				/>
			{/each}
		</div>
	{/if}
{/if}
