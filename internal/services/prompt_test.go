package services

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"
	"msg-classifier/pkg/jev"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newPromptService reuses the package newTestDB (Contact) and adds the JevPrompt table.
func newPromptService(t *testing.T, classifier *ClassificationService, extractor *ContactExtractor) (*PromptService, *gorm.DB) {
	t.Helper()
	db := newTestDB(t)
	require.NoError(t, db.AutoMigrate(&models.JevPrompt{}), "AutoMigrate(JevPrompt)")
	return NewPromptService(repository.NewPromptRepository(db), classifier, extractor), db
}

// failFirstJevClient fails the first Jev call, then succeeds — for per-prompt error capture.
type failFirstJevClient struct {
	resp  *jev.JevResponse
	calls int
}

func (m *failFirstJevClient) MakeJevRequestFromFile(state jev.JevState, fileName string) (*jev.JevResponse, error) {
	m.calls++
	if m.calls == 1 {
		return nil, errors.New("upstream boom")
	}
	return m.resp, nil
}

func TestPromptServiceAdd(t *testing.T) {
	service, db := newPromptService(t, nil, nil)

	prompt, err := service.Add(models.FlowClassification, "salva fulano", "contact:add")
	require.NoError(t, err)
	assert.NotZero(t, prompt.ID)
	assert.Equal(t, models.FlowClassification, prompt.Flow)

	var count int64
	require.NoError(t, db.Model(&models.JevPrompt{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestPromptServiceAddInvalidFlow(t *testing.T) {
	service, _ := newPromptService(t, nil, nil)

	_, err := service.Add("bogus", "msg", "expected")
	assert.ErrorIs(t, err, ErrInvalidPrompt)
}

func TestPromptServiceAddEmptyFields(t *testing.T) {
	service, _ := newPromptService(t, nil, nil)

	_, err := service.Add(models.FlowClassification, "  ", "expected")
	assert.ErrorIs(t, err, ErrInvalidPrompt)

	_, err = service.Add(models.FlowClassification, "msg", "")
	assert.ErrorIs(t, err, ErrInvalidPrompt)
}

func TestPromptServiceListByFlow(t *testing.T) {
	service, db := newPromptService(t, nil, nil)
	db.Create(&models.JevPrompt{Flow: models.FlowClassification, Message: "a", ExpectedResult: "contact:add"})
	db.Create(&models.JevPrompt{Flow: models.FlowName, Message: "b", ExpectedResult: "João"})

	prompts, err := service.ListByFlow(models.FlowClassification)
	require.NoError(t, err)
	require.Len(t, prompts, 1)
	assert.Equal(t, "a", prompts[0].Message)
}

func TestPromptServiceEvaluateClassification(t *testing.T) {
	mock := &mockJevClient{resp: choiceResponse("contact", "add", 0.95)}
	service, db := newPromptService(t, NewClassificationService(mock), nil)
	prompt := &models.JevPrompt{Flow: models.FlowClassification, Message: "salva fulano", ExpectedResult: "contact:add"}
	require.NoError(t, db.Create(prompt).Error)

	results, err := service.Evaluate(models.FlowClassification, []uint{prompt.ID})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "contact:add", results[0].ObtainedResult)
	assert.True(t, results[0].Match)
	assert.Nil(t, results[0].Segments)
}

func TestPromptServiceEvaluateClassificationCaseInsensitive(t *testing.T) {
	mock := &mockJevClient{resp: choiceResponse("Contact", "Add", 0.9)}
	service, db := newPromptService(t, NewClassificationService(mock), nil)
	prompt := &models.JevPrompt{Flow: models.FlowClassification, Message: "salva fulano", ExpectedResult: "contact:add"}
	require.NoError(t, db.Create(prompt).Error)

	results, err := service.Evaluate(models.FlowClassification, []uint{prompt.ID})
	require.NoError(t, err)
	assert.True(t, results[0].Match)
}

func TestPromptServiceEvaluateClassificationMismatch(t *testing.T) {
	mock := &mockJevClient{resp: choiceResponse("finance", "require", 0.9)}
	service, db := newPromptService(t, NewClassificationService(mock), nil)
	prompt := &models.JevPrompt{Flow: models.FlowClassification, Message: "quanto gastei?", ExpectedResult: "contact:add"}
	require.NoError(t, db.Create(prompt).Error)

	results, err := service.Evaluate(models.FlowClassification, []uint{prompt.ID})
	require.NoError(t, err)
	assert.Equal(t, "finance:require", results[0].ObtainedResult)
	assert.False(t, results[0].Match)
}

func TestPromptServiceEvaluateName(t *testing.T) {
	mock := &mockJevRequester{resp: noulResponse(0.99, 0.98, 0.98)}
	service, db := newPromptService(t, nil, NewContactExtractor(mock))
	prompt := &models.JevPrompt{Flow: models.FlowName, Message: "João da Silva", ExpectedResult: "João da Silva"}
	require.NoError(t, db.Create(prompt).Error)

	results, err := service.Evaluate(models.FlowName, []uint{prompt.ID})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "João da Silva", results[0].ObtainedResult)
	assert.True(t, results[0].Match)
	require.Len(t, results[0].Segments, 3)
	assert.True(t, results[0].Segments[0].Included)
}

func TestPromptServiceEvaluateNameAccentInsensitive(t *testing.T) {
	mock := &mockJevRequester{resp: noulResponse(0.99, 0.98, 0.98)}
	service, db := newPromptService(t, nil, NewContactExtractor(mock))
	prompt := &models.JevPrompt{Flow: models.FlowName, Message: "João da Silva", ExpectedResult: "JOAO DA SILVA"}
	require.NoError(t, db.Create(prompt).Error)

	results, err := service.Evaluate(models.FlowName, []uint{prompt.ID})
	require.NoError(t, err)
	assert.True(t, results[0].Match)
}

func TestPromptServiceEvaluateNameMismatch(t *testing.T) {
	mock := &mockJevRequester{resp: noulResponse(0.99, 0.1, 0.98)}
	service, db := newPromptService(t, nil, NewContactExtractor(mock))
	prompt := &models.JevPrompt{Flow: models.FlowName, Message: "João da Silva", ExpectedResult: "João da Silva"}
	require.NoError(t, db.Create(prompt).Error)

	results, err := service.Evaluate(models.FlowName, []uint{prompt.ID})
	require.NoError(t, err)
	assert.Equal(t, "João Silva", results[0].ObtainedResult)
	assert.False(t, results[0].Match)
}

func TestPromptServiceEvaluateFiltersSelectedIDs(t *testing.T) {
	mock := &mockJevClient{resp: choiceResponse("contact", "add", 0.95)}
	service, db := newPromptService(t, NewClassificationService(mock), nil)
	p1 := &models.JevPrompt{Flow: models.FlowClassification, Message: "a", ExpectedResult: "contact:add"}
	p2 := &models.JevPrompt{Flow: models.FlowClassification, Message: "b", ExpectedResult: "contact:add"}
	require.NoError(t, db.Create(p1).Error)
	require.NoError(t, db.Create(p2).Error)

	results, err := service.Evaluate(models.FlowClassification, []uint{p1.ID})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "a", results[0].Message)
}

func TestPromptServiceEvaluateNoSelection(t *testing.T) {
	mock := &mockJevClient{resp: choiceResponse("contact", "add", 0.95)}
	service, db := newPromptService(t, NewClassificationService(mock), nil)
	db.Create(&models.JevPrompt{Flow: models.FlowClassification, Message: "a", ExpectedResult: "contact:add"})

	results, err := service.Evaluate(models.FlowClassification, []uint{999})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestPromptServiceEvaluatePerPromptErrorCapture(t *testing.T) {
	mock := &failFirstJevClient{resp: choiceResponse("contact", "add", 0.95)}
	service, db := newPromptService(t, NewClassificationService(mock), nil)
	p1 := &models.JevPrompt{Flow: models.FlowClassification, Message: "primeiro", ExpectedResult: "contact:add"}
	p2 := &models.JevPrompt{Flow: models.FlowClassification, Message: "segundo", ExpectedResult: "contact:add"}
	require.NoError(t, db.Create(p1).Error)
	require.NoError(t, db.Create(p2).Error)

	results, err := service.Evaluate(models.FlowClassification, []uint{p1.ID, p2.ID})
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.False(t, results[0].Match)
	assert.Contains(t, results[0].ObtainedResult, "upstream boom")
	assert.True(t, results[1].Match)
	assert.Equal(t, "contact:add", results[1].ObtainedResult)
}

func TestPromptServiceExportCSV(t *testing.T) {
	oldDir := exportDir
	exportDir = t.TempDir()
	t.Cleanup(func() { exportDir = oldDir })

	mock := &mockJevClient{resp: choiceResponse("contact", "add", 0.95)}
	service, db := newPromptService(t, NewClassificationService(mock), nil)
	prompt := &models.JevPrompt{Flow: models.FlowClassification, Message: "salva fulano", ExpectedResult: "contact:add"}
	require.NoError(t, db.Create(prompt).Error)

	results, err := service.Evaluate(models.FlowClassification, []uint{prompt.ID})
	require.NoError(t, err)

	path, err := service.ExportCSV(models.FlowClassification, results)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(filepath.Base(path), "classification-"))
	assert.True(t, strings.HasSuffix(path, ".csv"))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.Len(t, lines, 2)
	assert.Equal(t, "id,message,expected_result,obtained_result,match,executed_at", lines[0])
	assert.Contains(t, lines[1], "salva fulano")
	assert.Contains(t, lines[1], "contact:add")
	assert.Contains(t, lines[1], "true")
}