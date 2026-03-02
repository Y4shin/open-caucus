---
title-en: Document Authoring Standards
title-de: Standards für Dokumenterstellung
---

# Document Authoring Standards

## Mandatory structure

- Use numbered directories/files for deterministic tree order.
- Every directory requires `index.en.md` and `index.de.md`.
- Every markdown file requires frontmatter with non-empty `title-en` and `title-de`.

## Content standards

- Keep EN and DE pages at parity in scope and maintenance level.
- Prefer route-accurate instructions using concrete endpoints.
- Document expected failures (forbidden access, validation constraints).

## Link and media standards

- Prefer `/docs/...` links for internal navigation.
- Store capture media in `doc/assets/captures/`.
- Remove stale capture references when scripts/assets are removed.
