<script lang="ts">
	import { locales } from '$lib/paraglide/runtime';
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import LegacyIcon from '$lib/components/ui/LegacyIcon.svelte';
	import { session } from '$lib/stores/session.svelte.js';
	import { pageActions } from '$lib/stores/page-actions.svelte.js';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	let { children } = $props();
	let mobileMenuOpen = $state(false);
	let themePreference = $state<'auto' | 'light' | 'dark'>('auto');

	const themeStorageKey = 'conference-tool.theme';

	onMount(() => {
		void import('htmx.org').then((mod) => {
			const htmx = mod.default ?? mod;
			(window as typeof window & { htmx?: unknown }).htmx = htmx;
			if (typeof (htmx as { process?: (node: Element | Document) => void }).process === 'function') {
				(htmx as { process: (node: Element | Document) => void }).process(document.body);
			}
		});
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
		goto('/docs/index');
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
				<a href={pageActions.backHref} class="btn btn-ghost btn-sm btn-square" aria-label="Back">
					<LegacyIcon name="left" />
				</a>
			{/if}
			<div class="flex-1">
				<div class="min-w-0 md:hidden">
					<h1 class="truncate text-base font-semibold sm:text-lg">Conference-Tool</h1>
				</div>
				<div class="min-w-0 hidden md:block">
					<h1 class="truncate text-base font-semibold sm:text-lg">
						Conference-Tool{pageActions.title ? ` - ${pageActions.title}` : ''}
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
						<p class="scaffold-auth-text text-xs">Logged in as {session.actor?.displayName ?? ''}</p>
						<button class="btn btn-ghost btn-sm" type="button" onclick={logout}>Logout</button>
					{:else}
						<a href="/login" class="btn btn-ghost btn-sm">Login</a>
					{/if}
				</div>
				<div class="dropdown dropdown-end md:hidden">
					<button class="btn btn-ghost btn-sm" onclick={() => (mobileMenuOpen = !mobileMenuOpen)}>
						Menu
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
									<p class="text-xs text-base-content/70">Logged in as {session.actor?.displayName ?? ''}</p>
									<button class="btn btn-sm btn-error" type="button" onclick={logout}>Logout</button>
								{:else}
									<a href="/login" class="btn btn-sm">Login</a>
								{/if}
								<div class="mt-3 flex items-center justify-between gap-2">
									<button class="btn btn-sm btn-outline" type="button" onclick={openDocs}>Help</button>
									<div class="flex items-center gap-2">
										<div class="join">
											{#each locales as locale}
												<form method="POST" action="/locale" class="inline flex-1">
													<input type="hidden" name="lang" value={locale} />
													<button
														type="submit"
														class={`btn btn-sm join-item w-full ${session.locale === locale ? 'btn-active' : ''}`}
													>
														{locale.toUpperCase()}
													</button>
												</form>
											{/each}
										</div>
										<div class="join">
											<button
												type="button"
												class={`btn btn-sm join-item flex-1 ${themePreference === 'light' ? 'btn-active' : ''}`}
												onclick={() => applyTheme('light')}
											>
												Light
											</button>
											<button
												type="button"
												class={`btn btn-sm join-item flex-1 ${themePreference === 'dark' ? 'btn-active' : ''}`}
												onclick={() => applyTheme('dark')}
											>
												Dark
											</button>
											<button
												type="button"
												class={`btn btn-sm join-item flex-1 ${themePreference === 'auto' ? 'btn-active' : ''}`}
												onclick={() => applyTheme('auto')}
											>
												Auto
											</button>
										</div>
									</div>
								</div>
								<p class="mt-2 text-xs text-base-content/70">Powered by Open Assembly</p>
							</div>
						</div>
					{/if}
				</div>
			</div>
		</nav>
		<div class="mx-auto flex w-full max-w-screen-xl min-h-0 flex-1 flex-col p-5">
			<div class="relative flex min-h-0 flex-1 flex-col gap-4 md:flex-row md:items-stretch">
				<section id="app-docs-target" class="hidden"></section>
				<main class="page-main flex min-h-0 w-full min-w-0 flex-1 flex-col gap-4 p-4">
					{@render children()}
				</main>
			</div>
		</div>
		<footer class="page-footer hidden border-t border-base-300 bg-base-100/70 md:block">
			<div class="mx-auto flex w-full max-w-screen-xl items-center justify-between gap-3 px-4 py-3 text-sm text-base-content/70 lg:px-5">
				<span class="text-xs sm:text-sm">Powered by Open Assembly</span>
				<div class="flex items-center gap-2">
					<button class="btn btn-ghost btn-sm" type="button" onclick={openDocs}>Help</button>
					<div class="join">
						{#each locales as locale}
							<form method="POST" action="/locale" class="inline">
								<input type="hidden" name="lang" value={locale} />
								<button
									type="submit"
									class={`btn btn-sm join-item ${session.locale === locale ? 'btn-active' : ''}`}
								>
									{locale.toUpperCase()}
								</button>
							</form>
						{/each}
					</div>
					<div class="join">
						<button
							type="button"
							class={`btn btn-sm join-item ${themePreference === 'light' ? 'btn-active' : ''}`}
							onclick={() => applyTheme('light')}
						>
							Light
						</button>
						<button
							type="button"
							class={`btn btn-sm join-item ${themePreference === 'dark' ? 'btn-active' : ''}`}
							onclick={() => applyTheme('dark')}
						>
							Dark
						</button>
						<button
							type="button"
							class={`btn btn-sm join-item ${themePreference === 'auto' ? 'btn-active' : ''}`}
							onclick={() => applyTheme('auto')}
						>
							Auto
						</button>
					</div>
				</div>
			</div>
		</footer>
	</div>
{/if}
