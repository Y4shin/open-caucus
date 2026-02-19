# Step 1 – Create `internal/locale` package

Create two new files.

---

## `internal/locale/locale.go`

```go
package locale

import "context"

type contextKey int

const (
	localeKey    contextKey = iota
	hadPrefixKey contextKey = iota
)

// PathPrefix returns the locale path prefix to prepend to a generated URL.
//
// Priority:
//  1. If override is non-empty, return "/" + override.
//  2. If the current request arrived with a locale URL prefix (as set by the
//     middleware), replicate it by returning "/" + locale from context.
//  3. Otherwise return "" (no prefix).
func PathPrefix(ctx context.Context, override string) string {
	if override != "" {
		return "/" + override
	}
	if HadURLPrefix(ctx) {
		if l, ok := GetLocale(ctx); ok {
			return "/" + l
		}
	}
	return ""
}

func WithLocale(ctx context.Context, l string) context.Context {
	return context.WithValue(ctx, localeKey, l)
}

func WithHadURLPrefix(ctx context.Context, had bool) context.Context {
	return context.WithValue(ctx, hadPrefixKey, had)
}

func GetLocale(ctx context.Context) (string, bool) {
	l, ok := ctx.Value(localeKey).(string)
	return l, ok
}

func HadURLPrefix(ctx context.Context) bool {
	had, _ := ctx.Value(hadPrefixKey).(bool)
	return had
}
```

---

## `internal/locale/middleware.go`

```go
package locale

import (
	"net/http"
	"strings"

	"github.com/invopop/ctxi18n"
)

// Config holds locale middleware settings.
type Config struct {
	Default   string   // e.g. "en"
	Supported []string // e.g. []string{"en", "de"}
}

// NewMiddleware returns an HTTP middleware that:
//  1. Checks for a locale URL prefix (e.g. /de/...), strips it, and records hadPrefix=true.
//  2. Falls back to the "locale" cookie.
//  3. Falls back to the Accept-Language header (first matching supported tag).
//  4. Falls back to cfg.Default.
//
// It stores the resolved locale in ctx via WithLocale / WithHadURLPrefix and also
// calls ctxi18n.WithLocale so that i18n.T(ctx, key) works in Templ templates.
func NewMiddleware(next http.Handler, cfg Config) http.Handler {
	supported := make(map[string]bool, len(cfg.Supported))
	for _, l := range cfg.Supported {
		supported[l] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loc, hadPrefix, cleanPath := detect(r, cfg.Default, supported)
		ctx := r.Context()
		ctx = WithLocale(ctx, loc)
		ctx = WithHadURLPrefix(ctx, hadPrefix)
		ctx = ctxi18n.WithLocale(ctx, loc)
		if hadPrefix {
			r2 := r.Clone(ctx)
			r2.URL.Path = cleanPath
			next.ServeHTTP(w, r2)
		} else {
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}

// detect resolves the locale for the request.
// Returns the locale tag, whether a URL prefix was found, and the cleaned path.
func detect(r *http.Request, defaultLocale string, supported map[string]bool) (string, bool, string) {
	// 1. URL prefix
	for l := range supported {
		prefix := "/" + l
		if r.URL.Path == prefix || strings.HasPrefix(r.URL.Path, prefix+"/") {
			clean := strings.TrimPrefix(r.URL.Path, prefix)
			if clean == "" {
				clean = "/"
			}
			return l, true, clean
		}
	}

	// 2. Cookie
	if cookie, err := r.Cookie("locale"); err == nil && supported[cookie.Value] {
		return cookie.Value, false, r.URL.Path
	}

	// 3. Accept-Language header
	for _, tag := range parseAcceptLanguage(r.Header.Get("Accept-Language")) {
		if supported[tag] {
			return tag, false, r.URL.Path
		}
	}

	return defaultLocale, false, r.URL.Path
}

// parseAcceptLanguage returns language tags in header order, normalised to
// lowercase base tags (e.g. "de-DE" → "de"). Quality values are ignored since
// browsers already order the list by preference.
func parseAcceptLanguage(header string) []string {
	if header == "" {
		return nil
	}
	var tags []string
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if idx := strings.IndexByte(part, ';'); idx >= 0 {
			part = strings.TrimSpace(part[:idx])
		}
		// Strip region code: "de-DE" → "de"
		if idx := strings.IndexAny(part, "-_"); idx >= 0 {
			part = part[:idx]
		}
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" {
			tags = append(tags, part)
		}
	}
	return tags
}
```

> **Note:** `ctxi18n.WithLocale` expects a `language.Tag` in some versions. Check the actual signature when adding the dependency. If it takes a `language.Tag`, use `language.Make(loc)` from `golang.org/x/text/language`.

---

## Dependency

```
go get github.com/invopop/ctxi18n
```
