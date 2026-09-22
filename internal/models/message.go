package models

import "fmt"

// ReceiveMessageRequest is the inbound DTO for POST /api/message.
type ReceiveMessageRequest struct {
	Message string `json:"message" form:"message"`
	UserID  string `json:"user_id" form:"user_id"`
}

// ReceiveMessageResponse is the view model for the "resultado" partial.
type ReceiveMessageResponse struct {
	Category map[string]any `json:"category"`
	Kind     map[string]any `json:"kind"`
}

// Classification is the domain result with raw numeric confidences.
type Classification struct {
	Category CategoryFinding `json:"category"`
	Kind     KindFinding     `json:"kind"`
}

type CategoryFinding struct {
	Choice     string  `json:"choice"`
	Confidence float64 `json:"confidence"`
}

type KindFinding struct {
	Score      float64           `json:"score"`
	Confidence float64           `json:"confidence"`
	Legend     map[string]string `json:"legend"`
	Value      string            `json:"value"`
}

// Action identifies the use case a classified message is dispatched to.
type Action string

const (
	ActionNone       Action = "none"
	ActionContactAdd Action = "contact_add"
)

// UseCaseOutcome carries the classification and the dispatched action.
type UseCaseOutcome struct {
	Classification *Classification
	Action         Action
}

// ToResponse formats confidences as percent strings for display.
func (cl *Classification) ToResponse() *ReceiveMessageResponse {
	return &ReceiveMessageResponse{
		Category: map[string]any{
			"choice":     cl.Category.Choice,
			"confidence": fmt.Sprintf("%.2f", cl.Category.Confidence*100),
		},
		Kind: map[string]any{
			"score":      cl.Kind.Score,
			"confidence": fmt.Sprintf("%.2f", cl.Kind.Confidence*100),
			"legend":     cl.Kind.Legend,
			"value":      cl.Kind.Value,
		},
	}
}
