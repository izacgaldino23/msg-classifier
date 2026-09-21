package handlers

import (
	"net/http"

	"github.com/donseba/go-htmx"
	"github.com/gin-gonic/gin"
)

type WebHandler struct {
	*htmx.HTMX
}

func NewWebHandler(htmx *htmx.HTMX) *WebHandler {
	return &WebHandler{htmx}
}

func (h *WebHandler) Home(c *gin.Context) {
	hCtx := c.MustGet("htmx").(*htmx.Handler)

	if hCtx.IsHxRequest() {
		// Renderiza apenas o bloco do conteúdo
		c.HTML(http.StatusOK, "page:content", nil)
		return
	}

	c.HTML(http.StatusOK, "base", nil)
}
