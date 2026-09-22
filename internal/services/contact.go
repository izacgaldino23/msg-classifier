package services

import (
	"msg-classifier/internal/models"
)

// ContactService handles the contact category use cases.
type ContactService struct{}

func NewContactService() *ContactService {
	return &ContactService{}
}

// Handle routes contact messages to the add or get use case.
func (s *ContactService) Handle(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	if classification.Kind.Value == "require" {
		// TODO: get path — fetch existing contact data.
		return &models.UseCaseOutcome{Classification: classification, Action: models.ActionNone}, nil
	}
	return s.Add(request, classification)
}

// Add is the stub for the contact add use case.
func (s *ContactService) Add(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	// TODO: extract contact data from request.Message and persist it.
	return &models.UseCaseOutcome{Classification: classification, Action: models.ActionContactAdd}, nil
}
