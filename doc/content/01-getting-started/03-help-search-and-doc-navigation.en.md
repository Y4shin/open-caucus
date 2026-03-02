---
title-en: Help Search and Doc Navigation
title-de: Hilfe-Suche und Dokumentnavigation
---

# Help Search and Doc Navigation

## Help panel behavior

The help element supports:

- full-text search via `/docs/search`
- a tree-based navigation with numbered ordering
- breadcrumb/path display derived from directory index + page title frontmatter

## Document structure requirements

- Every directory must provide `index.{en,de}.md`.
- Every page must include frontmatter keys `title-en` and `title-de`.
- Missing titles are a startup error and block application boot.

## Authoring references

- [Capture Guide](/docs/guides/01-capture)
- [Document Authoring Standards](/docs/guides/02-document-authoring-standards)
