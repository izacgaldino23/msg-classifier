package controllers

import (
	"errors"
	"net/http"

	"msg-classifier/internal/models"
	"msg-classifier/internal/services"
	"msg-classifier/internal/views"

	"github.com/gin-gonic/gin"
)

// MessageController handles message classification requests. HTTP concerns
// only: bind → service → render. No business logic, no Jev types, no
// template name literals.
type MessageController struct {
	classifier *services.ClassificationService
}

// NewMessageController wires the classification service into the controller.
func NewMessageController(classifier *services.ClassificationService) *MessageController {
	return &MessageController{classifier: classifier}
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
		if errors.Is(err, services.ErrUpstream) {
			views.RenderError(c, http.StatusBadGateway, err.Error())
			return
		}
		views.RenderError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Verify if user want save or get data
	// if user want save, save the data to database
	// if user want get, get the data from database
	// TODO

	views.RenderResult(c, classification)
}
