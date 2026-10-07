package finance

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"msg-classifier/internal/models"
	"msg-classifier/internal/ptbr"
	"msg-classifier/internal/repository"
)

// FinanceService handles the finance category use cases: register a transaction and
// search transactions or totals.
type FinanceService struct {
	extractor *FinanceExtractor
	repo      *repository.TransactionRepository
}

func NewService(extractor *FinanceExtractor, repo *repository.TransactionRepository) *FinanceService {
	return &FinanceService{extractor: extractor, repo: repo}
}

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
	amount, hasAmount := ptbr.ParseAmount(content)
	if !hasAmount {
		return financeNoData(classification, "o valor"), nil
	}

	now := time.Now()
	date, hasDate := ptbr.ParseEventDate(content, now)
	if !hasDate {
		date = ptbr.StartOfDay(now)
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

	var existing *models.Transaction
	if request.DupAction != "new" {
		existing, err = s.repo.FindDuplicate(transaction.Type, transaction.Amount, transaction.Date, transaction.Party)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("failed to check duplicate transaction: %w", err)
		}
	}
	if existing != nil {
		if request.DupAction == "update" {
			existing.Content = transaction.Content
			if transaction.Party != "" {
				existing.Party = transaction.Party
			}
			if err := s.repo.Save(existing); err != nil {
				return nil, fmt.Errorf("failed to persist transaction: %w", err)
			}
			return &models.UseCaseOutcome{
				Classification: classification,
				Action:         models.ActionTransactionAdd,
				Transactions:   []*models.Transaction{existing},
				Segments:       result.Segments,
			}, nil
		}
		return &models.UseCaseOutcome{
			Classification: classification,
			Action:         models.ActionTransactionDuplicate,
			Transactions:   []*models.Transaction{existing},
			Segments:       result.Segments,
		}, nil
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
