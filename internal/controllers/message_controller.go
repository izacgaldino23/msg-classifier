package controllers

import (
	"errors"
	"net/http"

	"msg-classifier/internal/jevq"
	"msg-classifier/internal/messages"
	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"
	"msg-classifier/internal/services"
	"msg-classifier/internal/views"

	"github.com/gin-gonic/gin"
)

// MessageController handles POST /api/message. HTTP concerns only.
type MessageController struct {
	classifier *services.ClassificationService
	dispatcher *services.Dispatcher
}

func NewMessageController(classifier *services.ClassificationService, dispatcher *services.Dispatcher) *MessageController {
	return &MessageController{classifier: classifier, dispatcher: dispatcher}
}

// ReceiveMessage handles POST /api/message.
func (ctrl *MessageController) ReceiveMessage(c *gin.Context) {
	request := &models.ReceiveMessageRequest{}

	if err := c.Bind(request); err != nil {
		views.RenderError(c, http.StatusBadRequest, messages.BadRequest())
		return
	}

	classification, err := ctrl.classifier.Classify(request)
	if err != nil {
		renderServiceError(c, err)
		return
	}

	outcome, err := ctrl.dispatcher.Dispatch(request, classification)
	if err != nil {
		renderServiceError(c, err)
		return
	}

	views.RenderResult(c, outcome)
}

// renderServiceError maps a service error to the error partial: the status from the
// sentinel, the wording from internal/messages.
func renderServiceError(c *gin.Context, err error) {
	if errors.Is(err, jevq.ErrUpstream) {
		views.RenderError(c, http.StatusBadGateway, messages.UpstreamUnavailable())
		return
	}
	views.RenderError(c, http.StatusInternalServerError, userText(err))
}

// userText gives a recognized sentinel its PT-BR wording and passes anything else
// through unchanged, so a diagnostic still reaches the developer. api.userText is the
// same switch over the same sentinels: sharing the function would mean one surface
// importing the other's packages.
func userText(err error) string {
	switch {
	case errors.Is(err, services.ErrInvalidFilter):
		return messages.InvalidFilter()
	case errors.Is(err, services.ErrInvalidData):
		return messages.InvalidData()
	case errors.Is(err, services.ErrInvalidPrompt):
		return messages.InvalidPrompt()
	case errors.Is(err, repository.ErrNotFound):
		return messages.NotFound()
	default:
		return err.Error()
	}
}
