package services

import (
	"regexp"
	"strings"
)

var (
	newlineItemPattern   = regexp.MustCompile(`\r?\n`)
	numberedItemPattern  = regexp.MustCompile(`(?m)(?:^|\s)\d+[.)]\s+`)
	separatorPattern     = regexp.MustCompile(`[,;]`)
	leadingNumberPattern = regexp.MustCompile(`^\d+[.)]\s+`)
	// leadingConjunctionPattern strips a conjunction left at the start of an item
	// ("leite, e ovos" → "ovos", "2. e comprar leite" → "comprar leite").
	leadingConjunctionPattern = regexp.MustCompile(`(?i)^(?:e\s+(?:os|as|o|a)\s+|e\s+)`)
	// itemTrimCutset is trimmed from both ends of an item.
	itemTrimCutset = " \t.;:!?-—"
)

// SplitTodoItems splits a to-do content into ordered items. Newlines win, then
// numbered markers ("1. ", "2) "), then commas/semicolons. Each item is trimmed,
// stripped of a leading number/conjunction and of trailing punctuation; empty
// items are dropped and the original order is preserved.
func SplitTodoItems(content string) []string {
	raw := splitRawItems(content)
	items := make([]string, 0, len(raw))
	for _, candidate := range raw {
		if item := cleanItem(candidate); item != "" {
			items = append(items, item)
		}
	}
	return items
}

// splitRawItems picks the split strategy and cuts the content.
func splitRawItems(content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	if strings.ContainsAny(trimmed, "\r\n") {
		return newlineItemPattern.Split(trimmed, -1)
	}
	if numberedItemPattern.MatchString(trimmed) {
		return numberedItemPattern.Split(trimmed, -1)
	}
	return separatorPattern.Split(trimmed, -1)
}

// cleanItem normalizes a single raw item.
func cleanItem(candidate string) string {
	item := leadingNumberPattern.ReplaceAllString(strings.TrimSpace(candidate), "")
	item = leadingConjunctionPattern.ReplaceAllString(strings.TrimSpace(item), "")
	return strings.Trim(strings.TrimSpace(item), itemTrimCutset)
}