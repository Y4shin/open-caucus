//go:build e2e

package e2e_test

import "regexp"

var allWhitespaceRe = regexp.MustCompile(`\s+`)

func compactText(raw string) string {
	return allWhitespaceRe.ReplaceAllString(raw, "")
}
