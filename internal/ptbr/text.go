// Package ptbr reads PT-BR text deterministically: accent-folded comparisons,
// dates, clock times and monetary amounts. No LLM, no service state — every
// function here is a pure function of (message, now), so the classification
// flows and the /data screen share one implementation of each rule.
package ptbr

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// TrimCutset is stripped from both ends of a segment by TrimSegment. Periods stay
// ("Jr." is a legitimate name ending).
const TrimCutset = `"'),;:!?-—`

// NormalizeName lowercases, strips diacritics (NFD + remove Mn marks), and
// collapses whitespace so name matching is robust to case and accents.
func NormalizeName(s string) string {
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

// TrimSegment strips the punctuation that can surround a word segment.
func TrimSegment(segment string) string {
	return strings.Trim(segment, TrimCutset)
}

// IsCapitalized reports whether s starts with an uppercase letter.
func IsCapitalized(s string) bool {
	if s == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

// StripPunctuation keeps letters and digits only, so a trailing "?" cannot leak
// into a LIKE term and miss every row.
func StripPunctuation(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}