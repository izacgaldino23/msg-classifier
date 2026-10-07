package services

import (
	"msg-classifier/internal/models"
)

// CategoryHandler runs the use case for a classified category.
type CategoryHandler interface {
	Handle(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error)
}

// Dispatcher routes a classification to the handler for its category.
type Dispatcher struct {
	handlers map[string]CategoryHandler
}

func NewDispatcher(handlers map[string]CategoryHandler) *Dispatcher {
	return &Dispatcher{handlers: handlers}
}

// Dispatch routes the classification to the handler for its category. The outcome
// carries the original message so the WEB duplicate screen can re-post it (DC-008).
func (d *Dispatcher) Dispatch(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	handler, ok := d.handlers[classification.Category.Choice]
	if !ok {
		return &models.UseCaseOutcome{Classification: classification, Action: models.ActionNone, Message: request.Message}, nil
	}
	outcome, err := handler.Handle(request, classification)
	if outcome != nil {
		outcome.Message = request.Message
	}
	return outcome, err
}
