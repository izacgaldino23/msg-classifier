package repository

import (
	"msg-classifier/internal/models"

	"gorm.io/gorm"
)

// PromptRepository owns all gorm queries for the JevPrompt entity.
type PromptRepository struct {
	db *gorm.DB
}

func NewPromptRepository(db *gorm.DB) *PromptRepository {
	return &PromptRepository{db: db}
}

// Create persists a new prompt.
func (r *PromptRepository) Create(prompt *models.JevPrompt) error {
	return r.db.Create(prompt).Error
}

// ListByFlow returns all prompts for the given flow, ordered by id.
func (r *PromptRepository) ListByFlow(flow string) ([]models.JevPrompt, error) {
	var prompts []models.JevPrompt
	err := r.db.Where("flow = ?", flow).Order("id").Find(&prompts).Error
	if err != nil {
		return nil, err
	}
	return prompts, nil
}