package finance

import (
	"testing"
	"time"

	"msg-classifier/internal/models"
	"msg-classifier/internal/ptbr"
	"msg-classifier/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinanceServiceAddDuplicateThenNew(t *testing.T) {
	db := newFinanceTestDB(t)
	service := NewService(NewExtractor(&mockJevRequester{resp: financeResponse(models.TransactionTypePurchase, 0.95)}),
		repository.NewTransactionRepository(db))
	msg := &models.ReceiveMessageRequest{Message: "comprei 20 reais no mercado"}

	first, err := service.Handle(msg, financeClassification())
	require.NoError(t, err)
	require.Equal(t, models.ActionTransactionAdd, first.Action)

	dup, err := service.Handle(msg, financeClassification())
	require.NoError(t, err)
	assert.Equal(t, models.ActionTransactionDuplicate, dup.Action)
	require.Len(t, dup.Transactions, 1)
	assert.Equal(t, first.Transactions[0].ID, dup.Transactions[0].ID)

	var count int64
	require.NoError(t, db.Model(&models.Transaction{}).Count(&count).Error)
	assert.Equal(t, int64(1), count, "a duplicate hit persists nothing")

	again, err := service.Handle(&models.ReceiveMessageRequest{Message: msg.Message, DupAction: "new"}, financeClassification())
	require.NoError(t, err)
	assert.Equal(t, models.ActionTransactionAdd, again.Action)
	require.NoError(t, db.Model(&models.Transaction{}).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

func TestFinanceServiceAddDuplicateUpdateMerges(t *testing.T) {
	db := newFinanceTestDB(t)
	seeded := &models.Transaction{
		Type: models.TransactionTypePurchase, Amount: 20,
		Date:    ptbr.StartOfDay(time.Now()),
		Party:   "mercado",
		Content: "conteúdo antigo",
	}
	require.NoError(t, db.Create(seeded).Error)

	service := NewService(NewExtractor(&mockJevRequester{resp: financeResponse(models.TransactionTypePurchase, 0.95)}),
		repository.NewTransactionRepository(db))

	outcome, err := service.Handle(&models.ReceiveMessageRequest{
		Message: "comprei 20 reais no mercado", DupAction: "update",
	}, financeClassification())
	require.NoError(t, err)
	assert.Equal(t, models.ActionTransactionAdd, outcome.Action)
	require.Len(t, outcome.Transactions, 1)
	merged := outcome.Transactions[0]
	assert.Equal(t, seeded.ID, merged.ID, "update keeps the existing row")
	assert.Equal(t, "comprei 20 reais no mercado", merged.Content, "the pending content overwrites")
	assert.Equal(t, "mercado", merged.Party)

	var count int64
	require.NoError(t, db.Model(&models.Transaction{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestFinanceServiceAddDuplicateUpdateMissFallsBackToCreate(t *testing.T) {
	db := newFinanceTestDB(t)
	service := NewService(NewExtractor(&mockJevRequester{resp: financeResponse(models.TransactionTypePurchase, 0.95)}),
		repository.NewTransactionRepository(db))

	outcome, err := service.Handle(&models.ReceiveMessageRequest{
		Message: "comprei 20 reais no mercado", DupAction: "update",
	}, financeClassification())
	require.NoError(t, err)
	assert.Equal(t, models.ActionTransactionAdd, outcome.Action, "update on a miss creates")

	var count int64
	require.NoError(t, db.Model(&models.Transaction{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
