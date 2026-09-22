package main

import (
	"html/template"

	"msg-classifier/internal/config"
	"msg-classifier/internal/controllers"
	"msg-classifier/internal/services"
	"msg-classifier/pkg/jev"

	"github.com/donseba/go-htmx"
	"github.com/gin-gonic/gin"
)

const (
	SourcePath = "web/templates"
)

func main() {
	router := gin.Default()

	tmpl := template.Must(template.ParseGlob(SourcePath + "/**/*.html"))
	router.SetHTMLTemplate(tmpl)

	// Single htmx instance: created once, used by the middleware. Controllers
	// reach the per-request *htmx.Handler through the context (key "htmx").
	h := htmx.New()

	router.Use(func(c *gin.Context) {
		c.Set("htmx", h.NewHandler(c.Writer, c.Request))
		c.Next()
	})

	// Composition root wiring: config → jev client → service → controllers → routes.
	// config.GetEnv() is the single godotenv load site (duplicate init removed).
	env := config.GetEnv()
	jevClient := jev.NewClient(env.TypesafeApiUrl, env.TypesafeToken, env.TypesafeModel)
	classifier := services.NewClassificationService(jevClient)

	webController := controllers.NewWebController()
	messageController := controllers.NewMessageController(classifier)

	router.GET("/", webController.Home)

	api := router.Group("/api")
	api.POST("/message", messageController.ReceiveMessage)

	_ = router.Run(":8080")
}
