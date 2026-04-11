<script lang="ts">
	import { Select } from 'bits-ui';

	type Item = { value: string; label: string; disabled?: boolean };

	let {
		value = $bindable(''),
		items,
		placeholder = '',
		disabled = false,
		size = 'sm',
		id = '',
		onValueChange
	}: {
		value?: string;
		items: Item[];
		placeholder?: string;
		disabled?: boolean;
		size?: 'xs' | 'sm' | 'md';
		id?: string;
		onValueChange?: (value: string) => void;
	} = $props();

	const sizeMap: Record<string, string> = { xs: 'input-xs text-xs', sm: 'input-sm text-sm', md: '' };
	let sizeClass = $derived(sizeMap[size] ?? '');

	let selectedLabel = $derived(items.find((i) => i.value === value)?.label ?? placeholder);
</script>

<Select.Root type="single" bind:value {disabled} onValueChange={onValueChange}>
	<Select.Trigger
		class="input input-bordered {sizeClass} flex w-full items-center justify-between gap-1"
		{id}
	>
		<span class="truncate">{selectedLabel}</span>
		<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4 shrink-0 opacity-50">
			<path fill-rule="evenodd" d="M5.22 8.22a.75.75 0 0 1 1.06 0L10 11.94l3.72-3.72a.75.75 0 1 1 1.06 1.06l-4.25 4.25a.75.75 0 0 1-1.06 0l-4.25-4.25a.75.75 0 0 1 0-1.06Z" clip-rule="evenodd" />
		</svg>
	</Select.Trigger>
	<Select.Portal>
		<Select.Content
			class="rounded-box border border-base-300 bg-base-100 py-1 shadow-lg z-50 max-h-60 overflow-auto"
			sideOffset={4}
			side="bottom"
		>
			<Select.Viewport>
				{#each items as item}
					<Select.Item
						value={item.value}
						label={item.label}
						disabled={item.disabled}
						class="cursor-pointer px-3 py-1.5 text-sm hover:bg-base-200 data-highlighted:bg-base-200 data-selected:font-semibold data-disabled:opacity-40"
					>
						{item.label}
					</Select.Item>
				{/each}
			</Select.Viewport>
		</Select.Content>
	</Select.Portal>
</Select.Root>
