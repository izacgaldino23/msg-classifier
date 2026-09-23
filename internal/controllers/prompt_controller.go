package controllers

import (
	"errors"
	"net/http"

	"msg-classifier/internal/models"
	"msg-classifier/internal/services"
	"msg-classifier/internal/views"

	"github.com/gin-gonic/gin"
)

// PromptController serves the Jev validation harness (/prompts). HTTP concerns only.
type PromptController struct {
	service *services.PromptService
}

func NewPromptController(service *services.PromptService) *PromptController {
	return &PromptController{service: service}
}

// Page handles GET /prompts.
func (ctrl *PromptController) Page(c *gin.Context) {
	views.RenderPage(c)
}

// Table handles GET /prompts/table?flow=... — renders the prompt table partial.
func (ctrl *PromptController) Table(c *gin.Context) {
	flow := c.Query("flow")
	if flow == "" {
		views.RenderError(c, http.StatusBadRequest, "missing flow")
		return
	}
	prompts, err := ctrl.service.ListByFlow(flow)
	if err != nil {
		renderServiceError(c, err)
		return
	}
	views.RenderPromptTable(c, flow, prompts)
}

// Add handles POST /prompts — persists a new prompt and re-renders the table.
func (ctrl *PromptController) Add(c *gin.Context) {
	request := &models.PromptForm{}
	if err := c.Bind(request); err != nil {
		views.RenderError(c, http.StatusBadRequest, "invalid request")
		return
	}
	if _, err := ctrl.service.Add(request.Flow, request.Message, request.Expected); err != nil {
		if errors.Is(err, services.ErrInvalidPrompt) {
			views.RenderError(c, http.StatusBadRequest, err.Error())
			return
		}
		renderServiceError(c, err)
		return
	}
	prompts, err := ctrl.service.ListByFlow(request.Flow)
	if err != nil {
		renderServiceError(c, err)
		return
	}
	views.RenderPromptTable(c, request.Flow, prompts)
}

// Evaluate handles POST /prompts/evaluate — runs the selected prompts and renders results.
func (ctrl *PromptController) Evaluate(c *gin.Context) {
	request := &models.EvaluateForm{}
	if err := c.Bind(request); err != nil {
		views.RenderError(c, http.StatusBadRequest, "invalid request")
		return
	}
	results, err := ctrl.service.Evaluate(request.Flow, request.IDs)
	if err != nil {
		renderServiceError(c, err)
		return
	}
	views.RenderEvaluationResults(c, request.Flow, results)
}

// Export handles POST /prompts/export — writes the CSV and renders the saved path.
func (ctrl *PromptController) Export(c *gin.Context) {
	request := &models.EvaluateForm{}
	if err := c.Bind(request); err != nil {
		views.RenderError(c, http.StatusBadRequest, "invalid request")
		return
	}
	path, err := ctrl.service.ExportCSV(request.Flow, request.IDs)
	if err != nil {
		renderServiceError(c, err)
		return
	}
	views.RenderExportResult(c, path)
}