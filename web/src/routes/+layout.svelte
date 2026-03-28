<script lang="ts">
	import { page } from '$app/state';
	import { locales, localizeHref } from '$lib/paraglide/runtime';
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { session } from '$lib/stores/session.svelte.js';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	let { children } = $props();
	let localeMenuOpen = $state(false);

	onMount(() => {
		session.load();
	});
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

{#if !session.loaded}
	<div class="flex min-h-screen items-center justify-center">
		<span class="loading loading-spinner loading-lg"></span>
	</div>
{:else}
	<div class="min-h-screen bg-base-100">
		<nav class="navbar bg-base-200 shadow-sm">
			<div class="navbar-start">
				<a href="/home" class="btn btn-ghost text-xl font-bold">Conference Tool</a>
			</div>
			<div class="navbar-center hidden gap-1 lg:flex">
				{#if session.authenticated}
					<a href="/home" class="btn btn-ghost btn-sm">Home</a>
					{#each session.availableCommittees as committee}
						<a href="/committee/{committee.slug}" class="btn btn-ghost btn-sm">{committee.name}</a>
					{/each}
					{#if session.isAdmin}
						<a href="/admin" class="btn btn-ghost btn-sm">Admin</a>
					{/if}
				{/if}
			</div>
			<div class="navbar-end gap-2">
				<div class="relative">
					<button
						class="btn btn-ghost btn-sm"
						onclick={() => (localeMenuOpen = !localeMenuOpen)}
						aria-haspopup="menu"
						aria-expanded={localeMenuOpen}
					>
						{session.locale.toUpperCase()}
					</button>
					{#if localeMenuOpen}
						<ul
							role="menu"
							class="menu absolute right-0 top-full z-10 w-24 rounded-box bg-base-200 p-2 shadow"
						>
							{#each locales as locale}
								<li role="menuitem">
									<a
										href={localizeHref(page.url.pathname, { locale })}
										onclick={() => (localeMenuOpen = false)}
										class="text-sm">{locale.toUpperCase()}</a
									>
								</li>
							{/each}
						</ul>
					{/if}
				</div>
				{#if session.authenticated}
					<p class="scaffold-auth-text hidden text-xs lg:block">{session.actor?.displayName ?? ''}</p>
					<button
						class="btn btn-ghost btn-sm"
						onclick={async () => {
							const { sessionClient } = await import('$lib/api/index.js');
							await sessionClient.logout({});
							session.clear();
							goto('/login');
						}}
					>
						Logout
					</button>
				{:else}
					<a href="/login" class="btn btn-primary btn-sm">Login</a>
				{/if}
			</div>
		</nav>
		<main class="container mx-auto px-4 py-6">
			{@render children()}
		</main>
	</div>
{/if}
