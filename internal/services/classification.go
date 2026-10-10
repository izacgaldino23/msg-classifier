package services

import (
	"fmt"

	"msg-classifier/internal/jevq"
	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"
)

// ClassificationService runs the Jev classification per message.
type ClassificationService struct {
	jev jevq.Client
}

func NewClassificationService(client jevq.Client) *ClassificationService {
	return &ClassificationService{jev: client}
}

// Classify runs the Jev classification and maps answers into a Classification.
func (s *ClassificationService) Classify(request *models.ReceiveMessageRequest) (*models.Classification, error) {
	state := jev.JevState(map[string]any{
		"user":    request.UserID,
		"message": request.Message,
	})

	categoryResp, err := s.jev.MakeJevRequestFromFile(state, "classification.json")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to call jev with template %q: %w", jevq.ErrUpstream, "classification.json", err)
	}

	category, err := jevq.AnswerChoice(categoryResp, "classification")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", jevq.ErrUpstream, err)
	}

	kind, err := jevq.AnswerChoice(categoryResp, "adding_or_requiring")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", jevq.ErrUpstream, err)
	}

	subtype, _ := jevq.AnswerChoice(categoryResp, "category_subtype")
	partySegmentsRaw, _ := jevq.AnswerNoulSegments(categoryResp)
	intent, _ := jevq.AnswerChoice(categoryResp, "request_intent")
	txCategory, _ := jevq.AnswerChoice(categoryResp, "transaction_category")
	noteTopic, _ := jevq.AnswerChoice(categoryResp, "note_topic")

	class := &models.Classification{
		Category: models.CategoryFinding{
			Choice:     category.Choice,
			Confidence: category.Confidence,
		},
		Kind: models.KindFinding{
			Choice:     kind.Choice,
			Confidence: kind.Confidence,
		},
	}
	if subtype != nil {
		class.Subtype = models.SubtypeFinding{
			Choice:     subtype.Choice,
			Confidence: subtype.Confidence,
		}
	}
	if len(partySegmentsRaw) > 0 {
		partySegments := make([]models.SegmentScore, len(partySegmentsRaw))
		for i, seg := range partySegmentsRaw {
			partySegments[i] = models.SegmentScore{Score: seg.Score, Included: seg.Included}
		}
		class.PartySegments = partySegments
	}
	if intent != nil {
		class.RequestIntent = models.IntentFinding{
			Choice:     intent.Choice,
			Confidence: intent.Confidence,
		}
	}
	if txCategory != nil {
		class.TransactionCategory = models.SubtypeFinding{
			Choice:     txCategory.Choice,
			Confidence: txCategory.Confidence,
		}
	}
	if noteTopic != nil {
		class.NoteTopic = models.SubtypeFinding{
			Choice:     noteTopic.Choice,
			Confidence: noteTopic.Confidence,
		}
	}
	return class, nil
}
