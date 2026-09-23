package repository

import (
	"testing"

	"msg-classifier/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

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

func strPtr(s string) *string { return &s }

func TestContactRepositoryCreate(t *testing.T) {
	db := newTestDB(t)
	repo := NewContactRepository(db)

	contact := &models.Contact{Name: "Fulano", Phone: strPtr("9292929290")}
	require.NoError(t, repo.Create(contact))
	assert.NotZero(t, contact.ID)

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestContactRepositoryFindByPhone(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano", Phone: strPtr("9292929290")})
	repo := NewContactRepository(db)

	contact, err := repo.FindByPhone("9292929290")
	require.NoError(t, err)
	assert.Equal(t, "Fulano", contact.Name)

	_, err = repo.FindByPhone("9999999999")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestContactRepositoryFindByEmail(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano", Email: strPtr("X@Y.COM")})
	repo := NewContactRepository(db)

	contact, err := repo.FindByEmail("x@y.com")
	require.NoError(t, err)
	assert.Equal(t, "Fulano", contact.Name)

	_, err = repo.FindByEmail("nope@y.com")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestContactRepositoryFindByName(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano Tal", NameNorm: "fulano tal"})
	repo := NewContactRepository(db)

	contact, err := repo.FindByName("fulano")
	require.NoError(t, err)
	assert.Equal(t, "Fulano Tal", contact.Name)

	_, err = repo.FindByName("zezinho")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestContactRepositoryListNeedingNameNorm(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "João da Silva"})            // empty NameNorm
	db.Create(&models.Contact{Name: "Maria", NameNorm: "maria"}) // already filled
	repo := NewContactRepository(db)

	contacts, err := repo.ListNeedingNameNorm()
	require.NoError(t, err)
	require.Len(t, contacts, 1)
	assert.Equal(t, "João da Silva", contacts[0].Name)
}

func TestContactRepositorySave(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "João da Silva"})
	repo := NewContactRepository(db)

	contacts, err := repo.ListNeedingNameNorm()
	require.NoError(t, err)
	contacts[0].NameNorm = "joao da silva"
	require.NoError(t, repo.Save(&contacts[0]))

	var reloaded models.Contact
	require.NoError(t, db.First(&reloaded, contacts[0].ID).Error)
	assert.Equal(t, "joao da silva", reloaded.NameNorm)
}

func TestContactRepositoryClosedDB(t *testing.T) {
	db := newTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	repo := NewContactRepository(db)

	_, err = repo.FindByPhone("9292929290")
	require.Error(t, err)

	require.Error(t, repo.Create(&models.Contact{Name: "Fulano"}))
}