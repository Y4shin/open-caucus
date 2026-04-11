<script lang="ts">
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import AppTooltip from '$lib/components/ui/AppTooltip.svelte';
	import * as m from '$lib/paraglide/messages';

	let {
		speakerType,
		quoted,
		firstSpeaker,
		priority,
		mine = false
	}: {
		speakerType: string;
		quoted: boolean;
		firstSpeaker: boolean;
		priority: boolean;
		mine?: boolean;
	} = $props();
</script>

{#if speakerType === 'ropm'}
	<AppTooltip text="Point of order" side="right">
		<span class="badge badge-warning badge-sm" data-testid="live-speaker-ropm-badge"><LegacyIcon name="scale" class="h-3.5 w-3.5" /></span>
	</AppTooltip>
{/if}
{#if quoted}
	<AppTooltip text="FLINTA*" side="right">
		<span class="badge badge-info badge-sm" data-testid="live-speaker-quoted-badge"><LegacyIcon name="transgender" class="h-3.5 w-3.5" /></span>
	</AppTooltip>
{/if}
{#if firstSpeaker}
	<AppTooltip text="First Time" side="right">
		<span class="badge badge-success badge-sm" data-testid="live-speaker-first-badge"><LegacyIcon name="person-raised" class="h-3.5 w-3.5" /></span>
	</AppTooltip>
{/if}
{#if priority}
	<AppTooltip text="Priority" side="right">
		<span class="badge badge-warning badge-sm badge-outline" data-testid="live-speaker-priority-icon-badge"><LegacyIcon name="star" class="h-3.5 w-3.5" /></span>
	</AppTooltip>
	<span class="badge badge-warning badge-sm" data-testid="live-speaker-priority-label-badge">{m.meeting_live_badge_priority()}</span>
{/if}
{#if mine}
	<span class="badge badge-primary badge-sm">{m.speaker_badge_you()}</span>
{/if}
