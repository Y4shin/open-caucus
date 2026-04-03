<script lang="ts">
	import { page } from '$app/state';
	import { getLocale, locales, setLocale } from '$lib/paraglide/runtime';
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import DocsOverlay from '$lib/components/docs/DocsOverlay.svelte';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import { docsClient } from '$lib/api/index.js';
	import {
		DOCS_HEADING_PARAM,
		DOCS_PATH_PARAM,
		DOCS_QUERY_PARAM,
		buildDocsOverlayHref
	} from '$lib/docs/navigation.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import { createRemoteState } from '$lib/utils/remote.svelte.js';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import * as m from '$lib/paraglide/messages';
	import { PUBLIC_APP_VERSION } from '$env/static/public';

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

	interface SearchHit {
		ref: string;
		path: string;
		heading?: string;
		title: string;
		snippet?: string;
	}

	interface DocsOverlayData {
		title: string;
		locale: string;
		pathDisplay: string;
		crumbs: Crumb[];
		tree: NavNode[];
		html?: string;
		query?: string;
		searchHits?: SearchHit[] | null;
		error?: string;
		notFound?: boolean;
	}

	let { children } = $props();
	let mobileMenuOpen = $state(false);
	let themePreference = $state<'auto' | 'light' | 'dark'>('auto');
	let docsOverlayState = $state(createRemoteState<DocsOverlayData>());
	let docsOverlayRequestId = 0;

	const themeStorageKey = 'conference-tool.theme';
	const docsOverlayPath = $derived(page.url.searchParams.get(DOCS_PATH_PARAM) ?? '');
	const docsOverlayHeading = $derived(page.url.searchParams.get(DOCS_HEADING_PARAM) ?? '');
	const docsOverlayQuery = $derived(page.url.searchParams.get(DOCS_QUERY_PARAM) ?? '');
	const standaloneDocsRoute = $derived(page.url.pathname.startsWith('/docs'));
	const docsOverlayOpen = $derived(!standaloneDocsRoute && docsOverlayPath !== '');

	onMount(() => {
		try {
			const stored = window.localStorage.getItem(themeStorageKey);
			if (stored === 'light' || stored === 'dark' || stored === 'auto') {
				themePreference = stored;
			}
		} catch {
			themePreference = 'auto';
		}
		applyTheme(themePreference);
		session.load();
	});

	function applyTheme(preference: 'auto' | 'light' | 'dark') {
		themePreference = preference;
		try {
			window.localStorage.setItem(themeStorageKey, preference);
		} catch {
			// Ignore storage failures.
		}
		if (preference === 'dark') {
			document.documentElement.setAttribute('data-theme', 'dark');
			return;
		}
		if (preference === 'light') {
			document.documentElement.setAttribute('data-theme', 'corporate');
			return;
		}
		document.documentElement.removeAttribute('data-theme');
	}

	async function logout(event: Event) {
		event.preventDefault();
		const { sessionClient } = await import('$lib/api/index.js');
		await sessionClient.logout({});
		session.clear();
		goto('/login');
	}

	function openDocs(event: Event) {
		event.preventDefault();
		goto(buildDocsOverlayHref('index', page.url));
	}

	function isActiveLocale(lang: (typeof locales)[number]) {
		return getLocale() === lang;
	}

	async function switchLocale(lang: (typeof locales)[number]) {
		document.cookie = `locale=${lang}; path=/; max-age=${365 * 24 * 60 * 60}; samesite=lax`;
		await setLocale(lang);
	}

	$effect(() => {
		const overlayPath = docsOverlayPath;
		const overlayHeading = docsOverlayHeading;
		const overlayQuery = docsOverlayQuery;
		const isStandaloneDocsRoute = standaloneDocsRoute;

		if (isStandaloneDocsRoute || !overlayPath) {
			docsOverlayState.data = null;
			docsOverlayState.error = '';
			docsOverlayState.loading = false;
			return;
		}

		void loadDocsOverlay(overlayPath, overlayHeading, overlayQuery);
	});

	async function loadDocsOverlay(path: string, heading: string, query: string) {
		const requestId = ++docsOverlayRequestId;
		docsOverlayState.loading = true;
		docsOverlayState.error = '';
		try {
			if (path === 'search') {
				const [shellResponse, searchResponse] = await Promise.all([
					docsClient.getPage({ path: 'index' }),
					docsClient.search({ query, limit: 10 })
				]);
				if (requestId !== docsOverlayRequestId) return;
				const shell = shellResponse.page;
				if (!shell) {
					docsOverlayState.data = null;
					docsOverlayState.error = 'Failed to load the documentation shell.';
					return;
				}
				docsOverlayState.data = {
					title: shell.title,
					locale: shell.locale,
					pathDisplay: shell.pathDisplay,
					crumbs: shell.crumbs as Crumb[],
					tree: shell.tree as NavNode[],
					query,
					searchHits: (searchResponse.hits ?? []) as SearchHit[]
				};
				return;
			}

			const response = await docsClient.getPage({ path, heading });
			if (requestId !== docsOverlayRequestId) return;
			const docsPage = response.page;
			if (!docsPage) {
				docsOverlayState.data = null;
				docsOverlayState.error = 'Failed to load the documentation page.';
				return;
			}
			docsOverlayState.data = {
				title: docsPage.title,
				locale: docsPage.locale,
				pathDisplay: docsPage.pathDisplay,
				crumbs: docsPage.crumbs as Crumb[],
				tree: docsPage.tree as NavNode[],
				html: docsPage.html,
				searchHits: null
			};
		} catch (err) {
			if (requestId !== docsOverlayRequestId) return;
			docsOverlayState.data = null;
			docsOverlayState.error = getDisplayError(err, 'Failed to load the documentation.');
		} finally {
			if (requestId === docsOverlayRequestId) {
				docsOverlayState.loading = false;
			}
		}
	}
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

{#if !session.loaded}
	<div class="flex min-h-screen items-center justify-center">
		<span class="loading loading-spinner loading-lg"></span>
	</div>
{:else}
	<div class="app-shell bg-base-100 text-base-content">
		<nav class="navbar sticky top-0 z-30 bg-base-100 shadow-sm">
			{#if pageActions.backHref}
				<a href={pageActions.backHref} class="btn btn-ghost btn-sm btn-square" aria-label={m.common_back()}>
					<LegacyIcon name="left" />
				</a>
			{/if}
			<div class="flex-1">
				<div class="min-w-0 md:hidden">
					<h1 class="truncate text-base font-semibold sm:text-lg">{m.common_app_name()}</h1>
				</div>
				<div class="min-w-0 hidden md:block">
					<h1 class="truncate text-base font-semibold sm:text-lg">
						{m.common_app_name()}{pageActions.title ? ` - ${pageActions.title}` : ''}
					</h1>
				</div>
			</div>
			<div class="flex-none flex flex-row items-center gap-2">
				<div class="hidden items-center gap-2 md:flex">
					{#if pageActions.actions.length > 0}
						<div class="flex items-center gap-2">
							{#each pageActions.actions as action}
								<a
									class={`btn btn-sm ${action.kind === 'ghost' ? 'btn-ghost' : 'btn-primary'}`}
									href={action.href}
								>
									{action.label}
								</a>
							{/each}
						</div>
					{/if}
					{#if session.authenticated}
						{#if session.isAdmin}
							<a href="/admin" class="btn btn-ghost btn-sm">Admin</a>
						{/if}
						<p class="scaffold-auth-text text-xs">{m.common_logged_in_as({ name: session.actor?.displayName ?? '' })}</p>
						<button class="btn btn-ghost btn-sm" type="button" onclick={logout}>{m.common_logout()}</button>
					{:else}
						<a href="/login" class="btn btn-ghost btn-sm">{m.login_button()}</a>
					{/if}
				</div>
				<div class="dropdown dropdown-end md:hidden">
					<button class="btn btn-ghost btn-sm" onclick={() => (mobileMenuOpen = !mobileMenuOpen)}>
						{m.scaffold_menu()}
					</button>
					{#if mobileMenuOpen}
						<div class="dropdown-content z-[1] mt-2 w-80 max-w-[calc(100vw-2rem)] rounded-box border border-base-300 bg-base-100 p-3 shadow">
							{#if pageActions.subtitle}
								<p class="mb-2 text-sm text-base-content/70">{pageActions.subtitle}</p>
							{/if}
							<div class="flex flex-col gap-2">
								{#if pageActions.actions.length > 0}
									{#each pageActions.actions as action}
										<a
											class={`btn btn-sm justify-start ${action.kind === 'ghost' ? 'btn-ghost' : 'btn-primary'}`}
											href={action.href}
										>
											{action.label}
										</a>
									{/each}
								{/if}
								{#if session.isAdmin}
									<a class="btn btn-sm justify-start" href="/admin">Admin</a>
								{/if}
								{#if session.authenticated}
									<p class="text-xs text-base-content/70">{m.common_logged_in_as({ name: session.actor?.displayName ?? '' })}</p>
									<button class="btn btn-sm btn-error" type="button" onclick={logout}>{m.common_logout()}</button>
								{:else}
									<a href="/login" class="btn btn-sm">{m.login_button()}</a>
								{/if}
								<div class="mt-3 flex items-center justify-between gap-2">
									<button class="btn btn-sm btn-outline" type="button" onclick={openDocs}>{m.common_help()}</button>
									<div class="flex items-center gap-2">
										<div class="join">
											{#each locales as locale}
												<button
													type="button"
													class={`btn btn-sm join-item flex-1 ${isActiveLocale(locale) ? 'btn-active' : ''}`}
													onclick={() => switchLocale(locale)}
												>
													{locale.toUpperCase()}
												</button>
											{/each}
										</div>
										<div class="join">
											<button
												type="button"
												class={`btn btn-sm join-item flex-1 ${themePreference === 'light' ? 'btn-active' : ''}`}
												onclick={() => applyTheme('light')}
											>
												{m.theme_switcher_light_button()}
											</button>
											<button
												type="button"
												class={`btn btn-sm join-item flex-1 ${themePreference === 'dark' ? 'btn-active' : ''}`}
												onclick={() => applyTheme('dark')}
											>
												{m.theme_switcher_dark_button()}
											</button>
											<button
												type="button"
												class={`btn btn-sm join-item flex-1 ${themePreference === 'auto' ? 'btn-active' : ''}`}
												onclick={() => applyTheme('auto')}
											>
												{m.theme_switcher_auto_button()}
											</button>
										</div>
									</div>
								</div>
								<p class="mt-2 text-xs text-base-content/70">Powered by Open Caucus v{PUBLIC_APP_VERSION}</p>
							</div>
						</div>
					{/if}
				</div>
			</div>
		</nav>
		<div class="mx-auto flex w-full max-w-screen-xl min-h-0 flex-1 flex-col p-5">
			<div class="relative flex min-h-0 flex-1 flex-col gap-4 md:flex-row md:items-stretch">
				<main class="page-main flex min-h-0 w-full min-w-0 flex-1 flex-col gap-4 p-4">
					{@render children()}
				</main>
				{#if docsOverlayOpen && docsOverlayState.data}
					<DocsOverlay
						title={docsOverlayState.data.title}
						locale={docsOverlayState.data.locale}
						pathDisplay={docsOverlayState.data.pathDisplay}
						crumbs={docsOverlayState.data.crumbs}
						tree={docsOverlayState.data.tree}
						html={docsOverlayState.data.html ?? ''}
						query={docsOverlayState.data.query ?? ''}
						error={docsOverlayState.data.error ?? ''}
						notFound={docsOverlayState.data.notFound ?? false}
						searchHits={docsOverlayState.data.searchHits ?? null}
						overlayMode
					/>
				{:else if docsOverlayOpen && docsOverlayState.loading}
					<section
						id="app-docs-target"
						class="fixed inset-0 z-50 md:inset-y-0 md:left-auto md:right-0 md:z-40 md:w-[33.333vw]"
						data-docs-open="1"
					>
						<div class="absolute inset-0 bg-neutral-950/70 md:hidden"></div>
						<div class="absolute inset-x-0 bottom-0 flex h-[67dvh] min-h-[22rem] max-h-[90dvh] min-w-0 flex-col rounded-t-2xl border border-base-300 bg-base-100 shadow-2xl md:relative md:inset-auto md:h-full md:min-h-0 md:max-h-none md:rounded-none md:border-0 md:border-l md:border-base-300">
							<div class="flex min-h-0 flex-1 items-center justify-center p-4 md:p-5">
								<span class="loading loading-spinner loading-lg" aria-label="Loading documentation"></span>
							</div>
						</div>
					</section>
				{:else if docsOverlayOpen && docsOverlayState.error}
					<section
						id="app-docs-target"
						class="fixed inset-0 z-50 md:inset-y-0 md:left-auto md:right-0 md:z-40 md:w-[33.333vw]"
						data-docs-open="1"
					>
						<div class="absolute inset-0 bg-neutral-950/70 md:hidden"></div>
						<div class="absolute inset-x-0 bottom-0 flex h-[67dvh] min-h-[22rem] max-h-[90dvh] min-w-0 flex-col rounded-t-2xl border border-base-300 bg-base-100 shadow-2xl md:relative md:inset-auto md:h-full md:min-h-0 md:max-h-none md:rounded-none md:border-0 md:border-l md:border-base-300">
							<div class="p-4 md:p-5">
								<div role="alert" class="alert alert-error">
									<span>{docsOverlayState.error}</span>
								</div>
							</div>
						</div>
					</section>
				{/if}
			</div>
		</div>
		<footer class="page-footer hidden border-t border-base-300 bg-base-100/70 md:block">
			<div class="mx-auto flex w-full max-w-screen-xl items-center justify-between gap-3 px-4 py-3 text-sm text-base-content/70 lg:px-5">
				<span class="text-xs sm:text-sm">Powered by Open Caucus v{PUBLIC_APP_VERSION}</span>
				<div class="flex items-center gap-2">
					<button class="btn btn-ghost btn-sm" type="button" onclick={openDocs}>{m.common_help()}</button>
					<div class="join">
						{#each locales as locale}
							<button
								type="button"
								class={`btn btn-sm join-item ${isActiveLocale(locale) ? 'btn-active' : ''}`}
								onclick={() => switchLocale(locale)}
							>
								{locale.toUpperCase()}
							</button>
						{/each}
					</div>
					<div class="join">
						<button
							type="button"
							class={`btn btn-sm join-item ${themePreference === 'light' ? 'btn-active' : ''}`}
							onclick={() => applyTheme('light')}
						>
							{m.theme_switcher_light_button()}
						</button>
						<button
							type="button"
							class={`btn btn-sm join-item ${themePreference === 'dark' ? 'btn-active' : ''}`}
							onclick={() => applyTheme('dark')}
						>
							{m.theme_switcher_dark_button()}
						</button>
						<button
							type="button"
							class={`btn btn-sm join-item ${themePreference === 'auto' ? 'btn-active' : ''}`}
							onclick={() => applyTheme('auto')}
						>
							{m.theme_switcher_auto_button()}
						</button>
					</div>
				</div>
			</div>
		</footer>
	</div>
{/if}
