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

	// Single htmx instance; controllers read it from the context per request.
	h := htmx.New()

	router.Use(func(c *gin.Context) {
		c.Set("htmx", h.NewHandler(c.Writer, c.Request))
		c.Next()
	})

	// Composition root: config → jev client → services → dispatcher → controllers → routes.
	env := config.GetEnv()
	jevClient := jev.NewClient(env.TypesafeApiUrl, env.TypesafeToken, env.TypesafeModel)
	classifier := services.NewClassificationService(jevClient)

	contactService := services.NewContactService()
	dispatcher := services.NewDispatcher(map[string]services.CategoryHandler{
		"contact": contactService,
	})

	webController := controllers.NewWebController()
	messageController := controllers.NewMessageController(classifier, dispatcher)

	router.GET("/", webController.Home)

	api := router.Group("/api")
	api.POST("/message", messageController.ReceiveMessage)

	_ = router.Run(":8080")
}
