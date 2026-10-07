// Package app is the composition root shared by every entrypoint: config → Jev
// client → DB → repositories → services → dispatcher. The three binaries differ
// only in how they present a UseCaseOutcome, so the wiring lives here once
// instead of drifting in three copies.
package app

import (
	"fmt"
	"strings"

	"msg-classifier/internal/config"
	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"
	"msg-classifier/internal/services"
	"msg-classifier/internal/services/contact"
	"msg-classifier/internal/services/finance"
	"msg-classifier/internal/services/notes"
	"msg-classifier/pkg/jev"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// App carries what every entrypoint needs: classify a message, dispatch it, and
// browse what the flows persisted.
type App struct {
	Classifier *services.ClassificationService
	Dispatcher *services.Dispatcher
	Data       *services.DataService
	Prompts    *services.PromptService
}

// New wires the whole core over dbPath. dbPath is a parameter rather than a config
// read so a test can pass ":memory:"; the Jev credentials still come from config
// because only the request path needs them and New never makes a call.
func New(dbPath string) (*App, error) {
	env := config.GetEnv()
	jevClient := jev.NewClient(env.TypesafeApiUrl, env.TypesafeToken, env.TypesafeModel)

	db, err := gorm.Open(sqlite.Open(dsn(dbPath)), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database %q: %w", dbPath, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database pool: %w", err)
	}
	// Single connection serializes SQLite access; concurrent connections are
	// unstable with the pure-Go driver (glebarez/modernc) on Windows.
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.Contact{}, &models.JevPrompt{}, &models.Note{}, &models.TodoItem{}, &models.Transaction{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	classifier := services.NewClassificationService(jevClient)
	extractor := contact.NewExtractor(jevClient)
	noteExtractor := notes.NewExtractor(jevClient)
	financeExtractor := finance.NewExtractor(jevClient)

	contactRepo := repository.NewContactRepository(db)
	notesRepo := repository.NewNotesRepository(db)
	transactionsRepo := repository.NewTransactionRepository(db)

	contactService := contact.NewService(extractor, contactRepo)
	// Idempotent and run once per process: all three entries read the same rows and
	// write the same values, and SQLite serialises the writes.
	if err := contactService.BackfillNameNorm(); err != nil {
		return nil, fmt.Errorf("failed to backfill name_norm: %w", err)
	}

	return &App{
		Classifier: classifier,
		Dispatcher: services.NewDispatcher(map[string]services.CategoryHandler{
			"contact": contactService,
			"notes":   notes.NewService(noteExtractor, notesRepo),
			"finance": finance.NewService(financeExtractor, transactionsRepo),
		}),
		Data: services.NewDataService(contactRepo, notesRepo, transactionsRepo),
		Prompts: services.NewPromptService(
			repository.NewPromptRepository(db), classifier, extractor, noteExtractor, financeExtractor,
		),
	}, nil
}

// dsn adds the two pragmas that make several processes safe on one SQLite file:
// busy_timeout turns a lock collision into a 5s wait instead of an error, and WAL
// lets a reader work while a writer commits. Without them the cost of three
// processes is an intermittent "database is locked" in production rather than a
// failing test. An in-memory DSN is returned untouched so tests keep the shape
// they already use.
func dsn(path string) string {
	if strings.Contains(path, ":memory:") {
		return path
	}
	return "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
}
