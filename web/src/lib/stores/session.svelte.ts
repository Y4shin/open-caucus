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
}

function createSessionStore() {
	let state = $state<SessionState>({
		loaded: false,
		authenticated: false,
		isAdmin: false,
		actor: undefined,
		availableCommittees: [],
		locale: 'en'
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
			state.loaded = true;
		},

		clear() {
			state.authenticated = false;
			state.isAdmin = false;
			state.actor = undefined;
			state.availableCommittees = [];
			state.loaded = false;
		}
	};
}

export const session = createSessionStore();
