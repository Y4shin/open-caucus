<script lang="ts">
	import { page } from '$app/state';
	import DocsOverlay from '$lib/components/docs/DocsOverlay.svelte';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { docsClient } from '$lib/api/index.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { onDestroy } from 'svelte';

	interface SearchHit {
		ref: string;
		path: string;
		heading: string;
		title: string;
		locale: string;
		snippet: string;
		score: number;
	}

	interface Crumb {
		title: string;
		path: string;
		current: boolean;
	}

	interface NavNode {
		title: string;
		path: string;
		current: boolean;
		expanded: boolean;
		children: NavNode[];
	}

	interface DocsPageData {
		path: string;
		locale: string;
		title: string;
		heading: string;
		html: string;
		pathDisplay: string;
		crumbs: Crumb[];
		tree: NavNode[];
	}

	let searchState = $state(createRemoteState<SearchHit[]>());
	let docsShellState = $state(createRemoteState<DocsPageData>());
	const query = $derived(page.url.searchParams.get('q') ?? '');

	onDestroy(() => {
		pageActions.clear();
	});

	$effect(() => {
		const currentSearch = page.url.search;
		const currentQuery = page.url.searchParams.get('q') ?? '';
		void currentSearch;
		loadDocsShell();
		loadSearch(currentQuery);
	});

	$effect(() => {
		if (!docsShellState.data) return;
		pageActions.set([], {
			title: docsShellState.data.title
		});
	});

	async function loadDocsShell() {
		docsShellState.loading = true;
		docsShellState.error = '';
		try {
			const response = await docsClient.getPage({ path: 'index' });
			docsShellState.data = (response.page ?? null) as DocsPageData | null;
		} catch (err) {
			docsShellState.error = getDisplayError(err, 'Failed to load the documentation shell.');
		} finally {
			docsShellState.loading = false;
		}
	}

	async function loadSearch(searchQuery: string) {
		searchState.loading = true;
		searchState.error = '';
		try {
			const response = await docsClient.search({ query: searchQuery, limit: 10 });
			searchState.data = (response.hits ?? []) as SearchHit[];
		} catch (err) {
			searchState.error = getDisplayError(err, 'Failed to search documentation.');
		} finally {
			searchState.loading = false;
		}
	}
</script>

{#if docsShellState.loading || searchState.loading}
	<AppSpinner label="Loading documentation" />
{:else if docsShellState.error}
	<AppAlert message={docsShellState.error} />
{:else if searchState.error}
	<AppAlert message={searchState.error} />
{:else if docsShellState.data}
	<DocsOverlay
		title={docsShellState.data.title}
		locale={docsShellState.data.locale}
		pathDisplay={docsShellState.data.pathDisplay}
		crumbs={docsShellState.data.crumbs}
		tree={docsShellState.data.tree}
		query={query}
		searchHits={searchState.data ?? []}
	/>
{/if}
