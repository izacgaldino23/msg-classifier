package messages

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"msg-classifier/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// keyShape is what an unresolved key looks like: dotted lowercase words. A wrapper
// returning one means the locale file lost the entry.
var keyShape = regexp.MustCompile(`^[a-z]+(\.[a-z_0-9]+)+$`)

func assertTranslated(t *testing.T, got string) {
	t.Helper()
	require.NotEmpty(t, got, "message is empty")
	require.NotContains(t, got, "%!", "unresolved format verb: %q", got)
	require.False(t, keyShape.MatchString(got), "looks like an unresolved key: %q", got)
}

func TestLocaleLoads(t *testing.T) {
	// The count guards against an entry dropped from the file; the encoding check guards
	// against an accented value saved in the wrong codepage, which renders as "?" here.
	assert.GreaterOrEqual(t, len(texts), 60, "locale file lost entries")
	assert.Equal(t, "extração:", SectionExtraction())
}

func TestEveryWrapperIsTranslated(t *testing.T) {
	for name, got := range map[string]string{
		"ContactSaved":           ContactSaved(7),
		"ContactFound":           ContactFound(),
		"ContactNotFound":        ContactNotFound("joão"),
		"ContactNoData":          ContactNoData(),
		"ContactDuplicate":       ContactDuplicate("Fulano"),
		"NoteSaved":              NoteSaved(models.NoteTypeNote, 7),
		"NoteSavedNoID":          NoteSavedNoID(),
		"NoteFound":              NoteFound(2, "pão"),
		"NoteNotFound":           NoteNotFound("pão"),
		"NoteNoData":             NoteNoData(),
		"NoteDuplicate":          NoteDuplicate("comprar pão"),
		"NoteDuplicateNoContent": NoteDuplicateNoContent(),
		"TransactionSaved":       TransactionSaved(7),
		"TransactionSavedNoID":   TransactionSavedNoID(),
		"TransactionFound":       TransactionFound(2, "mercado"),
		"TransactionNotFound":    TransactionNotFound("mercado"),
		"TransactionNoData":      TransactionNoData(MissingAmount()),
		"TransactionDuplicate":   TransactionDuplicate(),
		"MissingMessage":         MissingMessage(),
		"MissingAmount":          MissingAmount(),
		"MissingFilter":          MissingFilter(),
		"BadRequest":             BadRequest(),
		"BadKind":                BadKind(),
		"BadID":                  BadID(),
		"MissingFlow":            MissingFlow(),
		"MessageRequired":        MessageRequired(),
		"NotFound":               NotFound(),
		"InvalidFilter":          InvalidFilter(),
		"InvalidData":            InvalidData(),
		"InvalidPrompt":          InvalidPrompt(),
		"UpstreamUnavailable":    UpstreamUnavailable(),
		"Banner":                 Banner(),
		"Cancelled":              Cancelled(),
		"DupOptions":             DupOptions(),
		"SectionExtraction":      SectionExtraction(),
		"ErrorLine":              ErrorLine(errors.New("boom")),
		"FieldName":              Field("name"),
		"FieldParty":             Field("party"),
	} {
		t.Run(name, func(t *testing.T) { assertTranslated(t, got) })
	}
}

// The participle has to agree with the noun: this is the bug the package exists for.
func TestNoteSavedGender(t *testing.T) {
	assert.Equal(t, "Lembrete salvo · ID 3", NoteSaved(models.NoteTypeReminder, 3))
	assert.Equal(t, "Lista de tarefas salva · ID 3", NoteSaved(models.NoteTypeTodo, 3))
	assert.Equal(t, "Nota salva · ID 3", NoteSaved(models.NoteTypeNote, 3))
	assert.Equal(t, "Nota salva · ID 3", NoteSaved("outro", 3), "unknown falls back to note")
}

func TestLabel(t *testing.T) {
	assert.Equal(t, "Contato", Label("contact"))
	assert.Equal(t, "Transferência", Label("transferencia"))
	assert.Equal(t, "Lista de tarefas", Label("todo"))
	assert.Equal(t, "brand-new", Label("brand-new"), "unknown falls back to the input")
	assert.Equal(t, "", Label(nil), "a missing classification renders empty")
	assert.Equal(t, "", Label(42))
}

func TestTUnknownKeyReturnsKey(t *testing.T) {
	assert.Equal(t, "nope.missing", T("nope.missing"))
}

// A malformed locale must not panic in a request path: T degrades to returning keys
// and TestLocaleLoads fails instead.
func TestLoadIgnoresBrokenLocale(t *testing.T) {
	assert.Empty(t, load([]byte("{not json")))
	assert.Empty(t, load(nil))
	assert.Equal(t, map[string]string{"x.y": "z"}, load([]byte(`{"x.y":"z"}`)))
}

func TestFieldNamesAreAllPresent(t *testing.T) {
	for _, name := range []string{"name", "phone", "email", "content", "date", "time", "type", "amount", "party", "message", "total"} {
		assertTranslated(t, Field(name))
	}
	assert.True(t, strings.Contains(DupOptions(), "[u]"), "the dup options keep their keys")
}