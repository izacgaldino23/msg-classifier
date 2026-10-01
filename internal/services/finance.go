package services

import (
	"fmt"
	"strings"
	"time"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"
)

// FinanceService handles the finance category use cases: register a transaction and
// search transactions or totals.
type FinanceService struct {
	extractor *FinanceExtractor
	repo      *repository.TransactionRepository
}

func NewFinanceService(extractor *FinanceExtractor, repo *repository.TransactionRepository) *FinanceService {
	return &FinanceService{extractor: extractor, repo: repo}
}

// compile-time assertion that FinanceService satisfies the CategoryHandler seam.
var _ CategoryHandler = (*FinanceService)(nil)

// Handle routes finance messages to the add or the get use case.
func (s *FinanceService) Handle(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	if classification.Kind.Choice == "require" {
		return s.Get(request, classification)
	}
	return s.Add(request, classification)
}

// Add persists a transaction. The amount is the only mandatory field: without it the
// message is not a transaction, so it answers a no-data outcome naming what is
// missing, and the user appends it to the same message and sends it again. A missing
// date means today — the user is telling us about something that just happened.
func (s *FinanceService) Add(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	content := strings.TrimSpace(request.Message)
	if content == "" {
		return financeNoData(classification, "mensagem"), nil
	}
	amount, hasAmount := ParseAmount(content)
	if !hasAmount {
		return financeNoData(classification, "o valor"), nil
	}

	now := time.Now()
	date, hasDate := ParseEventDate(content, now)
	if !hasDate {
		date = startOfDay(now)
	}

	result, err := s.extractor.Extract(content)
	if err != nil {
		return nil, err
	}

	transaction := &models.Transaction{
		Type:    result.Type,
		Amount:  amount,
		Date:    date,
		Party:   result.Party,
		Content: content,
	}
	if err := s.repo.Create(transaction); err != nil {
		return nil, fmt.Errorf("failed to persist transaction: %w", err)
	}
	return &models.UseCaseOutcome{
		Classification: classification,
		Action:         models.ActionTransactionAdd,
		Transactions:   []*models.Transaction{transaction},
		Segments:       result.Segments,
	}, nil
}

// financeNoData is the shared "cannot classify this message" outcome of the add path.
func financeNoData(classification *models.Classification, missing string) *models.UseCaseOutcome {
	return &models.UseCaseOutcome{
		Classification: classification,
		Action:         models.ActionTransactionNoData,
		Missing:        missing,
	}
}
