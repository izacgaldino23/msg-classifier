package services

import (
	"errors"
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
		return s.Get(request, classification)
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

	// Duplicate check before name extraction (early return — no Jev spent).
	if hasPhone {
		existing, err := s.findDuplicate("phone = ?", phone)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactDuplicate, Contact: existing}, nil
		}
	} else if hasEmail {
		existing, err := s.findDuplicate("LOWER(email) = LOWER(?)", email)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactDuplicate, Contact: existing}, nil
		}
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

	contact := &models.Contact{Name: nameResult.Name, NameNorm: normalizeName(nameResult.Name)}
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

// findDuplicate returns the existing contact matching the query, or nil.
func (s *ContactService) findDuplicate(query string, arg string) (*models.Contact, error) {
	var existing models.Contact
	err := s.db.Where(query, arg).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check duplicate contact: %w", err)
	}
	if err == nil {
		return &existing, nil
	}
	return nil, nil
}

// BackfillNameNorm recomputes NameNorm for rows created before the column existed.
func (s *ContactService) BackfillNameNorm() error {
	var contacts []models.Contact
	if err := s.db.Where("name_norm = '' OR name_norm IS NULL").Find(&contacts).Error; err != nil {
		return fmt.Errorf("failed to load contacts for backfill: %w", err)
	}
	for i := range contacts {
		contacts[i].NameNorm = normalizeName(contacts[i].Name)
		if err := s.db.Save(&contacts[i]).Error; err != nil {
			return fmt.Errorf("failed to backfill name_norm: %w", err)
		}
	}
	return nil
}
