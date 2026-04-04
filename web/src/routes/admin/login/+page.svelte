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
				goto(res.session.isAdmin ? '/admin' : res.session.redirectTo || '/home');
			}
		} catch (err) {
			errorMsg = getDisplayError(err, 'Login failed.');
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex min-h-full items-center justify-center">
	<fieldset class="fieldset bg-base-200 border-base-300 rounded-box w-xs border p-4 gap-2">
		<legend class="fieldset-legend">{m.admin_login_legend()}</legend>

		{#if errorMsg}
			<AppAlert message={errorMsg} />
		{/if}

		{#if session.passwordEnabled}
			<form method="POST" action="/admin/login" class="flex flex-col gap-2" onsubmit={handleLogin}>
				<label class="label" for="username">{m.admin_login_username_label()}</label>
				<input
					class="input"
					type="text"
					id="username"
					name="username"
					value={username}
					placeholder={m.admin_login_username_label()}
					oninput={(event) => {
						username = (event.currentTarget as HTMLInputElement).value;
					}}
					required
				/>

				<label class="label" for="password">{m.admin_login_password_label()}</label>
				<input
					class="input"
					type="password"
					id="password"
					name="password"
					value={password}
					placeholder={m.admin_login_password_label()}
					oninput={(event) => {
						password = (event.currentTarget as HTMLInputElement).value;
					}}
					required
				/>

				<button class="btn btn-neutral mt-2" type="submit" disabled={loading}>{m.admin_login_button()}</button>
			</form>
		{/if}
		{#if session.oauthEnabled}
			<a class="btn btn-outline mt-1" href="/oauth/start?target=admin" data-sveltekit-reload>{m.admin_login_oauth_button()}</a>
		{/if}
		<a class="btn btn-ghost btn-sm mt-2" href="/login">{m.admin_login_back_button()}</a>
	</fieldset>
</div>
