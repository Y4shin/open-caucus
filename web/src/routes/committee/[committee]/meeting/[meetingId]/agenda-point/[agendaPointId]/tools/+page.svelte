<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { agendaClient } from '$lib/api/index.js';
	import type { AgendaPointToolsView, AttachmentRecord } from '$lib/gen/conference/agenda/v1/agenda_pb.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import * as m from '$lib/paraglide/messages';

	const slug = $derived(page.params.committee);
	const meetingId = $derived(page.params.meetingId);
	const agendaPointId = $derived(page.params.agendaPointId);

	let toolsState = $state(createRemoteState<AgendaPointToolsView>());
	let actionError = $state('');
	let label = $state('');
	let fileInput = $state<HTMLInputElement | null>(null);
	let uploadPending = $state(false);
	let currentAction = $state('');

	$effect(() => {
		if (!session.loaded) return;
		if (!session.authenticated) {
			goto('/login');
			return;
		}
		loadTools();
	});

	async function loadTools() {
		toolsState.loading = true;
		toolsState.error = '';
		try {
			const res = await agendaClient.getAgendaPointTools({
				committeeSlug: slug,
				meetingId,
				agendaPointId
			});
			toolsState.data = res.view ?? null;
		} catch (err) {
			toolsState.error = getDisplayError(err, 'Failed to load agenda-point tools.');
		} finally {
			toolsState.loading = false;
		}
	}

	async function uploadAttachment() {
		const file = fileInput?.files?.[0];
		if (!file || uploadPending) return;

		uploadPending = true;
		actionError = '';

		try {
			const body = new FormData();
			body.set('file', file);
			body.set('label', label.trim());

			const response = await fetch(
				`/api/committee/${slug}/meeting/${meetingId}/agenda-point/${agendaPointId}/attachments`,
				{
					method: 'POST',
					body
				}
			);

			if (!response.ok) {
				const payload = (await response.json().catch(() => null)) as { error?: string } | null;
				throw new Error(payload?.error || `upload failed (${response.status})`);
			}

			label = '';
			if (fileInput) {
				fileInput.value = '';
			}
			await loadTools();
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to upload attachment.');
		} finally {
			uploadPending = false;
		}
	}

	async function setCurrent(attachmentId: string) {
		currentAction = `set:${attachmentId}`;
		actionError = '';
		try {
			const res = await agendaClient.setCurrentAttachment({
				committeeSlug: slug,
				meetingId,
				agendaPointId,
				attachmentId
			});
			toolsState.data = res.view ?? toolsState.data;
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to publish the current document.');
		} finally {
			currentAction = '';
		}
	}

	async function clearCurrent() {
		currentAction = 'clear';
		actionError = '';
		try {
			const res = await agendaClient.clearCurrentDocument({
				committeeSlug: slug,
				meetingId,
				agendaPointId
			});
			toolsState.data = res.view ?? toolsState.data;
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to clear the current document.');
		} finally {
			currentAction = '';
		}
	}

	async function deleteAttachment(attachment: AttachmentRecord) {
		if (!window.confirm(m.attachment_delete_confirm())) {
			return;
		}

		currentAction = `delete:${attachment.attachmentId}`;
		actionError = '';
		try {
			const res = await agendaClient.deleteAttachment({
				committeeSlug: slug,
				meetingId,
				agendaPointId,
				attachmentId: attachment.attachmentId
			});
			toolsState.data = res.view ?? toolsState.data;
		} catch (err) {
			actionError = getDisplayError(err, 'Failed to delete the attachment.');
		} finally {
			currentAction = '';
		}
	}
</script>

<div class="manage-grid grid gap-4 lg:grid-cols-1">

	{#if toolsState.loading}
		<AppSpinner label="Loading agenda-point tools" />
	{:else}
		{#if toolsState.error}
			<AppAlert message={toolsState.error} />
		{/if}
		{#if actionError}
			<AppAlert message={actionError} />
		{/if}

		{#if toolsState.data}
			<section class="panel card bg-base-100 border border-base-300 shadow-sm rounded-box p-4 mb-4">
				<h2>{m.attachment_heading()}</h2>
				<div id="attachment-list-ap-{agendaPointId}">
					<h4>{m.meeting_moderate_attachments_for({ agendaPoint: toolsState.data.agendaPointTitle })}</h4>
					<form
						onsubmit={async (event) => {
							event.preventDefault();
							await uploadAttachment();
						}}
					>
						<div>
							<label for="attachment-label-{agendaPointId}">{m.attachment_label_label()}</label>
							<input
								class="input input-bordered input-sm"
								type="text"
								id="attachment-label-{agendaPointId}"
								name="label"
								bind:value={label}
							/>
						</div>
						<div>
							<label for="attachment-file-{agendaPointId}">{m.attachment_file_label()}</label>
							<input
								class="file-input file-input-bordered file-input-sm"
								id="attachment-file-{agendaPointId}"
								name="file"
								bind:this={fileInput}
								type="file"
								required
							/>
						</div>
						<button class="btn btn-sm" type="submit" disabled={uploadPending}>{m.attachment_upload_button()}</button>
					</form>
					{#if toolsState.data.attachments.length === 0}
						<p>{m.attachment_empty_state()}</p>
					{:else}
						<ul>
							{#each toolsState.data.attachments as attachment}
								<li id="attachment-item-{attachment.attachmentId}">
									<a href={attachment.downloadUrl}>
										{#if attachment.label}
											{attachment.label} ({attachment.filename})
										{:else}
											{attachment.filename}
										{/if}
									</a>
									<form
										class="inline-form inline"
										onsubmit={async (event) => {
											event.preventDefault();
											await deleteAttachment(attachment);
										}}
									>
										<button class="btn btn-sm" type="submit" disabled={currentAction === `delete:${attachment.attachmentId}`}>
											{m.attachment_delete_button()}
										</button>
									</form>
									{#if attachment.isCurrent}
										<form
											class="inline-form inline"
											onsubmit={async (event) => {
												event.preventDefault();
												await clearCurrent();
											}}
										>
											<button class="btn btn-sm" type="submit" disabled={currentAction === 'clear'}>
												{m.attachment_clear_button()}
											</button>
										</form>
									{:else}
										<form
											class="inline-form inline"
											onsubmit={async (event) => {
												event.preventDefault();
												await setCurrent(attachment.attachmentId);
											}}
										>
											<button class="btn btn-sm" type="submit" disabled={currentAction === `set:${attachment.attachmentId}`}>
												{m.attachment_set_current_button()}
											</button>
										</form>
									{/if}
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			</section>
		{/if}
	{/if}
</div>
