package models

import (
	"fmt"
	"time"
)

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
	Choice     string  `json:"choice"`
	Confidence float64 `json:"confidence"`
}

// Action identifies the use case a classified message is dispatched to.
type Action string

const (
	ActionNone             Action = "none"
	ActionContactAdd       Action = "contact_add"
	ActionContactNoData    Action = "contact_no_data"
	ActionContactFound     Action = "contact_found"
	ActionContactNotFound  Action = "contact_not_found"
	ActionContactDuplicate Action = "contact_duplicate"
)

// UseCaseOutcome carries the classification, the dispatched action, and any use-case result.
type UseCaseOutcome struct {
	Classification *Classification
	Action         Action
	Contact        *Contact
	Segments       []SegmentScore
	SearchTerm     string
}

// ToResponse formats confidences as percent strings for display.
func (cl *Classification) ToResponse() *ReceiveMessageResponse {
	return &ReceiveMessageResponse{
		Category: map[string]any{
			"choice":     cl.Category.Choice,
			"confidence": fmt.Sprintf("%.2f", cl.Category.Confidence*100),
		},
		Kind: map[string]any{
			"choice":     cl.Kind.Choice,
			"confidence": fmt.Sprintf("%.2f", cl.Kind.Confidence*100),
		},
	}
}

// Contact is the persisted contact entity (Gorm). Phone/Email are nullable.
type Contact struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	NameNorm  string    `gorm:"column:name_norm" json:"name_norm"`
	Phone     *string   `json:"phone"`
	Email     *string   `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SegmentScore is one judged name segment: the text, its noul score, and whether
// it was included in the extracted name (decided once in the extractor).
type SegmentScore struct {
	Text     string  `json:"text"`
	Score    float64 `json:"score"`
	Included bool    `json:"included"`
}
