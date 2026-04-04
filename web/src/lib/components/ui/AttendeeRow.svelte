<script lang="ts">
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import type { AttendeeRecord } from '$lib/gen/conference/attendees/v1/attendees_pb.js';
	import * as m from '$lib/paraglide/messages';

	let {
		attendee,
		attendeeActionPending,
		onRemove,
		onToggleChair,
		onToggleQuoted,
		onRecovery
	}: {
		attendee: AttendeeRecord;
		attendeeActionPending: string;
		onRemove: (attendeeId: string, fullName: string) => Promise<void>;
		onToggleChair: (attendee: AttendeeRecord) => Promise<void>;
		onToggleQuoted: (attendee: AttendeeRecord) => Promise<void>;
		onRecovery: (attendeeId: string) => void;
	} = $props();
</script>

<li class="list-row grid-cols-1 items-center gap-3" data-testid="manage-attendee-card">
	<div class="col-span-full w-full min-w-0">
		<div class="flex min-w-0 items-center gap-2">
			<div class="w-12 shrink-0 text-base-content/70">#{attendee.attendeeNumber.toString()}</div>
			<div class="min-w-0 flex-1 overflow-x-hidden">
				<div class="truncate overflow-x-hidden font-semibold">{attendee.fullName}</div>
				{#if attendee.isGuest || attendee.isChair || attendee.quoted}
					<div class="mt-1 hidden flex-wrap items-center gap-1 sm:flex">
						{#if attendee.isGuest}
							<span class="badge badge-neutral badge-sm">{m.meeting_moderate_guest_badge()}</span>
						{/if}
						{#if attendee.isChair}
							<span class="tooltip tooltip-right" data-tip="Chairperson">
								<span class="badge badge-success badge-sm"><LegacyIcon name="crown" class="h-3.5 w-3.5" /></span>
							</span>
						{/if}
						{#if attendee.quoted}
							<span class="tooltip tooltip-right" data-tip="FLINTA*">
								<span class="badge badge-info badge-sm" data-testid="manage-attendee-quoted-badge"><LegacyIcon name="transgender" class="h-3.5 w-3.5" /></span>
							</span>
						{/if}
					</div>
				{/if}
			</div>
			<div class="flex shrink-0 items-center gap-3">
				<div class="hidden flex-col gap-1 sm:flex">
					<label class="label cursor-pointer justify-start gap-2 p-0">
						<input class={attendee.isChair ? 'toggle toggle-xs toggle-primary' : 'toggle toggle-xs'} type="checkbox" checked={attendee.isChair} title="Chairperson" aria-label="Chairperson" disabled={attendeeActionPending !== ''} onchange={async (event) => { event.preventDefault(); event.stopPropagation(); await onToggleChair(attendee); }} />
						<span class="text-xs leading-none">{m.meeting_live_chair()}</span>
					</label>
					{#if attendee.isGuest}
						<label class="label cursor-pointer justify-start gap-2 p-0">
							<input class={attendee.quoted ? 'toggle toggle-xs toggle-info' : 'toggle toggle-xs'} type="checkbox" checked={attendee.quoted} title="FLINTA*" aria-label="FLINTA*" disabled={attendeeActionPending !== ''} onchange={async (event) => { event.preventDefault(); event.stopPropagation(); await onToggleQuoted(attendee); }} />
							<span class="text-xs leading-none">{m.meeting_join_quoted_label()}</span>
						</label>
					{/if}
				</div>
				<div class="join join-vertical">
					{#if attendee.isGuest}
						<button type="button" class="join-item btn btn-sm btn-square tooltip tooltip-left" data-tip="Recovery link" title="Recovery link" aria-label="Recovery link" onclick={() => onRecovery(attendee.attendeeId)}><LegacyIcon name="history" class="h-4 w-4" /></button>
					{/if}
					<form class="inline-flex" onsubmit={async (event) => { event.preventDefault(); await onRemove(attendee.attendeeId, attendee.fullName); }}>
						<button type="submit" class="join-item btn btn-sm btn-square btn-error tooltip tooltip-left" data-tip="Remove attendee" title="Remove attendee" aria-label="Remove attendee" disabled={attendeeActionPending !== ''}><LegacyIcon name="trash" class="h-4 w-4" /></button>
					</form>
				</div>
			</div>
		</div>
	</div>
</li>
