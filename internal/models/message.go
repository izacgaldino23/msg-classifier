package models

// ReceiveMessageRequest is the inbound DTO for POST /api/message.
type ReceiveMessageRequest struct {
	Message   string `json:"message" form:"message"`
	UserID    string `json:"user_id" form:"user_id"`
	DupAction string `json:"dup_action" form:"dup_action"` // "" = check, "new" = insert anyway, "update" = merge into the duplicate
}

// Classification is the domain result with raw numeric confidences.
type Classification struct {
	Category        CategoryFinding `json:"category"`
	Kind            KindFinding     `json:"kind"`
	Subtype         SubtypeFinding  `json:"subtype,omitempty"`
	PartySegments   []SegmentScore  `json:"party_segments,omitempty"`
	RequestIntent   IntentFinding   `json:"request_intent,omitempty"`
}

type CategoryFinding struct {
	Choice     string  `json:"choice"`
	Confidence float64 `json:"confidence"`
}

type KindFinding struct {
	Choice     string  `json:"choice"`
	Confidence float64 `json:"confidence"`
}

type SubtypeFinding struct {
	Choice     string  `json:"choice"`
	Confidence float64 `json:"confidence"`
}

type IntentFinding struct {
	Choice     string  `json:"choice"`
	Confidence float64 `json:"confidence"`
}

// Action identifies the use case a classified message is dispatched to.
type Action string

const (
	ActionNone                 Action = "none"
	ActionContactAdd           Action = "contact_add"
	ActionContactNoData        Action = "contact_no_data"
	ActionContactFound         Action = "contact_found"
	ActionContactNotFound      Action = "contact_not_found"
	ActionContactDuplicate     Action = "contact_duplicate"
	ActionNoteAdd              Action = "note_add"
	ActionNoteNoData           Action = "note_no_data"
	ActionNoteFound            Action = "note_found"
	ActionNoteNotFound         Action = "note_not_found"
	ActionNoteDuplicate        Action = "note_duplicate"
	ActionTransactionAdd       Action = "transaction_add"
	ActionTransactionNoData    Action = "transaction_no_data"
	ActionTransactionFound     Action = "transaction_found"
	ActionTransactionNotFound  Action = "transaction_not_found"
	ActionTransactionDuplicate Action = "transaction_duplicate"
)

// UseCaseOutcome carries the classification, the dispatched action, and any use-case result.
type UseCaseOutcome struct {
	Classification *Classification
	Action         Action
	Message        string // original message; the WEB duplicate screen re-posts it
	Contact        *Contact
	Notes          []*Note
	Transactions   []*Transaction
	Segments       []SegmentScore
	SearchTerm     string
	Total          float64
	Missing        string
}
