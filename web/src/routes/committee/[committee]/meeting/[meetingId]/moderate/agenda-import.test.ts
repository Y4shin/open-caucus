import { describe, it, expect } from 'vitest';
import {
	parseAgendaImportSource,
	buildImportedAgenda,
	matchTitlePairs,
	computeSubDiff,
	computeAgendaDiff,
	type AgendaImportLine,
	type AgendaPointLike
} from './agenda-import.js';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function pt(id: string, title: string, subPoints: AgendaPointLike[] = []): AgendaPointLike {
	return { agendaPointId: id, title, subPoints };
}

function lines(ops: Record<string, 'heading' | 'subheading' | 'ignore'>): AgendaImportLine[] {
	return Object.entries(ops).map(([text, state], i) => ({ lineNo: i + 1, text, rawLine: text, titleStart: 0, state }));
}

// ---------------------------------------------------------------------------
// parseAgendaImportSource
// ---------------------------------------------------------------------------

describe('parseAgendaImportSource - plain text, numbered', () => {
	it('parses "1. Title" as heading', () => {
		const result = parseAgendaImportSource('1. First\n2. Second');
		expect(result).toHaveLength(2);
		expect(result[0]).toMatchObject({ state: 'heading', text: 'First' });
		expect(result[1]).toMatchObject({ state: 'heading', text: 'Second' });
	});

	it('parses "1.1" as subheading', () => {
		const result = parseAgendaImportSource('1. First\n1.1 Sub\n2. Second');
		expect(result[1]).toMatchObject({ state: 'subheading', text: 'Sub' });
	});

	it('parses "TOP 1" prefix', () => {
		const result = parseAgendaImportSource('TOP 1. Motion A\nTOP 1.1 Detail');
		expect(result[0]).toMatchObject({ state: 'heading', text: 'Motion A' });
		expect(result[1]).toMatchObject({ state: 'subheading', text: 'Detail' });
	});

	it('parses various separators (colon, paren, space)', () => {
		for (const line of ['1: Alpha', '2) Beta', '3 - Gamma']) {
			const [r] = parseAgendaImportSource(line);
			expect(r.state).toBe('heading');
		}
	});

	it('marks unmatched lines as ignore', () => {
		const result = parseAgendaImportSource('1. First\nnot a number');
		expect(result[1]).toMatchObject({ state: 'ignore', text: 'not a number' });
	});

	it('skips empty lines', () => {
		const result = parseAgendaImportSource('1. A\n\n2. B');
		expect(result).toHaveLength(2);
		expect(result[0].lineNo).toBe(1);
		expect(result[1].lineNo).toBe(3);
	});
});

describe('parseAgendaImportSource - plain text, indentation', () => {
	it('treats every line as heading when no indentation', () => {
		const result = parseAgendaImportSource('Alpha\nBeta\nGamma');
		expect(result.every((r) => r.state === 'heading')).toBe(true);
	});

	it('uses first indent level as subheading', () => {
		const result = parseAgendaImportSource('Alpha\n  Sub-Alpha\nBeta\n  Sub-Beta');
		expect(result[0]).toMatchObject({ state: 'heading', text: 'Alpha' });
		expect(result[1]).toMatchObject({ state: 'subheading', text: 'Sub-Alpha' });
		expect(result[2]).toMatchObject({ state: 'heading', text: 'Beta' });
		expect(result[3]).toMatchObject({ state: 'subheading', text: 'Sub-Beta' });
	});

	it('treats deeper than second indent level as ignore', () => {
		const result = parseAgendaImportSource('A\n  B\n    C');
		expect(result[0].state).toBe('heading');
		expect(result[1].state).toBe('subheading');
		expect(result[2].state).toBe('ignore');
	});

	it('normalises tabs to 4 spaces for indent comparison', () => {
		const result = parseAgendaImportSource('A\n\tB');
		expect(result[0].state).toBe('heading');
		expect(result[1].state).toBe('subheading');
	});
});

describe('parseAgendaImportSource - markdown', () => {
	it('# is heading, ## is subheading, H3+ is ignored', () => {
		const result = parseAgendaImportSource('# Top\n## Sub\n### Deeper', 'markdown');
		expect(result[0]).toMatchObject({ state: 'heading', text: 'Top' });
		expect(result[1]).toMatchObject({ state: 'subheading', text: 'Sub' });
		expect(result[2]).toMatchObject({ state: 'ignore', text: '### Deeper' });
	});

	it('plain lines are ignored in markdown mode', () => {
		const result = parseAgendaImportSource('# Top\nsome prose', 'markdown');
		expect(result[1]).toMatchObject({ state: 'ignore', text: 'some prose' });
	});

	it('does not misinterpret numbered lines in markdown mode', () => {
		const result = parseAgendaImportSource('1. Item', 'markdown');
		expect(result[0].state).toBe('ignore');
	});
});

// ---------------------------------------------------------------------------
// buildImportedAgenda
// ---------------------------------------------------------------------------

describe('buildImportedAgenda', () => {
	it('builds flat headings', () => {
		const result = buildImportedAgenda(
			lines({ Alpha: 'heading', Beta: 'heading', Gamma: 'heading' })
		);
		expect(result).toHaveLength(3);
		expect(result.map((p) => p.title)).toEqual(['Alpha', 'Beta', 'Gamma']);
		expect(result.every((p) => p.children.length === 0)).toBe(true);
	});

	it('attaches subheadings to the preceding heading', () => {
		const result = buildImportedAgenda(
			lines({ Alpha: 'heading', Sub1: 'subheading', Sub2: 'subheading', Beta: 'heading' })
		);
		expect(result[0].children).toEqual(['Sub1', 'Sub2']);
		expect(result[1].children).toEqual([]);
	});

	it('ignores lines with state=ignore', () => {
		const result = buildImportedAgenda(
			lines({ Alpha: 'heading', 'skip me': 'ignore', Beta: 'heading' })
		);
		expect(result).toHaveLength(2);
	});

	it('drops orphan subheadings (before first heading)', () => {
		const result = buildImportedAgenda(lines({ Orphan: 'subheading', Alpha: 'heading' }));
		expect(result).toHaveLength(1);
		expect(result[0].children).toEqual([]);
	});
});

// ---------------------------------------------------------------------------
// matchTitlePairs
// ---------------------------------------------------------------------------

describe('matchTitlePairs', () => {
	it('matches identical lists', () => {
		const pairs = matchTitlePairs(
			[{ title: 'A' }, { title: 'B' }, { title: 'C' }],
			[{ title: 'A' }, { title: 'B' }, { title: 'C' }]
		);
		expect(pairs).toHaveLength(3);
		pairs.forEach((p) => expect(p.existingIdx).toBe(p.importedIdx));
	});

	it('matches swapped items', () => {
		const pairs = matchTitlePairs(
			[{ title: 'A' }, { title: 'B' }, { title: 'C' }],
			[{ title: 'C' }, { title: 'B' }, { title: 'A' }]
		);
		expect(pairs).toHaveLength(3);
		const byImp = Object.fromEntries(pairs.map((p) => [p.importedIdx, p.existingIdx]));
		expect(byImp[0]).toBe(2);
		expect(byImp[1]).toBe(1);
		expect(byImp[2]).toBe(0);
	});

	it('is case-insensitive', () => {
		const pairs = matchTitlePairs([{ title: 'hello world' }], [{ title: 'Hello World' }]);
		expect(pairs).toHaveLength(1);
	});

	it('returns no pairs when nothing matches', () => {
		const pairs = matchTitlePairs([{ title: 'A' }], [{ title: 'B' }]);
		expect(pairs).toHaveLength(0);
	});

	it('matches each item at most once', () => {
		const pairs = matchTitlePairs(
			[{ title: 'A' }, { title: 'A' }],
			[{ title: 'A' }, { title: 'A' }]
		);
		expect(pairs).toHaveLength(2);
		expect(new Set(pairs.map((p) => p.existingIdx)).size).toBe(2);
	});

	it('picks closest existing when duplicates exist', () => {
		const pairs = matchTitlePairs(
			[{ title: 'A' }, { title: 'B' }, { title: 'A' }],
			[{ title: 'X' }, { title: 'X' }, { title: 'A' }]
		);
		const aMatch = pairs.find((p) => p.importedIdx === 2);
		expect(aMatch?.existingIdx).toBe(2);
	});
});

// ---------------------------------------------------------------------------
// computeSubDiff (within-parent)
// ---------------------------------------------------------------------------

describe('computeSubDiff', () => {
	it('empty → empty is empty', () => {
		expect(computeSubDiff([], [])).toHaveLength(0);
	});

	it('preserves existingId on matched rows', () => {
		const rows = computeSubDiff([pt('s1', 'Alpha')], ['Alpha']);
		expect(rows[0].existingId).toBe('s1');
	});

	it('rename fallback pairs by position among unmatched', () => {
		const rows = computeSubDiff([pt('s1', 'X'), pt('s2', 'Y')], ['A', 'B']);
		expect(rows[0]).toMatchObject({ op: 'renamed', existingTitle: 'X', importedTitle: 'A' });
		expect(rows[1]).toMatchObject({ op: 'renamed', existingTitle: 'Y', importedTitle: 'B' });
	});

	it('detects within-parent reorder', () => {
		// [X, Y] → [Y, X]: both change relative rank
		const rows = computeSubDiff([pt('s1', 'X'), pt('s2', 'Y')], ['Y', 'X']);
		expect(rows.find((r) => r.existingTitle === 'X')?.op).toBe('reordered');
		expect(rows.find((r) => r.existingTitle === 'Y')?.op).toBe('reordered');
	});

	it('unchanged when positions and titles are same', () => {
		const rows = computeSubDiff([pt('s1', 'A'), pt('s2', 'B')], ['A', 'B']);
		expect(rows.every((r) => r.op === 'unchanged')).toBe(true);
	});
});

// ---------------------------------------------------------------------------
// computeAgendaDiff - top-level
// ---------------------------------------------------------------------------

describe('computeAgendaDiff - all unchanged', () => {
	it('marks everything unchanged when lists are identical', () => {
		const existing = [pt('1', 'A'), pt('2', 'B'), pt('3', 'C')];
		const imported = [
			{ title: 'A', children: [] },
			{ title: 'B', children: [] },
			{ title: 'C', children: [] }
		];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff.every((r) => r.op === 'unchanged')).toBe(true);
	});
});

describe('computeAgendaDiff - requirement 1: child changes do not affect parent op', () => {
	it('parent stays unchanged when only children change', () => {
		const existing = [pt('1', 'A', [pt('s1', 'Sub1')])];
		const imported = [{ title: 'A', children: ['Sub2'] }];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff[0].op).toBe('unchanged');
		expect(diff[0].subDiff.some((s) => s.op !== 'unchanged')).toBe(true);
	});

	it('parent stays unchanged when a child is added', () => {
		const existing = [pt('1', 'A', [])];
		const imported = [{ title: 'A', children: ['NewSub'] }];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff[0].op).toBe('unchanged');
	});

	it('parent stays unchanged when a child is deleted', () => {
		const existing = [pt('1', 'A', [pt('s1', 'OldSub')])];
		const imported = [{ title: 'A', children: [] }];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff[0].op).toBe('unchanged');
	});
});

describe('computeAgendaDiff - deletions', () => {
	it('marks missing point as deleted', () => {
		const existing = [pt('1', 'A'), pt('2', 'B'), pt('3', 'C')];
		const imported = [{ title: 'A', children: [] }, { title: 'C', children: [] }];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff.find((r) => r.op === 'deleted')?.existingTitle).toBe('B');
		expect(diff.find((r) => r.existingTitle === 'A')?.op).toBe('unchanged');
		expect(diff.find((r) => r.existingTitle === 'C')?.op).toBe('unchanged');
	});

	it('places deleted rows at the end', () => {
		const existing = [pt('1', 'A'), pt('2', 'B')];
		const imported = [{ title: 'A', children: [] }];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff[diff.length - 1].op).toBe('deleted');
	});
});

describe('computeAgendaDiff - additions', () => {
	it('marks extra point as added', () => {
		const existing = [pt('1', 'A'), pt('2', 'C')];
		const imported = [
			{ title: 'A', children: [] },
			{ title: 'B', children: [] },
			{ title: 'C', children: [] }
		];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff.find((r) => r.op === 'added')?.importedTitle).toBe('B');
	});
});

describe('computeAgendaDiff - rename', () => {
	it('marks same-position different-name as renamed', () => {
		const existing = [pt('1', 'A'), pt('2', 'B'), pt('3', 'C')];
		const imported = [
			{ title: 'A', children: [] },
			{ title: 'D', children: [] },
			{ title: 'C', children: [] }
		];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff.find((r) => r.importedTitle === 'D')?.op).toBe('renamed');
		expect(diff.find((r) => r.existingTitle === 'B')?.importedTitle).toBe('D');
	});
});

describe('computeAgendaDiff - reorder', () => {
	it('detects simple two-item swap as reordered', () => {
		const existing = [pt('1', 'A'), pt('2', 'B')];
		const imported = [{ title: 'B', children: [] }, { title: 'A', children: [] }];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff.find((r) => r.existingTitle === 'A')?.op).toBe('reordered');
		expect(diff.find((r) => r.existingTitle === 'B')?.op).toBe('reordered');
	});

	it('requirement 3: circular three-way move is all reordered', () => {
		// [A, B, F] → [B, F, A]
		const existing = [pt('1', 'A'), pt('2', 'B'), pt('3', 'F')];
		const imported = [
			{ title: 'B', children: [] },
			{ title: 'F', children: [] },
			{ title: 'A', children: [] }
		];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff.find((r) => r.existingTitle === 'A')?.op).toBe('reordered');
		expect(diff.find((r) => r.existingTitle === 'B')?.op).toBe('reordered');
		expect(diff.find((r) => r.existingTitle === 'F')?.op).toBe('reordered');
	});

	it('requirement 4: children follow parent, unchanged when parent swaps', () => {
		// Before: A[], B[C,E], F[G]  →  After: B[C,E], A[], F[G]
		const existing = [
			pt('1', 'A'),
			pt('2', 'B', [pt('s1', 'C'), pt('s2', 'E')]),
			pt('3', 'F', [pt('s3', 'G')])
		];
		const imported = [
			{ title: 'B', children: ['C', 'E'] },
			{ title: 'A', children: [] },
			{ title: 'F', children: ['G'] }
		];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff.find((r) => r.existingTitle === 'A')?.op).toBe('reordered');
		expect(diff.find((r) => r.existingTitle === 'B')?.op).toBe('reordered');
		expect(diff.find((r) => r.existingTitle === 'F')?.op).toBe('unchanged');
		// C and E stay with B — no cross-parent move, both unchanged
		const bRow = diff.find((r) => r.existingTitle === 'B')!;
		expect(bRow.subDiff.every((s) => s.op === 'unchanged')).toBe(true);
	});

	it('requirement 4 mirrored: B[C,E] first in existing → A first in imported, children stay with B', () => {
		// Before: B[C,E], A[], F[G]  →  After: A[], B[C,E], F[G]  (B and A swap, C/E follow B)
		const existing = [
			pt('1', 'B', [pt('s1', 'C'), pt('s2', 'E')]),
			pt('2', 'A'),
			pt('3', 'F', [pt('s3', 'G')])
		];
		const imported = [
			{ title: 'A', children: [] },
			{ title: 'B', children: ['C', 'E'] },
			{ title: 'F', children: ['G'] }
		];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff.find((r) => r.existingTitle === 'A')?.op).toBe('reordered');
		expect(diff.find((r) => r.existingTitle === 'B')?.op).toBe('reordered');
		expect(diff.find((r) => r.existingTitle === 'F')?.op).toBe('unchanged');
		// C and E stayed with B — unchanged, no cross-parent
		const bRow = diff.find((r) => r.existingTitle === 'B')!;
		expect(bRow.subDiff.every((s) => s.op === 'unchanged')).toBe(true);
	});

	it('cross-parent move with parent swap: C/E move from B to A when A and B reorder', () => {
		// Before: B[C,E], A[], F[G]  →  After: B[], A[C,E], F[G]
		// B and A both reorder; C and E actually moved to A → newParent
		const existing = [
			pt('1', 'B', [pt('s1', 'C'), pt('s2', 'E')]),
			pt('2', 'A'),
			pt('3', 'F', [pt('s3', 'G')])
		];
		const imported = [
			{ title: 'B', children: [] },
			{ title: 'A', children: ['C', 'E'] },
			{ title: 'F', children: ['G'] }
		];
		const diff = computeAgendaDiff(existing, imported);
		// B and A didn't move — only sub-items changed (req 1: parent op unaffected)
		expect(diff.find((r) => r.existingTitle === 'B')?.op).toBe('unchanged');
		expect(diff.find((r) => r.existingTitle === 'A')?.op).toBe('unchanged');
		// B has no subDiff entries (C/E left, covered by newParent under A)
		const bRow = diff.find((r) => r.existingTitle === 'B')!;
		expect(bRow.subDiff).toHaveLength(0);
		// A's subDiff shows C and E as newParent
		const aRow = diff.find((r) => r.existingTitle === 'A')!;
		expect(aRow.subDiff).toHaveLength(2);
		expect(aRow.subDiff.every((s) => s.op === 'newParent')).toBe(true);
		expect(aRow.subDiff.find((s) => s.importedTitle === 'C')?.existingTitle).toBe('C');
		expect(aRow.subDiff.find((s) => s.importedTitle === 'E')?.existingTitle).toBe('E');
	});
});

describe('computeAgendaDiff - changing between root and sub-point', () => {
	it('moving a root point to become a sub-point of another existing point is detected as newParent', () => {
		const existing = [pt('1', 'A'), pt('2', 'B')];
		const imported = [{ title: 'A', children: ['B'] }];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff).toHaveLength(1);
		expect(diff[0].existingTitle).toBe('A');
		expect(diff[0].op).toBe('unchanged');
		expect(diff[0].subDiff).toHaveLength(1);
		expect(diff[0].subDiff[0]).toMatchObject({
			op: 'newParent',
			existingTitle: 'B',
			importedTitle: 'B'
		});
	});

	it('moving a sub-point to become a root point is detected as newRoot', () => {
		const existing = [pt('1', 'A', [pt('s1', 'B')])];
		const imported = [{ title: 'A', children: [] }, { title: 'B', children: [] }];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff).toHaveLength(2);
		const aRow = diff[0];
		const bRow = diff[1];
		expect(aRow.existingTitle).toBe('A');
		expect(aRow.op).toBe('unchanged');
		expect(aRow.subDiff).toHaveLength(0);
		expect(bRow.existingTitle).toBe('B');
		expect(bRow.op).toBe('newRoot');
		expect(bRow.subDiff).toHaveLength(0);
	});

	it('moving a sub-point to root while further children are transfered to new root level item', () => {
		const existing = [pt('1', 'A', [pt('s1', 'B'), pt('s2', 'C')])];
		const imported = [
			{ title: 'A', children: [] },
			{ title: 'B', children: ['C'] },
		];
		const diff = computeAgendaDiff(existing, imported);
		expect(diff).toHaveLength(2);
		const aRow = diff[0];
		const bRow = diff[1];
		expect(aRow.existingTitle).toBe('A');
		expect(aRow.op).toBe('unchanged');
		expect(aRow.subDiff).toHaveLength(0);
		expect(bRow.existingTitle).toBe('B');
		expect(bRow.op).toBe('newRoot');
		expect(bRow.subDiff).toHaveLength(1);
		expect(bRow.subDiff[0]).toMatchObject({
			op: 'newParent',
			existingTitle: 'C',
			importedTitle: 'C'
		});
	});
});

describe('computeAgendaDiff - empty inputs', () => {
	it('all added when existing is empty', () => {
		const diff = computeAgendaDiff([], [{ title: 'A', children: [] }, { title: 'B', children: [] }]);
		expect(diff.every((r) => r.op === 'added')).toBe(true);
	});

	it('all deleted when imported is empty', () => {
		const diff = computeAgendaDiff([pt('1', 'A'), pt('2', 'B')], []);
		expect(diff.every((r) => r.op === 'deleted')).toBe(true);
	});

	it('returns empty diff for two empty lists', () => {
		expect(computeAgendaDiff([], [])).toHaveLength(0);
	});
});

// ---------------------------------------------------------------------------
// computeAgendaDiff - sub-point handling
// ---------------------------------------------------------------------------

describe('computeAgendaDiff - same-parent sub-point ops', () => {
	it('unchanged sub-point', () => {
		const existing = [pt('1', 'A', [pt('s1', 'Sub1')])];
		const diff = computeAgendaDiff(existing, [{ title: 'A', children: ['Sub1'] }]);
		expect(diff[0].subDiff[0].op).toBe('unchanged');
	});

	it('added sub-point', () => {
		const existing = [pt('1', 'A', [])];
		const diff = computeAgendaDiff(existing, [{ title: 'A', children: ['NewSub'] }]);
		expect(diff[0].subDiff[0]).toMatchObject({ op: 'added', importedTitle: 'NewSub' });
	});

	it('deleted sub-point', () => {
		const existing = [pt('1', 'A', [pt('s1', 'OldSub')])];
		const diff = computeAgendaDiff(existing, [{ title: 'A', children: [] }]);
		expect(diff[0].subDiff[0]).toMatchObject({ op: 'deleted', existingTitle: 'OldSub' });
	});

	it('renamed sub-point', () => {
		const existing = [pt('1', 'A', [pt('s1', 'OldName')])];
		const diff = computeAgendaDiff(existing, [{ title: 'A', children: ['NewName'] }]);
		expect(diff[0].subDiff[0]).toMatchObject({
			op: 'renamed',
			existingTitle: 'OldName',
			importedTitle: 'NewName'
		});
	});

	it('within-parent reordered sub-point', () => {
		// [C, E] → [E, C]
		const existing = [pt('1', 'B', [pt('s1', 'C'), pt('s2', 'E')])];
		const diff = computeAgendaDiff(existing, [{ title: 'B', children: ['E', 'C'] }]);
		expect(diff[0].subDiff.find((s) => s.existingTitle === 'C')?.op).toBe('reordered');
		expect(diff[0].subDiff.find((s) => s.existingTitle === 'E')?.op).toBe('reordered');
	});

	it('added top-level point carries added sub-points', () => {
		const diff = computeAgendaDiff([], [{ title: 'New', children: ['Child1', 'Child2'] }]);
		expect(diff[0].op).toBe('added');
		expect(diff[0].subDiff.every((s) => s.op === 'added')).toBe(true);
	});

	it('deleted top-level point carries deleted sub-points', () => {
		const diff = computeAgendaDiff([pt('1', 'A', [pt('s1', 'Sub1'), pt('s2', 'Sub2')])], []);
		expect(diff[0].op).toBe('deleted');
		expect(diff[0].subDiff.every((s) => s.op === 'deleted')).toBe(true);
	});
});

// ---------------------------------------------------------------------------
// computeAgendaDiff - requirement 2: cross-parent sub-item moves
// ---------------------------------------------------------------------------

describe('computeAgendaDiff - requirement 2: cross-parent sub moves', () => {
	// Before: A, B[C,E], F[G]   After: A, B[C,G], F[E]   (E↔G swap)
	function buildSwapFixture() {
		const existing = [
			pt('1', 'A'),
			pt('2', 'B', [pt('s1', 'C'), pt('s2', 'E')]),
			pt('3', 'F', [pt('s3', 'G')])
		];
		const imported = [
			{ title: 'A', children: [] },
			{ title: 'B', children: ['C', 'G'] },
			{ title: 'F', children: ['E'] }
		];
		return { existing, imported };
	}

	it('parent ops are unchanged (E and G are just cross-parent moved)', () => {
		const { existing, imported } = buildSwapFixture();
		const diff = computeAgendaDiff(existing, imported);
		expect(diff.find((r) => r.existingTitle === 'B')?.op).toBe('unchanged');
		expect(diff.find((r) => r.existingTitle === 'F')?.op).toBe('unchanged');
	});

	it('C stays unchanged within B', () => {
		const { existing, imported } = buildSwapFixture();
		const diff = computeAgendaDiff(existing, imported);
		const bRow = diff.find((r) => r.existingTitle === 'B')!;
		expect(bRow.subDiff.find((s) => s.existingTitle === 'C' || s.importedTitle === 'C')?.op).toBe(
			'unchanged'
		);
	});

	it('E left B — not present in B subDiff (covered by newParent in F)', () => {
		const { existing, imported } = buildSwapFixture();
		const diff = computeAgendaDiff(existing, imported);
		const bRow = diff.find((r) => r.existingTitle === 'B')!;
		expect(bRow.subDiff.find((s) => s.existingTitle === 'E')).toBeUndefined();
	});

	it('G arriving at B shows as newParent in B subDiff', () => {
		const { existing, imported } = buildSwapFixture();
		const diff = computeAgendaDiff(existing, imported);
		const bRow = diff.find((r) => r.existingTitle === 'B')!;
		const gEntry = bRow.subDiff.find((s) => s.importedTitle === 'G');
		expect(gEntry?.op).toBe('newParent');
		expect(gEntry?.existingTitle).toBe('G');
	});

	it('G left F — not present in F subDiff (covered by newParent in B)', () => {
		const { existing, imported } = buildSwapFixture();
		const diff = computeAgendaDiff(existing, imported);
		const fRow = diff.find((r) => r.existingTitle === 'F')!;
		expect(fRow.subDiff.find((s) => s.existingTitle === 'G')).toBeUndefined();
	});

	it('E arriving at F shows as newParent in F subDiff', () => {
		const { existing, imported } = buildSwapFixture();
		const diff = computeAgendaDiff(existing, imported);
		const fRow = diff.find((r) => r.existingTitle === 'F')!;
		const eEntry = fRow.subDiff.find((s) => s.importedTitle === 'E');
		expect(eEntry?.op).toBe('newParent');
		expect(eEntry?.existingTitle).toBe('E');
	});

	it('requirement 3+2 combined: circular top move with cross-parent sub moves', () => {
		// Before: A[], B[C,E], F[G]   After: B[], F[C,E], A[G]
		// B→A pos, F→B pos, A→F pos; C and E moved B→F; G moved F→A
		const existing = [
			pt('1', 'A'),
			pt('2', 'B', [pt('s1', 'C'), pt('s2', 'E')]),
			pt('3', 'F', [pt('s3', 'G')])
		];
		const imported = [
			{ title: 'B', children: [] },
			{ title: 'F', children: ['C', 'E'] },
			{ title: 'A', children: ['G'] }
		];
		const diff = computeAgendaDiff(existing, imported);
		// All top-level reordered
		expect(diff.find((r) => r.existingTitle === 'A')?.op).toBe('reordered');
		expect(diff.find((r) => r.existingTitle === 'B')?.op).toBe('reordered');
		expect(diff.find((r) => r.existingTitle === 'F')?.op).toBe('reordered');
		// B lost C and E — not in B's subDiff (covered by newParent in F)
		const bRow = diff.find((r) => r.existingTitle === 'B')!;
		expect(bRow.subDiff).toHaveLength(0);
		// F gained C and E → newParent in F
		const fRow = diff.find((r) => r.existingTitle === 'F')!;
		expect(fRow.subDiff.filter((s) => s.op === 'newParent').map((s) => s.importedTitle)).toEqual(
			expect.arrayContaining(['C', 'E'])
		);
		// A gained G → newParent in A
		const aRow = diff.find((r) => r.existingTitle === 'A')!;
		expect(aRow.subDiff.find((s) => s.importedTitle === 'G')?.op).toBe('newParent');
	});
});
