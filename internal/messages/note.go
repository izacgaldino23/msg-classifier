package messages

import "msg-classifier/internal/models"

// NoteSaved is the summary once a note is persisted. The participle travels with the
// noun because the two genders disagree — "Lembrete salvo", but "Nota salva" and
// "Lista de tarefas salva"; one shared key would print the feminine ones wrong.
func NoteSaved(noteType string, id uint) string { return format(noteSavedKey(noteType), id) }

func noteSavedKey(noteType string) string {
	switch noteType {
	case models.NoteTypeReminder:
		return "note.saved.reminder"
	case models.NoteTypeTodo:
		return "note.saved.todo"
	default:
		return "note.saved.note"
	}
}

// NoteSavedNoID is the fallback for a malformed outcome carrying no note at all.
func NoteSavedNoID() string { return T("note.saved_no_id") }

// NoteFound is the summary listing how many notes matched the search term.
func NoteFound(count int, term string) string { return format("note.found", count, term) }

// NoteNotFound is the summary when no note matched the search term.
func NoteNotFound(term string) string { return format("note.not_found", term) }

// NoteNoData is the summary when the sub-type or the reminder date was missing.
func NoteNoData() string { return T("note.no_data") }

// NoteDuplicate is the summary when the note already exists (DC-008).
func NoteDuplicate(content string) string { return format("note.duplicate", content) }

// NoteDuplicateNoContent is the fallback for a duplicate outcome carrying no note.
func NoteDuplicateNoContent() string { return T("note.duplicate_no_content") }