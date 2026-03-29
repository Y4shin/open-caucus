<script lang="ts">
	import { goto } from '$app/navigation';

	type Crumb = {
		title: string;
		path: string;
		current: boolean;
	};

	interface NavNode {
		title: string;
		path: string;
		current: boolean;
		expanded: boolean;
		children: NavNode[];
	}

	type SearchHit = {
		ref: string;
		path: string;
		heading?: string;
		title: string;
		snippet?: string;
	};

	let {
		title,
		locale,
		pathDisplay = '',
		crumbs = [],
		tree = [],
		query = '',
		html = '',
		error = '',
		notFound = false,
		searchHits = null
	}: {
		title: string;
		locale: string;
		pathDisplay?: string;
		crumbs?: Crumb[];
		tree?: Array<{
			title: string;
			path: string;
			current: boolean;
			expanded: boolean;
			children: unknown[];
		}>;
		query?: string;
		html?: string;
		error?: string;
		notFound?: boolean;
		searchHits?: SearchHit[] | null;
	} = $props();

	function closeDocs() {
		if (window.history.length > 1) {
			window.history.back();
			return;
		}
		goto('/home');
	}
	const resolvedPathDisplay = $derived(
		pathDisplay || (crumbs.length ? crumbs.map((crumb) => crumb.title).join(' / ') : '')
	);

	function hasVisibleHeading() {
		return html.toLowerCase().includes('<h1');
	}
</script>

<section
	id="app-docs-target"
	class="fixed inset-0 z-50 md:inset-y-0 md:left-auto md:right-0 md:z-40 md:w-[33.333vw]"
	data-docs-open="1"
>
	<button
		type="button"
		class="absolute inset-0 bg-neutral-950/70 md:hidden"
		aria-label="Close documentation"
		data-docs-close
		onclick={closeDocs}
	></button>
	<div class="absolute inset-x-0 bottom-0 flex h-[67dvh] min-h-[22rem] max-h-[90dvh] min-w-0 flex-col rounded-t-2xl border border-base-300 bg-base-100 shadow-2xl md:relative md:inset-auto md:h-full md:min-h-0 md:max-h-none md:rounded-none md:border-0 md:border-l md:border-base-300">
		<div class="min-h-0 flex-1 overflow-y-auto p-4 md:p-5">
			<div class="flex items-center justify-between gap-2">
				<div>
					<h2 class="text-lg font-semibold">{title}</h2>
					{#if resolvedPathDisplay}
						<p class="text-xs text-base-content/70">Path: {resolvedPathDisplay}</p>
					{/if}
				</div>
				<div class="flex items-center gap-2">
					<span class="text-xs text-base-content/70">{locale}</span>
					<button type="button" class="btn btn-ghost btn-xs" data-docs-close onclick={closeDocs}>Close</button>
				</div>
			</div>
			<form class="mt-3 flex gap-2" action="/docs/search" method="GET">
				<input class="input input-bordered input-sm flex-1" type="search" name="q" value={query} placeholder="Search documentation" />
			</form>
			<details class="collapse collapse-arrow mt-3 border border-base-300 bg-base-200/30" open>
				<summary class="collapse-title py-2 pr-8 text-sm font-medium">Browse Documentation</summary>
				<div class="collapse-content">
					{#if crumbs.length}
						<div class="mb-2 flex flex-wrap items-center gap-1 text-xs text-base-content/80">
							{#each crumbs as crumb, index}
								{#if index > 0}
									<span>/</span>
								{/if}
								<a href={`/docs/${crumb.path}`} class={crumb.current ? 'hover:underline font-semibold text-primary' : 'hover:underline'}>{crumb.title}</a>
							{/each}
						</div>
					{/if}
					<div class="max-h-64 overflow-y-auto rounded-box border border-base-300 bg-base-100 p-2">
						{#snippet navTree(nodes: Array<{ title: string; path: string; current: boolean; expanded: boolean; children: unknown[] }>)}
							<div class="space-y-1">
								{#each nodes as node}
									{#if node.children.length}
										<details class="collapse collapse-arrow border border-base-300 bg-base-100" open={node.expanded || node.current}>
											<summary class="collapse-title py-2 pr-8 text-sm">
												<a href={`/docs/${node.path}`} class={node.current ? 'font-medium text-primary hover:underline' : 'font-medium hover:underline'}>
													{node.title}
												</a>
											</summary>
											<div class="collapse-content pt-0 pb-2">
												<div class="pl-2 border-l border-base-300">
													{@render navTree(node.children as Array<{ title: string; path: string; current: boolean; expanded: boolean; children: unknown[] }>)}
												</div>
											</div>
										</details>
									{:else}
										<a
											href={`/docs/${node.path}`}
											class={node.current
												? 'block rounded px-2 py-1 text-sm hover:bg-base-200 bg-base-200 font-semibold text-primary'
												: 'block rounded px-2 py-1 text-sm hover:bg-base-200'}
										>
											{node.title}
										</a>
									{/if}
								{/each}
							</div>
						{/snippet}
						{#if tree.length}
							{@render navTree(tree)}
						{:else}
							<p class="text-sm text-base-content/70">No documentation tree available.</p>
						{/if}
					</div>
				</div>
			</details>
			{#if searchHits !== null}
				<div class="rounded-box border border-base-300 bg-base-200/30 p-3" id="docs-search-results">
					{#if error}
						<p class="text-sm text-error">{error}</p>
					{:else if searchHits.length === 0}
						<p class="text-sm text-base-content/70">No documentation results matched that query.</p>
					{:else}
						<ul class="space-y-2">
							{#each searchHits as hit}
								<li class="rounded-box border border-base-300 bg-base-100 p-2">
									<a class="font-medium text-sm" href={`/docs/${hit.ref}`}>{hit.title}</a>
									<p class="text-xs text-base-content/70">{hit.ref}</p>
									{#if hit.snippet}
										<p class="text-xs text-base-content/80">{hit.snippet}</p>
									{/if}
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			{:else}
				<div class="mt-3" id="docs-search-results"></div>
			{/if}
			{#if searchHits === null}
				{#if error}
					<p class="mt-3 text-sm text-error">{error}</p>
				{:else if notFound}
					<p class="mt-3 text-sm text-warning">Documentation page not found.</p>
				{:else if html}
					<div class="docs-markdown mt-4">
						{#if !hasVisibleHeading()}
							<h1>{title}</h1>
						{/if}
						{@html html}
					</div>
				{/if}
			{/if}
		</div>
	</div>
</section>
