package notes

import (
	"errors"
	"testing"

	"msg-classifier/internal/jevq"
	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noteTypeResponse is the shape of the note.json answer map.
func noteTypeResponse(choice string) *jev.JevResponse {
	return &jev.JevResponse{
		Model: "jev-latest",
		Answers: map[string]any{
			noteTypeAnswerKey: &jev.JevAnswerChoice{Choice: choice, Confidence: 0.9},
		},
	}
}

func TestNoteExtractorExtractType(t *testing.T) {
	mock := &mockJevClient{resp: noteTypeResponse(models.NoteTypeReminder)}
	extractor := NewExtractor(mock)

	choice, err := extractor.ExtractType(&models.ReceiveMessageRequest{Message: "me lembra dia 10", UserID: "u1"})
	require.NoError(t, err)
	assert.Equal(t, models.NoteTypeReminder, choice)
	assert.Equal(t, "note.json", mock.gotFile)

	state, ok := mock.gotState.(map[string]any)
	require.True(t, ok, "state = %T, want map[string]any", mock.gotState)
	assert.Equal(t, "me lembra dia 10", state["message"])
	assert.Equal(t, "u1", state["user"])
}

func TestNoteExtractorExtractTypeUpstreamFailure(t *testing.T) {
	mock := &mockJevClient{err: errors.New("boom")}
	extractor := NewExtractor(mock)

	_, err := extractor.ExtractType(&models.ReceiveMessageRequest{Message: "anota isso"})
	assert.ErrorIs(t, err, jevq.ErrUpstream)
}

func TestNoteExtractorExtractTypeMissingAnswer(t *testing.T) {
	mock := &mockJevClient{resp: &jev.JevResponse{Model: "jev-latest", Answers: map[string]any{}}}
	extractor := NewExtractor(mock)

	_, err := extractor.ExtractType(&models.ReceiveMessageRequest{Message: "anota isso"})
	assert.ErrorIs(t, err, jevq.ErrUpstream)
}

func TestNoteExtractorExtractTypeWrongAnswerType(t *testing.T) {
	mock := &mockJevClient{resp: &jev.JevResponse{
		Model: "jev-latest",
		Answers: map[string]any{
			noteTypeAnswerKey: &jev.JevAnswerNoul{Noul: 0.9},
		},
	}}
	extractor := NewExtractor(mock)

	_, err := extractor.ExtractType(&models.ReceiveMessageRequest{Message: "anota isso"})
	assert.ErrorIs(t, err, jevq.ErrUpstream)
}