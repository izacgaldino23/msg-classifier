package repository

import (
	"testing"
	"time"

	"msg-classifier/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newTransactionTestDB mirrors the package newTestDB but migrates the
// transactions schema.
func newTransactionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "gorm.Open()")
	sqlDB, err := db.DB()
	require.NoError(t, err, "db.DB()")
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&models.Transaction{}), "AutoMigrate()")
	return db
}

func txnDay(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// seedTransactions inserts a small month of spending so the filters have
// something to choose between.
func seedTransactions(t *testing.T, repo *TransactionRepository) {
	t.Helper()
	rows := []*models.Transaction{
		{Type: models.TransactionTypePurchase, Amount: 50, Date: txnDay(2026, 3, 2), Party: "supermercado", Content: "arroz, feijão"},
		{Type: models.TransactionTypePurchase, Amount: 30, Date: txnDay(2026, 3, 5), Party: "padaria", Content: "pão"},
		{Type: models.TransactionTypePayment, Amount: 1200, Date: txnDay(2026, 3, 10), Party: "Fulano", Content: "aluguel"},
		{Type: models.TransactionTypePurchase, Amount: 75.5, Date: txnDay(2026, 2, 20), Party: "supermercado", Content: "carne"},
	}
	for _, row := range rows {
		require.NoError(t, repo.Create(row))
	}
}

func TestTransactionRepositoryCreate(t *testing.T) {
	db := newTransactionTestDB(t)
	repo := NewTransactionRepository(db)

	transaction := &models.Transaction{
		Type:    models.TransactionTypePurchase,
		Amount:  50,
		Date:    txnDay(2026, 3, 15),
		Party:   "supermercado",
		Content: "arroz, feijão, alho",
	}
	require.NoError(t, repo.Create(transaction))
	assert.NotZero(t, transaction.ID)
	assert.NotZero(t, transaction.CreatedAt)
}

func TestTransactionRepositoryFindInRange(t *testing.T) {
	db := newTransactionTestDB(t)
	repo := NewTransactionRepository(db)
	seedTransactions(t, repo)

	// March only: the "until" bound is exclusive, so the 10/03 is inside.
	found, err := repo.Find(TransactionFilter{From: txnDay(2026, 3, 1), Until: txnDay(2026, 4, 1)})
	require.NoError(t, err)
	require.Len(t, found, 3)
	assert.Equal(t, 1200.0, found[0].Amount, "newest date first")
	assert.Equal(t, 30.0, found[1].Amount)
	assert.Equal(t, 50.0, found[2].Amount)

	// A single day.
	found, err = repo.Find(TransactionFilter{From: txnDay(2026, 3, 10), Until: txnDay(2026, 3, 11)})
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, "Fulano", found[0].Party)

	// No row in the range is ErrNotFound, not an empty slice.
	_, err = repo.Find(TransactionFilter{From: txnDay(2025, 1, 1), Until: txnDay(2025, 2, 1)})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestTransactionRepositoryFindByTypeAndTerm(t *testing.T) {
	db := newTransactionTestDB(t)
	repo := NewTransactionRepository(db)
	seedTransactions(t, repo)

	found, err := repo.Find(TransactionFilter{Type: models.TransactionTypePurchase})
	require.NoError(t, err)
	assert.Len(t, found, 3)

	// The term searches the content and the party.
	found, err = repo.Find(TransactionFilter{Term: "MERCA"})
	require.NoError(t, err)
	require.Len(t, found, 2)
	for _, transaction := range found {
		assert.Equal(t, "supermercado", transaction.Party)
	}

	found, err = repo.Find(TransactionFilter{Term: "feijão"})
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, 50.0, found[0].Amount)

	_, err = repo.Find(TransactionFilter{Term: "inexistente"})
	assert.ErrorIs(t, err, ErrNotFound)
}

// The term clause carries an OR, so the other clauses of the same filter must
// still apply — that is what the parentheses in scopeTransactions are for.
func TestTransactionRepositoryFilterCombines(t *testing.T) {
	db := newTransactionTestDB(t)
	repo := NewTransactionRepository(db)
	seedTransactions(t, repo)

	found, err := repo.Find(TransactionFilter{
		From:  txnDay(2026, 3, 1),
		Until: txnDay(2026, 4, 1),
		Type:  models.TransactionTypePurchase,
		Term:  "supermercado",
	})
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, 50.0, found[0].Amount, "the February row must stay out")

	_, err = repo.Find(TransactionFilter{Type: models.TransactionTypePurchase, Party: "Fulano"})
	assert.ErrorIs(t, err, ErrNotFound, "the payment row is excluded by type")
}

func TestTransactionRepositorySum(t *testing.T) {
	db := newTransactionTestDB(t)
	repo := NewTransactionRepository(db)
	seedTransactions(t, repo)

	total, err := repo.Sum(TransactionFilter{From: txnDay(2026, 3, 1), Until: txnDay(2026, 4, 1)})
	require.NoError(t, err)
	assert.Equal(t, 1280.0, total)

	total, err = repo.Sum(TransactionFilter{Term: "supermercado"})
	require.NoError(t, err)
	assert.Equal(t, 125.5, total)

	// A miss totals zero instead of erroring.
	total, err = repo.Sum(TransactionFilter{From: txnDay(2020, 1, 1), Until: txnDay(2020, 2, 1)})
	require.NoError(t, err)
	assert.Equal(t, 0.0, total)
}

func TestTransactionRepositoryListNeverNotFound(t *testing.T) {
	db := newTransactionTestDB(t)
	repo := NewTransactionRepository(db)

	listed, err := repo.List(TransactionFilter{})
	require.NoError(t, err, "an empty table is not a miss")
	assert.Empty(t, listed)

	seedTransactions(t, repo)
	listed, err = repo.List(TransactionFilter{})
	require.NoError(t, err)
	require.Len(t, listed, 4)
	assert.Equal(t, 75.5, listed[0].Amount, "newest id first, which is the February row")
}

func TestTransactionRepositorySaveAndDelete(t *testing.T) {
	db := newTransactionTestDB(t)
	repo := NewTransactionRepository(db)
	seedTransactions(t, repo)

	found, err := repo.Find(TransactionFilter{Term: "padaria"})
	require.NoError(t, err)
	require.Len(t, found, 1)

	found[0].Amount = 35
	found[0].Party = "padaria do bairro"
	require.NoError(t, repo.Save(found[0]))

	reloaded, err := repo.FindByID(found[0].ID)
	require.NoError(t, err)
	assert.Equal(t, 35.0, reloaded.Amount)
	assert.Equal(t, "padaria do bairro", reloaded.Party)

	_, err = repo.FindByID(999)
	assert.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, repo.DeleteByIDs([]uint{reloaded.ID}))
	_, err = repo.FindByID(reloaded.ID)
	assert.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, repo.DeleteByIDs(nil), "an empty list is a no-op")
}