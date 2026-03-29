import { voteClient } from '$lib/api/index.js';

export type StoredReceipt = {
	id: string;
	kind: 'open' | 'secret';
	voteId: string;
	voteName: string;
	receiptToken: string;
	receipt: string;
};

const STORAGE_KEY = 'conference_tool_receipts';

function canUseStorage() {
	return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined';
}

export function listReceipts(): StoredReceipt[] {
	if (!canUseStorage()) return [];

	try {
		const raw = window.localStorage.getItem(STORAGE_KEY);
		if (!raw) return [];
		const parsed = JSON.parse(raw) as StoredReceipt[];
		return Array.isArray(parsed) ? parsed : [];
	} catch {
		return [];
	}
}

export function saveReceipt(receipt: StoredReceipt) {
	if (!canUseStorage()) return;

	const receipts = listReceipts().filter((item) => item.id !== receipt.id);
	receipts.unshift(receipt);
	window.localStorage.setItem(STORAGE_KEY, JSON.stringify(receipts));
}

export function clearReceipts() {
	if (!canUseStorage()) return;
	window.localStorage.removeItem(STORAGE_KEY);
}

export async function verifyReceipt(receipt: StoredReceipt) {
	try {
		if (receipt.kind === 'secret') {
			return await voteClient.verifySecretReceipt({
				voteId: receipt.voteId,
				receiptToken: receipt.receiptToken
			});
		}
		return await voteClient.verifyOpenReceipt({
			voteId: receipt.voteId,
			receiptToken: receipt.receiptToken
		});
	} catch (err) {
		if (err instanceof Error) {
			throw err;
		}
		throw new Error('Verification failed.');
	}
}
