package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"msg-classifier/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestNewWiresEveryEntry(t *testing.T) {
	application, err := New(":memory:")
	require.NoError(t, err, "New()")
	require.NotNil(t, application.Classifier, "Classifier")
	require.NotNil(t, application.Dispatcher, "Dispatcher")
	require.NotNil(t, application.Data, "Data")
	require.NotNil(t, application.Prompts, "Prompts")
}

func TestNewRunsNameNormBackfill(t *testing.T) {
	path := scratchDB(t)
	require.NoError(t, seedLegacyContact(path), "seedLegacyContact()")

	application, err := New(path)
	require.NoError(t, err, "New()")

	contacts, err := application.Data.ListContacts("all")
	require.NoError(t, err, "ListContacts()")
	require.Len(t, contacts, 1, "the seeded row")
	assert.Equal(t, "maria da silva", contacts[0].NameNorm, "BackfillNameNorm() must fill the pre-migration row")
}

// scratchDB hands out a path for a scratch SQLite file and removes it on a
// best-effort basis. t.TempDir() is deliberately not used: New keeps the
// connection pool it opened, and Windows refuses to delete a file another
// handle holds open, so the framework's TempDir cleanup would fail the test
// over a handle no caller can release. ponytail: the scratch files stay in
// %TEMP% — give App a Close() when the entrypoints gain a shutdown path.
func scratchDB(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "app-test-")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "contacts.db")
}

// seedLegacyContact writes the shape a database created before the name_norm
// column had values: a contact whose name_norm is empty, which is exactly what
// BackfillNameNorm exists to fix.
func seedLegacyContact(path string) error {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return err
	}
	defer func() {
		if sqlDB, poolErr := db.DB(); poolErr == nil {
			_ = sqlDB.Close()
		}
	}()
	if err := db.AutoMigrate(&models.Contact{}); err != nil {
		return err
	}
	empty := ""
	return db.Create(&models.Contact{Name: "Maria da Silva", NameNorm: empty}).Error
}

func TestDSN(t *testing.T) {
	assert.Equal(t, "file:contacts.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", dsn("contacts.db"))
	assert.Equal(t, ":memory:", dsn(":memory:"))
}

// TestDSNPragmasTakeEffect is the guard on the driver's _pragma syntax. The DSN
// is load-bearing for three concurrent processes and the failure mode of getting
// it wrong is a random "database is locked" in production, never in a test — so
// assert the pragma actually lands instead of trusting the README.
func TestDSNPragmasTakeEffect(t *testing.T) {
	path := scratchDB(t)
	db, err := gorm.Open(sqlite.Open(dsn(path)), &gorm.Config{})
	require.NoError(t, err, "gorm.Open()")
	sqlDB, err := db.DB()
	require.NoError(t, err, "db.DB()")
	defer func() { require.NoError(t, sqlDB.Close()) }()

	var mode string
	require.NoError(t, db.Raw("PRAGMA journal_mode").Scan(&mode).Error, "PRAGMA journal_mode")
	assert.Equal(t, "wal", strings.ToLower(mode), "the DSN pragma was not applied by the driver")

	var timeout int
	require.NoError(t, db.Raw("PRAGMA busy_timeout").Scan(&timeout).Error, "PRAGMA busy_timeout")
	assert.Equal(t, 5000, timeout, "the DSN pragma was not applied by the driver")
}
