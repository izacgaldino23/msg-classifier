package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"
)

// financeTypeWords map a PT-BR word to the transaction type it names. They are also
// dropped from the search term, so "mostre minhas compras" filters by type instead of
// searching for the word "compras" inside the message text.
var financeTypeWords = map[string]string{
	"compra": models.TransactionTypePurchase, "compras": models.TransactionTypePurchase,
	"comprei": models.TransactionTypePurchase, "compre": models.TransactionTypePurchase,
	"venda": models.TransactionTypeSale, "vendas": models.TransactionTypeSale,
	"vendi":     models.TransactionTypeSale,
	"pagamento": models.TransactionTypePayment, "pagamentos": models.TransactionTypePayment,
	"paguei": models.TransactionTypePayment, "pagar": models.TransactionTypePayment,
	"recebimento": models.TransactionTypeReceipt, "recebimentos": models.TransactionTypeReceipt,
	"recebi": models.TransactionTypeReceipt, "receber": models.TransactionTypeReceipt,
	"transferencia": models.TransactionTypeTransfer, "transferencias": models.TransactionTypeTransfer,
	"transferi": models.TransactionTypeTransfer,
}

// The search term is the establishment or person the message names, read with the same
// candidate extractor the add path uses: "quanto gastei no mercado esse mês" searches
// "mercado". Deriving it from leftover words instead would drag in grammar the stopword
// list does not know ("liste os pagamentos" → "liste"), and a message that names no place
// is a type or period search, not a text one.
func financeSearchTerm(content string) string {
	return partyCandidateFrom(content).Raw
}

// totalMarkers are the words that turn a search into a sum.
var totalMarkers = []string{"quanto", "total", "soma", "gastei", "gasto", "custo", "custou"}

// Get searches stored transactions (require flow). The period, the type and the term
// compose, because "quanto gastei com mercado esse mês" is all three at once. When the
// message asks how much, the outcome also carries the sum of what was found.
func (s *FinanceService) Get(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	content := request.Message
	normalized := normalizeName(content)

	filter := repository.TransactionFilter{Type: transactionTypeFromMessage(normalized)}
	if from, until, ok := ParseRange(content, time.Now()); ok {
		filter.From, filter.Until = from, until
	}
	filter.Term = financeSearchTerm(content)

	if filter.Type == "" && filter.From.IsZero() && filter.Term == "" {
		return financeNoData(classification, "um período, um tipo ou um estabelecimento"), nil
	}

	transactions, err := s.repo.Find(filter)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("failed to search transactions: %w", err)
	}
	// ponytail: a multi-word term that misses is more likely grammar than a name ("quais
	// compras eu fiz" → "quais fiz"), so the explicit type and period answer instead. A
	// single word that misses stays a miss, otherwise "quanto gastei no parque ontem?"
	// would come back with yesterday's padaria. A classify-the-intent Jev question is
	// the way past this.
	if len(transactions) == 0 && len(strings.Fields(filter.Term)) > 1 {
		relaxed := filter
		relaxed.Term = ""
		transactions, err = s.repo.Find(relaxed)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("failed to search transactions: %w", err)
		}
		if len(transactions) > 0 {
			filter = relaxed
		}
	}
	if len(transactions) == 0 {
		return &models.UseCaseOutcome{
			Classification: classification,
			Action:         models.ActionTransactionNotFound,
			SearchTerm:     filterLabel(filter),
		}, nil
	}

	outcome := &models.UseCaseOutcome{
		Classification: classification,
		Action:         models.ActionTransactionFound,
		Transactions:   transactions,
		SearchTerm:     filterLabel(filter),
	}
	if hasTotalMarker(normalized) {
		total, err := s.repo.Sum(filter)
		if err != nil {
			return nil, fmt.Errorf("failed to sum transactions: %w", err)
		}
		outcome.Total = total
	}
	return outcome, nil
}

// transactionTypeFromMessage returns the type the message names, or "" when it names
// none ("quanto gastei no mercado" is a party search, not a type search).
func transactionTypeFromMessage(normalized string) string {
	for _, word := range strings.Fields(normalized) {
		if transactionType, ok := financeTypeWords[stripPunctuation(word)]; ok {
			return transactionType
		}
	}
	return ""
}

// filterLabel describes the filter for the result partial: the term when there is one,
// then the type, then the period the message asked for.
func filterLabel(filter repository.TransactionFilter) string {
	if filter.Term != "" {
		return filter.Term
	}
	if filter.Type != "" {
		return filter.Type
	}
	if filter.From.IsZero() || filter.Until.IsZero() {
		return ""
	}
	last := filter.Until.AddDate(0, 0, -1)
	if filter.From.Equal(last) {
		return filter.From.Format(dateLayout)
	}
	return filter.From.Format(dateLayout) + " a " + last.Format(dateLayout)
}

// hasTotalMarker reports whether the message asks how much was spent.
func hasTotalMarker(normalized string) bool {
	for _, marker := range totalMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}