package services

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// normalizeName lowercases, strips diacritics (NFD + remove Mn marks), and
// collapses whitespace so name matching is robust to case and accents.
func normalizeName(s string) string {
	s = norm.NFD.String(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
