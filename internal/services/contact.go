package services

import (
	"fmt"

	"msg-classifier/internal/models"

	"gorm.io/gorm"
)

// ContactService handles the contact category use cases.
type ContactService struct {
	extractor *ContactExtractor
	db        *gorm.DB
}

func NewContactService(extractor *ContactExtractor, db *gorm.DB) *ContactService {
	return &ContactService{extractor: extractor, db: db}
}

// Handle routes contact messages to the add or get use case.
func (s *ContactService) Handle(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	if classification.Kind.Choice == "require" {
		// TODO: get path — fetch existing contact data.
		return &models.UseCaseOutcome{Classification: classification, Action: models.ActionNone}, nil
	}
	return s.Add(request, classification)
}

// Add extracts contact data from the message and persists it.
func (s *ContactService) Add(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	phone, phoneSpan, hasPhone := s.extractor.ExtractPhone(request.Message)
	email, emailSpan, hasEmail := s.extractor.ExtractEmail(request.Message)

	if !hasPhone && !hasEmail {
		return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactNoData}, nil
	}

	spans := make([]Span, 0, 2)
	if hasPhone {
		spans = append(spans, phoneSpan)
	}
	if hasEmail {
		spans = append(spans, emailSpan)
	}

	nameResult, err := s.extractor.ExtractName(request.Message, spans)
	if err != nil {
		return nil, err
	}

	contact := &models.Contact{Name: nameResult.Name}
	if hasPhone {
		contact.Phone = &phone
	}
	if hasEmail {
		contact.Email = &email
	}

	if err := s.db.Create(contact).Error; err != nil {
		return nil, fmt.Errorf("failed to persist contact: %w", err)
	}

	return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactAdd, Contact: contact, Segments: nameResult.Segments}, nil
}
