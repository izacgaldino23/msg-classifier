package models

import "time"

// NoteType constants identify the sub-type of a stored note (the Jev note_type answer).
const (
	NoteTypeNote     = "note"
	NoteTypeReminder = "reminder"
	NoteTypeTodo     = "todo"
)

// NoteTopic constants identify the topic of a stored note (the Jev note_topic
// answer). F2 asks for it in the composite classification round trip, so a note
// knows what it is about, not only what kind it is.
const (
	NoteTopicProject   = "projeto"
	NoteTopicPersonal  = "pessoal"
	NoteTopicHealth    = "saude"
	NoteTopicShopping  = "compras"
	NoteTopicFinancial = "financeiro"
	NoteTopicOther     = "outro"
)

// IsNoteTopic reports whether a raw topic is one of the known criteria. The
// service drops any other Jev answer instead of persisting it.
func IsNoteTopic(value string) bool {
	switch value {
	case NoteTopicProject, NoteTopicPersonal, NoteTopicHealth,
		NoteTopicShopping, NoteTopicFinancial, NoteTopicOther:
		return true
	}
	return false
}

// Note is the persisted note entity (Gorm). All three sub-types share one table:
// Date is set only for reminders, Time is optional and Content is the note text.
type Note struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Type      string     `gorm:"index" json:"type"`
	Content   string     `gorm:"not null" json:"content"`
	Topic     string     `gorm:"index" json:"topic"`
	Date      *time.Time `gorm:"index" json:"date"`
	Time      *string    `json:"time"`
	Items     []TodoItem `gorm:"foreignKey:NoteID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// TodoItem is one item of a to-do list; Done tracks whether it was finished.
type TodoItem struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	NoteID   uint   `gorm:"index" json:"note_id"`
	Text     string `gorm:"not null" json:"text"`
	Done     bool   `gorm:"index" json:"done"`
	Position int    `gorm:"column:position" json:"position"`
}