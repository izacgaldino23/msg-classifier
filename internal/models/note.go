package models

import "time"

// NoteType constants identify the sub-type of a stored note (the Jev note_type answer).
const (
	NoteTypeNote     = "note"
	NoteTypeReminder = "reminder"
	NoteTypeTodo     = "todo"
)

// Note is the persisted note entity (Gorm). All three sub-types share one table:
// Date is set only for reminders, Time is optional and Content is the note text.
type Note struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Type      string     `gorm:"index" json:"type"`
	Content   string     `gorm:"not null" json:"content"`
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