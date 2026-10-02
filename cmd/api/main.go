package main

import (
	"html/template"
	"log"

	"msg-classifier/internal/config"
	"msg-classifier/internal/controllers"
	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"
	"msg-classifier/internal/services"
	"msg-classifier/internal/services/contact"
	"msg-classifier/internal/services/finance"
	"msg-classifier/internal/services/notes"
	"msg-classifier/internal/views"
	"msg-classifier/pkg/jev"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
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
	if err := db.AutoMigrate(&models.Contact{}, &models.JevPrompt{}, &models.Note{}, &models.TodoItem{}, &models.Transaction{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	classifier := services.NewClassificationService(jevClient)
	extractor := contact.NewExtractor(jevClient)
	noteExtractor := notes.NewExtractor(jevClient)
	financeExtractor := finance.NewExtractor(jevClient)

	contactRepo := repository.NewContactRepository(db)
	notesRepo := repository.NewNotesRepository(db)
	transactionsRepo := repository.NewTransactionRepository(db)

	contactService := contact.NewService(extractor, contactRepo)
	if err := contactService.BackfillNameNorm(); err != nil {
		log.Fatalf("failed to backfill name_norm: %v", err)
	}
	notesService := notes.NewService(noteExtractor, notesRepo)
	financeService := finance.NewService(financeExtractor, transactionsRepo)
	dispatcher := services.NewDispatcher(map[string]services.CategoryHandler{
		"contact": contactService,
		"notes":   notesService,
		"finance": financeService,
	})

	promptRepo := repository.NewPromptRepository(db)
	promptService := services.NewPromptService(promptRepo, classifier, extractor, noteExtractor, financeExtractor)
	promptController := controllers.NewPromptController(promptService)

	webController := controllers.NewWebController()
	messageController := controllers.NewMessageController(classifier, dispatcher)

	dataService := services.NewDataService(contactRepo, notesRepo, transactionsRepo)
	dataController := controllers.NewDataController(dataService)

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
