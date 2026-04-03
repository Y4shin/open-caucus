package docs

import (
	"context"
	"net/http"
	"strings"

	"github.com/Y4shin/open-caucus/internal/locale"
)

const themeCookieName = "conference-tool-theme"

func VariantFromRequest(ctx context.Context, r *http.Request) VariantContext {
	language := DefaultLocale
	if l, ok := locale.GetLocale(ctx); ok {
		n := normalizeLocale(l)
		if n != "" {
			language = n
		}
	}

	theme := ThemeLight
	if r != nil {
		if cookie, err := r.Cookie(themeCookieName); err == nil {
			parsed := normalizeTheme(cookie.Value)
			theme = parsed
		}
	}

	device := DeviceDesktop
	if r != nil {
		ua := strings.ToLower(r.UserAgent())
		if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
			device = DeviceMobile
		}
	}

	return VariantContext{
		Language: language,
		Theme:    theme,
		Device:   device,
	}
}
