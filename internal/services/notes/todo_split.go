package notes

import (
	"regexp"
	"strings"
)

var (
	newlineItemPattern  = regexp.MustCompile(`\r?\n`)
	numberedItemPattern = regexp.MustCompile(`(?m)(?:^|\s)\d+[.)]\s+`)
	separatorPattern    = regexp.MustCompile(`[,;]`)
	// inlineItemPattern also splits on the "e" conjunction, used only when the text
	// already proved to be a list ("arroz, feijão, alho e 2 cenouras").
	inlineItemPattern = regexp.MustCompile(`(?i)[,;]|\s+e\s+`)
	// leadingLabelPattern matches a marker at the very start ("Comprar: arroz, feijão").
	leadingLabelPattern  = regexp.MustCompile(`^[^\n:]{1,40}:[ \t]*`)
	leadingNumberPattern = regexp.MustCompile(`^\d+[.)]\s+`)
	// leadingConjunctionPattern strips a conjunction left at the start of an item
	// ("leite, e ovos" → "ovos", "2. e comprar leite" → "comprar leite").
	leadingConjunctionPattern = regexp.MustCompile(`(?i)^(?:e\s+(?:os|as|o|a)\s+|e\s+)`)
	// itemTrimCutset is trimmed from both ends of an item.
	itemTrimCutset = " \t.;:!?-—"
)

// ponytail: heuristic split with no LLM — a comma, a semicolon or a dropped label is the
// proof that the text is a list, and only then does " e " separate too. One pass, no
// cross-item semantics, so a sentence that ever arrives with a comma ("o preço do leite, do
// pão e da carne") splits even where it shouldn't. Raise the ceiling with an LLM split, or
// let the user fix the items on the /data screen.

// SplitTodoItems splits a to-do content into ordered items. Newlines win, then
// numbered markers ("1. ", "2) "), then commas/semicolons — and the "e"
// conjunction as well, but only in an inline list that already shows list
// structure. A leading marker ("Tarefas: ", "Comprar: ") is dropped when a list
// follows it. Each item is trimmed, stripped of a leading number/conjunction and
// of trailing punctuation; empty items are dropped and the original order is
// preserved.
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
	trimmed, isList := stripLeadingLabel(trimmed)
	if strings.ContainsAny(trimmed, "\r\n") {
		return newlineItemPattern.Split(trimmed, -1)
	}
	if numberedItemPattern.MatchString(trimmed) {
		return numberedItemPattern.Split(trimmed, -1)
	}
	if isList || separatorPattern.MatchString(trimmed) {
		return inlineItemPattern.Split(trimmed, -1)
	}
	return separatorPattern.Split(trimmed, -1)
}

// stripLeadingLabel drops a "Marker: " prefix and reports whether it did. It drops it
// only when a list follows ("Comprar: arroz, feijão" → "arroz, feijão", and a bare
// "Tarefas:" is left with nothing); without a list after the colon the text is not a
// marker, so "Revisar: contrato às 10:00" stays whole.
func stripLeadingLabel(trimmed string) (string, bool) {
	loc := leadingLabelPattern.FindStringIndex(trimmed)
	if loc == nil {
		return trimmed, false
	}
	rest := strings.TrimSpace(trimmed[loc[1]:])
	if rest == "" {
		return "", true
	}
	if !strings.ContainsAny(rest, ",;\r\n") && !numberedItemPattern.MatchString(rest) {
		return trimmed, false
	}
	return rest, true
}

// cleanItem normalizes a single raw item.
func cleanItem(candidate string) string {
	item := leadingNumberPattern.ReplaceAllString(strings.TrimSpace(candidate), "")
	item = leadingConjunctionPattern.ReplaceAllString(strings.TrimSpace(item), "")
	return strings.Trim(strings.TrimSpace(item), itemTrimCutset)
}