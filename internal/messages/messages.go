// Package messages owns the text the user reads. The wording lives in
// locales/<locale>.json and this package is the only thing that reads it, so the
// terminal, the JSON API and the web partials say the same thing instead of
// drifting apart on case and punctuation. A second language is one more file in
// locales/ with the same keys — nothing in the call sites changes.
//
// One exported function per message, named for what it says rather than which
// surface prints it. Call sites never see a key, a format string or a raw string,
// which is what keeps the arity and the layout ("· ID %d") in one place.
//
// Wording only: indentation, column layout and HTML stay with the surface. Templates
// never call these symbols either — views exposes Label through the FuncMap under a
// template name.
//
// It imports models only for the note-kind vocabulary; the sentinels (services,
// repository, jevq) stay out, so the controllers keep mapping errors to text.
package messages

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed locales/pt-BR.json
var ptBR []byte

// texts is the loaded locale. Only one language ships today; selecting another is a
// matter of loading a different file here.
var texts = load(ptBR)

func load(locale []byte) map[string]string {
	parsed := map[string]string{}
	if err := json.Unmarshal(locale, &parsed); err != nil {
		return map[string]string{}
	}
	return parsed
}

// T returns the text for key, or the key itself when the locale has no entry: an
// untranslated message shows its key instead of rendering nothing. A broken locale
// file is caught by the tests, not by a user staring at an empty card.
func T(key string) string {
	if text, ok := texts[key]; ok {
		return text
	}
	return key
}

// format renders the message for key with its arguments, so a wrapper stays one line.
func format(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}
