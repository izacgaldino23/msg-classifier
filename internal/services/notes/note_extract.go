package notes

import (
	"fmt"

	"msg-classifier/internal/jevq"
	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"
)

// noteTypeAnswerKey is the answer key of the note_type question in note.json.
const noteTypeAnswerKey = "note_type"

// noteTypeTemplate is the embedded template that asks for the sub-type.
const noteTypeTemplate = "note.json"

// NoteExtractor asks Jev which sub-type a notes message is. It reuses the jevq.Client
// seam: the note flow is template-file based, exactly like the classification call.
// If a pre-classified subtype is available from the composite request, it uses that
// instead of making a second Jev call.
type NoteExtractor struct {
	jev jevq.Client
}

func NewExtractor(client jevq.Client) *NoteExtractor {
	return &NoteExtractor{jev: client}
}

// ExtractType returns the raw note sub-type chosen by Jev ("note", "reminder" or
// "todo"). The raw choice is returned as-is; validating it is a service rule.
func (e *NoteExtractor) ExtractType(request *models.ReceiveMessageRequest) (string, error) {
	// ... (This function remains as an internal-only helper/fallback)
	// (Actual implementation remains identical to what was read)
	state := jev.JevState(map[string]any{
		"user":    request.UserID,
		"message": request.Message,
	})

	resp, err := e.jev.MakeJevRequestFromFile(state, noteTypeTemplate)
	if err != nil {
		return "", fmt.Errorf("%w: failed to call jev with template %q: %w", jevq.ErrUpstream, noteTypeTemplate, err)
	}

	answer, err := jevq.AnswerChoice(resp, noteTypeAnswerKey)
	if err != nil {
		return "", fmt.Errorf("%w: %w", jevq.ErrUpstream, err)
	}
	return answer.Choice, nil
}

// ExtractTypeFromClassification extracts the note type from a pre-classified
// composite request. The composite request has category_subtype with values like
// note_reminder, note_todo.
func (e *NoteExtractor) ExtractTypeFromClassification(request *models.ReceiveMessageRequest, classification *models.Classification) (string, error) {
	if classification != nil && classification.Subtype.Choice != "" {
		return mapSubtypeToNoteType(classification.Subtype.Choice), nil
	}
	// Fallback: no pre-classified subtype, call Jev.
	return e.ExtractType(request)
}

// mapSubtypeToNoteType maps the composite classification subtype to note type.
func mapSubtypeToNoteType(subtype string) string {
	switch subtype {
	case "note_reminder":
		return "reminder"
	case "note_todo":
		return "todo"
	default:
		return "note"
	}
}