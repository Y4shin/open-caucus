<script lang="ts">
	import { QuotationType } from '$lib/gen/conference/common/v1/common_pb.js';
	import * as m from '$lib/paraglide/messages';

	let { quotationOrder }: { quotationOrder: QuotationType[] } = $props();

	interface MockSpeaker {
		name: string;
		quoted: boolean;
		firstSpeaker: boolean;
	}

	const mockSpeakers: MockSpeaker[] = [
		{ name: 'Alice', quoted: true, firstSpeaker: true },
		{ name: 'Bob', quoted: false, firstSpeaker: false },
		{ name: 'Clara', quoted: true, firstSpeaker: false },
		{ name: 'David', quoted: false, firstSpeaker: true }
	];

	function sortSpeakers(speakers: MockSpeaker[], order: QuotationType[]): MockSpeaker[] {
		const list = [...speakers];

		if (order.length === 0) {
			return list; // no quotation, original order (request time)
		}

		const primary = order[0];
		const secondary = order.length > 1 ? order[1] : null;

		function withinGroupSort(a: MockSpeaker, b: MockSpeaker): number {
			if (secondary === QuotationType.GENDER) {
				if (a.quoted !== b.quoted) return a.quoted ? -1 : 1;
			}
			if (secondary === QuotationType.FIRST_SPEAKER) {
				if (a.firstSpeaker !== b.firstSpeaker) return a.firstSpeaker ? -1 : 1;
			}
			return 0;
		}

		if (primary === QuotationType.GENDER) {
			const quoted = list.filter((s) => s.quoted).sort(withinGroupSort);
			const nonQuoted = list.filter((s) => !s.quoted).sort(withinGroupSort);
			const result: MockSpeaker[] = [];
			const maxLen = Math.max(quoted.length, nonQuoted.length);
			for (let i = 0; i < maxLen; i++) {
				if (i < quoted.length) result.push(quoted[i]);
				if (i < nonQuoted.length) result.push(nonQuoted[i]);
			}
			return result;
		}

		if (primary === QuotationType.FIRST_SPEAKER) {
			const first = list.filter((s) => s.firstSpeaker).sort(withinGroupSort);
			const returning = list.filter((s) => !s.firstSpeaker).sort(withinGroupSort);
			const result: MockSpeaker[] = [];
			const maxLen = Math.max(first.length, returning.length);
			for (let i = 0; i < maxLen; i++) {
				if (i < first.length) result.push(first[i]);
				if (i < returning.length) result.push(returning[i]);
			}
			return result;
		}

		return list;
	}

	let sorted = $derived(sortSpeakers(mockSpeakers, quotationOrder));
</script>

<div class="rounded-box border border-base-300 bg-base-200/20 p-3">
	<h4 class="mb-1 text-xs font-semibold uppercase tracking-wide text-base-content/60">{m.quotation_preview_heading()}</h4>
	<p class="mb-2 text-xs text-base-content/50">{m.quotation_preview_description()}</p>
	<ol class="space-y-1">
		{#each sorted as speaker, i}
			<li class="flex items-center gap-2 rounded px-2 py-1 text-sm {i % 2 === 0 ? 'bg-base-100' : ''}">
				<span class="w-5 text-right font-mono text-xs text-base-content/40">{i + 1}</span>
				<span class="font-medium">{speaker.name}</span>
				{#if speaker.quoted}
					<span class="badge badge-info badge-xs">FLINTA*</span>
				{/if}
				{#if speaker.firstSpeaker}
					<span class="badge badge-success badge-xs">1st</span>
				{/if}
			</li>
		{/each}
	</ol>
</div>
