package views

import (
	"net/http"

	"msg-classifier/internal/models"

	"github.com/donseba/go-htmx"
	"github.com/gin-gonic/gin"
)

// Template name constants — no string literals at call sites.
const (
	BaseTemplate        = "base"
	PageContentTemplate = "page:content"
	ResultTemplate      = "resultado"
	ErrorTemplate       = "error"
)

// ErrorData is the view model for the "error" partial.
type ErrorData struct {
	Message string `json:"message"`
}

// RenderPage renders the full page, or only page:content for htmx requests.
func RenderPage(c *gin.Context) {
	if isHxRequest(c) {
		c.HTML(http.StatusOK, PageContentTemplate, nil)
		return
	}
	c.HTML(http.StatusOK, BaseTemplate, nil)
}

// RenderResult renders the "resultado" partial (HTTP 200).
func RenderResult(c *gin.Context, classification *models.Classification) {
	c.HTML(http.StatusOK, ResultTemplate, classification.ToResponse())
}

// RenderError renders the "error" partial with the given HTTP status.
func RenderError(c *gin.Context, status int, message string) {
	c.HTML(status, ErrorTemplate, ErrorData{Message: message})
}

// isHxRequest reports whether the request is an htmx request; false on any miss.
func isHxRequest(c *gin.Context) bool {
	value, ok := c.Get("htmx")
	if !ok {
		return false
	}
	handler, ok := value.(*htmx.Handler)
	if !ok {
		return false
	}
	return handler.IsHxRequest()
}
