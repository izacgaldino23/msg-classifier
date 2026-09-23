package services

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"
)

// ErrInvalidPrompt marks invalid Add input (unknown flow or empty fields);
// controllers map it to HTTP 400.
var ErrInvalidPrompt = errors.New("invalid prompt")

// exportDir is where CSV exports are saved; a var (not const) so tests can
// redirect it to a temp dir. Production default: exports/ at the project root.
var exportDir = "exports"

// PromptService orchestrates the Jev validation harness. It reuses the exact
// production Jev paths (ClassificationService.Classify, ContactExtractor.ExtractName).
type PromptService struct {
	repo       *repository.PromptRepository
	classifier *ClassificationService
	extractor  *ContactExtractor
}

func NewPromptService(repo *repository.PromptRepository, classifier *ClassificationService, extractor *ContactExtractor) *PromptService {
	return &PromptService{repo: repo, classifier: classifier, extractor: extractor}
}

// Add validates the input and persists a new prompt.
func (s *PromptService) Add(flow, message, expected string) (*models.JevPrompt, error) {
	if flow != models.FlowClassification && flow != models.FlowName {
		return nil, fmt.Errorf("%w: flow %q", ErrInvalidPrompt, flow)
	}
	if strings.TrimSpace(message) == "" || strings.TrimSpace(expected) == "" {
		return nil, fmt.Errorf("%w: message and expected result are required", ErrInvalidPrompt)
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

// Evaluate runs the selected prompts against the real Jev flow and compares the
// obtained result with the expected one. A Jev failure for one prompt is captured
// in its row (obtained = error string, match = false) and evaluation continues.
func (s *PromptService) Evaluate(flow string, ids []uint) ([]models.EvaluationResult, error) {
	prompts, err := s.ListByFlow(flow)
	if err != nil {
		return nil, err
	}

	selected := make(map[uint]bool, len(ids))
	for _, id := range ids {
		selected[id] = true
	}

	results := make([]models.EvaluationResult, 0, len(prompts))
	for _, prompt := range prompts {
		if !selected[prompt.ID] {
			continue
		}
		results = append(results, s.evaluateOne(flow, prompt))
	}
	return results, nil
}

func (s *PromptService) evaluateOne(flow string, prompt models.JevPrompt) models.EvaluationResult {
	result := models.EvaluationResult{
		PromptID:       prompt.ID,
		Message:        prompt.Message,
		ExpectedResult: prompt.ExpectedResult,
	}

	switch flow {
	case models.FlowClassification:
		classification, err := s.classifier.Classify(&models.ReceiveMessageRequest{Message: prompt.Message})
		if err != nil {
			result.ObtainedResult = err.Error()
			result.Match = false
			return result
		}
		result.ObtainedResult = classification.Category.Choice + ":" + classification.Kind.Choice
		result.Match = strings.EqualFold(result.ObtainedResult, prompt.ExpectedResult)
	case models.FlowName:
		nameResult, err := s.extractor.ExtractName(prompt.Message, nil)
		if err != nil {
			result.ObtainedResult = err.Error()
			result.Match = false
			return result
		}
		result.ObtainedResult = nameResult.Name
		result.Segments = nameResult.Segments
		result.Match = normalizeName(prompt.ExpectedResult) == normalizeName(nameResult.Name)
	}
	return result
}

// ExportCSV writes the given evaluation results to
// exports/<flow>-<yyyyMMdd-HHmmss>.csv (folder created on demand), returning the
// path. It does not re-run the evaluation — the CSV mirrors what was evaluated.
func (s *PromptService) ExportCSV(flow string, results []models.EvaluationResult) (string, error) {
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create export dir: %w", err)
	}

	path := filepath.Join(exportDir, fmt.Sprintf("%s-%s.csv", flow, time.Now().Format("20060102-150405")))
	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("failed to create export file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"id", "message", "expected_result", "obtained_result", "match", "executed_at"}); err != nil {
		return "", fmt.Errorf("failed to write csv header: %w", err)
	}
	executedAt := time.Now().Format(time.RFC3339)
	for _, result := range results {
		record := []string{
			fmt.Sprintf("%d", result.PromptID),
			result.Message,
			result.ExpectedResult,
			result.ObtainedResult,
			fmt.Sprintf("%t", result.Match),
			executedAt,
		}
		if err := writer.Write(record); err != nil {
			return "", fmt.Errorf("failed to write csv row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("failed to flush csv: %w", err)
	}
	return path, nil
}