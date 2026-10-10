package contact

import (
	"errors"
	"fmt"

	"msg-classifier/internal/jevq"
	"msg-classifier/internal/models"
	"msg-classifier/internal/ptbr"
	"msg-classifier/internal/repository"
)

// ContactService handles the contact category use cases.
type ContactService struct {
	extractor *ContactExtractor
	repo      *repository.ContactRepository
}

func NewService(extractor *ContactExtractor, repo *repository.ContactRepository) *ContactService {
	return &ContactService{extractor: extractor, repo: repo}
}

// Handle routes contact messages to the add or get use case.
func (s *ContactService) Handle(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	if classification.Kind.Choice == "require" {
		return s.Get(request, classification)
	}
	return s.Add(request, classification)
}

// Add extracts contact data from the message and persists it. Phone and email are
// checked before any Jev call; with neither of them the name extraction runs first
// and its result is the duplicate key (DC-008). A duplicate with dup_action=update
// still runs the extraction — only the confirmed merge pays for Jev.
func (s *ContactService) Add(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	phone, phoneSpan, hasPhone := s.extractor.ExtractPhone(request.Message)
	email, emailSpan, hasEmail := s.extractor.ExtractEmail(request.Message)

	// Duplicate check before name extraction (early return — no Jev spent).
	var existing *models.Contact
	if request.DupAction != "new" {
		var err error
		switch {
		case hasPhone:
			existing, err = s.repo.FindByPhone(phone)
		case hasEmail:
			existing, err = s.repo.FindByEmail(email)
		}
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("failed to check duplicate contact: %w", err)
		}
		if existing != nil && request.DupAction != "update" {
			return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactDuplicate, Contact: existing}, nil
		}
	}

	spans := make([]jevq.Span, 0, 2)
	if hasPhone {
		spans = append(spans, phoneSpan)
	}
	if hasEmail {
		spans = append(spans, emailSpan)
	}

	nameResult, err := s.extractor.ExtractNameFromClassification(request.Message, spans, classification)
	if err != nil {
		return nil, err
	}

	// Name-fallback check: no phone/email to key on, so the extracted name is the key.
	if existing == nil && request.DupAction != "new" && !hasPhone && !hasEmail && nameResult.Name != "" {
		existing, err = s.repo.FindByNameNorm(ptbr.NormalizeName(nameResult.Name))
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("failed to check duplicate contact: %w", err)
		}
		if existing != nil && request.DupAction != "update" {
			return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactDuplicate, Contact: existing, Segments: nameResult.Segments}, nil
		}
	}

	if nameResult.Name == "" {
		return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactNoData}, nil
	}

	contact := &models.Contact{Name: nameResult.Name, NameNorm: ptbr.NormalizeName(nameResult.Name)}
	if hasPhone {
		contact.Phone = &phone
	}
	if hasEmail {
		contact.Email = &email
	}

	if existing != nil {
		mergeContact(existing, contact)
		if err := s.repo.Save(existing); err != nil {
			return nil, fmt.Errorf("failed to persist contact: %w", err)
		}
		return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactAdd, Contact: existing, Segments: nameResult.Segments}, nil
	}

	if err := s.repo.Create(contact); err != nil {
		return nil, fmt.Errorf("failed to persist contact: %w", err)
	}

	return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactAdd, Contact: contact, Segments: nameResult.Segments}, nil
}

// mergeContact copies every non-empty field of pending onto existing; a nil or
// empty pending field leaves the stored value alone.
func mergeContact(existing, pending *models.Contact) {
	if pending == nil {
		return
	}
	if pending.Name != "" {
		existing.Name = pending.Name
		existing.NameNorm = pending.NameNorm
	}
	if pending.Phone != nil && *pending.Phone != "" {
		existing.Phone = pending.Phone
	}
	if pending.Email != nil && *pending.Email != "" {
		existing.Email = pending.Email
	}
}

// BackfillNameNorm recomputes NameNorm for rows created before the column existed.
func (s *ContactService) BackfillNameNorm() error {
	contacts, err := s.repo.ListNeedingNameNorm()
	if err != nil {
		return fmt.Errorf("failed to load contacts for backfill: %w", err)
	}
	for i := range contacts {
		contacts[i].NameNorm = ptbr.NormalizeName(contacts[i].Name)
		if err := s.repo.Save(&contacts[i]); err != nil {
			return fmt.Errorf("failed to backfill name_norm: %w", err)
		}
	}
	return nil
}