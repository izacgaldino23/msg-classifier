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

// ResultData is the view model for the "resultado" partial.
type ResultData struct {
	Category map[string]any
	Kind     map[string]any
	Action   models.Action
	Contact  *models.Contact
	Segments []models.SegmentScore
}

// RenderPage renders the full page, or only page:content for htmx requests.
func RenderPage(c *gin.Context) {
	if isHxRequest(c) {
		c.HTML(http.StatusOK, PageContentTemplate, nil)
		return
	}
	c.HTML(http.StatusOK, BaseTemplate, nil)
}

// RenderResult renders the "resultado" partial (HTTP 200) from the use case outcome.
func RenderResult(c *gin.Context, outcome *models.UseCaseOutcome) {
	data := ResultData{Action: outcome.Action, Contact: outcome.Contact, Segments: outcome.Segments}
	if outcome.Classification != nil {
		response := outcome.Classification.ToResponse()
		data.Category = response.Category
		data.Kind = response.Kind
	}
	c.HTML(http.StatusOK, ResultTemplate, data)
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
