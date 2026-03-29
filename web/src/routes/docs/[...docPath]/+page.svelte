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

	const docPath = $derived(page.params.docPath || 'index');
	let docsState = $state(createRemoteState<DocsPageData>());

	onDestroy(() => {
		pageActions.clear();
	});

	$effect(() => {
		loadDoc();
	});

	$effect(() => {
		if (!docsState.data) return;
		pageActions.set([], {
			title: docsState.data.title
		});
	});

	async function loadDoc() {
		docsState.loading = true;
		docsState.error = '';
		try {
			const response = await docsClient.getPage({ path: docPath });
			docsState.data = (response.page ?? null) as DocsPageData | null;
		} catch (err) {
			docsState.error = getDisplayError(err, 'Failed to load the documentation page.');
		} finally {
			docsState.loading = false;
		}
	}
</script>

{#if docsState.loading}
	<AppSpinner label="Loading documentation" />
{:else if docsState.error}
	<AppAlert message={docsState.error} />
{:else if docsState.data}
	<DocsOverlay
		title={docsState.data.title}
		locale={docsState.data.locale}
		pathDisplay={docsState.data.pathDisplay}
		crumbs={docsState.data.crumbs}
		tree={docsState.data.tree}
		html={docsState.data.html}
	/>
{/if}
