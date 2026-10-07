package main

import (
	"html/template"
	"log"

	"msg-classifier/internal/app"
	"msg-classifier/internal/config"
	"msg-classifier/internal/controllers"
	"msg-classifier/internal/views"

	"github.com/gin-gonic/gin"
)

const (
	SourcePath = "web/templates"
)

func main() {
	router := gin.Default()

	// Static assets (app.css) — CWD-relative, same convention as web/templates.
	router.Static("/static", "./web/static")

	// Templates: layouts+partials are shared; each page gets its own clone so
	// pages never collide on page:title/page:content block names.
	tmpl := template.Must(template.New("").Funcs(views.FuncMap).ParseGlob(SourcePath + "/layouts/*.html"))
	tmpl = template.Must(tmpl.ParseGlob(SourcePath + "/partial/*.html"))
	pagesRenderer, err := views.NewPagesRenderer(tmpl, map[string]string{
		views.HomePage:    SourcePath + "/pages/index.html",
		views.PromptsPage: SourcePath + "/pages/prompts.html",
		views.DataPage:    SourcePath + "/pages/data.html",
	})
	if err != nil {
		log.Fatalf("failed to build page templates: %v", err)
	}
	router.HTMLRender = pagesRenderer

	application, err := app.New(config.GetEnv().DBPath)
	if err != nil {
		log.Fatalf("failed to build application: %v", err)
	}

	promptController := controllers.NewPromptController(application.Prompts)
	webController := controllers.NewWebController()
	messageController := controllers.NewMessageController(application.Classifier, application.Dispatcher)
	dataController := controllers.NewDataController(application.Data)

	router.GET("/", webController.Home)
	router.GET("/prompts", promptController.Page)
	router.GET("/prompts/table", promptController.Table)
	router.POST("/prompts", promptController.Add)
	router.POST("/prompts/evaluate", promptController.Evaluate)

	// Data screen. The static segments are registered before the :kind/:id param
	// route so they are matched directly instead of being read as a kind.
	router.GET("/data", dataController.Page)
	router.GET("/data/table", dataController.Table)
	router.GET("/data/item-row", dataController.ItemRow)
	router.GET("/data/:kind/:id", dataController.Detail)
	router.POST("/data/:kind/:id", dataController.Update)
	router.POST("/data/delete", dataController.Delete)

	api := router.Group("/api")
	api.POST("/message", messageController.ReceiveMessage)

	_ = router.Run(":8080")
}
