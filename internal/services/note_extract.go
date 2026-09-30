package services

import (
	"fmt"

	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"
)

// noteTypeAnswerKey is the answer key of the note_type question in note.json.
const noteTypeAnswerKey = "note_type"

// noteTypeTemplate is the embedded template that asks for the sub-type.
const noteTypeTemplate = "note.json"

// NoteExtractor asks Jev which sub-type a notes message is. It reuses the jevClient
// seam from classification.go: the note flow is template-file based, exactly like
// the classification call.
type NoteExtractor struct {
	jev jevClient
}

func NewNoteExtractor(client jevClient) *NoteExtractor {
	return &NoteExtractor{jev: client}
}

// ExtractType returns the raw note sub-type chosen by Jev ("note", "reminder" or
// "todo"). The raw choice is returned as-is; validating it is a service rule.
func (e *NoteExtractor) ExtractType(request *models.ReceiveMessageRequest) (string, error) {
	state := jev.JevState(map[string]any{
		"user":    request.UserID,
		"message": request.Message,
	})

	resp, err := e.jev.MakeJevRequestFromFile(state, noteTypeTemplate)
	if err != nil {
		return "", fmt.Errorf("%w: failed to call jev with template %q: %w", ErrUpstream, noteTypeTemplate, err)
	}

	answer, err := answerAsChoice(resp, noteTypeAnswerKey)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUpstream, err)
	}
	return answer.Choice, nil
}