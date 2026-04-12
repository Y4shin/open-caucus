export type AgendaImportState = 'ignore' | 'heading' | 'subheading';
export type AgendaImportLine = {
	lineNo: number;
	text: string;        // extracted title (what gets saved)
	rawLine: string;     // original trimmed line (for display/highlighting)
	titleStart: number;  // character index where the extracted title starts in rawLine
	state: AgendaImportState;
};
export type AgendaImportPoint = { title: string; children: string[] };
// No 'modified': child changes do not affect the parent op (req 1).
export type SubDiffOp = 'unchanged' | 'renamed' | 'reordered' | 'renamed+reordered' | 'added' | 'deleted' | 'newParent';
export type TopDiffOp = 'unchanged' | 'renamed' | 'reordered' | 'renamed+reordered' | 'added' | 'deleted' | 'newRoot';
export type SubDiffRow = {
	op: SubDiffOp;
	existingId: string | null;
	existingTitle: string | null;
	importedTitle: string | null;
};
export type AgendaDiffRow = {
	op: TopDiffOp;
	existingId: string | null;
	existingTitle: string | null;
	importedTitle: string | null;
	subDiff: SubDiffRow[];
};

// Minimal interface so the diff functions don't depend on the generated protobuf types.
export type AgendaPointLike = {
	agendaPointId: string;
	title: string;
	subPoints: AgendaPointLike[];
};

export function parseAgendaImportSource(
	source: string,
	format: 'markdown' | 'plaintext' = 'plaintext'
): AgendaImportLine[] {
	const rawLines = source.split('\n');
	const indexed = rawLines
		.map((line, index) => ({ original: line, raw: line.trim(), lineNo: index + 1 }))
		.filter((line) => line.raw.length > 0);

	if (format === 'markdown') {
		return indexed.map(({ raw, lineNo }) => {
			const h1 = raw.match(/^(#{1}\s+)(.+)$/);
			if (h1) return { lineNo, text: h1[2].trim(), rawLine: raw, titleStart: h1[1].length, state: 'heading' as const };
			const h2 = raw.match(/^(#{2}\s+)(.+)$/);
			if (h2) return { lineNo, text: h2[2].trim(), rawLine: raw, titleStart: h2[1].length, state: 'subheading' as const };
			return { lineNo, text: raw, rawLine: raw, titleStart: 0, state: 'ignore' as const };
		});
	}

	const hasNumbers = indexed.some(({ raw }) =>
		/^(?:TOP\s*)?(\d+(?:\.\d+)?)[:.) -]/i.test(raw)
	);

	if (hasNumbers) {
		return indexed.map(({ raw, lineNo }) => {
			const match = raw.match(/^((?:TOP\s*)?\d+(?:\.\d+)?[:.) -]*\s*)(.+)$/i);
			if (match) {
				return {
					lineNo,
					text: match[2].trim(),
					rawLine: raw,
					titleStart: match[1].length,
					state: match[1].match(/\d+\.\d+/) ? ('subheading' as const) : ('heading' as const)
				};
			}
			return { lineNo, text: raw, rawLine: raw, titleStart: 0, state: 'ignore' as const };
		});
	}

	const indentOf = (line: string) => {
		const m = line.match(/^(\s+)/);
		if (!m) return 0;
		return m[1].replace(/\t/g, '    ').length;
	};

	const indents = indexed.map(({ original }) => indentOf(original));
	const uniqueIndents = [...new Set(indents)].sort((a, b) => a - b);
	const subheadingIndent = uniqueIndents.length > 1 ? uniqueIndents[1] : null;

	return indexed.map(({ raw, lineNo }, i) => {
		const indent = indents[i];
		// For indentation format, the entire trimmed line is the title (no prefix stripping).
		if (indent === uniqueIndents[0]) return { lineNo, text: raw, rawLine: raw, titleStart: 0, state: 'heading' as const };
		if (subheadingIndent !== null && indent === subheadingIndent)
			return { lineNo, text: raw, rawLine: raw, titleStart: 0, state: 'subheading' as const };
		return { lineNo, text: raw, rawLine: raw, titleStart: 0, state: 'ignore' as const };
	});
}

export function buildImportedAgenda(lines: AgendaImportLine[]): AgendaImportPoint[] {
	const points: AgendaImportPoint[] = [];
	let currentTop: AgendaImportPoint | null = null;
	for (const line of lines) {
		if (line.state === 'heading') {
			currentTop = { title: line.text, children: [] };
			points.push(currentTop);
			continue;
		}
		if (line.state === 'subheading' && currentTop) {
			currentTop.children.push(line.text);
		}
	}
	return points;
}

export function matchTitlePairs(
	existing: { title: string }[],
	imported: { title: string }[]
): Array<{ existingIdx: number; importedIdx: number }> {
	const byTitle = new Map<string, number[]>();
	existing.forEach((e, i) => {
		const key = e.title.trim().toLowerCase();
		if (!byTitle.has(key)) byTitle.set(key, []);
		byTitle.get(key)!.push(i);
	});
	const used = new Set<number>();
	const pairs: Array<{ existingIdx: number; importedIdx: number }> = [];
	imported.forEach((imp, impIdx) => {
		const candidates = (byTitle.get(imp.title.trim().toLowerCase()) ?? []).filter(
			(i) => !used.has(i)
		);
		if (candidates.length === 0) return;
		const best = candidates.reduce((a, b) =>
			Math.abs(a - impIdx) <= Math.abs(b - impIdx) ? a : b
		);
		used.add(best);
		pairs.push({ existingIdx: best, importedIdx: impIdx });
	});
	return pairs;
}

// computeSubDiff: pure within-parent diff, used for isolated testing.
// computeAgendaDiff uses global sub matching instead of calling this directly.
export function computeSubDiff(
	existingSubs: AgendaPointLike[],
	importedChildren: string[]
): SubDiffRow[] {
	const titlePairs = matchTitlePairs(
		existingSubs,
		importedChildren.map((c) => ({ title: c }))
	);
	const matchedEx = new Set(titlePairs.map((p) => p.existingIdx));
	const matchedImp = new Set(titlePairs.map((p) => p.importedIdx));
	const unmatchedEx = existingSubs.map((s, i) => ({ s, i })).filter(({ i }) => !matchedEx.has(i));
	const unmatchedImp = importedChildren.map((c, i) => ({ c, i })).filter(({ i }) => !matchedImp.has(i));
	const renames: Array<{ existingIdx: number; importedIdx: number }> = [];
	const minLen = Math.min(unmatchedEx.length, unmatchedImp.length);
	for (let k = 0; k < minLen; k++)
		renames.push({ existingIdx: unmatchedEx[k].i, importedIdx: unmatchedImp[k].i });
	const deletedIdxs = unmatchedEx.slice(minLen).map((x) => x.i);
	const allPairs = [...titlePairs, ...renames];
	const impToEx = new Map(allPairs.map((p) => [p.importedIdx, p.existingIdx]));

	// Relative-rank reorder detection within this parent
	const byEx = [...allPairs].sort((a, b) => a.existingIdx - b.existingIdx);
	const byImp = [...allPairs].sort((a, b) => a.importedIdx - b.importedIdx);
	const exRank = new Map(byEx.map((p, r) => [p.existingIdx, r]));
	const impRank = new Map(byImp.map((p, r) => [p.importedIdx, r]));

	const rows: SubDiffRow[] = [];
	importedChildren.forEach((child, impIdx) => {
		const exIdx = impToEx.get(impIdx);
		if (exIdx === undefined) {
			rows.push({ op: 'added', existingId: null, existingTitle: null, importedTitle: child });
			return;
		}
		const ex = existingSubs[exIdx];
		const titleChanged = ex.title !== child;
		const posChanged = exRank.get(exIdx) !== impRank.get(impIdx);
		let op: SubDiffOp;
		if (!titleChanged && !posChanged) op = 'unchanged';
		else if (titleChanged && posChanged) op = 'renamed+reordered';
		else if (titleChanged) op = 'renamed';
		else op = 'reordered';
		rows.push({ op, existingId: ex.agendaPointId, existingTitle: ex.title, importedTitle: child });
	});
	deletedIdxs.forEach((i) => {
		const ex = existingSubs[i];
		rows.push({ op: 'deleted', existingId: ex.agendaPointId, existingTitle: ex.title, importedTitle: null });
	});
	return rows;
}

export function computeAgendaDiff(
	existing: AgendaPointLike[],
	imported: AgendaImportPoint[]
): AgendaDiffRow[] {
	// ── Step 1: Top-level title matching ───────────────────────────────────
	const topPairs = matchTitlePairs(existing, imported);
	const matchedExTop = new Set(topPairs.map((p) => p.existingIdx));
	const matchedImpTop = new Set(topPairs.map((p) => p.importedIdx));
	const unmatchedExTopList = existing.map((e, i) => ({ e, i })).filter(({ i }) => !matchedExTop.has(i));
	const unmatchedImpTopList = imported.map((p, i) => ({ p, i })).filter(({ i }) => !matchedImpTop.has(i));

	// ── Step 2: Collect all sub-items globally ─────────────────────────────
	type ExSubInfo = { item: AgendaPointLike; parentExIdx: number; posWithinParent: number };
	type ImpSubInfo = { title: string; parentImpIdx: number; posWithinParent: number };

	const allExSubs: ExSubInfo[] = [];
	existing.forEach((e, eIdx) =>
		e.subPoints.forEach((s, sPos) =>
			allExSubs.push({ item: s, parentExIdx: eIdx, posWithinParent: sPos })
		)
	);
	const allImpSubs: ImpSubInfo[] = [];
	imported.forEach((p, pIdx) =>
		p.children.forEach((c, cPos) =>
			allImpSubs.push({ title: c, parentImpIdx: pIdx, posWithinParent: cPos })
		)
	);

	const exSubLookup = new Map<string, number>();
	allExSubs.forEach((s, i) => exSubLookup.set(`${s.parentExIdx}:${s.posWithinParent}`, i));
	const impSubLookup = new Map<string, number>();
	allImpSubs.forEach((s, i) => impSubLookup.set(`${s.parentImpIdx}:${s.posWithinParent}`, i));

	// ── Step 3: Global sub-item title matching ─────────────────────────────
	const globalSubPairs = matchTitlePairs(
		allExSubs.map((s) => ({ title: s.item.title })),
		allImpSubs.map((s) => ({ title: s.title }))
	);
	const matchedExSubGlobal = new Set(globalSubPairs.map((p) => p.existingIdx));
	const matchedImpSubGlobal = new Set(globalSubPairs.map((p) => p.importedIdx));

	// ── Step 4: Cross-level matching ───────────────────────────────────────
	// 4A: Unmatched existing top-level items → unmatched imported sub-items (root → sub)
	const unmatchedImpSubIdxs = allImpSubs.map((_, i) => i).filter((i) => !matchedImpSubGlobal.has(i));
	const crossARaw = matchTitlePairs(
		unmatchedExTopList.map(({ e }) => ({ title: e.title })),
		unmatchedImpSubIdxs.map((i) => ({ title: allImpSubs[i].title }))
	);
	const rootToSubMatches = crossARaw.map((p) => ({
		exTopIdx: unmatchedExTopList[p.existingIdx].i,
		impSubGlobalIdx: unmatchedImpSubIdxs[p.importedIdx]
	}));
	const rootToSubExTopSet = new Set(rootToSubMatches.map((m) => m.exTopIdx));
	const rootToSubImpSubSet = new Set(rootToSubMatches.map((m) => m.impSubGlobalIdx));
	const rootToSubImpSubToExTop = new Map(rootToSubMatches.map((m) => [m.impSubGlobalIdx, m.exTopIdx]));

	// 4B: Unmatched existing sub-items → unmatched imported top-level items (sub → root)
	const unmatchedExSubIdxs = allExSubs.map((_, i) => i).filter((i) => !matchedExSubGlobal.has(i));
	const crossBRaw = matchTitlePairs(
		unmatchedExSubIdxs.map((i) => ({ title: allExSubs[i].item.title })),
		unmatchedImpTopList.map(({ p }) => ({ title: p.title }))
	);
	const subToRootMatches = crossBRaw.map((p) => ({
		exSubGlobalIdx: unmatchedExSubIdxs[p.existingIdx],
		impTopIdx: unmatchedImpTopList[p.importedIdx].i
	}));
	const subToRootExSubSet = new Set(subToRootMatches.map((m) => m.exSubGlobalIdx));
	const subToRootImpTopSet = new Set(subToRootMatches.map((m) => m.impTopIdx));
	const subToRootImpTopToExSub = new Map(subToRootMatches.map((m) => [m.impTopIdx, m.exSubGlobalIdx]));

	// ── Step 5: Top-level positional renames (excluding cross-level matched) ─
	const truelyUnmatchedExTops = unmatchedExTopList.filter(({ i }) => !rootToSubExTopSet.has(i));
	const truelyUnmatchedImpTops = unmatchedImpTopList.filter(({ i }) => !subToRootImpTopSet.has(i));
	const topRenames: Array<{ existingIdx: number; importedIdx: number }> = [];
	const minTop = Math.min(truelyUnmatchedExTops.length, truelyUnmatchedImpTops.length);
	for (let k = 0; k < minTop; k++)
		topRenames.push({ existingIdx: truelyUnmatchedExTops[k].i, importedIdx: truelyUnmatchedImpTops[k].i });
	const deletedTopIdxs = truelyUnmatchedExTops.slice(minTop).map((x) => x.i);
	const allTopPairs = [...topPairs, ...topRenames];
	const impToExTop = new Map(allTopPairs.map((p) => [p.importedIdx, p.existingIdx]));
	const byExTop = [...allTopPairs].sort((a, b) => a.existingIdx - b.existingIdx);
	const byImpTop = [...allTopPairs].sort((a, b) => a.importedIdx - b.importedIdx);
	const exTopRank = new Map(byExTop.map((p, r) => [p.existingIdx, r]));
	const impTopRank = new Map(byImpTop.map((p, r) => [p.importedIdx, r]));

	// ── Step 6: Within-parent sub-item positional renames ─────────────────
	// Only pair unmatched subs, excluding cross-level matched.
	const unmatchedExSubByParent = new Map<number, number[]>();
	allExSubs.forEach((s, i) => {
		if (!matchedExSubGlobal.has(i) && !subToRootExSubSet.has(i)) {
			if (!unmatchedExSubByParent.has(s.parentExIdx)) unmatchedExSubByParent.set(s.parentExIdx, []);
			unmatchedExSubByParent.get(s.parentExIdx)!.push(i);
		}
	});
	const unmatchedImpSubByParent = new Map<number, number[]>();
	allImpSubs.forEach((s, i) => {
		if (!matchedImpSubGlobal.has(i) && !rootToSubImpSubSet.has(i)) {
			if (!unmatchedImpSubByParent.has(s.parentImpIdx)) unmatchedImpSubByParent.set(s.parentImpIdx, []);
			unmatchedImpSubByParent.get(s.parentImpIdx)!.push(i);
		}
	});
	const subRenames: Array<{ exSubIdx: number; impSubIdx: number }> = [];
	allTopPairs.forEach(({ existingIdx: exTopIdx, importedIdx: impTopIdx }) => {
		const unmEx = unmatchedExSubByParent.get(exTopIdx) ?? [];
		const unmImp = unmatchedImpSubByParent.get(impTopIdx) ?? [];
		const minLen = Math.min(unmEx.length, unmImp.length);
		for (let k = 0; k < minLen; k++)
			subRenames.push({ exSubIdx: unmEx[k], impSubIdx: unmImp[k] });
	});

	// Complete bidirectional sub mapping (sub-to-sub only)
	const impSubToExSub = new Map<number, number>([
		...globalSubPairs.map((p) => [p.importedIdx, p.existingIdx] as [number, number]),
		...subRenames.map((r) => [r.impSubIdx, r.exSubIdx] as [number, number])
	]);
	const exSubToImpSub = new Map<number, number>(
		[...impSubToExSub.entries()].map(([imp, ex]) => [ex, imp])
	);
	// All ex subs that are accounted for (either matched to an imp sub or promoted to root)
	const matchedExSubAll = new Set([...impSubToExSub.values(), ...subToRootExSubSet]);

	// ── Step 7: Within-parent relative-rank reorder detection ─────────────
	const exSubRankMap = new Map<number, number>();
	const impSubRankMap = new Map<number, number>();
	allTopPairs.forEach(({ existingIdx: exTopIdx, importedIdx: impTopIdx }) => {
		const withinPairs: Array<{ exSubIdx: number; impSubIdx: number }> = [];
		for (const [impSubIdx, exSubIdx] of impSubToExSub.entries()) {
			if (
				allExSubs[exSubIdx].parentExIdx === exTopIdx &&
				allImpSubs[impSubIdx].parentImpIdx === impTopIdx
			) {
				withinPairs.push({ exSubIdx, impSubIdx });
			}
		}
		if (withinPairs.length === 0) return;
		[...withinPairs]
			.sort((a, b) => allExSubs[a.exSubIdx].posWithinParent - allExSubs[b.exSubIdx].posWithinParent)
			.forEach((p, r) => exSubRankMap.set(p.exSubIdx, r));
		[...withinPairs]
			.sort((a, b) => allImpSubs[a.impSubIdx].posWithinParent - allImpSubs[b.impSubIdx].posWithinParent)
			.forEach((p, r) => impSubRankMap.set(p.impSubIdx, r));
	});

	// ── Step 8: Build result rows (in imported order) ──────────────────────
	const rows: AgendaDiffRow[] = [];

	// Helper: build subDiff for an imported item's children from the imp-side perspective.
	// Used for both standard rows and newRoot rows.
	// For newRoot rows, any existing sub matched is always cross-parent → always 'newParent'.
	function buildRightSideSubDiff(
		impIdx: number,
		exIdx: number | null // null = newRoot (no "owning" ex top)
	): SubDiffRow[] {
		const subDiff: SubDiffRow[] = [];
		imported[impIdx].children.forEach((child, cPos) => {
			const impSubGlobalIdx = impSubLookup.get(`${impIdx}:${cPos}`)!;
			const exSubGlobalIdx = impSubToExSub.get(impSubGlobalIdx);

			if (exSubGlobalIdx === undefined) {
				if (rootToSubImpSubSet.has(impSubGlobalIdx)) {
					// This imp sub was a former root-level item (root→sub)
					const exTopIdx = rootToSubImpSubToExTop.get(impSubGlobalIdx)!;
					const exTop = existing[exTopIdx];
					subDiff.push({ op: 'newParent', existingId: exTop.agendaPointId, existingTitle: exTop.title, importedTitle: child });
				} else {
					subDiff.push({ op: 'added', existingId: null, existingTitle: null, importedTitle: child });
				}
				return;
			}

			const exSub = allExSubs[exSubGlobalIdx];

			// Cross-parent: came from a different existing parent, or this is a newRoot row
			if (exIdx === null || exSub.parentExIdx !== exIdx) {
				subDiff.push({ op: 'newParent', existingId: exSub.item.agendaPointId, existingTitle: exSub.item.title, importedTitle: child });
				return;
			}

			// Same parent — detect title and position changes
			const titleCh = exSub.item.title !== child;
			const posCh = exSubRankMap.get(exSubGlobalIdx) !== impSubRankMap.get(impSubGlobalIdx);
			let subOp: SubDiffOp;
			if (!titleCh && !posCh) subOp = 'unchanged';
			else if (titleCh && posCh) subOp = 'renamed+reordered';
			else if (titleCh) subOp = 'renamed';
			else subOp = 'reordered';
			subDiff.push({ op: subOp, existingId: exSub.item.agendaPointId, existingTitle: exSub.item.title, importedTitle: child });
		});
		return subDiff;
	}

	imported.forEach((imp, impIdx) => {
		const exIdx = impToExTop.get(impIdx);

		if (exIdx !== undefined) {
			// ── Standard top-level pair ──────────────────────────────────
			const ex = existing[exIdx];
			const titleChanged = ex.title !== imp.title;
			const posChanged = exTopRank.get(exIdx) !== impTopRank.get(impIdx);
			let op: TopDiffOp;
			if (!titleChanged && !posChanged) op = 'unchanged';
			else if (titleChanged && posChanged) op = 'renamed+reordered';
			else if (titleChanged) op = 'renamed';
			else op = 'reordered';

			const subDiff = buildRightSideSubDiff(impIdx, exIdx);

			// Left-side: existing children that left this parent
			ex.subPoints.forEach((exSub, sPos) => {
				const exSubGlobalIdx = exSubLookup.get(`${exIdx}:${sPos}`)!;
				if (!matchedExSubAll.has(exSubGlobalIdx)) {
					subDiff.push({ op: 'deleted', existingId: exSub.agendaPointId, existingTitle: exSub.title, importedTitle: null });
					return;
				}
				if (subToRootExSubSet.has(exSubGlobalIdx)) return; // promoted to root, handled by newRoot row
				const impSubGlobalIdx = exSubToImpSub.get(exSubGlobalIdx)!;
				if (allImpSubs[impSubGlobalIdx].parentImpIdx !== impIdx) {
					// Cross-parent: new parent's right-side already emits 'newParent' — nothing here
				}
			});

			rows.push({ op, existingId: ex.agendaPointId, existingTitle: ex.title, importedTitle: imp.title, subDiff });
		} else if (subToRootImpTopSet.has(impIdx)) {
			// ── Sub → Root: this imported top was a sub-point in existing ─
			const exSubGlobalIdx = subToRootImpTopToExSub.get(impIdx)!;
			const exSub = allExSubs[exSubGlobalIdx];
			const subDiff = buildRightSideSubDiff(impIdx, null); // null = newRoot context
			rows.push({
				op: 'newRoot',
				existingId: exSub.item.agendaPointId,
				existingTitle: exSub.item.title,
				importedTitle: imp.title,
				subDiff
			});
		} else {
			// ── Truly new top-level item ──────────────────────────────────
			rows.push({
				op: 'added',
				existingId: null,
				existingTitle: null,
				importedTitle: imp.title,
				subDiff: imp.children.map((c) => ({
					op: 'added' as SubDiffOp,
					existingId: null,
					existingTitle: null,
					importedTitle: c
				}))
			});
		}
	});

	// Deleted top-level rows (with all their children)
	deletedTopIdxs.forEach((exIdx) => {
		const ex = existing[exIdx];
		rows.push({
			op: 'deleted',
			existingId: ex.agendaPointId,
			existingTitle: ex.title,
			importedTitle: null,
			subDiff: ex.subPoints.map((s) => ({
				op: 'deleted' as SubDiffOp,
				existingId: s.agendaPointId,
				existingTitle: s.title,
				importedTitle: null
			}))
		});
	});

	return rows;
}
