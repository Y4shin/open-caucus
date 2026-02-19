# Internationalization (i18n) Plan

## Overview

Add i18n to the conference tool with the following behaviour:

- **Locale detection priority** (highest to lowest): URL path prefix → cookie → `Accept-Language` header → configured default
- **URL prefix is optional** – the middleware strips it before routing so the router never sees it. HTMX partial requests automatically replicate the prefix through the path builders.
- **User override** – a locale-switcher endpoint sets a cookie that takes priority over Accept-Language on subsequent requests.
- **Template translation** – uses [`ctxi18n`](https://github.com/invopop/ctxi18n) with YAML files per locale. Templates call `i18n.T(ctx, "key")` to look up strings; `ctx` is already implicitly available in every Templ component.

## Key design decisions

### Path builders carry locale prefix automatically

All generated path builder methods gain two extra parameters:

```go
// No path params
func (Routes) AdminDashboardGet(ctx context.Context, override string) string

// With path params
func (r *CommitteeSlugRoute) CommitteePageGet(ctx context.Context, override string) string

// WithQuery variant
func (Routes) AdminDashboardGetWithQuery(ctx context.Context, override string, q AdminDashboardGetQueryParams) string
```

`locale.PathPrefix(ctx, override)` encodes the following logic:

```
if override != ""      → "/" + override
if HadURLPrefix(ctx)   → "/" + GetLocale(ctx)   // replicate the incoming prefix
otherwise              → ""
```

Static asset paths (`HtmxMinJs()`, etc.) are **not** locale-prefixed – they stay parameterless.

### Helper methods on input structs accept ctx

Every method on a template input struct that calls a path builder gains `ctx context.Context` as its first parameter, and passes `(ctx, "")` to the builder. These methods are called from within Templ component bodies where `ctx` is the implicit request context.

### No codegen changes to routes.yaml or routes_gen.go

The locale middleware wraps the entire mux at the server level (in `cmd/serve.go`) rather than being listed in `routes.yaml`. The router never knows about locales.

## Steps

1. [Create `internal/locale` package](step1_locale_package.md)
2. [Update the code generator for locale-aware path builders](step2_codegen.md)
3. [Update `cmd/serve.go` and `internal/routes/gen.go`](step3_serve.md)
4. [Update all `.templ` helper methods to accept and propagate `ctx`](step4_templates.md)
5. [Run code generation and fix compilation](step5_generate.md)
6. [Add `ctxi18n`, translation YAML files, and wire `i18n.T()` into templates](step6_translations.md)

## Directory layout after all steps

```
internal/locale/
  locale.go        ← context keys, PathPrefix, GetLocale, HadURLPrefix
  middleware.go    ← HTTP middleware (detect, strip prefix, set context)

locales/
  en.yaml          ← English strings (the reference locale)
  de.yaml          ← German strings (example second locale)

tools/routing/
  routebuilders.go ← MODIFIED: RouteGroup gets HasLocale/LocaleAlias; template updated
  generator.go     ← MODIFIED: PathsGenData gets LocalePackage; pathsGen imports block updated

internal/routes/
  gen.go           ← MODIFIED: -locale-package flag added to go:generate line

cmd/serve.go       ← MODIFIED: mux wrapped with locale.NewMiddleware; ctxi18n.Load called

internal/templates/
  *.templ          ← MODIFIED: all path-builder helper methods gain ctx param
```
