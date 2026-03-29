interface PageAction {
	label: string;
	href: string;
	kind?: 'primary' | 'ghost';
}

function createPageActionsStore() {
	let state = $state<{
		actions: PageAction[];
		backHref: string | null;
		title: string;
		subtitle: string;
	}>({
		actions: [],
		backHref: null,
		title: '',
		subtitle: ''
	});

	return {
		get actions() {
			return state.actions;
		},

		get backHref() {
			return state.backHref;
		},

		get title() {
			return state.title;
		},

		get subtitle() {
			return state.subtitle;
		},

		set(actions: PageAction[], options?: { backHref?: string | null; title?: string; subtitle?: string }) {
			state.actions = [...actions];
			state.backHref = options?.backHref ?? null;
			state.title = options?.title ?? '';
			state.subtitle = options?.subtitle ?? '';
		},

		clear() {
			state.actions = [];
			state.backHref = null;
			state.title = '';
			state.subtitle = '';
		}
	};
}

export type { PageAction };
export const pageActions = createPageActionsStore();
