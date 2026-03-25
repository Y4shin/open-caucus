export interface RemoteState<T> {
	data: T | null;
	error: string;
	loading: boolean;
}

export function createRemoteState<T>(initialData: T | null = null): RemoteState<T> {
	return {
		data: initialData,
		error: '',
		loading: false
	};
}
