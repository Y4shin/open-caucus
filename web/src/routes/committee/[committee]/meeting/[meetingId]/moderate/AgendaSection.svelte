<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { buildDocsOverlayHref } from '$lib/docs/navigation.js';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import AgendaPointCard from '$lib/components/ui/AgendaPointCard.svelte';
	import { agendaClient } from '$lib/api/index.js';
	import type { AgendaPointRecord } from '$lib/gen/conference/agenda/v1/agenda_pb.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import * as m from '$lib/paraglide/messages';
	import AgendaImportPreview from './AgendaImportPreview.svelte';
	import type { AgendaImportState, AgendaDiffRow } from './agenda-import.js';
	import { parseAgendaImportSource, buildImportedAgenda, computeAgendaDiff } from './agenda-import.js';

	let {
		agendaPoints,
		slug,
		meetingId,
		anyDialogOpen = $bindable(false),
		onError,
		onReload,
		onSetActivePoint
	}: {
		agendaPoints: AgendaPointRecord[];
		slug: string;
		meetingId: string;
		anyDialogOpen?: boolean;
		onError: (msg: string) => void;
		onReload: () => void;
		onSetActivePoint: (point: AgendaPointRecord | undefined) => void;
	} = $props();

	let agendaActionPending = $state('');
	let creatingAgenda = $state(false);
	let agendaEditOpen = $state(false);
	let agendaImportOpen = $state(false);
	let agendaTitle = $state('');
	let agendaParentId = $state('');
	let editingAgendaPointId = $state('');
	let editingAgendaPointTitle = $state('');
	let agendaImportRawText = $state('');
	let agendaImportLineStates = $state(new Map<number, AgendaImportState>());
	let agendaImportParsedText = $state('');
	let agendaImportFormat = $state<'markdown' | 'plaintext'>('plaintext');
	let agendaImportFormatManuallySet = $state(false);
	let agendaImportLines = $derived(
		parseAgendaImportSource(agendaImportParsedText, agendaImportFormat).map((line) => ({
			...line,
			state: agendaImportLineStates.get(line.lineNo) ?? line.state
		}))
	);
	let agendaImportDiff = $derived(
		buildImportedAgenda(agendaImportLines).length > 0
			? computeAgendaDiff(agendaPoints, buildImportedAgenda(agendaImportLines))
			: []
	);
	let agendaImportFingerprint = $state('');
	let agendaImportWarning = $state('');
	let agendaImportBusy = $state(false);
	let agendaImportStep = $state<'input' | 'diff'>('input');
	let agendaDiffHoverId = $state<string | null>(null);
	let agendaTitleInput = $state<HTMLInputElement | null>(null);
	let agendaEditDialogEl = $state<HTMLDialogElement | null>(null);
	let agendaImportDialogEl = $state<HTMLDialogElement | null>(null);

	$effect(() => {
		anyDialogOpen = agendaEditOpen || agendaImportOpen;
	});

	function isAgendaBusy(key: string) {
		return agendaActionPending !== '' && agendaActionPending !== key;
	}

	async function runAgendaAction(key: string, action: () => Promise<void>) {
		agendaActionPending = key;
		try {
			await action();
			return true;
		} catch (err) {
			onError(getDisplayError(err, 'Failed to update the agenda.'));
			onReload();
			return false;
		} finally {
			agendaActionPending = '';
		}
	}

	async function createAgendaPoint() {
		const title = agendaTitle.trim();
		if (!title || creatingAgenda) return;

		creatingAgenda = true;
		try {
			await agendaClient.createAgendaPoint({
				committeeSlug: slug,
				meetingId,
				title,
				parentAgendaPointId: agendaParentId
			});
			agendaTitle = '';
			agendaParentId = '';
			onReload();
			agendaTitleInput?.focus();
		} catch (err) {
			onError(getDisplayError(err, 'Failed to create the agenda point.'));
		} finally {
			creatingAgenda = false;
		}
	}

	async function handleAgendaTitleKeydown(event: KeyboardEvent) {
		if (event.key !== 'Enter') return;
		event.preventDefault();
		await createAgendaPoint();
	}

	async function activateAgendaPoint(agendaPointId: string, active: boolean) {
		await runAgendaAction(`activate-${agendaPointId}`, async () => {
			const res = await agendaClient.activateAgendaPoint({
				committeeSlug: slug,
				meetingId,
				agendaPointId: active ? '' : agendaPointId
			});
			onSetActivePoint(res.activeAgendaPoint);
			await onReload();
		});
	}

	async function moveAgendaPoint(agendaPointId: string, direction: 'up' | 'down') {
		await runAgendaAction(`move-${agendaPointId}-${direction}`, async () => {
			await agendaClient.moveAgendaPoint({
				committeeSlug: slug,
				meetingId,
				agendaPointId,
				direction
			});
			onReload();
		});
	}

	async function deleteAgendaPoint(agendaPointId: string) {
		if (!window.confirm(m.meeting_manage_delete_agenda_point_confirm())) return;
		await runAgendaAction(`delete-${agendaPointId}`, async () => {
			await agendaClient.deleteAgendaPoint({
				committeeSlug: slug,
				meetingId,
				agendaPointId
			});
			onReload();
		});
	}

	function startEditAgendaPoint(point: AgendaPointRecord) {
		editingAgendaPointId = point.agendaPointId;
		editingAgendaPointTitle = point.title;
	}

	function cancelEditAgendaPoint() {
		editingAgendaPointId = '';
		editingAgendaPointTitle = '';
	}

	async function saveEditAgendaPoint(agendaPointId: string) {
		const title = editingAgendaPointTitle.trim();
		if (!title) return;
		await runAgendaAction(`edit-${agendaPointId}`, async () => {
			await agendaClient.updateAgendaPoint({ committeeSlug: slug, meetingId, agendaPointId, title });
			onReload();
		});
		editingAgendaPointId = '';
		editingAgendaPointTitle = '';
	}

	function flattenAgenda(points: AgendaPointRecord[]) {
		const rows: Array<{ id: string; parentId: string; position: string; title: string }> = [];
		for (const point of points) {
			rows.push({
				id: point.agendaPointId,
				parentId: point.parentId,
				position: point.position.toString(),
				title: point.title.trim()
			});
			for (const child of point.subPoints) {
				rows.push({
					id: child.agendaPointId,
					parentId: child.parentId,
					position: child.position.toString(),
					title: child.title.trim()
				});
			}
		}
		return rows;
	}

	function agendaPointsFlat(points: AgendaPointRecord[] = agendaPoints) {
		const rows: AgendaPointRecord[] = [];
		const visit = (items: AgendaPointRecord[]) => {
			for (const item of items) {
				rows.push(item);
				if (item.subPoints.length) {
					visit(item.subPoints);
				}
			}
		};
		visit(points);
		return rows;
	}

	function agendaSiblings(point: AgendaPointRecord) {
		return agendaPointsFlat().filter((candidate) => candidate.parentId === point.parentId);
	}

	function agendaPointCanMoveUp(point: AgendaPointRecord) {
		return agendaSiblings(point).findIndex((candidate) => candidate.agendaPointId === point.agendaPointId) > 0;
	}

	function agendaPointCanMoveDown(point: AgendaPointRecord) {
		const siblings = agendaSiblings(point);
		const index = siblings.findIndex((candidate) => candidate.agendaPointId === point.agendaPointId);
		return index !== -1 && index < siblings.length - 1;
	}

	function legacyAgendaDisplayNumber(point: AgendaPointRecord) {
		if (point.displayNumber.startsWith('TOP')) return point.displayNumber;
		return `TOP ${point.displayNumber}`;
	}

	function currentAgendaFingerprint() {
		return JSON.stringify(flattenAgenda(agendaPoints));
	}

	async function fetchAgendaFingerprint() {
		const res = await agendaClient.listAgendaPoints({ committeeSlug: slug, meetingId });
		return JSON.stringify(flattenAgenda(res.agendaPoints));
	}

	function openAgendaImportDialog() {
		agendaImportOpen = true;
		agendaImportStep = 'input';
		agendaImportWarning = '';
		agendaImportRawText = '';
		agendaImportParsedText = '';
		agendaImportLineStates = new Map();
		agendaImportFormat = 'plaintext';
		agendaImportFormatManuallySet = false;
		agendaImportFingerprint = currentAgendaFingerprint();
		agendaImportDialogEl?.showModal();
	}

	function closeAgendaImportDialog() {
		agendaImportOpen = false;
		agendaImportStep = 'input';
		agendaImportWarning = '';
		agendaImportRawText = '';
		agendaImportParsedText = '';
		agendaImportLineStates = new Map();
		agendaImportFormat = 'plaintext';
		agendaImportFormatManuallySet = false;
		agendaImportDialogEl?.close();
	}

	function openAgendaEditDialog() {
		agendaEditOpen = true;
		agendaEditDialogEl?.showModal();
	}

	function closeAgendaEditDialog() {
		agendaEditOpen = false;
		agendaEditDialogEl?.close();
	}

	function agendaImportCurrentStep() {
		if (agendaImportStep === 'diff') return 2;
		return 1;
	}

	function setLinesFromSource(source: string) {
		const trimmed = source.trim();
		if (!trimmed) return;
		if (!agendaImportFormatManuallySet) {
			agendaImportFormat =
				/^#{1,6}\s/m.test(trimmed) || /^[-*+]\s+\S/m.test(trimmed) ? 'markdown' : 'plaintext';
		}
		agendaImportLineStates = new Map();
		agendaImportRawText = trimmed;
		agendaImportParsedText = trimmed;
	}

	function setAgendaImportFormat(fmt: 'markdown' | 'plaintext') {
		agendaImportFormat = fmt;
		agendaImportFormatManuallySet = true;
		agendaImportLineStates = new Map();
	}

	function runAgendaImportDetection() {
		const trimmed = agendaImportRawText.trim();
		if (!trimmed) return;
		agendaImportLineStates = new Map();
		agendaImportParsedText = trimmed;
	}

	function nextImportState(state: AgendaImportState): AgendaImportState {
		switch (state) {
			case 'ignore':
				return 'heading';
			case 'heading':
				return 'subheading';
			default:
				return 'ignore';
		}
	}

	function toggleAgendaImportLine(index: number) {
		const line = agendaImportLines[index];
		if (!line) return;
		agendaImportLineStates = new Map(agendaImportLineStates).set(line.lineNo, nextImportState(line.state));
	}

	function resetAgendaImportSource() {
		agendaImportRawText = '';
		agendaImportParsedText = '';
		agendaImportLineStates = new Map();
		agendaImportFormat = 'plaintext';
		agendaImportFormatManuallySet = false;
	}

	function generateAgendaDiff() {
		const trimmed = agendaImportRawText.trim();
		if (trimmed) agendaImportParsedText = trimmed;
		if (agendaImportLines.length === 0) {
			agendaImportWarning = m.agenda_import_error_empty_source();
			return;
		}
		if (buildImportedAgenda(agendaImportLines).length === 0) {
			agendaImportWarning = m.agenda_import_error_no_headings_after_correction();
			return;
		}
		agendaImportWarning = '';
		agendaImportFingerprint = currentAgendaFingerprint();
		agendaImportStep = 'diff';
	}

	async function applyAgendaImport() {
		if (agendaImportBusy) return;
		const diff = agendaImportDiff;
		if (diff.length === 0 || diff.every(r => r.op === 'unchanged')) {
			agendaImportWarning = 'No changes to apply.';
			return;
		}
		agendaImportBusy = true;
		agendaImportWarning = '';
		try {
			const latestFingerprint = await fetchAgendaFingerprint();
			if (agendaImportFingerprint !== latestFingerprint) {
				agendaImportWarning = m.agenda_import_warning_stale_diff();
				return;
			}

			// Phase 1: delete removed top-level points
			for (const row of diff) {
				if (row.op === 'deleted' && row.existingId) {
					await agendaClient.deleteAgendaPoint({ committeeSlug: slug, meetingId, agendaPointId: row.existingId });
				}
			}

			// Phase 2: rename surviving top-level points
			for (const row of diff) {
				if (row.existingId && row.importedTitle && row.existingTitle !== row.importedTitle) {
					await agendaClient.updateAgendaPoint({ committeeSlug: slug, meetingId, agendaPointId: row.existingId, title: row.importedTitle });
				}
			}

			// Phase 3: apply sub-point changes for surviving top-level points
			for (const row of diff) {
				if (!row.existingId || row.op === 'deleted') continue;
				for (const sub of row.subDiff) {
					if (sub.op === 'deleted' && sub.existingId) {
						await agendaClient.deleteAgendaPoint({ committeeSlug: slug, meetingId, agendaPointId: sub.existingId });
					} else if (sub.op === 'renamed' && sub.existingId && sub.importedTitle) {
						await agendaClient.updateAgendaPoint({ committeeSlug: slug, meetingId, agendaPointId: sub.existingId, title: sub.importedTitle });
					} else if (sub.op === 'added' && sub.importedTitle) {
						await agendaClient.createAgendaPoint({ committeeSlug: slug, meetingId, title: sub.importedTitle, parentAgendaPointId: row.existingId });
					} else if (sub.op === 'newParent' && sub.existingId && sub.importedTitle) {
						await agendaClient.deleteAgendaPoint({ committeeSlug: slug, meetingId, agendaPointId: sub.existingId });
						await agendaClient.createAgendaPoint({ committeeSlug: slug, meetingId, title: sub.importedTitle, parentAgendaPointId: row.existingId });
					}
				}
			}

			// Phase 4: add new top-level points with their sub-points
			const addedTopIds: string[] = [];
			for (const row of diff) {
				if (row.op !== 'added' || !row.importedTitle) continue;
				const res = await agendaClient.createAgendaPoint({ committeeSlug: slug, meetingId, title: row.importedTitle });
				const newId = res.agendaPoint?.agendaPointId;
				if (!newId) continue;
				addedTopIds.push(newId);
				for (const sub of row.subDiff) {
					if (sub.op === 'added' && sub.importedTitle) {
						await agendaClient.createAgendaPoint({ committeeSlug: slug, meetingId, title: sub.importedTitle, parentAgendaPointId: newId });
					}
				}
			}

			// Phase 5: reorder top-level points to match imported order
			const desiredIds: string[] = [];
			let addedIter = 0;
			for (const row of diff) {
				if (row.op === 'deleted') continue;
				if (row.existingId) desiredIds.push(row.existingId);
				else if (addedIter < addedTopIds.length) desiredIds.push(addedTopIds[addedIter++]);
			}
			if (desiredIds.length > 1) {
				const freshRes = await agendaClient.listAgendaPoints({ committeeSlug: slug, meetingId });
				const currentOrder = freshRes.agendaPoints.map(p => p.agendaPointId);
				for (let i = 0; i < desiredIds.length; i++) {
					const targetId = desiredIds[i];
					let j = currentOrder.indexOf(targetId);
					while (j > i) {
						await agendaClient.moveAgendaPoint({ committeeSlug: slug, meetingId, agendaPointId: targetId, direction: 'up' });
						currentOrder.splice(j, 1);
						currentOrder.splice(j - 1, 0, targetId);
						j--;
					}
				}
			}

			closeAgendaImportDialog();
			onReload();
		} catch (err) {
			agendaImportWarning = getDisplayError(err, 'Failed to apply the agenda import.');
		} finally {
			agendaImportBusy = false;
		}
	}

	function handleAgendaImportFile(event: Event) {
		const input = event.currentTarget as HTMLInputElement | null;
		const file = input?.files?.[0];
		if (!file) return;
		file.text().then((content) => {
			setLinesFromSource(content);
		});
	}
</script>

<section class="min-h-0" data-testid="manage-agenda-card">
	<div class="mb-3 flex items-center justify-between gap-2">
		<h2 class="text-lg font-semibold">{m.meeting_manage_agenda_points()}</h2>
		<div class="flex items-center gap-2">
			<button type="button" class="btn btn-sm btn-square btn-ghost" title="Open agenda help" aria-label="Open agenda help" onclick={() => goto(buildDocsOverlayHref('03-chairperson/03-agenda-management-and-import', page.url, { heading: 'agenda-routes' }))}><LegacyIcon name="help" class="h-4 w-4" /></button>
			<button type="button" class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Edit agenda" data-manage-dialog-open aria-controls="moderate-agenda-edit-dialog" title="Edit agenda" aria-label="Edit agenda" onclick={openAgendaEditDialog}><LegacyIcon name="settings" class="h-4 w-4" /></button>
		</div>
	</div>
	<div id="moderate-agenda-compact" class="space-y-2">
		{#if agendaPointsFlat().length === 0}
			<p class="text-sm text-base-content/70">{m.meeting_manage_no_agenda_points()}</p>
		{:else}
			<ul class="list rounded-box border border-base-300 bg-base-100">
				{#each agendaPointsFlat() as point}
					<li class={point.isActive ? 'list-row items-center gap-3 bg-primary/10' : point.parentId ? 'list-row items-center gap-3 pl-8' : 'list-row items-center gap-3'}>
						<span class="badge badge-outline">{legacyAgendaDisplayNumber(point)}</span>
						<span class={point.isActive ? 'flex-1 truncate font-semibold' : 'flex-1 truncate'}>{point.title}</span>
						{#if point.enteredAt}
							<span class="shrink-0 font-mono text-[0.65rem] text-base-content/50">{new Date(point.enteredAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
						{/if}
						{#if point.isActive}
							<span class="badge badge-success badge-sm">{m.meeting_manage_agenda_point_active_badge()}</span>
						{:else}
							<form class="inline" onsubmit={async (event) => { event.preventDefault(); await activateAgendaPoint(point.agendaPointId, point.isActive); }}>
								<button type="submit" class="btn btn-sm btn-square tooltip tooltip-left" data-tip="Activate agenda point" title="Activate agenda point" aria-label="Activate agenda point" disabled={isAgendaBusy(`activate-${point.agendaPointId}`)}><LegacyIcon name="check-circle" class="h-4 w-4" /></button>
							</form>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
	</div>
	<dialog
		id="moderate-agenda-edit-dialog"
		class="modal"
		data-manage-dialog
		bind:this={agendaEditDialogEl}
		onclose={() => {
			agendaEditOpen = false;
		}}
	>
		<div class="modal-box w-11/12 max-w-5xl">
			<div class="mb-4 flex items-center justify-between gap-2">
				<h3 class="text-lg font-semibold">{m.meeting_moderate_edit_agenda_title()}</h3>
				<button type="button" class="btn btn-sm btn-ghost" data-manage-dialog-close onclick={closeAgendaEditDialog}>{m.common_close()}</button>
			</div>
			<div id="agenda-point-list-container" class="space-y-3" data-import-open={agendaImportOpen ? 'true' : 'false'} data-import-top-prefix="TOP">
				<div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
					<div class="rounded-box border border-base-300 bg-base-100 p-3 lg:col-span-1">
						<form class="space-y-3" data-testid="manage-agenda-add-form" onsubmit={async (event) => { event.preventDefault(); await createAgendaPoint(); }}>
							<fieldset class="fieldset rounded-box border border-base-300 p-3">
								<legend class="fieldset-legend px-1 text-sm font-semibold">{m.meeting_manage_add_agenda_point()}</legend>
								<label class="label p-0 text-sm font-medium" for="ap_title">{m.meeting_manage_edit_agenda_point_title()}</label>
								<input class="input input-bordered input-sm w-full" type="text" id="ap_title" name="title" required placeholder={m.meeting_manage_agenda_point_title_placeholder()} bind:value={agendaTitle} bind:this={agendaTitleInput} onkeydown={handleAgendaTitleKeydown} />
								<label class="label mt-2 p-0 text-sm font-medium" for="ap_parent_id">{m.meeting_manage_agenda_point_parent_label()}</label>
								<select class="select select-bordered select-sm w-full" id="ap_parent_id" name="parent_id" bind:value={agendaParentId}>
									<option value="">-- top-level --</option>
									{#each agendaPointsFlat() as point}
										<option value={point.agendaPointId}>{point.title}</option>
									{/each}
								</select>
								<button type="submit" class="btn btn-sm mt-3 w-full"><LegacyIcon name="arrow-forward" class="h-4 w-4" />{m.meeting_manage_add_agenda_point()}</button>
								<button type="button" class="btn btn-sm btn-outline mt-2 w-full" data-manage-dialog-open aria-controls="moderate-agenda-import-dialog" onclick={openAgendaImportDialog}>{m.agenda_import_open_button()}</button>
							</fieldset>
						</form>
					</div>
					<div class="rounded-box border border-base-300 bg-base-100 p-3 lg:col-span-2">
						{#if agendaPointsFlat().length === 0}
							<p class="text-sm text-base-content/70">No agenda points have been created yet.</p>
						{:else}
							<div class="grid gap-3">
								{#each agendaPointsFlat() as point}
									<AgendaPointCard
										{point}
										isEditing={editingAgendaPointId === point.agendaPointId}
										bind:editTitle={editingAgendaPointTitle}
										canMoveUp={agendaPointCanMoveUp(point)}
										canMoveDown={agendaPointCanMoveDown(point)}
										{slug}
										{meetingId}
										isBusy={isAgendaBusy}
										onSave={async () => saveEditAgendaPoint(point.agendaPointId)}
										onCancelEdit={cancelEditAgendaPoint}
										onMoveUp={async () => moveAgendaPoint(point.agendaPointId, 'up')}
										onMoveDown={async () => moveAgendaPoint(point.agendaPointId, 'down')}
										onActivate={async () => activateAgendaPoint(point.agendaPointId, point.isActive)}
										onStartEdit={() => startEditAgendaPoint(point)}
										onDelete={async () => deleteAgendaPoint(point.agendaPointId)}
									/>
								{/each}
							</div>
						{/if}
					</div>
				</div>
				<dialog
					id="moderate-agenda-import-dialog"
					class="modal"
					data-manage-dialog
					bind:this={agendaImportDialogEl}
					onclose={() => {
						agendaImportOpen = false;
						agendaImportStep = 'input';
						agendaImportWarning = '';
					}}
				>
					<div class="modal-box w-11/12 max-w-5xl">
						<div class="mb-4 flex items-center justify-between gap-2">
							<h3 class="text-lg font-semibold">{m.agenda_import_title()}</h3>
							<button type="button" class="btn btn-sm btn-ghost" data-manage-dialog-close onclick={closeAgendaImportDialog}>{m.agenda_import_close()}</button>
						</div>
						<div class="space-y-4">
							{#if agendaImportWarning}
								<div class="alert alert-warning text-sm">{agendaImportWarning}</div>
							{/if}
							<div class="space-y-4" data-agenda-import-flow data-agenda-import-step={agendaImportCurrentStep().toString()}>
								<ul class="steps steps-horizontal w-full">
									<li class={agendaImportCurrentStep() >= 1 ? 'step step-primary' : 'step'} data-agenda-import-step-item="1">{m.agenda_import_step_source()}</li>
									<li class={agendaImportCurrentStep() >= 2 ? 'step step-primary' : 'step'} data-agenda-import-step-item="2">{m.agenda_import_step_diff()}</li>
								</ul>
								{#if agendaImportStep === 'input'}
									<div data-agenda-import-panel="1" class="space-y-3">
										<div class="flex items-center justify-between gap-2">
											<div class="flex items-center gap-2">
												<h4 class="text-sm font-medium">{m.agenda_import_correction_heading()}</h4>
											</div>
											<div class="flex items-center gap-2">
												<input type="file" class="file-input file-input-bordered file-input-sm" accept=".txt,.md,text/plain,text/markdown" data-agenda-import-file onchange={handleAgendaImportFile} />
												{#if agendaImportRawText.trim()}
													<button type="button" class="btn btn-xs btn-ghost" onclick={resetAgendaImportSource}>{m.agenda_import_back_button()}</button>
												{/if}
											</div>
										</div>
										<AgendaImportPreview
											bind:rawText={agendaImportRawText}
											lines={agendaImportLines}
											onToggle={toggleAgendaImportLine}
											onPasteText={setLinesFromSource}
										/>
										<div class="flex flex-wrap items-center justify-between gap-2">
											<div class="join">
												<button
													type="button"
													class={['join-item btn btn-sm', agendaImportFormat === 'plaintext' ? 'btn-active' : 'btn-ghost'].join(' ')}
													onclick={() => setAgendaImportFormat('plaintext')}
												>{m.agenda_import_format_plaintext()}</button>
												<button
													type="button"
													class={['join-item btn btn-sm', agendaImportFormat === 'markdown' ? 'btn-active' : 'btn-ghost'].join(' ')}
													onclick={() => setAgendaImportFormat('markdown')}
												>{m.agenda_import_format_markdown()}</button>
											</div>
											<div class="flex gap-2">
												<button type="button" class="btn btn-sm btn-outline" onclick={runAgendaImportDetection} disabled={!agendaImportRawText.trim()}>{m.agenda_import_detect_button()}</button>
												<button type="button" class="btn btn-sm btn-primary" onclick={generateAgendaDiff} disabled={agendaImportLines.length === 0}>{m.agenda_import_generate_diff_button()}</button>
											</div>
										</div>
									</div>
								{/if}
								{#if agendaImportStep === 'diff'}
									<div data-agenda-import-panel="2">
										<div class="space-y-3">
											<h4 class="text-base font-semibold">{m.agenda_import_diff_heading()}</h4>
											{#if agendaImportDiff.every(r => r.op === 'unchanged')}
												<div class="alert alert-info text-sm">{m.agenda_import_diff_no_changes()}</div>
											{:else}
												<div class="rounded-box border border-base-300 bg-base-100">
													<div class="grid grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,2fr)] border-b border-base-300 px-3 py-1.5 text-xs font-semibold uppercase tracking-wide text-base-content/50">
														<div>{m.agenda_import_diff_column_current()}</div>
														<div class="text-center">{m.agenda_import_diff_column_change()}</div>
														<div>{m.agenda_import_diff_column_imported()}</div>
													</div>
													<ul>
														{#each agendaImportDiff as row, rowIdx}
															{@const topKey = `t:${rowIdx}`}
															{@const topHovered = agendaDiffHoverId === topKey}
															{@const exTopNum = agendaImportDiff.slice(0, rowIdx + 1).filter(r => r.existingTitle !== null).length}
															{@const impTopNum = agendaImportDiff.slice(0, rowIdx + 1).filter(r => r.importedTitle !== null).length}
															<li
																class={['grid grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,2fr)] border-t border-base-300/60 text-sm',
																	row.op === 'unchanged' ? 'opacity-40' : ''].join(' ')}
															>
																<div
																	class={['min-h-10 px-3 py-2 transition-colors',
																		topHovered ? 'bg-base-200/60' : '',
																		row.op === 'deleted' ? 'text-error' : ''].join(' ')}
																	role="presentation"
																	onmouseenter={() => (agendaDiffHoverId = topKey)}
																	onmouseleave={() => (agendaDiffHoverId = null)}
																>
																	{#if row.existingTitle !== null}
																		<div class="text-xs text-base-content/40">TOP {exTopNum}</div>
																		<div class="font-medium">{row.existingTitle}</div>
																	{/if}
																</div>
																<div class="flex min-h-10 flex-col items-center justify-center gap-0.5 px-1">
																	{#if row.op === 'added'}
																		<span class="rounded bg-success/15 px-1.5 py-0.5 text-xs font-semibold text-success">{m.agenda_import_diff_op_added()}</span>
																	{:else if row.op === 'deleted'}
																		<span class="rounded bg-error/15 px-1.5 py-0.5 text-xs font-semibold text-error">{m.agenda_import_diff_op_deleted()}</span>
																	{:else if row.op === 'renamed'}
																		<span class="rounded bg-warning/15 px-1.5 py-0.5 text-xs font-semibold text-warning">{m.agenda_import_diff_op_renamed()}</span>
																	{:else if row.op === 'reordered'}
																		<span class="rounded bg-info/15 px-1.5 py-0.5 text-xs font-semibold text-info">{m.agenda_import_diff_op_reordered()}</span>
																	{:else if row.op === 'renamed+reordered'}
																		<span class="rounded bg-warning/15 px-1.5 py-0.5 text-xs font-semibold text-warning">{m.agenda_import_diff_op_renamed_reordered()}</span>
																	{:else if row.op === 'newRoot'}
																		<span class="rounded bg-secondary/15 px-1.5 py-0.5 text-xs font-semibold text-secondary">{m.agenda_import_diff_op_new_root()}</span>
																	{/if}
																</div>
																<div
																	class={['min-h-10 px-3 py-2 transition-colors',
																		topHovered ? 'bg-base-200/60' : '',
																		row.op === 'added' ? 'text-success' : ''].join(' ')}
																	role="presentation"
																	onmouseenter={() => (agendaDiffHoverId = topKey)}
																	onmouseleave={() => (agendaDiffHoverId = null)}
																>
																	{#if row.importedTitle !== null}
																		<div class="text-xs text-base-content/40">TOP {impTopNum}</div>
																		<div class="font-medium">{row.importedTitle}</div>
																	{/if}
																</div>
															</li>
															{#each row.subDiff as sub, subIdx}
																{@const subKey = `s:${rowIdx}:${subIdx}`}
																{@const subHovered = agendaDiffHoverId === subKey}
																<li
																	class={['grid grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,2fr)] border-t border-base-300/40 text-sm',
																		sub.op === 'unchanged' ? 'opacity-40' : ''].join(' ')}
																>
																	<div
																		class={['min-h-8 py-1.5 pl-8 pr-3 transition-colors',
																			subHovered ? 'bg-base-200/60' : '',
																			sub.op === 'deleted' ? 'text-error' : 'text-base-content/70'].join(' ')}
																		role="presentation"
																		onmouseenter={() => (agendaDiffHoverId = subKey)}
																		onmouseleave={() => (agendaDiffHoverId = null)}
																	>
																		{#if sub.existingTitle !== null}
																			<div class="text-xs italic">{sub.existingTitle}</div>
																		{/if}
																	</div>
																	<div class="flex min-h-8 flex-col items-center justify-center px-1">
																		{#if sub.op !== 'unchanged'}
																			{#if sub.op === 'added'}
																				<span class="rounded bg-success/15 px-1.5 py-0.5 text-xs font-semibold text-success">{m.agenda_import_diff_op_added()}</span>
																			{:else if sub.op === 'deleted'}
																				<span class="rounded bg-error/15 px-1.5 py-0.5 text-xs font-semibold text-error">{m.agenda_import_diff_op_deleted()}</span>
																			{:else if sub.op === 'renamed'}
																				<span class="rounded bg-warning/15 px-1.5 py-0.5 text-xs font-semibold text-warning">{m.agenda_import_diff_op_renamed()}</span>
																			{:else if sub.op === 'reordered'}
																				<span class="rounded bg-info/15 px-1.5 py-0.5 text-xs font-semibold text-info">{m.agenda_import_diff_op_reordered()}</span>
																			{:else if sub.op === 'renamed+reordered'}
																				<span class="rounded bg-warning/15 px-1.5 py-0.5 text-xs font-semibold text-warning">{m.agenda_import_diff_op_renamed_reordered()}</span>
																			{:else if sub.op === 'newParent'}
																				<span class="rounded bg-secondary/15 px-1.5 py-0.5 text-xs font-semibold text-secondary">{m.agenda_import_diff_op_new_parent()}</span>
																			{/if}
																		{/if}
																	</div>
																	<div
																		class={['min-h-8 py-1.5 pl-8 pr-3 transition-colors',
																			subHovered ? 'bg-base-200/60' : '',
																			sub.op === 'added' ? 'text-success' : 'text-base-content/70'].join(' ')}
																		role="presentation"
																		onmouseenter={() => (agendaDiffHoverId = subKey)}
																		onmouseleave={() => (agendaDiffHoverId = null)}
																	>
																		{#if sub.importedTitle !== null}
																			<div class="text-xs italic">{sub.importedTitle}</div>
																		{/if}
																	</div>
																</li>
															{/each}
														{/each}
													</ul>
												</div>
												{#if agendaImportDiff.some(r => r.op === 'deleted')}
													<div class="alert alert-warning text-sm">{m.agenda_import_diff_has_deletions_warning()}</div>
												{/if}
											{/if}
											<div class="flex flex-wrap gap-2">
												<button type="button" class="btn btn-sm btn-ghost" data-agenda-import-back="1" onclick={() => (agendaImportStep = 'input')}>{m.agenda_import_back_button()}</button>
												<form class="inline-flex" onsubmit={(event) => { event.preventDefault(); void applyAgendaImport(); }}>
													<button type="submit" class="btn btn-sm btn-primary" disabled={agendaImportBusy || agendaImportDiff.every(r => r.op === 'unchanged')}>{m.agenda_import_accept_button()}</button>
												</form>
												<button type="button" class="btn btn-sm btn-ghost" data-manage-dialog-close onclick={closeAgendaImportDialog}>{m.agenda_import_deny_button()}</button>
											</div>
										</div>
									</div>
								{/if}
							</div>
						</div>
					</div>
					<form method="dialog" class="modal-backdrop">
						<button aria-label="Close">Close</button>
					</form>
				</dialog>
			</div>
		</div>
		<form method="dialog" class="modal-backdrop">
			<button aria-label="Close">Close</button>
		</form>
	</dialog>
</section>
