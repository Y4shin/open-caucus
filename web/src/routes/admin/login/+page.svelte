<script lang="ts">
	import { goto } from '$app/navigation';
	import { sessionClient } from '$lib/api/index.js';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import { session } from '$lib/stores/session.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';

	let username = $state('');
	let password = $state('');
	let errorMsg = $state('');
	let loading = $state(false);

	async function handleLogin(e: SubmitEvent) {
		e.preventDefault();
		if (!session.passwordEnabled) {
			return;
		}
		errorMsg = '';
		loading = true;
		try {
			const res = await sessionClient.login({ username, password });
			if (res.session) {
				session.update(res.session);
				goto(res.session.isAdmin ? '/admin' : res.session.redirectTo || '/home');
			}
		} catch (err) {
			errorMsg = getDisplayError(err, 'Login failed.');
		} finally {
			loading = false;
		}
	}
</script>

{#if errorMsg}
	<AppAlert message={errorMsg} />
{/if}

{#if session.passwordEnabled}
	<form method="POST" action="/admin/login" onsubmit={handleLogin}>
		<div>
			<label for="username">Username:</label>
			<input
				class="input input-bordered input-sm"
				type="text"
				id="username"
				name="username"
				bind:value={username}
				oninput={(event) => {
					username = (event.currentTarget as HTMLInputElement).value;
				}}
				required
				autofocus
			/>
		</div>
		<div>
			<label for="password">Password:</label>
			<input
				class="input input-bordered input-sm"
				type="password"
				id="password"
				name="password"
				bind:value={password}
				oninput={(event) => {
					password = (event.currentTarget as HTMLInputElement).value;
				}}
				required
			/>
		</div>
		<button class="btn btn-sm" type="submit" disabled={loading}>Login</button>
	</form>
{/if}
{#if session.oauthEnabled}
	<a class="btn btn-sm btn-outline mt-2" href="/oauth/start?target=admin" data-sveltekit-reload>Login with OAuth</a>
{/if}
