package api

import (
	"errors"
	"net/http"

	"msg-classifier/internal/jevq"
	"msg-classifier/internal/messages"
	"msg-classifier/internal/repository"
	"msg-classifier/internal/services"

	"github.com/gin-gonic/gin"
)

// ErrorResponse is the JSON body of every non-2xx answer.
type ErrorResponse struct {
	Error string `json:"error"`
}

// RenderError writes the JSON error body with the given status.
func RenderError(c *gin.Context, status int, message string) {
	c.JSON(status, ErrorResponse{Error: message})
}

// StatusFor maps a service error to its HTTP status. It restates the rules
// controllers.renderServiceError applies on the web — upstream → 502, anything
// else → 500 — plus the data screen's 400/404. Sharing the function itself would
// mean importing internal/controllers, so the mapping is a second small switch.
func StatusFor(err error) int {
	switch {
	case errors.Is(err, services.ErrInvalidFilter), errors.Is(err, services.ErrInvalidData):
		return http.StatusBadRequest
	case errors.Is(err, repository.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, jevq.ErrUpstream):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// userText gives a recognized sentinel its PT-BR wording and passes anything else
// through unchanged, so a diagnostic still reaches the client. The web surface keeps
// the same switch in controllers.userText: sharing the function would mean one surface
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
	case errors.Is(err, jevq.ErrUpstream):
		return messages.UpstreamUnavailable()
	default:
		return err.Error()
	}
}
