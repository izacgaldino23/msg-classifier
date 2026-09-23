package services

import (
	"errors"
	"fmt"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"
)

// Get searches for an existing contact by phone, email, or name (require flow).
// Priority: phone → email → name; phone/email searches skip Jev entirely.
func (s *ContactService) Get(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	phone, _, hasPhone := s.extractor.ExtractPhone(request.Message)
	if hasPhone {
		contact, err := s.repo.FindByPhone(phone)
		return s.searchResult(classification, contact, err, phone, nil)
	}

	email, _, hasEmail := s.extractor.ExtractEmail(request.Message)
	if hasEmail {
		contact, err := s.repo.FindByEmail(email)
		return s.searchResult(classification, contact, err, email, nil)
	}

	nameResult, err := s.extractor.ExtractName(request.Message, nil)
	if err != nil {
		return nil, err
	}
	term := normalizeName(nameResult.Name)
	if term == "" {
		return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactNoData}, nil
	}
	contact, err := s.repo.FindByName(term)
	return s.searchResult(classification, contact, err, term, nameResult.Segments)
}

// searchResult maps a repo lookup to a found/not-found outcome.
// repository.ErrNotFound is a "not found" outcome, not an error.
func (s *ContactService) searchResult(classification *models.Classification, contact *models.Contact, err error, searchTerm string, segments []models.SegmentScore) (*models.UseCaseOutcome, error) {
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("failed to search contact: %w", err)
	}
	if contact != nil {
		return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactFound, Contact: contact, SearchTerm: searchTerm, Segments: segments}, nil
	}
	return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactNotFound, SearchTerm: searchTerm, Segments: segments}, nil
}