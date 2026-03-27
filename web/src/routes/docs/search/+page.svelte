<script lang="ts">
	import { page } from '$app/state';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';

	interface SearchHit {
		ref: string;
		path: string;
		heading: string;
		title: string;
		locale: string;
		snippet: string;
		score: number;
	}

	let searchState = $state(createRemoteState<SearchHit[]>());
	const query = $derived(page.url.searchParams.get('q') ?? '');

	$effect(() => {
		loadSearch();
	});

	async function loadSearch() {
		searchState.loading = true;
		searchState.error = '';
		try {
			const response = await fetch(`/api/docs/search?q=${encodeURIComponent(query)}`);
			if (!response.ok) {
				throw new Error(`search failed (${response.status})`);
			}
			const payload = (await response.json()) as { hits?: SearchHit[] };
			searchState.data = payload.hits ?? [];
		} catch (err) {
			searchState.error = getDisplayError(err, 'Failed to search documentation.');
		} finally {
			searchState.loading = false;
		}
	}
</script>

<div class="space-y-6">
	<div class="space-y-2">
		<h1 class="text-3xl font-bold">Documentation Search</h1>
		<p class="text-base-content/70">Results for “{query || '...'}”.</p>
	</div>

	{#if searchState.loading}
		<AppSpinner label="Searching documentation" />
	{:else if searchState.error}
		<AppAlert message={searchState.error} />
	{:else if searchState.data?.length}
		<div class="space-y-3" id="docs-search-results">
			{#each searchState.data as hit}
				<AppCard title={hit.title}>
					<p class="text-sm text-base-content/70">{hit.path}</p>
					{#if hit.snippet}
						<p class="mt-2 text-sm">{hit.snippet}</p>
					{/if}
					<div class="card-actions justify-end">
						<a class="btn btn-primary btn-sm" href={`/docs/${hit.path}`}>Open</a>
					</div>
				</AppCard>
			{/each}
		</div>
	{:else}
		<AppAlert tone="info" message="No documentation results matched that query." />
	{/if}
</div>
