package services

import (
	"errors"
	"fmt"

	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"
)

// ErrUpstream marks Jev/TypeSafe API failures; controllers map it to HTTP 502.
var ErrUpstream = errors.New("upstream classification failed")

// jevClient is the service boundary, allowing unit tests without HTTP.
type jevClient interface {
	MakeJevRequestFromFile(state jev.JevState, fileName string) (*jev.JevResponse, error)
}

// ClassificationService orchestrates the two Jev calls per message.
type ClassificationService struct {
	jev jevClient
}

func NewClassificationService(client jevClient) *ClassificationService {
	return &ClassificationService{jev: client}
}

// Classify runs both Jev classifications and maps answers into a Classification.
func (s *ClassificationService) Classify(request *models.ReceiveMessageRequest) (*models.Classification, error) {
	state := jev.JevState(map[string]any{
		"user":    request.UserID, // TODO: change this userId to user name
		"message": request.Message,
	})

	categoryResp, err := s.makeRequest(state, "classification.json")
	if err != nil {
		return nil, err
	}

	kindResp, err := s.makeRequest(state, "request_kind.json")
	if err != nil {
		return nil, err
	}

	// Verify if user want save or get data
	// if user want save, save the data to database
	// if user want get, get the data from database
	// TODO

	category, err := answerAsChoice(categoryResp, "classification")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpstream, err)
	}

	kind, err := answerAsScore(kindResp, "adding_or_requiring")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUpstream, err)
	}

	return &models.Classification{
		Category: models.CategoryFinding{
			Choice:     category.Choice,
			Confidence: category.Confidence,
		},
		Kind: models.KindFinding{
			Score:      kind.Score,
			Confidence: kind.Confidence,
			Legend:     kind.Legend,
		},
	}, nil
}

func (s *ClassificationService) makeRequest(state jev.JevState, template string) (*jev.JevResponse, error) {
	resp, err := s.jev.MakeJevRequestFromFile(state, template)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to call jev with template %q: %w", ErrUpstream, template, err)
	}
	return resp, nil
}

// answerAsChoice extracts a choice answer with a checked assertion.
func answerAsChoice(resp *jev.JevResponse, key string) (*jev.JevAnswerChoice, error) {
	answer, ok := resp.Answers[key]
	if !ok {
		return nil, fmt.Errorf("missing answer %q in jev response", key)
	}
	choice, ok := answer.(*jev.JevAnswerChoice)
	if !ok {
		return nil, fmt.Errorf("answer %q has type %T, want *jev.JevAnswerChoice", key, answer)
	}
	return choice, nil
}

// answerAsScore extracts a score answer with a checked assertion.
func answerAsScore(resp *jev.JevResponse, key string) (*jev.JevAnswerScore, error) {
	answer, ok := resp.Answers[key]
	if !ok {
		return nil, fmt.Errorf("missing answer %q in jev response", key)
	}
	score, ok := answer.(*jev.JevAnswerScore)
	if !ok {
		return nil, fmt.Errorf("answer %q has type %T, want *jev.JevAnswerScore", key, answer)
	}
	return score, nil
}
