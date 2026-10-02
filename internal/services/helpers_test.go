package services

import (
	"fmt"
	"testing"

	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func strPtr(s string) *string { return &s }

// newTestDB opens an in-memory SQLite with every schema the root services touch.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "gorm.Open()")
	sqlDB, err := db.DB()
	require.NoError(t, err, "db.DB()")
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&models.Contact{}), "AutoMigrate()")
	return db
}

// mockJevRequester fakes the dynamic-request seam the extractors use. The harness
// drives the extractors from this package, so the mock is repeated here.
type mockJevRequester struct {
	resp  *jev.JevResponse
	err   error
	got   *jev.JevRequest
	calls int
}

func (m *mockJevRequester) MakeJevRequest(request *jev.JevRequest) (*jev.JevResponse, error) {
	m.got = request
	m.calls++
	return m.resp, m.err
}

// noulResponse serves one Noul verdict per segment, keyed like jevq does.
func noulResponse(nouls ...float64) *jev.JevResponse {
	answers := make(map[string]any, len(nouls))
	for i, n := range nouls {
		answers[fmt.Sprintf("segment_%d", i)] = &jev.JevAnswerNoul{Noul: n}
	}
	return &jev.JevResponse{Model: "jev-latest", Answers: answers}
}

// financeResponse serves the transaction type choice plus one verdict per segment.
func financeResponse(choice string, nouls ...float64) *jev.JevResponse {
	answers := map[string]any{
		"transaction_type": &jev.JevAnswerChoice{Choice: choice, Confidence: 0.9},
	}
	for i, noul := range nouls {
		answers[fmt.Sprintf("segment_%d", i)] = &jev.JevAnswerNoul{Noul: noul}
	}
	return &jev.JevResponse{Model: "jev-latest", Answers: answers}
}

// noteTypeResponse is the shape of the note.json answer map.
func noteTypeResponse(choice string) *jev.JevResponse {
	return &jev.JevResponse{
		Model: "jev-latest",
		Answers: map[string]any{
			"note_type": &jev.JevAnswerChoice{Choice: choice, Confidence: 0.9},
		},
	}
}