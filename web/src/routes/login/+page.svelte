<script lang="ts">
	import { goto } from '$app/navigation';
	import { sessionClient } from '$lib/api/index.js';
	import { session } from '$lib/stores/session.svelte.js';
	import { ConnectError } from '@connectrpc/connect';

	let username = $state('');
	let password = $state('');
	let errorMsg = $state('');
	let loading = $state(false);

	async function handleLogin(e: SubmitEvent) {
		e.preventDefault();
		errorMsg = '';
		loading = true;
		try {
			const res = await sessionClient.login({ username, password });
			if (res.session) {
				session.update(res.session);
				const redirect = res.session.redirectTo || '/';
				goto(redirect);
			}
		} catch (err) {
			if (err instanceof ConnectError) {
				errorMsg = err.rawMessage || 'Login failed.';
			} else {
				errorMsg = 'An unexpected error occurred.';
			}
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex min-h-[60vh] items-center justify-center">
	<div class="card w-full max-w-sm bg-base-200 shadow-xl">
		<div class="card-body">
			<h2 class="card-title text-2xl">Sign In</h2>

			{#if errorMsg}
				<div role="alert" class="alert alert-error">
					<span>{errorMsg}</span>
				</div>
			{/if}

			<form onsubmit={handleLogin} class="flex flex-col gap-4">
				<label class="form-control">
					<div class="label"><span class="label-text">Username</span></div>
					<input
						type="text"
						class="input input-bordered"
						bind:value={username}
						autocomplete="username"
						required
					/>
				</label>

				<label class="form-control">
					<div class="label"><span class="label-text">Password</span></div>
					<input
						type="password"
						class="input input-bordered"
						bind:value={password}
						autocomplete="current-password"
						required
					/>
				</label>

				<button type="submit" class="btn btn-primary" disabled={loading}>
					{#if loading}
						<span class="loading loading-spinner loading-sm"></span>
					{/if}
					Sign In
				</button>
			</form>
		</div>
	</div>
</div>
