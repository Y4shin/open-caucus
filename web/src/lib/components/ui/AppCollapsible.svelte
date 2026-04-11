<script lang="ts">
	import { Collapsible } from 'bits-ui';
	import type { Snippet } from 'svelte';

	let {
		open = $bindable(false),
		triggerClass = '',
		contentClass = '',
		class: rootClass = '',
		onOpenChange,
		trigger,
		children
	}: {
		open?: boolean;
		triggerClass?: string;
		contentClass?: string;
		class?: string;
		onOpenChange?: (open: boolean) => void;
		trigger: Snippet;
		children: Snippet;
	} = $props();
</script>

<Collapsible.Root bind:open {onOpenChange} class={rootClass}>
	<Collapsible.Trigger class={triggerClass}>
		{@render trigger()}
	</Collapsible.Trigger>
	<Collapsible.Content class={contentClass}>
		{@render children()}
	</Collapsible.Content>
</Collapsible.Root>
