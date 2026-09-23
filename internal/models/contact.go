package models

import "time"

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