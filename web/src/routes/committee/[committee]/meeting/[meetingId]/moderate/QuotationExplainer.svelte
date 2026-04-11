<script lang="ts">
	import { Dialog } from 'bits-ui';
	import { QuotationType } from '$lib/gen/conference/common/v1/common_pb.js';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import * as m from '$lib/paraglide/messages';

	let { quotationOrder }: { quotationOrder: QuotationType[] } = $props();

	let open = $state(false);
	let currentStep = $state(0);

	// Transition state machine: idle → shrinking → moving → expanding → idle
	let phase = $state<'idle' | 'shrinking' | 'moving' | 'expanding'>('idle');
	let movedIds = $state(new Set<string>());
	let prevRoot = $state<Bucket | null>(null);
	let moveOrder = $state<string[]>([]);
	let moveTimer = $state<ReturnType<typeof setTimeout> | null>(null);

	interface Speaker {
		id: string;
		name: string;
		quoted: boolean;
		firstSpeaker: boolean;
	}

	const speakers: Speaker[] = [
		{ id: 'a', name: 'Alice', quoted: true, firstSpeaker: true },
		{ id: 'b', name: 'Bob', quoted: false, firstSpeaker: false },
		{ id: 'c', name: 'Clara', quoted: true, firstSpeaker: false },
		{ id: 'd', name: 'David', quoted: false, firstSpeaker: true },
		{ id: 'e', name: 'Emma', quoted: true, firstSpeaker: false },
		{ id: 'f', name: 'Frank', quoted: false, firstSpeaker: false },
		{ id: 'g', name: 'Grace', quoted: false, firstSpeaker: true },
		{ id: 'h', name: 'Henry', quoted: true, firstSpeaker: true }
	];

	interface Bucket {
		label: string;
		borderColor: string;
		speakers: Speaker[];
		children: Bucket[];
	}

	interface Step {
		title: string;
		description: string;
		root: Bucket;
	}

	function ruleLabel(qt: QuotationType): string {
		switch (qt) {
			case QuotationType.GENDER: return m.meeting_moderate_flinta_quotation_label();
			case QuotationType.FIRST_SPEAKER: return m.meeting_moderate_first_speaker_bonus_label();
			default: return '';
		}
	}

	function leaf(spkrs: Speaker[], label = '', borderColor = ''): Bucket {
		return { label, borderColor, speakers: spkrs, children: [] };
	}

	function branch(children: Bucket[], label = '', borderColor = ''): Bucket {
		return { label, borderColor, speakers: [], children };
	}

	function allSpeakers(b: Bucket): Speaker[] {
		if (b.children.length > 0) return b.children.flatMap(allSpeakers);
		return b.speakers;
	}

	function collectOrder(b: Bucket): string[] {
		if (b.children.length > 0) return b.children.flatMap(collectOrder);
		return b.speakers.map(s => s.id);
	}

	function divideLeaf(list: Speaker[], rule: QuotationType): [Bucket, Bucket] {
		if (rule === QuotationType.GENDER) {
			return [
				leaf(list.filter(s => s.quoted), m.quotation_bucket_flinta(), 'border-info'),
				leaf(list.filter(s => !s.quoted), m.quotation_bucket_non_flinta(), 'border-neutral')
			];
		}
		return [
			leaf(list.filter(s => s.firstSpeaker), m.quotation_bucket_first_time(), 'border-success'),
			leaf(list.filter(s => !s.firstSpeaker), m.quotation_bucket_returning(), 'border-warning')
		];
	}

	function divideBucket(b: Bucket, rule: QuotationType): Bucket {
		if (b.children.length > 0) {
			return branch(b.children.map(c => divideBucket(c, rule)), b.label, b.borderColor);
		}
		const [a, bb] = divideLeaf(b.speakers, rule);
		return branch([a, bb].filter(x => x.speakers.length > 0), b.label, b.borderColor);
	}

	function recombineBucket(b: Bucket, rule: QuotationType): Bucket {
		if (b.children.length === 0) return b;
		const hasGrandchildren = b.children.some(c => c.children.length > 0);
		if (hasGrandchildren) {
			return branch(b.children.map(c => recombineBucket(c, rule)), b.label, b.borderColor);
		}
		if (b.children.length < 2) {
			const child = b.children[0];
			return leaf(child.speakers, b.label, b.borderColor);
		}
		const a = b.children[0];
		const bb = b.children[1];
		let combined: Speaker[];
		if (rule === QuotationType.GENDER) {
			combined = [];
			const maxLen = Math.max(a.speakers.length, bb.speakers.length);
			for (let i = 0; i < maxLen; i++) {
				if (i < a.speakers.length) combined.push(a.speakers[i]);
				if (i < bb.speakers.length) combined.push(bb.speakers[i]);
			}
		} else {
			combined = [...a.speakers, ...bb.speakers];
		}
		return leaf(combined, b.label, b.borderColor);
	}

	function buildSteps(order: QuotationType[]): Step[] {
		const steps: Step[] = [];
		let current = leaf([...speakers]);
		steps.push({
			title: m.quotation_step_original(),
			description: m.quotation_step_original_desc(),
			root: current
		});
		if (order.length === 0) {
			steps.push({
				title: m.quotation_step_result(),
				description: m.quotation_step_result_desc(),
				root: current
			});
			return steps;
		}
		for (const rule of order) {
			current = divideBucket(current, rule);
			steps.push({
				title: m.quotation_step_divide({ rule: ruleLabel(rule) }),
				description: rule === QuotationType.GENDER
					? m.quotation_step_divide_desc_gender()
					: m.quotation_step_divide_desc_first_speaker(),
				root: current
			});
		}
		for (let i = order.length - 1; i >= 0; i--) {
			current = recombineBucket(current, order[i]);
			steps.push({
				title: m.quotation_step_recombine({ rule: ruleLabel(order[i]) }),
				description: order[i] === QuotationType.GENDER
					? m.quotation_step_recombine_desc_gender()
					: m.quotation_step_recombine_desc_first_speaker(),
				root: current
			});
		}
		steps.push({
			title: m.quotation_step_result(),
			description: m.quotation_step_result_desc(),
			root: leaf(allSpeakers(current))
		});
		return steps;
	}

	let steps = $derived(buildSteps(quotationOrder));
	let step = $derived(steps[currentStep] ?? steps[0]);
	let totalSteps = $derived(steps.length);
	let isLastStep = $derived(currentStep === totalSteps - 1);
	let transitioning = $derived(phase !== 'idle');

	function cancelTransition() {
		if (moveTimer) clearTimeout(moveTimer);
		moveTimer = null;
		phase = 'idle';
		prevRoot = null;
		movedIds = new Set();
		moveOrder = [];
	}

	function openModal() {
		cancelTransition();
		currentStep = 0;
		open = true;
	}

	function startMoveSequence() {
		const order = [...moveOrder];
		let i = 0;
		function moveNext() {
			if (i >= order.length) {
				phase = 'expanding';
				moveTimer = setTimeout(() => {
					phase = 'idle';
					prevRoot = null;
					movedIds = new Set();
					moveOrder = [];
					moveTimer = null;
				}, 400);
				return;
			}
			movedIds = new Set([...movedIds, order[i]]);
			i++;
			moveTimer = setTimeout(moveNext, 350);
		}
		moveNext();
	}

	function next() {
		if (currentStep >= totalSteps - 1 || transitioning) return;
		cancelTransition();

		const nextIndex = currentStep + 1;
		const isNextLast = nextIndex === totalSteps - 1;

		if (isNextLast) {
			// Final step: no animation, just advance
			currentStep = nextIndex;
			return;
		}

		prevRoot = step.root;
		currentStep = nextIndex;
		moveOrder = collectOrder(steps[currentStep].root);
		movedIds = new Set();

		phase = 'shrinking';
		moveTimer = setTimeout(() => {
			phase = 'moving';
			moveTimer = setTimeout(startMoveSequence, 150);
		}, 400);
	}

	function prev() {
		if (currentStep <= 0 || transitioning) return;
		cancelTransition();
		currentStep--;
	}
</script>

{#snippet speakerChip(speaker: Speaker, visible: boolean)}
	<div
		class="flex items-center gap-1.5 rounded-lg border border-base-300 px-2.5 py-1.5 text-sm transition-all duration-300
			{visible ? 'bg-base-100 opacity-100 translate-x-0' : 'bg-base-200/30 opacity-0 -translate-x-2'}"
	>
		<span class="font-medium">{speaker.name}</span>
		{#if speaker.quoted}
			<span class="badge badge-info badge-sm"><LegacyIcon name="transgender" class="h-3.5 w-3.5" /></span>
		{/if}
		{#if speaker.firstSpeaker}
			<span class="badge badge-success badge-sm"><LegacyIcon name="person-raised" class="h-3.5 w-3.5" /></span>
		{/if}
	</div>
{/snippet}

{#snippet renderBucket(bucket: Bucket, depth: number, showItems: boolean, itemFilter?: Set<string>)}
	{#if bucket.children.length > 0}
		<div class="rounded-box border-2 {bucket.borderColor || 'border-base-300'} p-2 space-y-1.5 {depth > 0 ? 'bg-base-200/15' : ''}">
			{#if bucket.label}
				<div class="text-xs font-semibold text-base-content/60">{bucket.label}</div>
			{/if}
			{#each bucket.children as child}
				{@render renderBucket(child, depth + 1, showItems, itemFilter)}
			{/each}
		</div>
	{:else if bucket.speakers.length > 0}
		{#if bucket.label}
			<div class="rounded-box border-2 {bucket.borderColor || 'border-base-300'} p-2 {depth > 0 ? 'bg-base-200/15' : ''}">
				<div class="text-xs font-semibold text-base-content/60 mb-1">{bucket.label}</div>
				<div class="flex flex-col gap-1">
					{#each bucket.speakers as speaker (speaker.id)}
						{@const visible = showItems && (!itemFilter || itemFilter.has(speaker.id))}
						{@render speakerChip(speaker, visible)}
					{/each}
				</div>
			</div>
		{:else}
			<div class="flex flex-col gap-1">
				{#each bucket.speakers as speaker (speaker.id)}
					{@const visible = showItems && (!itemFilter || itemFilter.has(speaker.id))}
					{@render speakerChip(speaker, visible)}
				{/each}
			</div>
		{/if}
	{/if}
{/snippet}

{#snippet leftSideBucket(bucket: Bucket, depth: number)}
	{#if bucket.children.length > 0}
		<div class="rounded-box border-2 {bucket.borderColor || 'border-base-300'} p-2 space-y-1.5 {depth > 0 ? 'bg-base-200/15' : ''}">
			{#if bucket.label}
				<div class="text-xs font-semibold text-base-content/60">{bucket.label}</div>
			{/if}
			{#each bucket.children as child}
				{@render leftSideBucket(child, depth + 1)}
			{/each}
		</div>
	{:else if bucket.speakers.length > 0}
		{#if bucket.label}
			<div class="rounded-box border-2 {bucket.borderColor || 'border-base-300'} p-2 {depth > 0 ? 'bg-base-200/15' : ''}">
				<div class="text-xs font-semibold text-base-content/60 mb-1">{bucket.label}</div>
				<div class="flex flex-col gap-1">
					{#each bucket.speakers as speaker (speaker.id)}
						{@render speakerChip(speaker, !movedIds.has(speaker.id))}
					{/each}
				</div>
			</div>
		{:else}
			<div class="flex flex-col gap-1">
				{#each bucket.speakers as speaker (speaker.id)}
					{@render speakerChip(speaker, !movedIds.has(speaker.id))}
				{/each}
			</div>
		{/if}
	{/if}
{/snippet}

<button type="button" class="btn btn-primary btn-sm gap-1.5" onclick={openModal}>
	<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4">
		<path fill-rule="evenodd" d="M18 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0ZM8.94 6.94a.75.75 0 1 1-1.061-1.061 3 3 0 1 1 2.871 5.026v.345a.75.75 0 0 1-1.5 0v-.5c0-.72.57-1.172 1.081-1.287A1.5 1.5 0 1 0 8.94 6.94ZM10 15a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z" clip-rule="evenodd" />
	</svg>
	{m.quotation_explain_button()}
</button>

<Dialog.Root bind:open onOpenChange={(o) => { if (!o) cancelTransition(); }}>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/50" />
		<Dialog.Content class="fixed left-1/2 top-1/2 z-50 -translate-x-1/2 -translate-y-1/2 w-11/12 max-w-2xl max-h-[90vh] overflow-y-auto rounded-box bg-base-100 p-6 shadow-xl">
			<Dialog.Title class="text-lg font-semibold mb-4">{m.quotation_explain_title()}</Dialog.Title>

			<!-- Step indicator -->
			<ul class="steps steps-horizontal w-full mb-4 text-xs">
				{#each steps as _, i}
					<li class={i <= currentStep ? 'step step-primary' : 'step'}>
						{#if i === 0}
							{m.quotation_step_original()}
						{:else if i === totalSteps - 1}
							{m.quotation_step_result()}
						{:else}
							{i}
						{/if}
					</li>
				{/each}
			</ul>

			<!-- Step content -->
			<div class="min-h-72">
				<h4 class="font-semibold text-sm">{step.title}</h4>
				<p class="text-xs text-base-content/60 mb-3">{step.description}</p>

				{#if phase === 'idle'}
					{@render renderBucket(step.root, 0, true)}
				{:else}
					<div class="flex gap-3 transition-all duration-400">
						<div class="transition-all duration-400 overflow-hidden {phase === 'expanding' ? 'w-0 opacity-0' : 'w-1/2 opacity-100'}">
							{#if prevRoot}
								{@render leftSideBucket(prevRoot, 0)}
							{/if}
						</div>
						<div class="transition-all duration-400 {phase === 'expanding' ? 'w-full' : 'w-1/2'}">
							{@render renderBucket(step.root, 0, true, movedIds)}
						</div>
					</div>
				{/if}
			</div>

			<!-- Navigation -->
			<div class="mt-4 flex items-center justify-between">
				<button type="button" class="btn btn-ghost btn-sm" disabled={currentStep === 0 || transitioning} onclick={prev}>
					{m.common_back()}
				</button>
				<span class="text-xs text-base-content/50">{currentStep + 1} / {totalSteps}</span>
				{#if !isLastStep}
					<button type="button" class="btn btn-primary btn-sm" disabled={transitioning} onclick={next}>
						{m.wizard_next()}
					</button>
				{:else}
					<button type="button" class="btn btn-primary btn-sm" onclick={() => (open = false)}>
						{m.common_close()}
					</button>
				{/if}
			</div>
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
