package repository

import (
	"testing"

	"msg-classifier/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newPromptTestDB mirrors the package's newTestDB but migrates JevPrompt.
func newPromptTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "gorm.Open()")
	sqlDB, err := db.DB()
	require.NoError(t, err, "db.DB()")
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&models.JevPrompt{}), "AutoMigrate()")
	return db
}

func TestPromptRepositoryCreate(t *testing.T) {
	db := newPromptTestDB(t)
	repo := NewPromptRepository(db)

	prompt := &models.JevPrompt{Flow: models.FlowClassification, Message: "salva fulano", ExpectedResult: "contact:add"}
	require.NoError(t, repo.Create(prompt))
	assert.NotZero(t, prompt.ID)

	var count int64
	require.NoError(t, db.Model(&models.JevPrompt{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestPromptRepositoryListByFlow(t *testing.T) {
	db := newPromptTestDB(t)
	db.Create(&models.JevPrompt{Flow: models.FlowClassification, Message: "msg 1", ExpectedResult: "contact:add"})
	db.Create(&models.JevPrompt{Flow: models.FlowName, Message: "nome 1", ExpectedResult: "João"})
	db.Create(&models.JevPrompt{Flow: models.FlowClassification, Message: "msg 2", ExpectedResult: "finance:require"})
	repo := NewPromptRepository(db)

	prompts, err := repo.ListByFlow(models.FlowClassification)
	require.NoError(t, err)
	require.Len(t, prompts, 2)
	assert.Equal(t, "msg 1", prompts[0].Message)
	assert.Equal(t, "msg 2", prompts[1].Message)
}

func TestPromptRepositoryListByFlowEmpty(t *testing.T) {
	db := newPromptTestDB(t)
	repo := NewPromptRepository(db)

	prompts, err := repo.ListByFlow(models.FlowName)
	require.NoError(t, err)
	assert.Empty(t, prompts)
}

func TestPromptRepositoryClosedDB(t *testing.T) {
	db := newPromptTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	repo := NewPromptRepository(db)

	require.Error(t, repo.Create(&models.JevPrompt{Flow: models.FlowClassification, Message: "x", ExpectedResult: "y"}))
	_, err = repo.ListByFlow(models.FlowClassification)
	require.Error(t, err)
}