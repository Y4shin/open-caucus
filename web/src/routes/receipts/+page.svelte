<script lang="ts">
	import { onMount } from 'svelte';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import * as m from '$lib/paraglide/messages';
	import {
		clearReceipts,
		listReceipts,
		type StoredReceipt,
		verifyReceipt
	} from '$lib/utils/receipts.js';

	let receipts = $state<StoredReceipt[]>([]);
	let status = $state('');
	let error = $state('');
	let verifyingId = $state('');
	let verifyResults = $state<Record<string, string>>({});
	const textNoStoredReceipts = 'No stored receipts.';
	const textUnknown = 'unknown';
	const textVote = 'Vote';
	const textVotePrefix = 'vote #';
	const textVerify = 'Verify';
	const textVerifying = 'verifying...';
	const textOkPrefix = 'OK: ';
	const textNoChoices = 'no choices';
	const textCommitmentReturned = 'commitment returned';
	const textErrorPrefix = 'Error: ';
	const textLoadingReceipts = 'Loading receipts...';
	const textLoadedReceipts = 'Loaded receipt(s): ';
	const textFailedLoadReceipts = 'Failed to load receipts: ';
	const textFailedClearReceipts = 'Failed to clear receipts: ';

	onMount(() => {
		pageActions.set([], { title: m.votes_receipt_vault_title() });
		loadReceipts();
		return () => pageActions.clear();
	});

	function sortedReceipts(items: StoredReceipt[]) {
		return [...items].sort((a, b) => {
			const voteCmp = a.voteId.localeCompare(b.voteId, undefined, { numeric: true });
			if (voteCmp !== 0) {
				return voteCmp;
			}
			return a.id.localeCompare(b.id);
		});
	}

	function setStatus(message: string, isError: boolean) {
		status = message;
		error = isError ? message : '';
	}

	function loadReceipts() {
		try {
			setStatus(textLoadingReceipts, false);
			receipts = sortedReceipts(listReceipts());
			setStatus(`${textLoadedReceipts}${receipts.length}.`, false);
		} catch (err) {
			const message = err instanceof Error ? err.message : 'unknown error';
			receipts = [];
			setStatus(`${textFailedLoadReceipts}${message}`, true);
		}
	}

	function clearAll() {
		try {
			clearReceipts();
			verifyResults = {};
			loadReceipts();
		} catch (err) {
			const message = err instanceof Error ? err.message : 'unknown error';
			setStatus(`${textFailedClearReceipts}${message}`, true);
		}
	}

	async function verifyOne(receipt: StoredReceipt) {
		verifyingId = receipt.id;
		error = '';
		try {
			const payload = await verifyReceipt(receipt);
			const labels =
				payload && 'choiceLabels' in payload && Array.isArray(payload.choiceLabels)
					? payload.choiceLabels.join(', ')
					: '';
			verifyResults[receipt.id] = `${textOkPrefix}${labels || textNoChoices}`;
		} catch (err) {
			verifyResults[receipt.id] = `${textErrorPrefix}${err instanceof Error ? err.message : 'Verification failed.'}`;
		} finally {
			verifyingId = '';
		}
	}
</script>

<div
	id="receipts-vault-content"
	class="mx-auto w-full max-w-5xl space-y-4"
	data-text-no-stored={textNoStoredReceipts}
	data-text-unknown={textUnknown}
	data-text-vote={textVote}
	data-text-vote-prefix={textVotePrefix}
	data-text-verify={textVerify}
	data-text-verifying={textVerifying}
	data-text-ok-prefix={textOkPrefix}
	data-text-no-choices={textNoChoices}
	data-text-commitment-returned={textCommitmentReturned}
	data-text-error-prefix={textErrorPrefix}
	data-text-loading-receipts={textLoadingReceipts}
	data-text-loaded-receipts={textLoadedReceipts}
	data-text-failed-load-receipts={textFailedLoadReceipts}
	data-text-failed-clear-receipts={textFailedClearReceipts}
	data-text-verify-failed-status="verify failed with status "
>
	<h1 class="text-2xl font-semibold">{m.votes_receipt_vault_title()}</h1>
	<p class="text-sm text-base-content/70">{m.votes_receipt_vault_description()}</p>

	{#if error}
		<AppAlert message={error} />
	{/if}

	<div class="rounded-box border border-base-300 bg-base-100 p-3 space-y-3">
		<div class="flex flex-wrap items-center gap-2">
			<button id="receipts-refresh" class="btn btn-sm btn-outline" type="button" onclick={loadReceipts}>{m.common_refresh()}</button>
			<button id="receipts-clear" class="btn btn-sm btn-error btn-outline" type="button" onclick={clearAll}>{m.votes_clear_all()}</button>
		</div>

		<div id="receipts-status" class={error ? 'text-sm text-error' : 'text-sm text-base-content/70'}>
			{status}
		</div>

		{#if receipts.length}
			<div id="receipts-list" class="space-y-2">
				{#each receipts as receipt}
					<div class="rounded-box border border-base-300 bg-base-200/30 p-3 space-y-2">
						<div class="flex flex-wrap items-center gap-2">
							<span class="badge badge-outline badge-sm">{receipt.kind || m.votes_unknown()}</span>
							<span class="font-semibold">{receipt.voteName || m.votes_vote()}</span>
							<span class="text-xs text-base-content/70">{m.votes_vote_number_prefix()}{receipt.voteId || '?'}</span>
						</div>
						<div class="text-xs text-base-content/70 break-all">{receipt.receipt}</div>
						<div class="flex flex-wrap items-center gap-2">
							<button
								class="btn btn-xs btn-primary"
								type="button"
								disabled={verifyingId === receipt.id}
								onclick={() => verifyOne(receipt)}
							>
								{verifyingId === receipt.id ? m.votes_verifying() : m.votes_verify()}
							</button>
							<span
								data-result
								class={verifyResults[receipt.id]?.startsWith(textErrorPrefix)
									? 'text-xs text-error'
									: 'text-xs text-success'}
							>{verifyResults[receipt.id] || ''}</span>
						</div>
					</div>
				{/each}
			</div>
		{:else}
			<div id="receipts-list" class="space-y-2"><p class="text-sm text-base-content/70">{m.votes_no_stored_receipts()}</p></div>
		{/if}
	</div>
</div>
