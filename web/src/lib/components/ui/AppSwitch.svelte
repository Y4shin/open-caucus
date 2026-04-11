<script lang="ts">
	import { Switch } from 'bits-ui';

	let {
		checked = $bindable(false),
		disabled = false,
		size = 'sm',
		color = 'primary',
		id = '',
		label = '',
		onCheckedChange
	}: {
		checked?: boolean;
		disabled?: boolean;
		size?: 'xs' | 'sm' | 'md';
		color?: 'primary' | 'info' | '';
		id?: string;
		label?: string;
		onCheckedChange?: (checked: boolean) => void;
	} = $props();

	const sizeClasses = {
		xs: { root: 'h-4 w-8', thumb: 'h-3 w-3', translate: 'translate-x-4' },
		sm: { root: 'h-5 w-10', thumb: 'h-4 w-4', translate: 'translate-x-5' },
		md: { root: 'h-6 w-12', thumb: 'h-5 w-5', translate: 'translate-x-6' }
	}[size];

	const colorClasses = {
		primary: 'bg-primary',
		info: 'bg-info',
		'': 'bg-base-content/30'
	}[color];
</script>

<label class="label cursor-pointer justify-start gap-2 p-0">
	<Switch.Root
		bind:checked
		{disabled}
		{id}
		{onCheckedChange}
		class="relative inline-flex shrink-0 cursor-pointer items-center rounded-full transition-colors {sizeClasses.root} {checked ? colorClasses : 'bg-base-content/20'} {disabled ? 'opacity-50 cursor-not-allowed' : ''}"
	>
		<Switch.Thumb
			class="pointer-events-none inline-block rounded-full bg-base-100 shadow transition-transform {sizeClasses.thumb} {checked ? sizeClasses.translate : 'translate-x-0.5'}"
		/>
	</Switch.Root>
	{#if label}
		<span class="text-xs leading-none">{label}</span>
	{/if}
</label>
