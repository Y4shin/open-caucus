export function legacyAttrs(node: HTMLElement, attrs: Record<string, string | boolean | null | undefined>) {
	function apply(next: Record<string, string | boolean | null | undefined>) {
		for (const [name, value] of Object.entries(next)) {
			if (value === false || value === null || value === undefined) {
				node.removeAttribute(name);
				continue;
			}
			if (value === true) {
				node.setAttribute(name, '');
				continue;
			}
			node.setAttribute(name, value);
		}
	}

	apply(attrs);

	return {
		update(next: Record<string, string | boolean | null | undefined>) {
			apply(next);
		}
	};
}

export function legacyValueAttr(
	node: HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement,
	value: string | number | null | undefined
) {
	function apply(next: string | number | null | undefined) {
		if (next === null || next === undefined) {
			node.removeAttribute('value');
			return;
		}
		node.setAttribute('value', String(next));
	}

	apply(value);

	return {
		update(next: string | number | null | undefined) {
			apply(next);
		}
	};
}
