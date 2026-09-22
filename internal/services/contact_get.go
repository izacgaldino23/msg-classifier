package services

import (
	"errors"
	"fmt"

	"msg-classifier/internal/models"

	"gorm.io/gorm"
)

// Get searches for an existing contact by phone, email, or name (require flow).
// Priority: phone → email → name; phone/email searches skip Jev entirely.
func (s *ContactService) Get(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	phone, _, hasPhone := s.extractor.ExtractPhone(request.Message)
	if hasPhone {
		return s.searchByField(classification, "phone = ?", phone, phone, nil)
	}

	email, _, hasEmail := s.extractor.ExtractEmail(request.Message)
	if hasEmail {
		return s.searchByField(classification, "LOWER(email) = LOWER(?)", email, email, nil)
	}

	nameResult, err := s.extractor.ExtractName(request.Message, nil)
	if err != nil {
		return nil, err
	}
	term := normalizeName(nameResult.Name)
	if term == "" {
		return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactNoData}, nil
	}
	return s.searchByField(classification, "name_norm LIKE ?", "%"+term+"%", term, nameResult.Segments)
}

// searchByField runs the query and maps the result to found/not-found.
// gorm.ErrRecordNotFound is a "not found" outcome, not an error.
func (s *ContactService) searchByField(classification *models.Classification, query string, arg string, searchTerm string, segments []models.SegmentScore) (*models.UseCaseOutcome, error) {
	var contact models.Contact
	err := s.db.Where(query, arg).Order("id").First(&contact).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to search contact: %w", err)
	}
	if err == nil {
		return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactFound, Contact: &contact, SearchTerm: searchTerm, Segments: segments}, nil
	}
	return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactNotFound, SearchTerm: searchTerm, Segments: segments}, nil
}
