package controllers

import (
	"msg-classifier/internal/views"

	"github.com/gin-gonic/gin"
)

// WebController serves HTML page routes.
type WebController struct{}

func NewWebController() *WebController {
	return &WebController{}
}

// Home handles GET /.
func (ctrl *WebController) Home(c *gin.Context) {
	views.RenderPage(c, views.HomePage, views.HomePageContent)
}
