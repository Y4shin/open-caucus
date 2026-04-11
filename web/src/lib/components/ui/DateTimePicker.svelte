<script lang="ts">
	import { DatePicker } from 'bits-ui';
	import {
		CalendarDateTime,
		getLocalTimeZone,
		today,
		type DateValue
	} from '@internationalized/date';

	let {
		value = $bindable<CalendarDateTime | undefined>(undefined),
		label = '',
		id = '',
		minValue = undefined as DateValue | undefined
	}: {
		value?: CalendarDateTime | undefined;
		label?: string;
		id?: string;
		minValue?: DateValue | undefined;
	} = $props();

	const todayDate = today(getLocalTimeZone());
	const placeholder = new CalendarDateTime(todayDate.year, todayDate.month, todayDate.day, 12, 0);

	let dialogEl = $state<HTMLDialogElement | null>(null);

	function openCalendarDialog() {
		dialogEl?.showModal();
	}

	function closeCalendarDialog() {
		dialogEl?.close();
	}

	function clearValue() {
		value = undefined;
		closeCalendarDialog();
	}
</script>

<div class="flex flex-col gap-1">
	{#if label}
		<label class="label text-sm font-medium" for={id}>{label}</label>
	{/if}

	<DatePicker.Root bind:value granularity="minute" {placeholder} {minValue} open={false}>
		<div class="flex items-center gap-1">
			<DatePicker.Input
				class="input input-bordered input-sm flex w-full items-center gap-0.5 px-2 font-mono text-sm"
				{id}
			>
				{#snippet children({ segments })}
					{#each segments as { part, value: segVal }}
						{#if part === 'literal'}
							<span class="text-base-content/50">{segVal}</span>
						{:else}
							<DatePicker.Segment
								{part}
								class="rounded px-0.5 focus:bg-primary focus:text-primary-content focus:outline-none"
							>
								{segVal}
							</DatePicker.Segment>
						{/if}
					{/each}
				{/snippet}
			</DatePicker.Input>
			<button type="button" class="btn btn-ghost btn-sm btn-square" aria-label="Open calendar" onclick={openCalendarDialog}>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					viewBox="0 0 20 20"
					fill="currentColor"
					class="h-4 w-4"
				>
					<path
						fill-rule="evenodd"
						d="M5.75 2a.75.75 0 0 1 .75.75V4h7V2.75a.75.75 0 0 1 1.5 0V4h.25A2.75 2.75 0 0 1 18 6.75v8.5A2.75 2.75 0 0 1 15.25 18H4.75A2.75 2.75 0 0 1 2 15.25v-8.5A2.75 2.75 0 0 1 4.75 4H5V2.75A.75.75 0 0 1 5.75 2Zm-1 5.5c-.69 0-1.25.56-1.25 1.25v6.5c0 .69.56 1.25 1.25 1.25h10.5c.69 0 1.25-.56 1.25-1.25v-6.5c0-.69-.56-1.25-1.25-1.25H4.75Z"
						clip-rule="evenodd"
					/>
				</svg>
			</button>
		</div>
	</DatePicker.Root>

	<!-- Hidden native input for a11y fallback -->
	<input type="datetime-local" class="sr-only" tabindex={-1} aria-hidden="true" />
</div>

<!-- Calendar modal overlay — renders on top of the parent wizard modal -->
<dialog class="modal" bind:this={dialogEl}>
	<div class="modal-box w-auto max-w-xs p-4">
		<DatePicker.Root bind:value granularity="minute" {placeholder} {minValue} open={true}>
			<DatePicker.Calendar>
				{#snippet children({ months, weekdays })}
					<DatePicker.Header class="flex items-center justify-between pb-2">
						<DatePicker.PrevButton class="btn btn-ghost btn-xs btn-circle">
							<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4">
								<path fill-rule="evenodd" d="M11.78 5.22a.75.75 0 0 1 0 1.06L8.06 10l3.72 3.72a.75.75 0 1 1-1.06 1.06l-4.25-4.25a.75.75 0 0 1 0-1.06l4.25-4.25a.75.75 0 0 1 1.06 0Z" clip-rule="evenodd" />
							</svg>
						</DatePicker.PrevButton>
						<DatePicker.Heading class="text-sm font-semibold" />
						<DatePicker.NextButton class="btn btn-ghost btn-xs btn-circle">
							<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4">
								<path fill-rule="evenodd" d="M8.22 5.22a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.75.75 0 1 1-1.06-1.06L11.94 10 8.22 6.28a.75.75 0 0 1 0-1.06Z" clip-rule="evenodd" />
							</svg>
						</DatePicker.NextButton>
					</DatePicker.Header>

					{#each months as month}
						<DatePicker.Grid class="w-full">
							<DatePicker.GridHead>
								<DatePicker.GridRow class="grid grid-cols-7">
									{#each weekdays as day}
										<DatePicker.HeadCell class="text-center text-xs font-medium text-base-content/50 py-1">
											{day}
										</DatePicker.HeadCell>
									{/each}
								</DatePicker.GridRow>
							</DatePicker.GridHead>
							<DatePicker.GridBody>
								{#each month.weeks as weekDates}
									<DatePicker.GridRow class="grid grid-cols-7">
										{#each weekDates as date}
											<DatePicker.Cell {date} month={month.value} class="flex items-center justify-center p-0.5">
												<DatePicker.Day
													class="btn btn-ghost btn-xs btn-circle text-xs
														data-selected:btn-primary
														data-today:border data-today:border-primary
														data-outside-month:text-base-content/30
														data-disabled:opacity-30
														data-unavailable:line-through data-unavailable:opacity-30"
												>
													{date.day}
												</DatePicker.Day>
											</DatePicker.Cell>
										{/each}
									</DatePicker.GridRow>
								{/each}
							</DatePicker.GridBody>
						</DatePicker.Grid>
					{/each}
				{/snippet}
			</DatePicker.Calendar>

			{#if value}
				<div class="mt-3 flex items-center justify-center gap-1 border-t border-base-300 pt-3">
					<input
						type="number"
						min="0"
						max="23"
						class="input input-bordered input-sm w-16 text-center font-mono"
						value={String(value.hour).padStart(2, '0')}
						onchange={(e) => {
							const h = Math.min(23, Math.max(0, parseInt(e.currentTarget.value) || 0));
							if (value) value = value.set({ hour: h });
						}}
					/>
					<span class="text-base-content/50 text-lg font-bold">:</span>
					<input
						type="number"
						min="0"
						max="59"
						step="5"
						class="input input-bordered input-sm w-16 text-center font-mono"
						value={String(value.minute).padStart(2, '0')}
						onchange={(e) => {
							const m = Math.min(59, Math.max(0, parseInt(e.currentTarget.value) || 0));
							if (value) value = value.set({ minute: m });
						}}
					/>
				</div>
			{/if}
		</DatePicker.Root>

		<div class="modal-action mt-3">
			<button type="button" class="btn btn-ghost btn-sm" onclick={clearValue}>Clear</button>
			<button type="button" class="btn btn-primary btn-sm" onclick={closeCalendarDialog}>Done</button>
		</div>
	</div>
	<form method="dialog" class="modal-backdrop"><button aria-label="Close">Close</button></form>
</dialog>
