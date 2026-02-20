package locale

import "context"

type contextKey int

const (
	localeKey contextKey = iota
	hadPrefixKey
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
