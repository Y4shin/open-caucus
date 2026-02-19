# Step 3 – Wire up locale middleware and update go:generate

---

## `internal/routes/gen.go`

Add `-locale-package` to the existing `go:generate` directive:

```go
//go:generate go run ../../tools/routing/cmd/route-codegen/main.go -config ../../routes.yaml -output routes_gen.go -package routes -paths-output paths/paths_gen.go -paths-package paths -static-dir static -locale-package github.com/Y4shin/conference-tool/internal/locale
```

---

## `cmd/serve.go`

Three additions:

1. Embed and load translation files at startup.
2. Wrap the mux with `locale.NewMiddleware`.
3. (Optional) Register a locale-switcher endpoint.

```go
import (
    // existing imports ...
    "embed"

    "github.com/Y4shin/conference-tool/internal/locale"
    "github.com/invopop/ctxi18n"
)

//go:embed locales
var localeFS embed.FS
```

In `RunE`, after building `mux := router.RegisterRoutes()`:

```go
// Load translations (must happen before any request is served).
if err := ctxi18n.Load(localeFS); err != nil {
    return fmt.Errorf("failed to load translations: %w", err)
}

// Wrap the router with locale detection middleware.
handler := locale.NewMiddleware(mux, locale.Config{
    Default:   "en",
    Supported: []string{"en", "de"},
})

// (Optional) locale-switcher: POST /locale sets the "locale" cookie and redirects.
mux.HandleFunc("POST /locale", func(w http.ResponseWriter, r *http.Request) {
    lang := r.FormValue("lang")
    supported := map[string]bool{"en": true, "de": true}
    if !supported[lang] {
        http.Error(w, "unsupported locale", http.StatusBadRequest)
        return
    }
    http.SetCookie(w, &http.Cookie{
        Name:     "locale",
        Value:    lang,
        Path:     "/",
        MaxAge:   365 * 24 * 60 * 60,
        SameSite: http.SameSiteLaxMode,
    })
    ref := r.Header.Get("Referer")
    if ref == "" {
        ref = "/"
    }
    http.Redirect(w, r, ref, http.StatusSeeOther)
})

return http.ListenAndServe(addr, handler) // was: mux
```

> **Note on locale switcher:** The `POST /locale` handler is registered on `mux` (before wrapping), so the locale middleware still runs for it. The redirect destination will be the referer, preserving any locale prefix the user came from.

---

## `locales/` directory

Create `locales/en.yaml` as a minimal skeleton (see [step 6](step6_translations.md) for the full string catalogue):

```yaml
# locales/en.yaml
login:
  title: "Conference Tool Login"
  committee_label: "Committee:"
  username_label: "Username:"
  password_label: "Password:"
  button: "Login"
```

```yaml
# locales/de.yaml
login:
  title: "Konferenztool-Anmeldung"
  committee_label: "Ausschuss:"
  username_label: "Benutzername:"
  password_label: "Passwort:"
  button: "Anmelden"
```
