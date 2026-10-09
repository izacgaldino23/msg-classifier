package services

import (
	"errors"
	"fmt"
	"strings"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"
)

// ErrInvalidPrompt marks invalid Add input (unknown flow or empty fields) and
// invalid prompt-file rows; controllers map it to HTTP 400.
var ErrInvalidPrompt = errors.New("invalid prompt")

// PromptService owns the persisted side of the validation harness (Add,
// ListByFlow, selection + export). The evaluation itself lives in
// PromptEvaluator, which has no repository and opens no database so the same
// flows run from a file (cmd/prompts) without one.
type PromptService struct {
	repo *repository.PromptRepository
	eval *PromptEvaluator
}

func NewPromptService(repo *repository.PromptRepository, eval *PromptEvaluator) *PromptService {
	return &PromptService{repo: repo, eval: eval}
}

// validatePrompt is the one validity rule shared by the web Add form and the
// prompt-file loader: a known flow and non-empty message and expected result.
func validatePrompt(flow, message, expected string) error {
	if flow != models.FlowClassification && flow != models.FlowName && flow != models.FlowNote && flow != models.FlowFinance {
		return fmt.Errorf("%w: flow %q", ErrInvalidPrompt, flow)
	}
	if strings.TrimSpace(message) == "" || strings.TrimSpace(expected) == "" {
		return fmt.Errorf("%w: message and expected result are required", ErrInvalidPrompt)
	}
	return nil
}

// Add validates the input and persists a new prompt.
func (s *PromptService) Add(flow, message, expected string) (*models.JevPrompt, error) {
	if err := validatePrompt(flow, message, expected); err != nil {
		return nil, err
	}
	prompt := &models.JevPrompt{Flow: flow, Message: message, ExpectedResult: expected}
	if err := s.repo.Create(prompt); err != nil {
		return nil, fmt.Errorf("failed to persist prompt: %w", err)
	}
	return prompt, nil
}

// ListByFlow returns the prompts for the given flow, ordered by id.
func (s *PromptService) ListByFlow(flow string) ([]models.JevPrompt, error) {
	prompts, err := s.repo.ListByFlow(flow)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}
	return prompts, nil
}

// Evaluate uses the PromptService's flow list but delegates the actual run to
// the shared evaluator. The web harness keeps loading from the database; the
// file runner goes straight to the evaluator.
func (s *PromptService) Evaluate(flow string, ids []uint) ([]models.EvaluationResult, error) {
	prompts, err := s.ListByFlow(flow)
	if err != nil {
		return nil, err
	}

	selected := make(map[uint]bool, len(ids))
	for _, id := range ids {
		selected[id] = true
	}

	picked := make([]models.JevPrompt, 0, len(prompts))
	for _, prompt := range prompts {
		if selected[prompt.ID] {
			picked = append(picked, prompt)
		}
	}
	return s.eval.Evaluate(picked), nil
}

// ExportCSV delegates to the evaluator; the web export keeps its shape.
func (s *PromptService) ExportCSV(flow string, results []models.EvaluationResult) (string, error) {
	return s.eval.ExportCSV(flow, results)
}