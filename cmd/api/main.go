package main

import (
	"html/template"
	"log"

	"msg-classifier/internal/config"
	"msg-classifier/internal/controllers"
	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"
	"msg-classifier/internal/services"
	"msg-classifier/internal/views"
	"msg-classifier/pkg/jev"

	"github.com/donseba/go-htmx"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

const (
	SourcePath = "web/templates"
)

func main() {
	router := gin.Default()

	// Templates: layouts+partials are shared; each page gets its own clone so
	// pages never collide on page:title/page:content block names.
	tmpl := template.Must(template.ParseGlob(SourcePath + "/layouts/*.html"))
	tmpl = template.Must(tmpl.ParseGlob(SourcePath + "/partial/*.html"))
	pagesRenderer, err := views.NewPagesRenderer(tmpl, map[string]string{
		views.HomePage:    SourcePath + "/pages/index.html",
		views.PromptsPage: SourcePath + "/pages/prompts.html",
	})
	if err != nil {
		log.Fatalf("failed to build page templates: %v", err)
	}
	router.HTMLRender = pagesRenderer

	// Single htmx instance; controllers read it from the context per request.
	h := htmx.New()

	router.Use(func(c *gin.Context) {
		c.Set("htmx", h.NewHandler(c.Writer, c.Request))
		c.Next()
	})

	// Composition root: config → jev client → services → dispatcher → controllers → routes.
	env := config.GetEnv()
	jevClient := jev.NewClient(env.TypesafeApiUrl, env.TypesafeToken, env.TypesafeModel)

	db, err := gorm.Open(sqlite.Open(env.DBPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database %q: %v", env.DBPath, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get database pool: %v", err)
	}
	// Single connection serializes SQLite access; concurrent connections are
	// unstable with the pure-Go driver (glebarez/modernc) on Windows.
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.Contact{}, &models.JevPrompt{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	classifier := services.NewClassificationService(jevClient)
	extractor := services.NewContactExtractor(jevClient)
	contactService := services.NewContactService(extractor, repository.NewContactRepository(db))
	if err := contactService.BackfillNameNorm(); err != nil {
		log.Fatalf("failed to backfill name_norm: %v", err)
	}
	dispatcher := services.NewDispatcher(map[string]services.CategoryHandler{
		"contact": contactService,
	})

	promptRepo := repository.NewPromptRepository(db)
	promptService := services.NewPromptService(promptRepo, classifier, extractor)
	promptController := controllers.NewPromptController(promptService)

	webController := controllers.NewWebController()
	messageController := controllers.NewMessageController(classifier, dispatcher)

	router.GET("/", webController.Home)
	router.GET("/prompts", promptController.Page)
	router.GET("/prompts/table", promptController.Table)
	router.POST("/prompts", promptController.Add)
	router.POST("/prompts/evaluate", promptController.Evaluate)
	router.POST("/prompts/export", promptController.Export)

	api := router.Group("/api")
	api.POST("/message", messageController.ReceiveMessage)

	_ = router.Run(":8080")
}
