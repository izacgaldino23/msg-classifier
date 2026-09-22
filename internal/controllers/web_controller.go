package controllers

import (
	"msg-classifier/internal/views"

	"github.com/gin-gonic/gin"
)

// WebController serves HTML page routes. HTTP concerns only — the
// page/partial switch lives in the view layer.
type WebController struct{}

// NewWebController returns a WebController.
func NewWebController() *WebController {
	return &WebController{}
}

// Home handles GET /.
func (ctrl *WebController) Home(c *gin.Context) {
	views.RenderPage(c)
}
