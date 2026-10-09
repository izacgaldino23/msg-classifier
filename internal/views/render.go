package views

import (
	"fmt"
	"html/template"
	"net/http"

	"msg-classifier/internal/messages"
	"msg-classifier/internal/models"
	"msg-classifier/internal/ptbr"

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

	contactsTableTmpl     = "contacts_table"
	notesTableTmpl        = "notes_table"
	transactionsTableTmpl = "transactions_table"
	dataDetailTmpl        = "data_detail"
	dataItemRowTmpl       = "data_item_row"
)

const (
	HomePage           = "home"
	HomePageContent    = "home:content"
	PromptsPage        = "prompts"
	PromptsPageContent = "prompts:content"
	DataPage           = "data"
	DataPageContent    = "data:content"
)

// ErrorData is the view model for the "error" partial.
type ErrorData struct {
	Message string `json:"message"`
}

// PageData is the view model for the "base" layout — it carries the page name
// so the layout can mark the active nav link without a per-page template block.
type PageData struct {
	Page string
}

// RenderPage renders the full page, or only its content for htmx requests.
func RenderPage(c *gin.Context, page, content string) {
	if isHxRequest(c) {
		c.HTML(http.StatusOK, content, nil)
		return
	}
	c.HTML(http.StatusOK, page, PageData{Page: page})
}

// RenderResult renders the "resultado" partial (HTTP 200) from the use case outcome.
// The outcome goes to the template as-is: UseCaseOutcome already exposes every field
// the partial reads, so a view model here would be a second copy to keep in sync.
func RenderResult(c *gin.Context, outcome *models.UseCaseOutcome) {
	c.HTML(http.StatusOK, ResultTemplate, outcome)
}

// RenderError renders the "error" partial with the given HTTP status.
func RenderError(c *gin.Context, status int, message string) {
	c.HTML(status, ErrorTemplate, ErrorData{Message: message})
}

// isHxRequest reports whether the request came from htmx; false on a direct visit.
func isHxRequest(c *gin.Context) bool {
	return c.GetHeader("HX-Request") == "true"
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

// DataDetailData is the view model for the "data_detail" partial. Only one of
// Contact, Note or Transaction is set, matching Kind.
type DataDetailData struct {
	Kind        string
	Mode        string
	Contact     *models.Contact
	Note        *models.Note
	Transaction *models.Transaction
}

// ItemRowData is the view model for the "data_item_row" partial — one to-do item
// line. The zero value renders the blank row the "Adicionar item" button appends.
type ItemRowData struct {
	Text string
	Done bool
}

// RenderContactsTable renders the "contacts_table" partial (HTTP 200). The filter
// and the search term ride back to the delete endpoint as hidden inputs, so the
// view the user was on comes back after a delete.
func RenderContactsTable(c *gin.Context, filter, search string, contacts []models.Contact) {
	c.HTML(http.StatusOK, contactsTableTmpl, gin.H{"Filter": filter, "Search": search, "Contacts": contacts})
}

// RenderNotesTable renders the "notes_table" partial (HTTP 200).
func RenderNotesTable(c *gin.Context, filter, search string, notes []*models.Note) {
	c.HTML(http.StatusOK, notesTableTmpl, gin.H{"Filter": filter, "Search": search, "Notes": notes})
}

// RenderTransactionsTable renders the "transactions_table" partial (HTTP 200).
func RenderTransactionsTable(c *gin.Context, filter, search string, transactions []*models.Transaction) {
	c.HTML(http.StatusOK, transactionsTableTmpl, gin.H{"Filter": filter, "Search": search, "Transactions": transactions})
}

// RenderDataDetail renders the "data_detail" partial (HTTP 200).
func RenderDataDetail(c *gin.Context, data DataDetailData) {
	c.HTML(http.StatusOK, dataDetailTmpl, data)
}

// RenderDataItemRow renders the "data_item_row" partial (HTTP 200) — one to-do row.
func RenderDataItemRow(c *gin.Context) {
	c.HTML(http.StatusOK, dataItemRowTmpl, ItemRowData{})
}

// BadgeVariant returns the Web Awesome <wa-badge> variant carrying a choice's
// tone. Web Awesome ships brand / neutral / success / warning / danger only, so
// the informational tones (notes, recebimento) share brand.
func BadgeVariant(choice any) string {
	s, _ := choice.(string)
	switch s {
	case "contact", "require", "note", "notes", "venda", "recebimento":
		return "brand"
	case "finance", "add", "todo", "compra":
		return "success"
	case "schedule", "both", "reminder", "pagamento":
		return "warning"
	default:
		return "neutral"
	}
}

// Deref dereferences an optional string; nil renders "".
func Deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Confidence formats a Jev confidence as a display percentage.
func Confidence(f float64) string {
	return fmt.Sprintf("%.2f", f*100)
}

// FuncMap exposes template helpers to the shared template set.
// money and dateBR live in internal/ptbr beside the parsers that read them back, label
// in internal/messages beside the locale that holds the wording; the rest are screen
// presentation. Templates call these names, never the Go symbols, so a helper can move
// packages without touching a single template.
var FuncMap = template.FuncMap{
	"label":      messages.Label,
	"money":      ptbr.MoneyBRL,
	"confidence": Confidence,
	"dateBR":     ptbr.DateBR,
	"deref":      Deref,
	"badge":      BadgeVariant,
}