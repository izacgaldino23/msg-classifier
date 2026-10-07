package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"msg-classifier/internal/jevq"
	"msg-classifier/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeClassifier struct {
	classification *models.Classification
	err            error
	seen           *models.ReceiveMessageRequest
}

func (f *fakeClassifier) Classify(request *models.ReceiveMessageRequest) (*models.Classification, error) {
	f.seen = request
	return f.classification, f.err
}

type fakeDispatcher struct {
	outcome *models.UseCaseOutcome
	err     error
}

func (f fakeDispatcher) Dispatch(*models.ReceiveMessageRequest, *models.Classification) (*models.UseCaseOutcome, error) {
	return f.outcome, f.err
}

func postMessage(t *testing.T, ctrl *MessageController, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/messages", ctrl.ReceiveMessage)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}

// The happy path must carry the outcome's CONTENT, not just a 200 — the same rule
// that governs every result.html branch.
func TestReceiveMessageReturnsTheOutcome(t *testing.T) {
	ctrl := NewMessageController(
		&fakeClassifier{classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "finance", Confidence: 0.94},
			Kind:     models.KindFinding{Choice: "add", Confidence: 0.86},
		}},
		fakeDispatcher{outcome: &models.UseCaseOutcome{
			Action:  models.ActionTransactionNoData,
			Missing: "o valor",
		}},
	)

	recorder := postMessage(t, ctrl, `{"message":"comprei arroz","user_id":"1"}`)
	require.Equal(t, http.StatusOK, recorder.Code)

	var response MessageResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response), "body must decode into MessageResponse")
	assert.Equal(t, string(models.ActionTransactionNoData), response.Action)
	assert.Equal(t, "Não consegui classificar a transação: o valor.", response.Message)
	assert.Equal(t, "o valor", response.Missing)
}

func TestReceiveMessageRequiresAMessage(t *testing.T) {
	classifier := &fakeClassifier{classification: &models.Classification{}}
	ctrl := NewMessageController(classifier, fakeDispatcher{})

	recorder := postMessage(t, ctrl, `{"user_id":"1"}`)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "message is required")
	assert.Nil(t, classifier.seen, "Classify() must not run on an empty message")
}

func TestReceiveMessageRejectsMalformedJSON(t *testing.T) {
	ctrl := NewMessageController(&fakeClassifier{classification: &models.Classification{}}, fakeDispatcher{})

	recorder := postMessage(t, ctrl, `not json`)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "invalid request")
}

func TestReceiveMessageMapsUpstreamTo502(t *testing.T) {
	ctrl := NewMessageController(
		&fakeClassifier{err: fmt.Errorf("failed to call jev: %w", jevq.ErrUpstream)},
		fakeDispatcher{},
	)

	recorder := postMessage(t, ctrl, `{"message":"oi"}`)
	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "failed to call jev")
}

func TestReceiveMessageMapsDispatchFailureTo500(t *testing.T) {
	ctrl := NewMessageController(
		&fakeClassifier{classification: &models.Classification{}},
		fakeDispatcher{err: fmt.Errorf("failed to persist contact")},
	)

	recorder := postMessage(t, ctrl, `{"message":"oi"}`)
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "failed to persist contact")
}

// The bound request must reach the classifier verbatim — a dropped UserID is
// invisible in the response and only shows up as a worse Jev state.
func TestReceiveMessageForwardsTheRequest(t *testing.T) {
	classifier := &fakeClassifier{classification: &models.Classification{}}
	ctrl := NewMessageController(classifier, fakeDispatcher{outcome: &models.UseCaseOutcome{Action: models.ActionNone}})

	postMessage(t, ctrl, `{"message":"salva o fulano","user_id":"42"}`)

	require.NotNil(t, classifier.seen, "Classify() was never called")
	assert.Equal(t, "salva o fulano", classifier.seen.Message)
	assert.Equal(t, "42", classifier.seen.UserID)
}
