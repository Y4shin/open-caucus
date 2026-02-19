# Step 5 – Regenerate and fix compilation

Run these commands in order.

---

## 1. Regenerate the routes and path builders

```bash
task generate:routes
# or directly:
go generate ./internal/routes/...
```

This overwrites `internal/routes/paths/paths_gen.go` with locale-aware path builder signatures:

```go
// Example of what the generated file will look like:
func (Routes) AdminDashboardGet(ctx context.Context, override string) string {
    return locale.PathPrefix(ctx, override) + "/admin"
}

func (Routes) AdminDashboardGetWithQuery(ctx context.Context, override string, q AdminDashboardGetQueryParams) string {
    path := locale.PathPrefix(ctx, override) + "/admin"
    // ... query string building
}

func (r *CommitteeSlugRoute) CommitteePageGet(ctx context.Context, override string) string {
    return locale.PathPrefix(ctx, override) + "/committee/" + r.Slug
}
```

---

## 2. Fix compile errors in `.templ` files

After step 5, `go build ./...` will report errors for every call site of a path builder that hasn't been updated yet. Work through the error list from [step 4](step4_templates.md).

---

## 3. Regenerate templates

After all `.templ` files are updated:

```bash
task generate:templates
# or:
go generate ./internal/templates/...
```

This regenerates all `*_templ.go` files.

---

## 4. Run all checks

```bash
task ci
```

This runs format, vet, lint, and unit tests. E2E tests can be run separately with `task test:e2e` once the app starts correctly.

---

## Verification

1. Start the app: `task dev`
2. Open `http://localhost:8080/` – English UI (no cookie, browser Accept-Language likely "en")
3. Open `http://localhost:8080/de/` – German UI (once translations are populated in step 6)
4. POST to `/locale` with `lang=de` – should set the `locale=de` cookie and redirect; subsequent requests without a URL prefix will use German
5. POST to `/locale` with `lang=en` – reverts to English cookie

---

## Common pitfalls

- **`context.Background()` in tests** – unit tests that construct input structs and call helper methods directly will need to pass `context.Background()` as ctx.
- **E2E tests** – the E2E tests navigate using full URLs. They continue to work unchanged since the locale middleware handles URL prefixes transparently. If a test explicitly asserts a URL path (e.g., checking `page.URL()`), it will still see the clean path (the middleware rewrites before routing).
- **SSE routes** – SSE handlers write directly to the ResponseWriter and don't use path builders, so they need no changes.
- **`ServeBlobDownload`** – this is a raw handler (no template). It also doesn't use path builders. However the *link to* a blob download in templates (`BlobDownloadGetStr()`) does use a path builder and needs updating per step 4.
