package services

import (
	"errors"
	"testing"

	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockJevClient struct {
	resp     *jev.JevResponse
	err      error
	gotState jev.JevState
	gotFile  string
}

func (m *mockJevClient) MakeJevRequestFromFile(state jev.JevState, fileName string) (*jev.JevResponse, error) {
	m.gotState = state
	m.gotFile = fileName
	return m.resp, m.err
}

func choiceResponse(classification, kind string, confidence float64) *jev.JevResponse {
	return &jev.JevResponse{
		Model: "jev-latest",
		Answers: map[string]jev.JevAnswer{
			"classification":      &jev.JevAnswerChoice{Type: jev.ChoiceQuestionType, Choice: classification, Confidence: confidence},
			"adding_or_requiring": &jev.JevAnswerChoice{Type: jev.ChoiceQuestionType, Choice: kind, Confidence: confidence},
		},
	}
}

func TestClassificationServiceClassify(t *testing.T) {
	mock := &mockJevClient{resp: choiceResponse("contact", "add", 0.95)}
	service := NewClassificationService(mock)

	got, err := service.Classify(&models.ReceiveMessageRequest{Message: "salva fulano", UserID: "u1"})
	require.NoError(t, err)
	assert.Equal(t, "contact", got.Category.Choice)
	assert.Equal(t, 0.95, got.Category.Confidence)
	assert.Equal(t, "add", got.Kind.Choice)
	assert.Equal(t, 0.95, got.Kind.Confidence)
}

func TestClassificationServiceClassifyStateAndTemplate(t *testing.T) {
	mock := &mockJevClient{resp: choiceResponse("contact", "add", 0.9)}
	service := NewClassificationService(mock)

	_, err := service.Classify(&models.ReceiveMessageRequest{Message: "salva fulano", UserID: "u1"})
	require.NoError(t, err)
	assert.Equal(t, "classification.json", mock.gotFile)
	state, ok := mock.gotState.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "u1", state["user"])
	assert.Equal(t, "salva fulano", state["message"])
}

func TestClassificationServiceClassifyJevFailure(t *testing.T) {
	mock := &mockJevClient{err: errors.New("boom")}
	service := NewClassificationService(mock)

	_, err := service.Classify(&models.ReceiveMessageRequest{Message: "oi"})
	assert.ErrorIs(t, err, ErrUpstream)
}

func TestClassificationServiceClassifyMissingAnswer(t *testing.T) {
	mock := &mockJevClient{resp: &jev.JevResponse{Model: "jev-latest", Answers: map[string]jev.JevAnswer{}}}
	service := NewClassificationService(mock)

	_, err := service.Classify(&models.ReceiveMessageRequest{Message: "oi"})
	assert.ErrorIs(t, err, ErrUpstream)
}

func TestClassificationServiceClassifyWrongAnswerType(t *testing.T) {
	mock := &mockJevClient{resp: &jev.JevResponse{Model: "jev-latest", Answers: map[string]jev.JevAnswer{
		"classification": &jev.JevAnswerNoul{Type: jev.NoulQuestionType, Noul: 0.5},
	}}}
	service := NewClassificationService(mock)

	_, err := service.Classify(&models.ReceiveMessageRequest{Message: "oi"})
	assert.ErrorIs(t, err, ErrUpstream)
}