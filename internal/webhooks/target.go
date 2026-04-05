package webhooks

import "strings"

// webhookTarget pairs a destination URL with an optional per-target
// authentication header parsed from the WEBHOOK_URLS entry.
type webhookTarget struct {
	URL         string
	HeaderName  string
	HeaderValue string
}

// parseTarget parses one WEBHOOK_URLS entry.
//
// Format: <url>[@<header-name>:<header-value>]
//
// The @ is treated as the auth delimiter only when the text after it has the
// form of a valid HTTP header name (letters/digits/hyphens/underscores,
// starting with a letter) followed by a colon. This distinguishes the auth
// suffix from the @ that appears in URL userinfo (e.g. user@host), because
// hostnames contain dots which are not allowed in our header name subset.
//
// Examples:
//
//	https://example.com/hook                        → plain URL
//	https://example.com/hook@X-Api-Key:secret       → URL + header
//	https://example.com/hook@Authorization:Bearer t → URL + header with spaces in value
//	https://user@host.example.com/hook              → plain URL (host has dots)
func parseTarget(entry string) webhookTarget {
	at := strings.LastIndex(entry, "@")
	if at == -1 {
		return webhookTarget{URL: entry}
	}

	rest := entry[at+1:]
	colon := strings.Index(rest, ":")
	if colon == -1 {
		return webhookTarget{URL: entry}
	}

	name := rest[:colon]
	if !isValidHeaderName(name) {
		return webhookTarget{URL: entry}
	}

	return webhookTarget{
		URL:         entry[:at],
		HeaderName:  name,
		HeaderValue: rest[colon+1:],
	}
}

// parseTargets parses all entries from cfg.URLs.
func parseTargets(urls []string) []webhookTarget {
	targets := make([]webhookTarget, len(urls))
	for i, u := range urls {
		targets[i] = parseTarget(u)
	}
	return targets
}

// isValidHeaderName reports whether s is a usable HTTP header field name for
// our purposes: starts with a letter, followed by letters, digits, hyphens,
// or underscores. Dots are intentionally excluded so that URL hostnames are
// never mistaken for header names.
func isValidHeaderName(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, c := range s {
		switch {
		case c >= 'A' && c <= 'Z':
		case c >= 'a' && c <= 'z':
		case (c >= '0' && c <= '9') && i > 0:
		case (c == '-' || c == '_') && i > 0:
		default:
			return false
		}
	}
	return true
}
