<script lang="ts">
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
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

	function loadReceipts() {
		receipts = listReceipts();
		status = `Loaded ${receipts.length} receipt${receipts.length === 1 ? '' : 's'}.`;
		error = '';
	}

	function clearAll() {
		clearReceipts();
		verifyResults = {};
		loadReceipts();
	}

	async function verifyOne(receipt: StoredReceipt) {
		verifyingId = receipt.id;
		error = '';
		try {
			const payload = await verifyReceipt(receipt);
			if (receipt.kind === 'open') {
				const labels = Array.isArray(payload?.choice_labels)
					? (payload.choice_labels as string[]).join(', ')
					: '';
				verifyResults[receipt.id] = labels || 'Verified.';
			} else {
				verifyResults[receipt.id] = 'Verified secret-ballot commitment.';
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Verification failed.';
		} finally {
			verifyingId = '';
		}
	}

	loadReceipts();
</script>

<div class="space-y-6">
	<div class="space-y-2">
		<h1 class="text-3xl font-bold">Receipts Vault</h1>
		<p class="text-base-content/70">
			Review locally stored ballot receipts and verify them against the backend.
		</p>
	</div>

	{#if error}
		<AppAlert message={error} />
	{/if}

	<AppCard title="Stored Receipts">
		<div class="mb-4 flex flex-wrap gap-2">
			<button class="btn btn-outline btn-sm" type="button" onclick={loadReceipts}>Refresh</button>
			<button class="btn btn-error btn-outline btn-sm" type="button" onclick={clearAll}>Clear All</button>
		</div>

		<p class="mb-4 text-sm text-base-content/70">{status}</p>

		{#if receipts.length}
			<div class="space-y-3">
				{#each receipts as receipt}
					<div class="rounded-box border border-base-300 bg-base-100 p-4">
						<div class="flex flex-wrap items-start justify-between gap-3">
							<div class="space-y-2">
								<div class="flex flex-wrap items-center gap-2">
									<span class="badge badge-outline">{receipt.kind}</span>
									<span class="font-medium">{receipt.voteName}</span>
								</div>
								<p class="text-sm text-base-content/70">Vote #{receipt.voteId}</p>
								<p class="font-mono text-xs break-all">{receipt.receipt}</p>
								{#if verifyResults[receipt.id]}
									<p class="text-sm text-success">{verifyResults[receipt.id]}</p>
								{/if}
							</div>
							<button
								class="btn btn-primary btn-sm"
								type="button"
								disabled={verifyingId === receipt.id}
								onclick={() => verifyOne(receipt)}
							>
								{verifyingId === receipt.id ? 'Verifying...' : 'Verify'}
							</button>
						</div>
					</div>
				{/each}
			</div>
		{:else}
			<AppAlert tone="info" message="No stored receipts yet." />
		{/if}
	</AppCard>
</div>
