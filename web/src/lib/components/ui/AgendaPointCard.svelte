<script lang="ts">
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import type { AgendaPointRecord } from '$lib/gen/conference/agenda/v1/agenda_pb.js';
	import * as m from '$lib/paraglide/messages';

	let {
		point,
		isEditing,
		editTitle = $bindable(''),
		canMoveUp,
		canMoveDown,
		slug,
		meetingId,
		isBusy,
		onSave,
		onCancelEdit,
		onMoveUp,
		onMoveDown,
		onActivate,
		onStartEdit,
		onDelete
	}: {
		point: AgendaPointRecord;
		isEditing: boolean;
		editTitle: string;
		canMoveUp: boolean;
		canMoveDown: boolean;
		slug: string;
		meetingId: string;
		isBusy: (key: string) => boolean;
		onSave: () => Promise<void>;
		onCancelEdit: () => void;
		onMoveUp: () => Promise<void>;
		onMoveDown: () => Promise<void>;
		onActivate: () => Promise<void>;
		onStartEdit: () => void;
		onDelete: () => Promise<void>;
	} = $props();

	function displayNumber() {
		if (point.displayNumber.startsWith('TOP')) return point.displayNumber;
		return `TOP ${point.displayNumber}`;
	}

	function cardClass() {
		const base = 'card rounded-box border border-base-300 bg-base-100 p-3 shadow-sm';
		if (point.isActive) return `${base} bg-primary/5 border-primary/40${point.parentId ? ' ml-5' : ''}`;
		if (point.parentId) return `${base} ml-5`;
		return base;
	}
</script>

<div id={`agenda-point-card-${point.agendaPointId}`} class={cardClass()} data-testid="manage-agenda-point-card">
	{#if isEditing}
		<form class="flex items-center gap-2" data-testid="manage-agenda-point-edit-form" onsubmit={async (event) => { event.preventDefault(); await onSave(); }}>
			<input class="input input-bordered input-sm flex-1" type="text" name="title" bind:value={editTitle} required disabled={isBusy(`edit-${point.agendaPointId}`)} data-testid="manage-agenda-point-edit-input" />
			<button type="submit" class="btn btn-sm btn-primary" disabled={isBusy(`edit-${point.agendaPointId}`)}>{m.common_save()}</button>
			<button type="button" class="btn btn-sm btn-ghost" onclick={onCancelEdit}>{m.common_cancel()}</button>
		</form>
	{:else}
		<div class="flex items-start gap-3">
			<span class="badge badge-outline shrink-0">{displayNumber()}</span>
			<div class="min-w-0 flex-1">
				<div class="truncate font-semibold">{point.title}</div>
				<div class="mt-1 flex flex-wrap gap-1">
					{#if point.parentId}
						<span class="badge badge-outline">{m.agenda_point_child_badge()}</span>
					{/if}
					{#if point.isActive}
						<span class="badge badge-outline badge-success" data-testid="manage-agenda-active-badge">{m.meeting_manage_agenda_point_active_badge()}</span>
					{/if}
					{#if point.enteredAt}
						<span class="badge badge-ghost badge-sm font-mono text-[0.65rem]" data-testid="manage-agenda-entered-at">{new Date(point.enteredAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
					{/if}
					{#if point.leftAt && point.enteredAt}
						{@const durationMs = new Date(point.leftAt).getTime() - new Date(point.enteredAt).getTime()}
						{@const durationMin = Math.floor(durationMs / 60000)}
						{@const durationSec = Math.floor((durationMs % 60000) / 1000)}
						<span class="badge badge-ghost badge-sm font-mono text-[0.65rem]">{String(durationMin).padStart(2, '0')}:{String(durationSec).padStart(2, '0')}</span>
					{/if}
				</div>
			</div>
		</div>
		<div class="mt-2 flex items-center gap-2">
			<form class="inline-flex" onsubmit={async (event) => { event.preventDefault(); await onMoveUp(); }}>
				<button type="submit" class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Move up" title="Move up" aria-label="Move up" disabled={!canMoveUp || isBusy(`move-${point.agendaPointId}-up`)}><LegacyIcon name="left" class="h-4 w-4 rotate-90" /></button>
			</form>
			<form class="inline-flex" onsubmit={async (event) => { event.preventDefault(); await onMoveDown(); }}>
				<button type="submit" class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Move down" title="Move down" aria-label="Move down" disabled={!canMoveDown || isBusy(`move-${point.agendaPointId}-down`)}><LegacyIcon name="right" class="h-4 w-4 rotate-90" /></button>
			</form>
			{#if !point.isActive}
				<form class="inline-flex" onsubmit={async (event) => { event.preventDefault(); await onActivate(); }}>
					<button type="submit" class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Activate agenda point" title="Activate agenda point" aria-label="Activate agenda point" disabled={isBusy(`activate-${point.agendaPointId}`)}><LegacyIcon name="check-circle" class="h-4 w-4" /></button>
				</form>
			{/if}
			<button type="button" class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Edit agenda point" title="Edit agenda point" aria-label="Edit agenda point" data-testid="manage-agenda-point-edit-btn" onclick={onStartEdit}><LegacyIcon name="edit" class="h-4 w-4" /></button>
			<a href={`/committee/${slug}/meeting/${meetingId}/agenda-point/${point.agendaPointId}/tools`} class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Open tools" title="Open tools" aria-label="Open tools"><LegacyIcon name="settings" class="h-4 w-4" /></a>
			<form class="inline-flex" onsubmit={async (event) => { event.preventDefault(); await onDelete(); }}>
				<button type="submit" class="btn btn-sm btn-square btn-error tooltip tooltip-left" data-tip="Delete agenda point" title="Delete agenda point" aria-label="Delete agenda point" disabled={isBusy(`delete-${point.agendaPointId}`)}><LegacyIcon name="trash" class="h-4 w-4" /></button>
			</form>
		</div>
	{/if}
</div>
