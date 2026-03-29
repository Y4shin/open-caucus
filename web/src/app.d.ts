// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
import 'svelte/elements';

declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

declare module 'svelte/elements' {
	interface HTMLAttributes<T> {
		'hx-get'?: string;
		'hx-post'?: string;
		'hx-target'?: string;
		'hx-swap'?: string;
		'hx-swap-oob'?: string;
		'hx-trigger'?: string;
		'hx-vals'?: string;
		'hx-include'?: string;
		'hx-confirm'?: string;
		'hx-on::after-request'?: string;
		'sse-swap'?: string;
	}
}

export {};
