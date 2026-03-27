import type { SessionBootstrap } from '$lib/gen/conference/session/v1/session_pb.js';
import type { CommitteeReference, ActorSummary } from '$lib/gen/conference/common/v1/common_pb.js';
import { sessionClient } from '$lib/api/index.js';

interface SessionState {
	loaded: boolean;
	authenticated: boolean;
	isAdmin: boolean;
	actor: ActorSummary | undefined;
	availableCommittees: CommitteeReference[];
	locale: string;
	passwordEnabled: boolean;
	oauthEnabled: boolean;
}

function createSessionStore() {
	let state = $state<SessionState>({
		loaded: false,
		authenticated: false,
		isAdmin: false,
		actor: undefined,
		availableCommittees: [],
		locale: 'en',
		passwordEnabled: true,
		oauthEnabled: false
	});

	return {
		get loaded() {
			return state.loaded;
		},
		get authenticated() {
			return state.authenticated;
		},
		get isAdmin() {
			return state.isAdmin;
		},
		get actor() {
			return state.actor;
		},
		get availableCommittees() {
			return state.availableCommittees;
		},
		get locale() {
			return state.locale;
		},
		get passwordEnabled() {
			return state.passwordEnabled;
		},
		get oauthEnabled() {
			return state.oauthEnabled;
		},

		async load() {
			try {
				const res = await sessionClient.getSession({});
				const s = res.session;
				if (s) {
					state.authenticated = s.authenticated;
					state.isAdmin = s.isAdmin;
					state.actor = s.actor;
					state.availableCommittees = [...s.availableCommittees];
					state.locale = s.locale || 'en';
					state.passwordEnabled = s.passwordEnabled;
					state.oauthEnabled = s.oauthEnabled;
				}
			} catch {
				state.authenticated = false;
			}
			state.loaded = true;
		},

		update(bootstrap: SessionBootstrap) {
			state.authenticated = bootstrap.authenticated;
			state.isAdmin = bootstrap.isAdmin;
			state.actor = bootstrap.actor;
			state.availableCommittees = [...bootstrap.availableCommittees];
			state.locale = bootstrap.locale || 'en';
			state.passwordEnabled = bootstrap.passwordEnabled;
			state.oauthEnabled = bootstrap.oauthEnabled;
			state.loaded = true;
		},

		clear() {
			state.authenticated = false;
			state.isAdmin = false;
			state.actor = undefined;
			state.availableCommittees = [];
			state.passwordEnabled = true;
			state.oauthEnabled = false;
			state.loaded = false;
		}
	};
}

export const session = createSessionStore();
