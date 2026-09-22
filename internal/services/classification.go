package services

import (
	"errors"
	"fmt"

	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"
)

// ErrUpstream marks failures that originate from the Jev/TypeSafe API or its
// responses. Controllers map it to HTTP 502; any other error maps to 500.
var ErrUpstream = errors.New("upstream classification failed")

// jevClient is the service boundary for the Jev client. It is defined here
// (not imported as a concrete type in method signatures) so the service can
// be unit-tested later without HTTP.
type jevClient interface {
	MakeJevRequestFromFile(state jev.JevState, fileName string) (*jev.JevResponse, error)
}

// ClassificationService orchestrates the two sequential Jev calls that
// classify a single message.
type ClassificationService struct {
	jev jevClient
}

// NewClassificationService wires an injected Jev client into the service.
func NewClassificationService(client jevClient) *ClassificationService {
	return &ClassificationService{jev: client}
}

// Classify runs the category and request-kind classifications and maps the
// answers into a domain Classification. It never panics: missing or
// mistyped answers become descriptive errors.
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

// makeRequest calls the Jev client with the given prompt template, wrapping
// any failure in ErrUpstream.
func (s *ClassificationService) makeRequest(state jev.JevState, template string) (*jev.JevResponse, error) {
	resp, err := s.jev.MakeJevRequestFromFile(state, template)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to call jev with template %q: %w", ErrUpstream, template, err)
	}
	return resp, nil
}

// answerAsChoice performs a checked extraction of a choice answer —
// replaces the former unchecked assertion on Answers["classification"].
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

// answerAsScore performs a checked extraction of a score answer —
// replaces the former unchecked assertion on Answers["adding_or_requiring"].
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
