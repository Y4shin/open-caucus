# Documentation Rewrite Workspace

This directory is the shared memory for the documentation overhaul.

## Purpose

The existing docs often describe capabilities ("what") but not procedures ("how").
This rewrite project focuses on user-facing, step-by-step guides for the core workflows in the app.

Primary goals:

1. Turn feature descriptions into practical procedures.
2. Keep technical details only when users need them to succeed.
3. Keep docs aligned with real UI behavior and current captures.
4. Track progress file-by-file so rewrites are systematic and reviewable.

## Rewrite Principles

1. Every important user task should have numbered steps.
2. Screenshots/GIFs should point at the exact action area (highlights where useful).
3. Include "Before you start" prerequisites when relevant.
4. Include "What happens next" and common failure/recovery notes.
5. Separate user-facing guidance from implementation detail.
6. If technical detail is needed, keep it in a dedicated "Technical note" block.

## How We Will Work

1. Rewrite one English file at a time.
2. Mark status in `04-file-inventory.md`.
3. Record global style/process decisions in `01-rules.md` and `05-decisions-log.md`.
4. Capture reusable user workflows in `02-core-workflows.md`.
5. Capture implementation details users may need in `03-technical-notes.md`.
6. Translate/update German in parallel once English wording is stable per file.

## Index

- Project status: [00-status.md](./00-status.md)
- Rewrite rules and style guide: [01-rules.md](./01-rules.md)
- Core user workflows (step templates): [02-core-workflows.md](./02-core-workflows.md)
- Technical notes for user-relevant internals: [03-technical-notes.md](./03-technical-notes.md)
- English file inventory and rewrite tracker: [04-file-inventory.md](./04-file-inventory.md)
- Decisions and changelog for the rewrite process: [05-decisions-log.md](./05-decisions-log.md)

## Definition of Done (Per File)

A file is considered reworked when all are true:

1. The guide contains concrete numbered steps for core tasks.
2. It includes prerequisites and expected outcomes where needed.
3. It includes troubleshooting notes for common failure points.
4. It avoids route/endpoint-level detail unless explicitly user-relevant.
5. Related screenshots/GIFs are accurate.
6. File status is updated in `04-file-inventory.md`.
