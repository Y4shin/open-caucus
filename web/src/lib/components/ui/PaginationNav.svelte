<script lang="ts">
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import * as m from '$lib/paraglide/messages';
</script>

<nav class="pagination-nav join">
	<button type="button" disabled class="ui-icon-label btn btn-sm">
		<LegacyIcon name="left" class="ui-icon--left" />
		<span class="ui-icon-text">{m.pagination_previous()}</span>
	</button>
	<button class="btn btn-sm" type="button" disabled>1</button>
	<button type="button" disabled class="ui-icon-label btn btn-sm">
		<span class="ui-icon-text">{m.pagination_next()}</span>
		<LegacyIcon name="right" class="ui-icon--right" />
	</button>
</nav>
