package models

import "fmt"

// ReceiveMessageRequest is the inbound DTO for POST /api/message.
type ReceiveMessageRequest struct {
	Message string `json:"message" form:"message"`
	UserID  string `json:"user_id" form:"user_id"`
}

// ReceiveMessageResponse is the view model rendered by the "resultado" partial.
// Confidence values are pre-formatted as percent strings for display.
type ReceiveMessageResponse struct {
	Category map[string]any `json:"category"`
	Kind     map[string]any `json:"kind"`
}

// Classification is the domain result produced by services.ClassificationService.
// It carries raw numeric confidences (0..1); presentation formatting happens
// in ToResponse.
type Classification struct {
	Category CategoryFinding `json:"category"`
	Kind     KindFinding     `json:"kind"`
}

// CategoryFinding is the category classification outcome (choice + confidence).
type CategoryFinding struct {
	Choice     string  `json:"choice"`
	Confidence float64 `json:"confidence"`
}

// KindFinding is the request-kind outcome (score + confidence + legend).
type KindFinding struct {
	Score      float64           `json:"score"`
	Confidence float64           `json:"confidence"`
	Legend     map[string]string `json:"legend"`
}

// ToResponse maps a Classification into the template-ready response shape,
// formatting confidence as a percent string (successor of the former
// fromJevResponse helper).
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
		},
	}
}
