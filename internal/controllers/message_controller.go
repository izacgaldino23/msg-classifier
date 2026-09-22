package controllers

import (
	"errors"
	"net/http"

	"msg-classifier/internal/models"
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
		views.RenderError(c, http.StatusBadRequest, "invalid request")
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

// renderServiceError maps a service error to the error partial.
func renderServiceError(c *gin.Context, err error) {
	if errors.Is(err, services.ErrUpstream) {
		views.RenderError(c, http.StatusBadGateway, err.Error())
		return
	}
	views.RenderError(c, http.StatusInternalServerError, err.Error())
}
