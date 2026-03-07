---
title-en: Document Authoring Standards
title-de: Standards für Dokumenterstellung
---

# Document Authoring Standards

Use this checklist when creating or editing documentation pages.

## Who this is for

Documentation maintainers and reviewers.

## Required structure

1. Use numbered directories/files for stable page order.
2. Every directory needs both:
   - `index.en.md`
   - `index.de.md`
3. Every markdown page needs frontmatter with non-empty:
   - `title-en`
   - `title-de`

## Content standards

1. Keep English and German pages aligned in scope.
2. Write task-oriented steps for the target user.
3. Include common failure/recovery notes where relevant.
4. Keep wording consistent with current UI labels.

## Link and media standards

1. Use `/docs/...` links for internal navigation.
2. Store captures in `doc/assets/captures/`.
3. Remove outdated capture references when assets/scripts change.

## Before marking a page ready

1. Verify links resolve correctly.
2. Verify capture references exist and match UI.
3. Verify EN/DE structure parity for the directory.
4. Verify frontmatter titles are complete.

## If something goes wrong

- Broken docs links:
  Recheck `/docs/...` target paths and section filenames.
- Missing image in page:
  Confirm the file exists under `doc/assets/captures/` and the relative path is correct.
- EN/DE mismatch:
  Align structure first, then fill remaining content differences.
