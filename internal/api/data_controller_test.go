package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"
	"msg-classifier/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "gorm.Open()")
	sqlDB, err := db.DB()
	require.NoError(t, err, "db.DB()")
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&models.Contact{}, &models.Note{}, &models.TodoItem{}, &models.Transaction{}), "AutoMigrate()")
	return db
}

func getRoute(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := newTestDB(t)

	require.NoError(t, repository.NewContactRepository(db).Create(&models.Contact{
		Name: "Maria da Silva", NameNorm: "maria da silva", Phone: strPtr("9292929290"),
	}), "seed contact")

	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	require.NoError(t, repository.NewNotesRepository(db).Create(&models.Note{
		Type: models.NoteTypeReminder, Content: "pagar a conta de luz", Date: &date, Time: strPtr("14:30"),
	}, nil), "seed note")

	require.NoError(t, repository.NewTransactionRepository(db).Create(&models.Transaction{
		Type: models.TransactionTypePurchase, Amount: 1234.56, Date: date, Party: "supermercado",
		Content: "compras no supermercado",
	}), "seed transaction")

	ctrl := NewDataController(services.NewDataService(
		repository.NewContactRepository(db),
		repository.NewNotesRepository(db),
		repository.NewTransactionRepository(db),
	))

	router := gin.New()
	router.GET("/api/v1/contacts", ctrl.ListContacts)
	router.GET("/api/v1/notes", ctrl.ListNotes)
	router.GET("/api/v1/transactions", ctrl.ListTransactions)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

// Each browse endpoint must carry the row's CONTENT. An empty array and a
// swallowed error look identical from the outside, so assert on the fields.
func TestBrowseEndpointsCarryContent(t *testing.T) {
	t.Run("contacts", func(t *testing.T) {
		recorder := getRoute(t, "/api/v1/contacts")
		require.Equal(t, http.StatusOK, recorder.Code)
		var contacts []models.Contact
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &contacts), "body")
		require.Len(t, contacts, 1)
		assert.Equal(t, "Maria da Silva", contacts[0].Name)
		require.NotNil(t, contacts[0].Phone)
		assert.Equal(t, "9292929290", *contacts[0].Phone)
	})

	t.Run("notes", func(t *testing.T) {
		recorder := getRoute(t, "/api/v1/notes?filter=reminder")
		require.Equal(t, http.StatusOK, recorder.Code)
		var notes []models.Note
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &notes), "body")
		require.Len(t, notes, 1)
		assert.Equal(t, models.NoteTypeReminder, notes[0].Type)
		assert.Equal(t, "pagar a conta de luz", notes[0].Content)
		require.NotNil(t, notes[0].Time)
		assert.Equal(t, "14:30", *notes[0].Time)
	})

	t.Run("transactions", func(t *testing.T) {
		recorder := getRoute(t, "/api/v1/transactions?filter=compra&search=supermercado")
		require.Equal(t, http.StatusOK, recorder.Code)
		var transactions []models.Transaction
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &transactions), "body")
		require.Len(t, transactions, 1)
		assert.Equal(t, models.TransactionTypePurchase, transactions[0].Type)
		assert.InDelta(t, 1234.56, transactions[0].Amount, 0.001)
		assert.Equal(t, "supermercado", transactions[0].Party)
	})
}

// An empty result must be [] and not null — a client iterating a null array
// breaks. Defence in depth: gorm's Scan already hands back a non-nil slice
// (it MakeSlices the destination before the row loop), but the Find-based
// paths in this repo do return nil for "no rows", so the guards keep the []
// contract if List ever follows that shape.
func TestBrowseEndpointsReturnEmptyArray(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newTestDB(t) // one database for all three repos, as app.New() wires it
	ctrl := NewDataController(services.NewDataService(
		repository.NewContactRepository(db),
		repository.NewNotesRepository(db),
		repository.NewTransactionRepository(db),
	))
	router := gin.New()
	router.GET("/api/v1/contacts", ctrl.ListContacts)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/contacts", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, "[]", recorder.Body.String())
}

func TestBrowseRejectsAnUnknownFilter(t *testing.T) {
	recorder := getRoute(t, "/api/v1/contacts?filter=bogus")
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Filtro inválido.")
}
