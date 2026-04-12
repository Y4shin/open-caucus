<script lang="ts">
	import { DateRangePicker } from 'bits-ui';
	import {
		CalendarDateTime,
		getLocalTimeZone,
		today,
		type DateValue
	} from '@internationalized/date';

	let {
		startValue = $bindable<CalendarDateTime | undefined>(undefined),
		endValue = $bindable<CalendarDateTime | undefined>(undefined),
		startLabel = 'Start',
		endLabel = 'End',
		id = ''
	}: {
		startValue?: CalendarDateTime | undefined;
		endValue?: CalendarDateTime | undefined;
		startLabel?: string;
		endLabel?: string;
		id?: string;
	} = $props();

	const todayDate = today(getLocalTimeZone());
	const placeholder = new CalendarDateTime(todayDate.year, todayDate.month, todayDate.day, 12, 0);

	// Sync between the DateRangePicker value (DateRange) and our individual start/end props.
	let rangeValue = $state<{ start: CalendarDateTime; end: CalendarDateTime } | undefined>(
		startValue && endValue ? { start: startValue, end: endValue } : undefined
	);

	let dialogEl = $state<HTMLDialogElement | null>(null);

	function openCalendarDialog() {
		dialogEl?.showModal();
	}

	function closeCalendarDialog() {
		dialogEl?.close();
	}

	function clearValue() {
		rangeValue = undefined;
		startValue = undefined;
		endValue = undefined;
		closeCalendarDialog();
	}

	function applyValue() {
		if (rangeValue) {
			startValue = rangeValue.start;
			endValue = rangeValue.end;
		}
		closeCalendarDialog();
	}
</script>

<div class="flex flex-col gap-1">
	<DateRangePicker.Root bind:value={rangeValue} granularity="minute" {placeholder} onValueChange={(v) => { if (v?.start && v?.end) { startValue = v.start as CalendarDateTime; endValue = v.end as CalendarDateTime; } }}>
	<div class="flex items-end gap-2">
		<div class="flex-1">
			<label class="label text-sm font-medium" for={id ? `${id}-start` : undefined}>{startLabel}</label>
				<DateRangePicker.Input type="start"
					class="input input-bordered input-sm flex w-full items-center gap-0.5 px-2 font-mono text-sm"
					id={id ? `${id}-start` : undefined}
				>
					{#snippet children({ segments })}
						{#each segments as { part, value: segVal }}
							{#if part === 'literal'}
								<span class="text-base-content/50">{segVal}</span>
							{:else}
								<DateRangePicker.Segment
									{part}
									class="rounded px-0.5 focus:bg-primary focus:text-primary-content focus:outline-none"
								>
									{segVal}
								</DateRangePicker.Segment>
							{/if}
						{/each}
					{/snippet}
				</DateRangePicker.Input>
		</div>
		<span class="pb-2 text-base-content/50">—</span>
		<div class="flex-1">
			<label class="label text-sm font-medium" for={id ? `${id}-end` : undefined}>{endLabel}</label>
				<DateRangePicker.Input type="end"
					class="input input-bordered input-sm flex w-full items-center gap-0.5 px-2 font-mono text-sm"
					id={id ? `${id}-end` : undefined}
				>
					{#snippet children({ segments })}
						{#each segments as { part, value: segVal }}
							{#if part === 'literal'}
								<span class="text-base-content/50">{segVal}</span>
							{:else}
								<DateRangePicker.Segment
									{part}
									class="rounded px-0.5 focus:bg-primary focus:text-primary-content focus:outline-none"
								>
									{segVal}
								</DateRangePicker.Segment>
							{/if}
						{/each}
					{/snippet}
				</DateRangePicker.Input>
		</div>
		<button type="button" class="btn btn-ghost btn-sm btn-square mb-0.5" aria-label="Open calendar" onclick={openCalendarDialog}>
			<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4">
				<path fill-rule="evenodd" d="M5.75 2a.75.75 0 0 1 .75.75V4h7V2.75a.75.75 0 0 1 1.5 0V4h.25A2.75 2.75 0 0 1 18 6.75v8.5A2.75 2.75 0 0 1 15.25 18H4.75A2.75 2.75 0 0 1 2 15.25v-8.5A2.75 2.75 0 0 1 4.75 4H5V2.75A.75.75 0 0 1 5.75 2Zm-1 5.5c-.69 0-1.25.56-1.25 1.25v6.5c0 .69.56 1.25 1.25 1.25h10.5c.69 0 1.25-.56 1.25-1.25v-6.5c0-.69-.56-1.25-1.25-1.25H4.75Z" clip-rule="evenodd" />
			</svg>
		</button>
	</div>
	</DateRangePicker.Root>

	<!-- Hidden native inputs for a11y fallback -->
	<input type="datetime-local" class="sr-only" tabindex={-1} aria-hidden="true" />
</div>

<!-- Calendar modal -->
<dialog class="modal" bind:this={dialogEl}>
	<div class="modal-box w-auto max-w-md p-4">
		<DateRangePicker.Root bind:value={rangeValue} granularity="minute" {placeholder}>
			<DateRangePicker.Calendar>
				{#snippet children({ months, weekdays })}
					<DateRangePicker.Header class="flex items-center justify-between pb-2">
						<DateRangePicker.PrevButton class="btn btn-ghost btn-xs btn-circle">
							<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4">
								<path fill-rule="evenodd" d="M11.78 5.22a.75.75 0 0 1 0 1.06L8.06 10l3.72 3.72a.75.75 0 1 1-1.06 1.06l-4.25-4.25a.75.75 0 0 1 0-1.06l4.25-4.25a.75.75 0 0 1 1.06 0Z" clip-rule="evenodd" />
							</svg>
						</DateRangePicker.PrevButton>
						<DateRangePicker.Heading class="text-sm font-semibold" />
						<DateRangePicker.NextButton class="btn btn-ghost btn-xs btn-circle">
							<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="h-4 w-4">
								<path fill-rule="evenodd" d="M8.22 5.22a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.75.75 0 1 1-1.06-1.06L11.94 10 8.22 6.28a.75.75 0 0 1 0-1.06Z" clip-rule="evenodd" />
							</svg>
						</DateRangePicker.NextButton>
					</DateRangePicker.Header>

					{#each months as month}
						<DateRangePicker.Grid class="w-full">
							<DateRangePicker.GridHead>
								<DateRangePicker.GridRow class="grid grid-cols-7">
									{#each weekdays as day}
										<DateRangePicker.HeadCell class="text-center text-xs font-medium text-base-content/50 py-1">
											{day}
										</DateRangePicker.HeadCell>
									{/each}
								</DateRangePicker.GridRow>
							</DateRangePicker.GridHead>
							<DateRangePicker.GridBody>
								{#each month.weeks as weekDates}
									<DateRangePicker.GridRow class="grid grid-cols-7">
										{#each weekDates as date}
											<DateRangePicker.Cell {date} month={month.value} class="flex items-center justify-center p-0.5">
												<DateRangePicker.Day
													class="btn btn-ghost btn-xs btn-circle text-xs
														data-selected:btn-primary
														data-today:border data-today:border-primary
														data-outside-month:text-base-content/30
														data-disabled:opacity-30
														data-unavailable:line-through data-unavailable:opacity-30
														data-selection-start:rounded-l-full
														data-selection-end:rounded-r-full"
												>
													{date.day}
												</DateRangePicker.Day>
											</DateRangePicker.Cell>
										{/each}
									</DateRangePicker.GridRow>
								{/each}
							</DateRangePicker.GridBody>
						</DateRangePicker.Grid>
					{/each}
				{/snippet}
			</DateRangePicker.Calendar>

			{#if rangeValue?.start && rangeValue?.end}
				<div class="mt-3 border-t border-base-300 pt-3 space-y-2">
					<div class="flex items-center gap-2">
						<span class="text-xs font-medium text-base-content/60 w-12">{startLabel}:</span>
						<div class="flex items-center gap-1">
							<input type="number" min="0" max="23" class="input input-bordered input-xs w-14 text-center font-mono"
								value={String(rangeValue.start.hour).padStart(2, '0')}
								onchange={(e) => { if (rangeValue?.start) rangeValue = { ...rangeValue, start: rangeValue.start.set({ hour: Math.min(23, Math.max(0, parseInt(e.currentTarget.value) || 0)) }) }; }} />
							<span class="text-base-content/50 font-bold">:</span>
							<input type="number" min="0" max="59" step="5" class="input input-bordered input-xs w-14 text-center font-mono"
								value={String(rangeValue.start.minute).padStart(2, '0')}
								onchange={(e) => { if (rangeValue?.start) rangeValue = { ...rangeValue, start: rangeValue.start.set({ minute: Math.min(59, Math.max(0, parseInt(e.currentTarget.value) || 0)) }) }; }} />
						</div>
					</div>
					<div class="flex items-center gap-2">
						<span class="text-xs font-medium text-base-content/60 w-12">{endLabel}:</span>
						<div class="flex items-center gap-1">
							<input type="number" min="0" max="23" class="input input-bordered input-xs w-14 text-center font-mono"
								value={String(rangeValue.end.hour).padStart(2, '0')}
								onchange={(e) => { if (rangeValue?.end) rangeValue = { ...rangeValue, end: rangeValue.end.set({ hour: Math.min(23, Math.max(0, parseInt(e.currentTarget.value) || 0)) }) }; }} />
							<span class="text-base-content/50 font-bold">:</span>
							<input type="number" min="0" max="59" step="5" class="input input-bordered input-xs w-14 text-center font-mono"
								value={String(rangeValue.end.minute).padStart(2, '0')}
								onchange={(e) => { if (rangeValue?.end) rangeValue = { ...rangeValue, end: rangeValue.end.set({ minute: Math.min(59, Math.max(0, parseInt(e.currentTarget.value) || 0)) }) }; }} />
						</div>
					</div>
				</div>
			{/if}
		</DateRangePicker.Root>

		<div class="modal-action mt-3">
			<button type="button" class="btn btn-ghost btn-sm" onclick={clearValue}>Clear</button>
			<button type="button" class="btn btn-primary btn-sm" onclick={applyValue}>Done</button>
		</div>
	</div>
	<form method="dialog" class="modal-backdrop"><button aria-label="Close">Close</button></form>
</dialog>
