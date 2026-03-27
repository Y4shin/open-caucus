<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { agendaClient } from '$lib/api/index.js';
	import type { AgendaPointToolsView, AttachmentRecord } from '$lib/gen/conference/agenda/v1/agenda_pb.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';

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
		if (!window.confirm(`Delete "${attachment.label || attachment.filename}"?`)) {
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

<div class="space-y-6">
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div class="space-y-2">
			<h1 class="text-3xl font-bold">{toolsState.data?.agendaPointTitle ?? 'Agenda Point Tools'}</h1>
			<p class="text-base-content/70">Manage attachments and the currently published live document.</p>
		</div>
		<a class="btn btn-outline" href="/committee/{slug}/meeting/{meetingId}/moderate">Back to Moderation</a>
	</div>

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
			<div class="stats shadow">
				<div class="stat">
					<div class="stat-title">Agenda Point</div>
					<div class="stat-value text-lg">{toolsState.data.agendaPointTitle}</div>
				</div>
				<div class="stat">
					<div class="stat-title">Attachments</div>
					<div class="stat-value">{toolsState.data.attachments.length}</div>
				</div>
				<div class="stat">
					<div class="stat-title">Published</div>
					<div class="stat-value text-lg">
						{toolsState.data.currentAttachmentId ? 'Yes' : 'No'}
					</div>
				</div>
			</div>

			<div class="grid gap-6 xl:grid-cols-[minmax(0,24rem)_minmax(0,1fr)]">
				<AppCard title="Upload Attachment">
					<form
						class="space-y-4"
						onsubmit={async (event) => {
							event.preventDefault();
							await uploadAttachment();
						}}
					>
						<label class="form-control gap-2">
							<span class="label-text font-medium">Label</span>
							<input
								class="input input-bordered"
								bind:value={label}
								placeholder="Budget Proposal"
							/>
						</label>

						<label class="form-control gap-2">
							<span class="label-text font-medium">File</span>
							<input class="file-input file-input-bordered" bind:this={fileInput} type="file" required />
						</label>

						<button class="btn btn-primary" type="submit" disabled={uploadPending}>
							{uploadPending ? 'Uploading...' : 'Upload Attachment'}
						</button>
					</form>
				</AppCard>

				<AppCard title="Attachments">
					{#if toolsState.data.attachments.length}
						<div class="space-y-3">
							{#each toolsState.data.attachments as attachment}
								<div class="rounded-box border border-base-300 bg-base-100 p-4">
									<div class="flex flex-wrap items-start justify-between gap-3">
										<div class="space-y-2">
											<div class="flex flex-wrap items-center gap-2">
												<span class="font-medium">
													{attachment.label || attachment.filename}
												</span>
												{#if attachment.isCurrent}
													<span class="badge badge-primary">Current</span>
												{/if}
											</div>
											<p class="text-sm text-base-content/70">{attachment.filename}</p>
											<a
												class="link link-primary text-sm"
												href={attachment.downloadUrl}
												target="_blank"
												rel="noreferrer"
											>
												Download
											</a>
										</div>

										<div class="flex flex-wrap gap-2">
											{#if attachment.isCurrent}
												<button
													class="btn btn-sm btn-outline"
													type="button"
													disabled={currentAction === 'clear'}
													onclick={clearCurrent}
												>
													{currentAction === 'clear' ? 'Clearing...' : 'Clear'}
												</button>
											{:else}
												<button
													class="btn btn-sm btn-primary"
													type="button"
													disabled={currentAction === `set:${attachment.attachmentId}`}
													onclick={() => setCurrent(attachment.attachmentId)}
												>
													{currentAction === `set:${attachment.attachmentId}`
														? 'Publishing...'
														: 'Set as Current'}
												</button>
											{/if}
											<button
												class="btn btn-sm btn-error btn-outline"
												type="button"
												disabled={currentAction === `delete:${attachment.attachmentId}`}
												onclick={() => deleteAttachment(attachment)}
											>
												{currentAction === `delete:${attachment.attachmentId}`
													? 'Deleting...'
													: 'Delete'}
											</button>
										</div>
									</div>
								</div>
							{/each}
						</div>
					{:else}
						<AppAlert tone="info" message="No attachments have been uploaded for this agenda point yet." />
					{/if}
				</AppCard>
			</div>
		{/if}
	{/if}
</div>
