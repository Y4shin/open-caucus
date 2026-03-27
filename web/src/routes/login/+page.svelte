<script lang="ts">
	import { goto } from '$app/navigation';
	import { sessionClient } from '$lib/api/index.js';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import AppCard from '$lib/components/ui/AppCard.svelte';
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
				const redirect = res.session.redirectTo || '/';
				goto(redirect);
			}
		} catch (err) {
			errorMsg = getDisplayError(err, 'Login failed.');
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex min-h-[60vh] items-center justify-center">
	<div class="w-full max-w-sm">
		<AppCard title="Sign In">
			{#if errorMsg}
				<AppAlert message={errorMsg} />
			{/if}

			{#if session.passwordEnabled}
				<form onsubmit={handleLogin} class="flex flex-col gap-4">
					<label class="form-control">
						<div class="label"><span class="label-text">Username</span></div>
						<input
							type="text"
							name="username"
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
							name="password"
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
			{/if}

			{#if session.oauthEnabled}
				<a class="btn btn-outline" href="/oauth/start?target=user">Sign in with OAuth</a>
			{/if}

			{#if !session.passwordEnabled && !session.oauthEnabled}
				<AppAlert message="No login method is currently enabled." />
			{/if}
		</AppCard>
	</div>
</div>
