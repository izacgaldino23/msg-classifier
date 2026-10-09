package services

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"msg-classifier/internal/models"
)

// LoadPromptsFile reads a .json or .csv file of {flow, message, expected_result}
// examples and validates every row with the same rule as PromptService.Add — a
// known flow and non-empty message and expected result. A JSON file is a plain
// array of JevPrompt (id/timestamps ignored); a CSV file must start with the
// header flow,message,expected_result. An empty file is an error: a generation
// step running against an empty file must not wipe the seeds silently.
func LoadPromptsFile(path string) ([]models.JevPrompt, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompts file: %w", err)
	}

	var prompts []models.JevPrompt
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		if err := json.Unmarshal(data, &prompts); err != nil {
			return nil, fmt.Errorf("failed to parse prompts json: %w", err)
		}
	case ".csv":
		prompts, err = parsePromptsCSV(data)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported prompts file %q (use .json or .csv)", path)
	}

	if len(prompts) == 0 {
		return nil, fmt.Errorf("%w: no prompts in %s", ErrInvalidPrompt, path)
	}
	for i := range prompts {
		if err := validatePrompt(prompts[i].Flow, prompts[i].Message, prompts[i].ExpectedResult); err != nil {
			return nil, fmt.Errorf("prompt #%d in %s: %w", i+1, path, err)
		}
	}
	return prompts, nil
}

func parsePromptsCSV(data []byte) ([]models.JevPrompt, error) {
	records, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompts csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("%w: empty csv", ErrInvalidPrompt)
	}
	header := records[0]
	if len(header) != 3 || header[0] != "flow" || header[1] != "message" || header[2] != "expected_result" {
		return nil, fmt.Errorf("%w: csv header must be exactly flow,message,expected_result", ErrInvalidPrompt)
	}
	prompts := make([]models.JevPrompt, 0, len(records)-1)
	for _, record := range records[1:] {
		if len(record) != 3 {
			return nil, fmt.Errorf("%w: csv row must have exactly 3 fields", ErrInvalidPrompt)
		}
		prompts = append(prompts, models.JevPrompt{
			Flow:           strings.TrimSpace(record[0]),
			Message:        record[1],
			ExpectedResult: record[2],
		})
	}
	return prompts, nil
}

// SeedSQL renders the prompts back as the scripts/sql/seed_prompts.sql file the
// web harness loads (wipe + re-seed, a single INSERT with the rows grouped by
// flow in the canonical order), so the JSON examples are the single source and
// the SQL cannot drift by hand. The shape — one INSERT, no trailing newline —
// matches the committed file so a regeneration is a no-op diff.
func SeedSQL(prompts []models.JevPrompt) string {
	var b strings.Builder
	b.WriteString("-- Wipe and re-seed the Jev validation examples.\n")
	b.WriteString("DELETE FROM jev_prompts;\n\n")
	b.WriteString("INSERT INTO jev_prompts (flow, message, expected_result) VALUES\n")

	first := true
	for _, flow := range []string{models.FlowClassification, models.FlowName, models.FlowNote, models.FlowFinance} {
		for _, prompt := range prompts {
			if prompt.Flow != flow {
				continue
			}
			if first {
				first = false
			} else {
				b.WriteString(",\n")
			}
			fmt.Fprintf(&b, "('%s', '%s', '%s')", prompt.Flow, sqlEscape(prompt.Message), sqlEscape(prompt.ExpectedResult))
		}
	}
	b.WriteString(";\n")
	return strings.TrimSuffix(b.String(), "\n")
}

func sqlEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}