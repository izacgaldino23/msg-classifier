package api

import (
	"errors"
	"net/http"

	"msg-classifier/internal/jevq"
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
