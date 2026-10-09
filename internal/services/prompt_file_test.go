package services

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"msg-classifier/internal/models"

	"github.com/stretchr/testify/require"
)

func writePromptFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestLoadPromptsJSON(t *testing.T) {
	path := writePromptFile(t, "exemplos.json", `[
  {"flow": "classification", "message": "Salva o telefone do João", "expected_result": "contact:add"},
  {"flow": "finance", "message": "Gastei 50 no mercado", "expected_result": "compra"}
]`)

	prompts, err := LoadPromptsFile(path)
	require.NoError(t, err)
	require.Len(t, prompts, 2)
	require.Equal(t, "classification", prompts[0].Flow)
	require.Equal(t, "Salva o telefone do João", prompts[0].Message)
	require.Equal(t, "contact:add", prompts[0].ExpectedResult)
	require.Equal(t, models.FlowFinance, prompts[1].Flow)
}

func TestLoadPromptsCSV(t *testing.T) {
	path := writePromptFile(t, "exemplos.csv",
		"flow,message,expected_result\nclassification,Salva o telefone do João,contact:add\nfinance,Gastei 50 no mercado,compra\n")

	prompts, err := LoadPromptsFile(path)
	require.NoError(t, err)
	require.Len(t, prompts, 2)
	require.Equal(t, "contact:add", prompts[0].ExpectedResult)
	require.Equal(t, models.FlowFinance, prompts[1].Flow)
}

func TestLoadPromptsCSVBadHeader(t *testing.T) {
	// Wrong header words (still 3 fields) must be rejected as an invalid prompt.
	path := writePromptFile(t, "exemplos.csv", "flow,message,result\nclassification,x,contact:add\n")

	_, err := LoadPromptsFile(path)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidPrompt)
}

func TestLoadPromptsJSONInvalidFlow(t *testing.T) {
	path := writePromptFile(t, "exemplos.json", `[{"flow": "unknown", "message": "x", "expected_result": "y"}]`)

	_, err := LoadPromptsFile(path)
	require.ErrorIs(t, err, ErrInvalidPrompt)
}

func TestLoadPromptsJSONMissingField(t *testing.T) {
	path := writePromptFile(t, "exemplos.json", `[{"flow": "note", "message": "", "expected_result": "reminder"}]`)

	_, err := LoadPromptsFile(path)
	require.ErrorIs(t, err, ErrInvalidPrompt)
}

func TestLoadPromptsEmptyIsError(t *testing.T) {
	path := writePromptFile(t, "exemplos.json", `[]`)

	_, err := LoadPromptsFile(path)
	require.ErrorIs(t, err, ErrInvalidPrompt)
}

func TestLoadPromptsUnsupportedExtension(t *testing.T) {
	path := writePromptFile(t, "exemplos.txt", `nope`)

	_, err := LoadPromptsFile(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), ".json or .csv")
}

func TestSeedSQLShapeAndEscaping(t *testing.T) {
	prompts := []models.JevPrompt{
		{Flow: models.FlowClassification, Message: "Salva o telefone", ExpectedResult: "contact:add"},
		{Flow: models.FlowClassification, Message: "Paga a fatura", ExpectedResult: "finance:add"},
		{Flow: models.FlowNote, Message: "Comprar pão d'água", ExpectedResult: "reminder"},
	}

	seed := SeedSQL(prompts)

	require.True(t, strings.HasPrefix(seed, "-- Wipe and re-seed the Jev validation examples.\nDELETE FROM jev_prompts;\n"))
	require.Equal(t, 1, strings.Count(seed, "INSERT INTO jev_prompts (flow, message, expected_result) VALUES\n"))
	require.Contains(t, seed, "'contact:add'")
	// Apostrophes are doubled: the generated SQL must survive a raw reader.
	require.Contains(t, seed, "d''água")
	require.Contains(t, seed, "'contact:add'")
	// Rows within a flow keep the source order.
	require.True(t, strings.Index(seed, "Salva o telefone") < strings.Index(seed, "Paga a fatura"))
}

func TestLoadPromptsEmptyFileIsRealError(t *testing.T) {
	// An empty file is a parse error, not a "no prompts" ErrInvalidPrompt.
	path := writePromptFile(t, "exemplos.json", "")
	_, err := LoadPromptsFile(path)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrInvalidPrompt))
}