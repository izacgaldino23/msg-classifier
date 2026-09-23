package services

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"

	"golang.org/x/text/unicode/norm"
)

// ContactService handles the contact category use cases.
type ContactService struct {
	extractor *ContactExtractor
	repo      *repository.ContactRepository
}

func NewContactService(extractor *ContactExtractor, repo *repository.ContactRepository) *ContactService {
	return &ContactService{extractor: extractor, repo: repo}
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
		existing, err := s.repo.FindByPhone(phone)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("failed to check duplicate contact: %w", err)
		}
		if existing != nil {
			return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactDuplicate, Contact: existing}, nil
		}
	} else if hasEmail {
		existing, err := s.repo.FindByEmail(email)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("failed to check duplicate contact: %w", err)
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

	if err := s.repo.Create(contact); err != nil {
		return nil, fmt.Errorf("failed to persist contact: %w", err)
	}

	return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactAdd, Contact: contact, Segments: nameResult.Segments}, nil
}

// BackfillNameNorm recomputes NameNorm for rows created before the column existed.
func (s *ContactService) BackfillNameNorm() error {
	contacts, err := s.repo.ListNeedingNameNorm()
	if err != nil {
		return fmt.Errorf("failed to load contacts for backfill: %w", err)
	}
	for i := range contacts {
		contacts[i].NameNorm = normalizeName(contacts[i].Name)
		if err := s.repo.Save(&contacts[i]); err != nil {
			return fmt.Errorf("failed to backfill name_norm: %w", err)
		}
	}
	return nil
}

// normalizeName lowercases, strips diacritics (NFD + remove Mn marks), and
// collapses whitespace so name matching is robust to case and accents.
func normalizeName(s string) string {
	s = norm.NFD.String(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return strings.Join(strings.Fields(b.String()), " ")
}