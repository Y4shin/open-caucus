<script lang="ts">
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import AppSwitch from '$lib/components/ui/AppSwitch.svelte';
	import AppTooltip from '$lib/components/ui/AppTooltip.svelte';
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
							<AppTooltip text="Chairperson" side="right">
								<span class="badge badge-success badge-sm"><LegacyIcon name="crown" class="h-3.5 w-3.5" /></span>
							</AppTooltip>
						{/if}
						{#if attendee.quoted}
							<AppTooltip text="FLINTA*" side="right">
								<span class="badge badge-info badge-sm" data-testid="manage-attendee-quoted-badge" aria-label="FLINTA*"><LegacyIcon name="transgender" class="h-3.5 w-3.5" /></span>
							</AppTooltip>
						{/if}
					</div>
				{/if}
			</div>
			<div class="flex shrink-0 items-center gap-3">
				<div class="hidden flex-col gap-1 sm:flex">
					<AppSwitch
						checked={attendee.isChair}
						size="xs"
						color={attendee.isChair ? 'primary' : ''}
						disabled={attendeeActionPending !== ''}
						label={m.meeting_live_chair()}
						onCheckedChange={async () => { await onToggleChair(attendee); }}
					/>
					{#if attendee.isGuest}
						<AppSwitch
							checked={attendee.quoted}
							size="xs"
							color={attendee.quoted ? 'info' : ''}
							disabled={attendeeActionPending !== ''}
							label={m.meeting_join_quoted_label()}
							onCheckedChange={async () => { await onToggleQuoted(attendee); }}
						/>
					{/if}
				</div>
				<div class="join join-vertical">
					{#if attendee.isGuest}
						<AppTooltip text="Recovery link" side="left">
							<button type="button" class="join-item btn btn-sm btn-square" aria-label="Recovery link" onclick={() => onRecovery(attendee.attendeeId)}><LegacyIcon name="history" class="h-4 w-4" /></button>
						</AppTooltip>
					{/if}
					<form class="inline-flex" onsubmit={async (event) => { event.preventDefault(); await onRemove(attendee.attendeeId, attendee.fullName); }}>
						<AppTooltip text="Remove attendee" side="left">
							<button type="submit" class="join-item btn btn-sm btn-square btn-error" aria-label="Remove attendee" disabled={attendeeActionPending !== ''}><LegacyIcon name="trash" class="h-4 w-4" /></button>
						</AppTooltip>
					</form>
				</div>
			</div>
		</div>
	</div>
</li>
