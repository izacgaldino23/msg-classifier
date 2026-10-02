package notes

import (
	"testing"

	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func strPtr(s string) *string { return &s }

// newTestDB opens an in-memory SQLite with the notes schema.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "gorm.Open()")
	sqlDB, err := db.DB()
	require.NoError(t, err, "db.DB()")
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&models.Note{}, &models.TodoItem{}), "AutoMigrate()")
	return db
}

// mockJevClient fakes the template-file seam the sub-type extractor uses.
type mockJevClient struct {
	resp     *jev.JevResponse
	err      error
	gotState jev.JevState
	gotFile  string
}

func (m *mockJevClient) MakeJevRequestFromFile(state jev.JevState, fileName string) (*jev.JevResponse, error) {
	m.gotState = state
	m.gotFile = fileName
	return m.resp, m.err
}