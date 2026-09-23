package models

import "time"

// Flow constants identify the Jev flows the validation harness can evaluate.
const (
	FlowClassification = "classification"
	FlowName           = "name"
)

// JevPrompt is a persisted example message with its expected Jev result.
type JevPrompt struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Flow           string    `json:"flow"`
	Message        string    `json:"message"`
	ExpectedResult string    `json:"expected_result"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// EvaluationResult compares one prompt's expected result with what Jev returned.
type EvaluationResult struct {
	PromptID       uint           `json:"prompt_id"`
	Message        string         `json:"message"`
	ExpectedResult string         `json:"expected_result"`
	ObtainedResult string         `json:"obtained_result"`
	Match          bool           `json:"match"`
	Segments       []SegmentScore `json:"segments,omitempty"` // set only for the name flow (UI trace)
}

// PromptForm is the inbound DTO for POST /prompts (add prompt).
type PromptForm struct {
	Flow     string `json:"flow" form:"flow"`
	Message  string `json:"message" form:"message"`
	Expected string `json:"expected" form:"expected"`
}

// EvaluateForm is the inbound DTO for POST /prompts/evaluate and /prompts/export.
type EvaluateForm struct {
	Flow string `json:"flow" form:"flow"`
	IDs  []uint `json:"ids" form:"ids"`
}