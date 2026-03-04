# Technical Notes (User-Relevant Internals)

These notes capture implementation details that affect user-facing docs.

## Agenda Import Parsing

Source: `internal/handlers/agenda_import.go`

### Input Limits

1. Empty input is rejected.
2. Oversized input is rejected (`agendaImportMaxBytes`).

### Parsing Strategy

1. Non-empty lines are extracted first.
2. Parser tries Markdown heading detection first.
3. If no Markdown headings are found, parser falls back to plain-text detection.

### Markdown Rules

1. Supports ATX headings (`#` through `######`).
2. Ignores headings inside fenced code blocks.
3. Default mapping:
   - `#` -> heading
   - `##` -> subheading
4. Special case:
   - If exactly one H1 exists and there are at least two H2/H3 lines, then:
   - `##` lines become headings
   - `###` lines become subheadings

### Plain Text Rules

1. Attempts numbered-line parsing first (e.g. `1.`, `1.1`, `2)`).
2. Requires continuous numbering consistency to use numbering mode.
3. If numbering is not valid, uses indentation depth:
   - minimum indent -> heading
   - deeper indent -> subheading
4. If no indentation distinction exists, all lines become headings.

### Validation and Application Behavior

1. A subheading without a prior heading fails validation.
2. If all lines are ignored/invalid after correction, apply is rejected.
3. Apply uses a diff process with operations like insert/delete/move/rename/unchanged.
4. If data changed between diff and apply, stale-diff warning can be produced.

## Receipt Vault Behavior

Source: `internal/templates/votes.templ`

1. Receipts are stored in browser IndexedDB (`conference_tool_receipts`).
2. The vault page loads local receipts and verifies them via API endpoints.
3. Clearing receipts only clears local browser storage.
4. Verification results depend on currently available backend verification endpoints.

Doc implication: user guides should clarify that receipts are browser-local and can be lost when storage is cleared.

## Captures and Highlighting

Sources: `tools/docscapture/scripts/app.go`, `tools/docscapture/capture.go`

1. Capture scripts can apply red highlight outlines to important controls.
2. GIF scripts now use slowed interactions and a visible demo cursor.
3. Full regeneration command: `task docs:capture:all`.

Doc implication: when rewriting steps, align screenshot/GIF references with highlighted target controls.
