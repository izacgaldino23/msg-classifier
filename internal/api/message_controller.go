package api

import (
	"net/http"

	"msg-classifier/internal/models"

	"github.com/gin-gonic/gin"
)

// classifier and dispatcher are the only two things the handler needs. They are
// interfaces so the test can drive the HTTP surface without a Jev call, the same
// reason services define jevClient at its own boundary.
type classifier interface {
	Classify(request *models.ReceiveMessageRequest) (*models.Classification, error)
}

type dispatcher interface {
	Dispatch(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error)
}

// MessageController handles POST /api/v1/messages. HTTP concerns only.
type MessageController struct {
	classifier classifier
	dispatcher dispatcher
}

func NewMessageController(c classifier, d dispatcher) *MessageController {
	return &MessageController{classifier: c, dispatcher: d}
}

// ReceiveMessage classifies a message and answers the outcome as JSON.
//
//	@Summary		Classify a message
//	@Description	Runs the same Classify → Dispatch core as the web and the CLI, and answers the outcome as JSON. `action` is the discriminant: contact_add, contact_found, contact_not_found, contact_duplicate, contact_no_data, note_add, note_found, note_not_found, note_no_data, note_duplicate, transaction_add, transaction_found, transaction_not_found, transaction_no_data, transaction_duplicate or none. On a duplicate the API only informs (DC-008): there is no confirmation path — re-posting the same message with `dup_action` set to `new` or `update` performs the confirmed save, but the web and CLI are the surfaces designed for that loop.
//	@Tags			messages
//	@Accept			json
//	@Produce		json
//	@Param			message	body		models.ReceiveMessageRequest	true	"Message to classify"
//	@Success		200		{object}	MessageResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		502		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/messages [post]
func (ctrl *MessageController) ReceiveMessage(c *gin.Context) {
	request := &models.ReceiveMessageRequest{}
	if err := c.ShouldBindJSON(request); err != nil {
		RenderError(c, http.StatusBadRequest, "invalid request")
		return
	}
	// Trust boundary: an empty message is a client bug, not a classification.
	if request.Message == "" {
		RenderError(c, http.StatusBadRequest, "message is required")
		return
	}

	classification, err := ctrl.classifier.Classify(request)
	if err != nil {
		RenderError(c, StatusFor(err), err.Error())
		return
	}

	outcome, err := ctrl.dispatcher.Dispatch(request, classification)
	if err != nil {
		RenderError(c, StatusFor(err), err.Error())
		return
	}

	c.JSON(http.StatusOK, Outcome(outcome))
}
