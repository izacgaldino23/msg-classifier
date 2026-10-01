package services

import (
	"errors"
	"testing"
	"time"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"
	"msg-classifier/pkg/jev"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newFinanceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "gorm.Open()")
	sqlDB, err := db.DB()
	require.NoError(t, err, "db.DB()")
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&models.Transaction{}), "AutoMigrate()")
	return db
}

func financeClassification() *models.Classification {
	return &models.Classification{
		Category: models.CategoryFinding{Choice: "finance", Confidence: 0.9},
		Kind:     models.KindFinding{Choice: "add", Confidence: 0.9},
	}
}

func TestFinanceServiceAdd(t *testing.T) {
	tests := []struct {
		name    string
		message string
		resp    *jev.JevResponse
		want    models.Transaction
	}{
		{
			name:    "amount date and shop",
			message: "Comprei pão, leite e ovos no supermercado. O total foi de 50 reais.",
			resp:    financeResponse(models.TransactionTypePurchase, 0.95),
			want: models.Transaction{
				Type:    models.TransactionTypePurchase,
				Amount:  50,
				Date:    startOfDay(time.Now()),
				Party:   "supermercado",
				Content: "Comprei pão, leite e ovos no supermercado. O total foi de 50 reais.",
			},
		},
		{
			name:    "installment keeps the installment amount",
			message: "Compra parcelada em 3 vezes de 300 reais no cartão",
			resp:    financeResponse(models.TransactionTypePurchase, 0.8),
			want: models.Transaction{
				Type:    models.TransactionTypePurchase,
				Amount:  300,
				Date:    startOfDay(time.Now()),
				Party:   "cartão",
				Content: "Compra parcelada em 3 vezes de 300 reais no cartão",
			},
		},
		{
			name:    "absolute date",
			message: "Paguei 1200 reais do aluguel em 15/10/2023",
			resp:    financeResponse(models.TransactionTypePayment, 0.9),
			want: models.Transaction{
				Type:    models.TransactionTypePayment,
				Amount:  1200,
				Date:    time.Date(2023, 10, 15, 0, 0, 0, 0, time.UTC),
				Party:   "aluguel",
				Content: "Paguei 1200 reais do aluguel em 15/10/2023",
			},
		},
		{
			name:    "period collapses to the last day of the period",
			message: "Gastei 80 reais na padaria semana passada",
			resp:    financeResponse(models.TransactionTypePurchase, 0.9),
			want: models.Transaction{
				Type:    models.TransactionTypePurchase,
				Amount:  80,
				Date:    startOfDay(time.Now()).AddDate(0, 0, -1),
				Party:   "padaria",
				Content: "Gastei 80 reais na padaria semana passada",
			},
		},
		{
			name:    "no date means today",
			message: "comprei 20 reais no mercado",
			resp:    financeResponse(models.TransactionTypePurchase, 0.9),
			want: models.Transaction{
				Type:    models.TransactionTypePurchase,
				Amount:  20,
				Date:    startOfDay(time.Now()),
				Party:   "mercado",
				Content: "comprei 20 reais no mercado",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newFinanceTestDB(t)
			service := NewFinanceService(NewFinanceExtractor(&mockJevRequester{resp: tt.resp}), repository.NewTransactionRepository(db))

			outcome, err := service.Handle(&models.ReceiveMessageRequest{Message: tt.message}, financeClassification())
			require.NoError(t, err)

			assert.Equal(t, models.ActionTransactionAdd, outcome.Action)
			require.Len(t, outcome.Transactions, 1)
			got := outcome.Transactions[0]
			assert.Equal(t, tt.want.Type, got.Type)
			assert.Equal(t, tt.want.Amount, got.Amount)
			assert.Equal(t, tt.want.Party, got.Party)
			assert.Equal(t, tt.want.Content, got.Content)
			assert.True(t, tt.want.Date.Equal(got.Date), "date = %s, want %s", got.Date, tt.want.Date)
			assert.NotZero(t, got.ID, "the transaction must be persisted")

			var stored models.Transaction
			require.NoError(t, db.First(&stored, got.ID).Error)
			assert.Equal(t, got.Amount, stored.Amount)
		})
	}
}

// Without an amount the message is not a transaction: the answer must name what is
// missing so the user can append it and send the same message again.
func TestFinanceServiceAddWithoutAmount(t *testing.T) {
	db := newFinanceTestDB(t)
	mock := &mockJevRequester{}
	service := NewFinanceService(NewFinanceExtractor(mock), repository.NewTransactionRepository(db))

	outcome, err := service.Handle(&models.ReceiveMessageRequest{Message: "comprei pão no supermercado"}, financeClassification())
	require.NoError(t, err)

	assert.Equal(t, models.ActionTransactionNoData, outcome.Action)
	assert.Equal(t, "o valor", outcome.Missing)
	assert.Empty(t, outcome.Transactions)
	assert.Zero(t, mock.calls, "an unclassifiable message must not spend a Jev call")

	var count int64
	require.NoError(t, db.Model(&models.Transaction{}).Count(&count).Error)
	assert.Zero(t, count, "nothing must be persisted")
}

func TestFinanceServiceAddPropagatesUpstreamError(t *testing.T) {
	db := newFinanceTestDB(t)
	mock := &mockJevRequester{err: errors.New("boom")}
	service := NewFinanceService(NewFinanceExtractor(mock), repository.NewTransactionRepository(db))

	_, err := service.Handle(&models.ReceiveMessageRequest{Message: "comprei 50 reais"}, financeClassification())
	assert.ErrorIs(t, err, ErrUpstream)
}

// The repository failure must be wrapped with the service context, like every other
// persist path.
func TestFinanceServiceAddWrapsRepositoryError(t *testing.T) {
	db := newFinanceTestDB(t)
	require.NoError(t, db.Migrator().DropTable(&models.Transaction{}))
	mock := &mockJevRequester{resp: financeResponse(models.TransactionTypePurchase)}
	service := NewFinanceService(NewFinanceExtractor(mock), repository.NewTransactionRepository(db))

	_, err := service.Handle(&models.ReceiveMessageRequest{Message: "comprei 50 reais no mercado"}, financeClassification())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to persist transaction")
}
