<script lang="ts">
	import { prepare, layout } from '@chenglou/pretext';
	import * as m from '$lib/paraglide/messages';

	type AgendaImportState = 'ignore' | 'heading' | 'subheading';
	type AgendaImportLine = {
		lineNo: number;
		text: string;
		state: AgendaImportState;
	};

	let {
		rawText = $bindable(''),
		lines,
		onToggle,
		onPasteText
	}: {
		rawText: string;
		lines: AgendaImportLine[];
		onToggle: (index: number) => void;
		onPasteText: (text: string) => void;
	} = $props();

	// Layout constants (px)
	const GUTTER_W = 32; // line-number column
	const CHIP_W = 148; // chip column
	const PT = 8; // padding-top / padding-bottom
	const PH = 8; // inner horizontal padding beside gutter/chip

	let textareaEl = $state<HTMLTextAreaElement | null>(null);

	type LinePos = { y: number; h: number };
	let linePositions = $state<LinePos[]>([]);

	function reflow() {
		if (!textareaEl) return;
		const cs = window.getComputedStyle(textareaEl);
		const font = cs.font;
		const lhRaw = cs.lineHeight;
		const lineHeight = lhRaw === 'normal' ? parseFloat(cs.fontSize) * 1.2 : parseFloat(lhRaw);
		const contentWidth = textareaEl.clientWidth - GUTTER_W - CHIP_W - PH * 2;
		if (contentWidth <= 0) return;

		const logicalLines = rawText.split('\n');
		let yOffset = PT;
		const positions: LinePos[] = logicalLines.map((lineText) => {
			const prepared = prepare(lineText.length ? lineText : ' ', font);
			const { height } = layout(prepared, contentWidth, lineHeight);
			const h = Math.max(height, lineHeight);
			const y = yOffset;
			yOffset += h;
			return { y, h };
		});
		linePositions = positions;

		// Auto-resize to content
		textareaEl.style.height = 'auto';
		textareaEl.style.height = `${yOffset + PT}px`;
	}

	// Re-measure whenever text changes or the element mounts
	$effect(() => {
		void rawText;
		if (textareaEl) requestAnimationFrame(reflow);
	});

	// Also re-measure on container resize (e.g. modal opening)
	$effect(() => {
		if (!textareaEl) return;
		const ro = new ResizeObserver(reflow);
		ro.observe(textareaEl);
		return () => ro.disconnect();
	});

	function importPrefix(index: number): string {
		let top = 0;
		let sub = 0;
		for (let i = 0; i <= index; i++) {
			const line = lines[i];
			if (line.state === 'heading') {
				top += 1;
				sub = 0;
			} else if (line.state === 'subheading' && top > 0) {
				sub += 1;
			}
		}
		const line = lines[index];
		if (line.state === 'heading' && top > 0) return `TOP ${top}`;
		if (line.state === 'subheading' && top > 0 && sub > 0) return `TOP ${top}.${sub}`;
		return '';
	}

	function importStateLabel(state: AgendaImportState): string {
		switch (state) {
			case 'heading':
				return m.agenda_import_state_heading();
			case 'subheading':
				return m.agenda_import_state_subheading();
			default:
				return m.agenda_import_state_ignore();
		}
	}

	function handlePaste(event: ClipboardEvent) {
		const text = event.clipboardData?.getData('text') ?? '';
		if (text.trim()) {
			event.preventDefault();
			onPasteText(text);
		}
	}
</script>

<div
	class="relative overflow-hidden rounded-box border border-base-300 bg-base-100 focus-within:outline focus-within:outline-2 focus-within:outline-offset-0 focus-within:outline-primary/30"
	data-agenda-import-lines
>
	<!-- Line-number gutter (left overlay) — one entry per raw line, including empty ones -->
	<div
		class="pointer-events-none absolute left-0 top-0 select-none"
		style="width: {GUTTER_W}px;"
		aria-hidden="true"
	>
		{#each linePositions as pos, i}
			<span
				class="absolute right-1 font-mono text-xs tabular-nums text-base-content/30"
				style="top: {pos.y}px; line-height: {pos.h}px;"
			>{i + 1}</span>
		{/each}
	</div>

	<!-- Textarea — natural editing, arrow keys, selection all work here -->
	<textarea
		bind:this={textareaEl}
		bind:value={rawText}
		class="block w-full resize-none bg-transparent text-sm leading-6 outline-none"
		style="padding: {PT}px {CHIP_W + PH}px {PT}px {GUTTER_W + PH}px; min-height: {PT * 2 + 24}px;"
		placeholder={m.agenda_import_source_placeholder()}
		spellcheck="false"
		autocomplete="off"
		onpaste={handlePaste}
	></textarea>

	<!-- Chip column (right overlay) — one chip per raw line, full-height filled blocks -->
	<div
		class="pointer-events-none absolute right-0 top-0"
		style="width: {CHIP_W}px;"
	>
		{#each linePositions as pos, i}
			{@const rawLineNo = i + 1}
			{@const contentIdx = lines.findIndex((l) => l.lineNo === rawLineNo)}
			{@const line = contentIdx >= 0 ? lines[contentIdx] : null}
			{@const isFirst = i === 0}
			{@const isLast = i === linePositions.length - 1}
			{@const chipTop = isFirst ? 0 : pos.y}
			{@const chipHeight = isFirst && isLast ? pos.y + pos.h + PT : isFirst ? pos.y + pos.h : isLast ? pos.h + PT : pos.h}
			{#if line !== null}
				{@const prefix = importPrefix(contentIdx)}
				<button
					type="button"
					class={[
						'pointer-events-auto absolute inset-x-0 flex items-center gap-1 px-2 text-xs font-semibold transition-opacity hover:opacity-80',
						line.state === 'heading'
							? 'bg-primary text-primary-content'
							: line.state === 'subheading'
								? 'bg-info text-info-content'
								: 'bg-base-200 text-base-content/50'
					].join(' ')}
					style="top: {chipTop}px; height: {chipHeight}px;"
					onmousedown={(e) => e.preventDefault()}
					onclick={() => onToggle(contentIdx)}
				>
					{#if prefix}
						<span class="font-mono tabular-nums">{prefix}</span>
						<span class="opacity-50">·</span>
					{/if}
					<span>{importStateLabel(line.state)}</span>
				</button>
			{:else}
				<div
					class="pointer-events-none absolute inset-x-0 flex items-center px-2 text-xs font-semibold bg-base-200 text-base-content/20"
					style="top: {chipTop}px; height: {chipHeight}px;"
				>
					<span>{importStateLabel('ignore')}</span>
				</div>
			{/if}
		{/each}
	</div>

	<!-- Dividing lines between rows, drawn across the full width -->
	<div class="pointer-events-none absolute inset-0" aria-hidden="true">
		{#each linePositions as pos, i}
			{#if i > 0}
				<div
					class="absolute inset-x-0 border-t border-base-300/60"
					style="top: {pos.y}px;"
				></div>
			{/if}
		{/each}
	</div>
</div>
