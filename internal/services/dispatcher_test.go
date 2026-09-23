package services

import (
	"errors"
	"testing"

	"msg-classifier/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCategoryHandler struct {
	gotRequest        *models.ReceiveMessageRequest
	gotClassification *models.Classification
	outcome           *models.UseCaseOutcome
	err               error
}

func (f *fakeCategoryHandler) Handle(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	f.gotRequest = request
	f.gotClassification = classification
	return f.outcome, f.err
}

func TestDispatcherDispatchRoutesToHandler(t *testing.T) {
	handler := &fakeCategoryHandler{outcome: &models.UseCaseOutcome{Action: models.ActionContactAdd}}
	d := NewDispatcher(map[string]CategoryHandler{"contact": handler})

	request := &models.ReceiveMessageRequest{Message: "oi"}
	classification := &models.Classification{Category: models.CategoryFinding{Choice: "contact"}}
	outcome, err := d.Dispatch(request, classification)
	require.NoError(t, err)
	assert.Same(t, handler.outcome, outcome)
	assert.Same(t, request, handler.gotRequest)
	assert.Same(t, classification, handler.gotClassification)
}

func TestDispatcherDispatchUnknownCategory(t *testing.T) {
	handler := &fakeCategoryHandler{}
	d := NewDispatcher(map[string]CategoryHandler{"contact": handler})

	outcome, err := d.Dispatch(&models.ReceiveMessageRequest{Message: "oi"}, &models.Classification{Category: models.CategoryFinding{Choice: "finance"}})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNone, outcome.Action)
	assert.Nil(t, handler.gotRequest, "handler must not be called for unknown category")
}

func TestDispatcherDispatchHandlerError(t *testing.T) {
	handler := &fakeCategoryHandler{err: errors.New("boom")}
	d := NewDispatcher(map[string]CategoryHandler{"contact": handler})

	_, err := d.Dispatch(&models.ReceiveMessageRequest{Message: "oi"}, &models.Classification{Category: models.CategoryFinding{Choice: "contact"}})
	require.Error(t, err)
}