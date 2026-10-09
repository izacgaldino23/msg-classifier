package services

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"msg-classifier/internal/jevq"
	"msg-classifier/internal/models"
	"msg-classifier/internal/ptbr"
	"msg-classifier/internal/services/contact"
	"msg-classifier/internal/services/finance"
	"msg-classifier/internal/services/notes"
)

// exportDir is where CSV exports are saved; a var (not const) so tests can
// redirect it to a temp dir. Production default: exports/ at the project root.
var exportDir = "exports"

// PromptEvaluator runs the exact production Jev paths (ClassificationService,
// ContactExtractor, NoteExtractor, FinanceExtractor) over a list of prompts and
// compares obtained vs expected. It deliberately has no repository and opens no
// database, so the web harness (PromptService) and the file runner (cmd/prompts)
// share the same evaluation.
type PromptEvaluator struct {
	classifier       *ClassificationService
	extractor        *contact.ContactExtractor
	noteExtractor    *notes.NoteExtractor
	financeExtractor *finance.FinanceExtractor
}

func NewPromptEvaluator(classifier *ClassificationService, extractor *contact.ContactExtractor, noteExtractor *notes.NoteExtractor, financeExtractor *finance.FinanceExtractor) *PromptEvaluator {
	return &PromptEvaluator{classifier: classifier, extractor: extractor, noteExtractor: noteExtractor, financeExtractor: financeExtractor}
}

// Evaluate runs every prompt through the real Jev flow in order. A Jev failure
// for one prompt is captured in its row (obtained = error string, match = false)
// and evaluation continues.
func (e *PromptEvaluator) Evaluate(prompts []models.JevPrompt) []models.EvaluationResult {
	results := make([]models.EvaluationResult, 0, len(prompts))
	for _, prompt := range prompts {
		results = append(results, e.evaluateOne(prompt.Flow, prompt))
	}
	return results
}

func (e *PromptEvaluator) evaluateOne(flow string, prompt models.JevPrompt) models.EvaluationResult {
	result := models.EvaluationResult{
		PromptID:       prompt.ID,
		Message:        prompt.Message,
		ExpectedResult: prompt.ExpectedResult,
	}

	switch flow {
	case models.FlowClassification:
		classification, err := e.classifier.Classify(&models.ReceiveMessageRequest{Message: prompt.Message})
		if err != nil {
			result.ObtainedResult = err.Error()
			result.Match = false
			return result
		}
		result.ObtainedResult = classification.Category.Choice + ":" + classification.Kind.Choice
		result.Match = strings.EqualFold(result.ObtainedResult, prompt.ExpectedResult)
	case models.FlowName:
		// Mirror the production contact path: strip phone/email spans before the
		// name fan-out, otherwise the email/phone itself becomes a candidate
		// segment and can leak into the extracted name.
		spans := make([]jevq.Span, 0, 2)
		if _, span, ok := e.extractor.ExtractPhone(prompt.Message); ok {
			spans = append(spans, span)
		}
		if _, span, ok := e.extractor.ExtractEmail(prompt.Message); ok {
			spans = append(spans, span)
		}
		nameResult, err := e.extractor.ExtractName(prompt.Message, spans)
		if err != nil {
			result.ObtainedResult = err.Error()
			result.Match = false
			return result
		}
		result.ObtainedResult = nameResult.Name
		result.Segments = nameResult.Segments
		result.Match = ptbr.NormalizeName(prompt.ExpectedResult) == ptbr.NormalizeName(nameResult.Name)
	case models.FlowNote:
		noteType, err := e.noteExtractor.ExtractType(&models.ReceiveMessageRequest{Message: prompt.Message})
		if err != nil {
			result.ObtainedResult = err.Error()
			result.Match = false
			return result
		}
		result.ObtainedResult = noteType
		result.Match = strings.EqualFold(strings.TrimSpace(noteType), strings.TrimSpace(prompt.ExpectedResult))
	case models.FlowFinance:
		financeResult, err := e.financeExtractor.Extract(prompt.Message)
		if err != nil {
			result.ObtainedResult = err.Error()
			result.Match = false
			return result
		}
		result.ObtainedResult = financeResult.Type
		result.Segments = financeResult.Segments
		result.Match = strings.EqualFold(strings.TrimSpace(financeResult.Type), strings.TrimSpace(prompt.ExpectedResult))
	}
	return result
}

// ExportCSV writes the given evaluation results to
// exports/<flow>-<yyyyMMdd-HHmmss>.csv (folder created on demand), returning the
// path. It does not re-run the evaluation — the CSV mirrors what was evaluated.
// Used by the web harness; the file runner writes to an explicit -csv path via
// WriteResultsCSV.
func (e *PromptEvaluator) ExportCSV(flow string, results []models.EvaluationResult) (string, error) {
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create export dir: %w", err)
	}

	path := filepath.Join(exportDir, fmt.Sprintf("%s-%s.csv", flow, time.Now().Format("20060102-150405")))
	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("failed to create export file: %w", err)
	}
	defer file.Close()

	if err := WriteResultsCSV(file, results); err != nil {
		return "", err
	}
	return path, nil
}

// WriteResultsCSV writes the shared CSV layout (id, message, expected_result,
// obtained_result, match, executed_at).
func WriteResultsCSV(w io.Writer, results []models.EvaluationResult) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"id", "message", "expected_result", "obtained_result", "match", "executed_at"}); err != nil {
		return fmt.Errorf("failed to write csv header: %w", err)
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
			return fmt.Errorf("failed to write csv row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("failed to flush csv: %w", err)
	}
	return nil
}