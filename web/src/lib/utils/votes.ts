export function voteStateBadgeClass(state: string): string {
	switch (state) {
		case 'draft':
			return 'badge badge-neutral badge-sm';
		case 'open':
			return 'badge badge-success badge-sm';
		case 'counting':
			return 'badge badge-warning badge-sm';
		case 'closed':
			return 'badge badge-info badge-sm';
		case 'archived':
			return 'badge badge-ghost badge-sm';
		default:
			return 'badge badge-sm';
	}
}

export function voteVisibilityBadgeClass(visibility: string): string {
	return visibility === 'secret'
		? 'badge badge-warning badge-outline badge-sm'
		: 'badge badge-primary badge-outline badge-sm';
}
