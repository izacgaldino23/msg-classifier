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

// RenderPage renders the full page for regular requests and only the
// page:content block for htmx requests (switch absorbed from web_handler).
// A missing or mistyped htmx context falls back to the full page —
// replaces the former MustGet + unchecked assertion (no panic).
func RenderPage(c *gin.Context) {
	if isHxRequest(c) {
		c.HTML(http.StatusOK, PageContentTemplate, nil)
		return
	}
	c.HTML(http.StatusOK, BaseTemplate, nil)
}

// RenderResult renders the "resultado" partial with the classification
// formatted for display (HTTP 200).
func RenderResult(c *gin.Context, classification *models.Classification) {
	c.HTML(http.StatusOK, ResultTemplate, classification.ToResponse())
}

// RenderError renders the "error" partial with the given HTTP status so
// htmx swaps the error card into #resultado instead of injecting JSON.
func RenderError(c *gin.Context, status int, message string) {
	c.HTML(status, ErrorTemplate, ErrorData{Message: message})
}

// isHxRequest safely reads the htmx handler set by the middleware in
// cmd/api/main.go; false on any miss.
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
