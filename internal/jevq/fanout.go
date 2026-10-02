package jevq

import (
	"fmt"

	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"
)

// Span locates an extracted value within the original message.
type Span struct {
	Start int
	End   int
}

// NoulThreshold is the single site where a Noul score becomes an include/exclude
// verdict.
const NoulThreshold = 0.5

// SegmentQuestion describes what a per-segment Noul fan-out is asking about. The
// prompt gets the segment index so the instruction can point at `segments[i]`.
type SegmentQuestion struct {
	Prompt        string
	TrueCriteria  string
	FalseCriteria string
}

// SegmentKey is the answer and state key of the Noul question for segment i. It is
// exported so a caller building a fake response can name the same answers.
func SegmentKey(i int) string {
	return fmt.Sprintf("segment_%d", i)
}

// NoulSegments asks Jev one Noul question per segment and returns the response
// plus the per-segment trace. It is shared by the contact name and the finance
// party: both are the same fan-out — "which of these words is the thing we
// want" — with different criteria.
//
// extra rides in the same request (nil for a pure fan-out), so a caller can ask a
// choice question without paying a second call; that is why the raw response is
// returned as well. With no segments and no extra questions there is nothing to
// ask and no request is made.
func NoulSegments(client Requester, message string, segments []string, question SegmentQuestion, extra map[string]jev.JevQuestionInterface) (*jev.JevResponse, []models.SegmentScore, error) {
	if len(segments) == 0 && len(extra) == 0 {
		return nil, nil, nil
	}

	questions := make(map[string]jev.JevQuestionInterface, len(segments)+len(extra))
	for name, q := range extra {
		questions[name] = q
	}
	for i := range segments {
		questions[SegmentKey(i)] = &jev.JevQuestionNoul{
			JevQuestion: jev.JevQuestion{
				Type:         jev.NoulQuestionType,
				Instructions: fmt.Sprintf(question.Prompt, i),
			},
			Criteria: jev.JevNoulCriteria{
				True:  question.TrueCriteria,
				False: question.FalseCriteria,
			},
		}
	}

	resp, err := client.MakeJevRequest(&jev.JevRequest{
		State: jev.JevState(map[string]any{
			"message":  message,
			"segments": segments,
		}),
		Questions: questions,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("%w: failed to extract segments via jev: %w", ErrUpstream, err)
	}

	trace := make([]models.SegmentScore, 0, len(segments))
	for i, segment := range segments {
		answer, err := AnswerNoul(resp, SegmentKey(i))
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %w", ErrUpstream, err)
		}
		trace = append(trace, models.SegmentScore{Text: segment, Score: answer.Noul, Included: answer.Noul > NoulThreshold})
	}
	return resp, trace, nil
}