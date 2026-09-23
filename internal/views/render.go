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

	PromptTableTemplate       = "prompt_table"
	EvaluationResultsTemplate = "evaluation_results"
)

const (
	HomePage           = "home"
	HomePageContent    = "home:content"
	PromptsPage        = "prompts"
	PromptsPageContent = "prompts:content"
)

// ErrorData is the view model for the "error" partial.
type ErrorData struct {
	Message string `json:"message"`
}

// ResultData is the view model for the "resultado" partial.
type ResultData struct {
	Category   map[string]any
	Kind       map[string]any
	Action     models.Action
	Contact    *models.Contact
	Segments   []models.SegmentScore
	SearchTerm string
}

// RenderPage renders the full page, or only its content for htmx requests.
func RenderPage(c *gin.Context, page, content string) {
	if isHxRequest(c) {
		c.HTML(http.StatusOK, content, nil)
		return
	}
	c.HTML(http.StatusOK, page, nil)
}

// RenderResult renders the "resultado" partial (HTTP 200) from the use case outcome.
func RenderResult(c *gin.Context, outcome *models.UseCaseOutcome) {
	data := ResultData{Action: outcome.Action, Contact: outcome.Contact, Segments: outcome.Segments, SearchTerm: outcome.SearchTerm}
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

// PromptTableData is the view model for the "prompt_table" partial.
type PromptTableData struct {
	Flow    string
	Prompts []models.JevPrompt
}

// EvaluationResultsData is the view model for the "evaluation_results" partial.
type EvaluationResultsData struct {
	Flow    string
	Results []models.EvaluationResult
	CsvPath string // set when the results were exported as CSV
}

// RenderPromptTable renders the "prompt_table" partial (HTTP 200).
func RenderPromptTable(c *gin.Context, flow string, prompts []models.JevPrompt) {
	c.HTML(http.StatusOK, PromptTableTemplate, PromptTableData{Flow: flow, Prompts: prompts})
}

// RenderEvaluationResults renders the "evaluation_results" partial (HTTP 200);
// csvPath is shown when the results were exported as CSV.
func RenderEvaluationResults(c *gin.Context, flow string, results []models.EvaluationResult, csvPath string) {
	c.HTML(http.StatusOK, EvaluationResultsTemplate, EvaluationResultsData{Flow: flow, Results: results, CsvPath: csvPath})
}
