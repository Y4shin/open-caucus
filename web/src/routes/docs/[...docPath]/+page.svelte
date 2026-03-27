<script lang="ts">
	import { page } from '$app/state';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
	import AppSpinner from '$lib/components/ui/AppSpinner.svelte';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';

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
		path_display: string;
		crumbs: Crumb[];
		tree: NavNode[];
	}

	const docPath = $derived(page.params.docPath || 'index');
	let docsState = $state(createRemoteState<DocsPageData>());

	$effect(() => {
		loadDoc();
	});

	async function loadDoc() {
		docsState.loading = true;
		docsState.error = '';
		try {
			const response = await fetch(`/api/docs/page/${docPath}`);
			if (!response.ok) {
				throw new Error(`docs page failed (${response.status})`);
			}
			docsState.data = (await response.json()) as DocsPageData;
		} catch (err) {
			docsState.error = getDisplayError(err, 'Failed to load the documentation page.');
		} finally {
			docsState.loading = false;
		}
	}
</script>

<div class="space-y-6">
	{#if docsState.loading}
		<AppSpinner label="Loading documentation" />
	{:else if docsState.error}
		<AppAlert message={docsState.error} />
	{:else if docsState.data}
		<div class="space-y-2">
			<h1 class="text-3xl font-bold">{docsState.data.title}</h1>
			{#if docsState.data.path_display}
				<p class="text-base-content/70">Path: {docsState.data.path_display}</p>
			{/if}
		</div>

		<div class="grid gap-6 xl:grid-cols-[18rem_minmax(0,1fr)]">
			<AppCard title="Browse Documentation">
				<div class="space-y-2 text-sm">
					{#snippet navTree(nodes: NavNode[])}
						<ul class="space-y-2">
							{#each nodes as node}
								<li>
									<a
										class={`block rounded px-2 py-1 ${node.current ? 'bg-base-300 font-medium' : 'hover:bg-base-200'}`}
										href={`/docs/${node.path}`}
									>
										{node.title}
									</a>
									{#if node.children.length}
										<div class="ml-3 mt-2">
											{@render navTree(node.children)}
										</div>
									{/if}
								</li>
							{/each}
						</ul>
					{/snippet}

					{@render navTree(docsState.data.tree)}
				</div>
			</AppCard>

			<AppCard title={docsState.data.title}>
				<div class="mb-4 flex flex-wrap gap-2">
					{#if docsState.data.crumbs.length}
						{#each docsState.data.crumbs as crumb}
							<a class="badge badge-outline p-3" href={`/docs/${crumb.path}`}>{crumb.title}</a>
						{/each}
					{/if}
				</div>
				<div class="prose max-w-none">
					{@html docsState.data.html}
				</div>
			</AppCard>
		</div>
	{/if}
</div>
