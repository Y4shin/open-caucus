import { ConnectError } from '@connectrpc/connect';

export function getDisplayError(error: unknown, fallback: string): string {
	if (error instanceof ConnectError) {
		return error.rawMessage || fallback;
	}
	if (error instanceof Error && error.message) {
		return error.message;
	}
	return fallback;
}
