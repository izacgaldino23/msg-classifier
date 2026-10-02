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

func financeRequireClassification() *models.Classification {
	return &models.Classification{
		Category: models.CategoryFinding{Choice: "finance", Confidence: 0.9},
		Kind:     models.KindFinding{Choice: "require", Confidence: 0.9},
	}
}

func seedTransactions(t *testing.T, repo *repository.TransactionRepository) {
	t.Helper()
	today := ptbr.StartOfDay(time.Now())
	rows := []*models.Transaction{
		{Type: models.TransactionTypePurchase, Amount: 50, Date: today, Party: "supermercado", Content: "mercado de hoje"},
		{Type: models.TransactionTypePurchase, Amount: 30, Date: today.AddDate(0, 0, -1), Party: "padaria", Content: "padaria de ontem"},
		{Type: models.TransactionTypePayment, Amount: 1200, Date: today.AddDate(0, 0, -2), Party: "aluguel", Content: "aluguel"},
	}
	for _, row := range rows {
		require.NoError(t, repo.Create(row))
	}
}

func newFinanceGetService(t *testing.T) (*FinanceService, *repository.TransactionRepository) {
	t.Helper()
	db := newFinanceTestDB(t)
	repo := repository.NewTransactionRepository(db)
	seedTransactions(t, repo)
	return NewService(NewExtractor(&mockJevRequester{}), repo), repo
}

func TestFinanceServiceGetFilters(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		wantCount int
		wantTotal float64
	}{
		{"period and party", "quanto gastei no supermercado esse mes", 1, 50},
		{"period only", "quando gastei ontem", 1, 30},
		{"type only", "mostre minhas compras", 2, 0},
		{"party only", "quanto gastei na padaria", 1, 30},
		{"type word is not a search term", "liste os pagamentos", 1, 0},
		{"nothing matches", "quanto gastei na padaria semana que vem", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _ := newFinanceGetService(t)

			outcome, err := service.Handle(&models.ReceiveMessageRequest{Message: tt.message}, financeRequireClassification())
			require.NoError(t, err)

			if tt.wantCount == 0 {
				assert.Equal(t, models.ActionTransactionNotFound, outcome.Action)
				return
			}
			assert.Equal(t, models.ActionTransactionFound, outcome.Action)
			assert.Len(t, outcome.Transactions, tt.wantCount)
			assert.Equal(t, tt.wantTotal, outcome.Total, "total of what was found")
		})
	}
}

// A search that is not a question about money must not compute a sum.
func TestFinanceServiceGetWithoutTotalMarker(t *testing.T) {
	service, _ := newFinanceGetService(t)

	outcome, err := service.Handle(&models.ReceiveMessageRequest{Message: "quais compras eu fiz"}, financeRequireClassification())
	require.NoError(t, err)

	assert.Equal(t, models.ActionTransactionFound, outcome.Action)
	assert.Zero(t, outcome.Total)
}

// "quanto gastei" with nothing extractable cannot be answered: the outcome must say
// what would make it answerable instead of returning everything.
func TestFinanceServiceGetWithoutFilter(t *testing.T) {
	service, _ := newFinanceGetService(t)

	outcome, err := service.Handle(&models.ReceiveMessageRequest{Message: "quanto gastei?"}, financeRequireClassification())
	require.NoError(t, err)

	assert.Equal(t, models.ActionTransactionNoData, outcome.Action)
	assert.NotEmpty(t, outcome.Missing)
	assert.Empty(t, outcome.Transactions)
}

// The sum must respect the same filter as the list, otherwise "quanto gastei no
// supermercado esse mês" would answer with the total of everything.
func TestFinanceServiceGetTotalRespectsTheFilter(t *testing.T) {
	service, repo := newFinanceGetService(t)

	outcome, err := service.Handle(&models.ReceiveMessageRequest{Message: "quanto gastei hoje"}, financeRequireClassification())
	require.NoError(t, err)

	assert.Equal(t, models.ActionTransactionFound, outcome.Action)
	assert.Equal(t, 50.0, outcome.Total)

	total, err := repo.Sum(repository.TransactionFilter{Party: "supermercado"})
	require.NoError(t, err)
	assert.Equal(t, 50.0, total)
}

func TestFinanceServiceGetFilterLabel(t *testing.T) {
	// "esse mês" spans whole days: ParseRange answers the first through the last
	// day of the current month, not a month counted from today.
	monthFirst := ptbr.StartOfDay(time.Now())
	monthFirst = time.Date(monthFirst.Year(), monthFirst.Month(), 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		message string
		want    string
	}{
		{"term wins", "quanto gastei no supermercado", "supermercado"},
		{"type when no term", "mostre os pagamentos", "pagamento"},
		{"period when nothing else", "quanto gastei ontem", ptbr.StartOfDay(time.Now()).AddDate(0, 0, -1).Format(ptbr.DateLayout)},
		{"period of many days", "quanto gastei esse mes", monthFirst.Format(ptbr.DateLayout) + " a " + monthFirst.AddDate(0, 1, -1).Format(ptbr.DateLayout)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _ := newFinanceGetService(t)

			outcome, err := service.Handle(&models.ReceiveMessageRequest{Message: tt.message}, financeRequireClassification())
			require.NoError(t, err)

			assert.Equal(t, tt.want, outcome.SearchTerm)
		})
	}
}
