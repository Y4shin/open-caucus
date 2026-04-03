<script lang="ts">
	import { goto } from '$app/navigation';
	import { sessionClient } from '$lib/api/index.js';
	import AppAlert from '$lib/components/ui/AppAlert.svelte';
	import { session } from '$lib/stores/session.svelte.js';
	import { getDisplayError } from '$lib/utils/errors.js';
	import * as m from '$lib/paraglide/messages';

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

{#if errorMsg}
	<AppAlert message={errorMsg} />
{/if}

<div class="w-full h-full flex justify-center align-center">
	<fieldset class="fieldset bg-base-200 border-base-300 rounded-box w-xs border p-4 gap-2">
		<legend class="fieldset-legend">{m.login_legend()}</legend>
		{#if session.passwordEnabled}
			<form method="POST" action="/login" class="flex flex-col gap-2" onsubmit={handleLogin}>
				<label class="label" for="username">{m.login_username_label()}</label>
				<input
					type="text"
					name="username"
					id="username"
					class="input"
					value={username}
					placeholder={m.login_username_label()}
					oninput={(event) => {
						username = (event.currentTarget as HTMLInputElement).value;
					}}
					required
				/>

				<label class="label" for="password">{m.login_password_label()}</label>
				<input
					type="password"
					name="password"
					id="password"
					class="input"
					value={password}
					placeholder={m.login_password_label()}
					oninput={(event) => {
						password = (event.currentTarget as HTMLInputElement).value;
					}}
					required
				/>

				<button class="btn btn-neutral mt-2" disabled={loading}>{m.login_button()}</button>
			</form>
		{/if}
		{#if session.oauthEnabled}
			<a class="btn btn-outline mt-1" href="/oauth/start?target=user" data-sveltekit-reload>{m.login_oauth_button()}</a>
		{/if}
	</fieldset>
</div>
