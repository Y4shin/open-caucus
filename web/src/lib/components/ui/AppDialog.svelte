<script lang="ts">
	import { Dialog } from 'bits-ui';
	import type { Snippet } from 'svelte';

	let {
		open = $bindable(false),
		title = '',
		class: contentClass = 'w-11/12 max-w-lg',
		children
	}: {
		open?: boolean;
		title?: string;
		class?: string;
		children: Snippet;
	} = $props();

	export function show() {
		open = true;
	}

	export function close() {
		open = false;
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<Dialog.Overlay class="fixed inset-0 z-40 bg-black/50" />
		<Dialog.Content
			class="fixed left-1/2 top-1/2 z-50 -translate-x-1/2 -translate-y-1/2 rounded-box bg-base-100 p-6 shadow-xl {contentClass}"
		>
			{#if title}
				<Dialog.Title class="text-lg font-semibold">{title}</Dialog.Title>
			{/if}
			{@render children()}
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
