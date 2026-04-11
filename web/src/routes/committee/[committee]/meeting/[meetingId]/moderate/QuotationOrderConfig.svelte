<script lang="ts">
	import { QuotationType } from '$lib/gen/conference/common/v1/common_pb.js';
	import * as m from '$lib/paraglide/messages';
	import QuotationExplainer from './QuotationExplainer.svelte';

	let {
		quotationOrder = $bindable<QuotationType[]>([]),
		disabled = false,
		onOrderChange
	}: {
		quotationOrder?: QuotationType[];
		disabled?: boolean;
		onOrderChange: (order: QuotationType[]) => void;
	} = $props();

	const allTypes: { type: QuotationType; label: () => string; icon: string; description: () => string }[] = [
		{
			type: QuotationType.GENDER,
			label: () => m.meeting_moderate_flinta_quotation_label(),
			icon: '⚧',
			description: () => m.quotation_gender_description()
		},
		{
			type: QuotationType.FIRST_SPEAKER,
			label: () => m.meeting_moderate_first_speaker_bonus_label(),
			icon: '🎤',
			description: () => m.quotation_first_speaker_description()
		}
	];

	let enabledTypes = $derived(quotationOrder.filter((t): t is QuotationType => t !== QuotationType.UNSPECIFIED));
	let disabledTypes = $derived(
		allTypes.filter((t) => !enabledTypes.includes(t.type as QuotationType)).map((t) => t.type as QuotationType)
	);

	let draggedType = $state<QuotationType | null>(null);
	let dragOverZone = $state<'enabled' | 'disabled' | null>(null);
	let dragOverIndex = $state<number | null>(null);

	function typeInfo(t: QuotationType) {
		return allTypes.find((a) => a.type === t)!;
	}

	function handleDragStart(t: QuotationType) {
		if (disabled) return;
		draggedType = t;
	}

	function handleDragEnd() {
		draggedType = null;
		dragOverZone = null;
		dragOverIndex = null;
	}

	function handleDropEnabled(targetIndex: number) {
		if (draggedType === null || disabled) return;
		const newEnabled = enabledTypes.filter((t) => t !== draggedType);
		newEnabled.splice(targetIndex, 0, draggedType);
		quotationOrder = newEnabled;
		onOrderChange(newEnabled);
		handleDragEnd();
	}

	function handleDropDisabled() {
		if (draggedType === null || disabled) return;
		const newEnabled = enabledTypes.filter((t) => t !== draggedType);
		quotationOrder = newEnabled;
		onOrderChange(newEnabled);
		handleDragEnd();
	}

	function moveUp(index: number) {
		if (index <= 0 || disabled) return;
		const newOrder = [...enabledTypes];
		[newOrder[index - 1], newOrder[index]] = [newOrder[index], newOrder[index - 1]];
		quotationOrder = newOrder;
		onOrderChange(newOrder);
	}

	function moveDown(index: number) {
		if (index >= enabledTypes.length - 1 || disabled) return;
		const newOrder = [...enabledTypes];
		[newOrder[index], newOrder[index + 1]] = [newOrder[index + 1], newOrder[index]];
		quotationOrder = newOrder;
		onOrderChange(newOrder);
	}

	function toggleType(t: QuotationType) {
		if (disabled) return;
		if (enabledTypes.includes(t)) {
			const newOrder = enabledTypes.filter((e) => e !== t);
			quotationOrder = newOrder;
			onOrderChange(newOrder);
		} else {
			const newOrder = [...enabledTypes, t];
			quotationOrder = newOrder;
			onOrderChange(newOrder);
		}
	}
</script>

<div class="space-y-3">
	<div>
		<p class="text-xs text-base-content/60 leading-relaxed">{m.quotation_explanation()}</p>
		<div class="mt-2 flex justify-center">
			<QuotationExplainer {quotationOrder} />
		</div>
	</div>

	<!-- Enabled zone -->
	<div>
		<div class="mb-1 text-xs font-semibold uppercase tracking-wide text-base-content/60">{m.quotation_enabled_heading()}</div>
		<div
			class="min-h-12 rounded-box border-2 p-2 space-y-1 transition-colors {dragOverZone === 'enabled' ? 'border-primary bg-primary/5' : 'border-base-300 bg-base-200/30'}"
			role="list"
			ondragover={(e) => { e.preventDefault(); dragOverZone = 'enabled'; }}
			ondragleave={() => { if (dragOverZone === 'enabled') dragOverZone = null; }}
			ondrop={() => handleDropEnabled(enabledTypes.length)}
		>
			{#each enabledTypes as t, i (t)}
				{@const info = typeInfo(t)}
				<div
					class="flex items-center gap-2 rounded-box border border-base-300 bg-base-100 px-3 py-2 shadow-sm {disabled ? 'opacity-50' : 'cursor-grab'}"
					role="listitem"
					draggable={!disabled}
					ondragstart={() => handleDragStart(t)}
					ondragend={handleDragEnd}
					ondragover={(e) => { e.preventDefault(); e.stopPropagation(); dragOverZone = 'enabled'; dragOverIndex = i; }}
					ondrop={(e) => { e.preventDefault(); e.stopPropagation(); handleDropEnabled(i); }}
				>
					<span class="text-base-content/40 font-mono text-sm font-bold">{i + 1}</span>
					<span class="text-lg" aria-hidden="true">{info.icon}</span>
					<div class="flex-1 min-w-0">
						<div class="text-sm font-medium">{info.label()}</div>
						<div class="text-xs text-base-content/60">{info.description()}</div>
					</div>
					<div class="flex items-center gap-1">
						<button type="button" class="btn btn-ghost btn-xs btn-square" aria-label="Move up" {disabled} onclick={() => moveUp(i)} class:invisible={i === 0}>
							<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4"><path fill-rule="evenodd" d="M9.47 6.47a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 1 1-1.06 1.06L10 8.06l-3.72 3.72a.75.75 0 0 1-1.06-1.06l4.25-4.25Z" clip-rule="evenodd" /></svg>
						</button>
						<button type="button" class="btn btn-ghost btn-xs btn-square" aria-label="Move down" {disabled} onclick={() => moveDown(i)} class:invisible={i === enabledTypes.length - 1}>
							<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4"><path fill-rule="evenodd" d="M5.22 8.22a.75.75 0 0 1 1.06 0L10 11.94l3.72-3.72a.75.75 0 1 1 1.06 1.06l-4.25 4.25a.75.75 0 0 1-1.06 0l-4.25-4.25a.75.75 0 0 1 0-1.06Z" clip-rule="evenodd" /></svg>
						</button>
						<button type="button" class="btn btn-ghost btn-xs btn-square text-error" aria-label="Disable" {disabled} onclick={() => toggleType(t)}>
							<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4"><path d="M6.28 5.22a.75.75 0 0 0-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 1 0 1.06 1.06L10 11.06l3.72 3.72a.75.75 0 1 0 1.06-1.06L11.06 10l3.72-3.72a.75.75 0 0 0-1.06-1.06L10 8.94 6.28 5.22Z" /></svg>
						</button>
					</div>
				</div>
			{:else}
				<p class="text-center text-sm text-base-content/50 py-2">{m.quotation_none_enabled()}</p>
			{/each}
		</div>
	</div>

	<!-- Disabled zone -->
	{#if disabledTypes.length > 0}
		<div>
			<div class="mb-1 text-xs font-semibold uppercase tracking-wide text-base-content/40">{m.quotation_disabled_heading()}</div>
			<div
				class="min-h-10 rounded-box border-2 border-dashed p-2 space-y-1 transition-colors {dragOverZone === 'disabled' ? 'border-warning bg-warning/5' : 'border-base-300 bg-base-200/10'}"
				role="list"
				ondragover={(e) => { e.preventDefault(); dragOverZone = 'disabled'; }}
				ondragleave={() => { if (dragOverZone === 'disabled') dragOverZone = null; }}
				ondrop={() => handleDropDisabled()}
			>
				{#each disabledTypes as t (t)}
					{@const info = typeInfo(t)}
					<div
						class="flex items-center gap-2 rounded-box border border-base-300 bg-base-200/50 px-3 py-2 opacity-60 {disabled ? '' : 'cursor-grab'}"
						role="listitem"
						draggable={!disabled}
						ondragstart={() => handleDragStart(t)}
						ondragend={handleDragEnd}
					>
						<span class="text-lg" aria-hidden="true">{info.icon}</span>
						<div class="flex-1 min-w-0">
							<div class="text-sm font-medium">{info.label()}</div>
						</div>
						<button type="button" class="btn btn-ghost btn-xs btn-square text-success" aria-label="Enable" {disabled} onclick={() => toggleType(t)}>
							<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4"><path fill-rule="evenodd" d="M10 18a8 8 0 1 0 0-16 8 8 0 0 0 0 16Zm3.857-9.809a.75.75 0 0 0-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 1 0-1.06 1.061l2.5 2.5a.75.75 0 0 0 1.137-.089l4-5.5Z" clip-rule="evenodd" /></svg>
						</button>
					</div>
				{/each}
			</div>
		</div>
	{/if}
</div>
