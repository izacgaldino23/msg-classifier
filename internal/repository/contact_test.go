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

func TestContactRepositoryListAllNewestFirst(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano", Phone: strPtr("9292929290")})
	db.Create(&models.Contact{Name: "Doutrina"})
	repo := NewContactRepository(db)

	contacts, err := repo.List("all")
	require.NoError(t, err)
	require.Len(t, contacts, 2)
	assert.Equal(t, "Doutrina", contacts[0].Name, "newest first (id DESC)")
	assert.Equal(t, "Fulano", contacts[1].Name)
}

func TestContactRepositoryListFilters(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Só telefone", Phone: strPtr("1111111111")})
	db.Create(&models.Contact{Name: "Só email", Email: strPtr("a@b.com")})
	db.Create(&models.Contact{Name: "Só nome"})
	db.Create(&models.Contact{Name: "Vazio", Phone: strPtr(""), Email: strPtr("")})
	repo := NewContactRepository(db)

	phone, err := repo.List("phone")
	require.NoError(t, err)
	require.Len(t, phone, 1)
	assert.Equal(t, "Só telefone", phone[0].Name)

	email, err := repo.List("email")
	require.NoError(t, err)
	require.Len(t, email, 1)
	assert.Equal(t, "Só email", email[0].Name)

	// "name" = phone AND email both absent or empty. Newest first (id DESC),
	// so "Vazio" (created last) comes before "Só nome".
	name, err := repo.List("name")
	require.NoError(t, err)
	require.Len(t, name, 2)
	assert.Equal(t, "Vazio", name[0].Name)
	assert.Equal(t, "Só nome", name[1].Name)
}

func TestContactRepositoryListUnknownFilterReturnsAll(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano"})
	repo := NewContactRepository(db)

	contacts, err := repo.List("bogus")
	require.NoError(t, err, "the repository never errors on an unknown filter - the service is the gate")
	assert.Len(t, contacts, 1)
}

func TestContactRepositoryListEmpty(t *testing.T) {
	repo := NewContactRepository(newTestDB(t))

	contacts, err := repo.List("all")
	require.NoError(t, err)
	assert.Empty(t, contacts)
}

func TestContactRepositoryFindByID(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano", Phone: strPtr("9292929290")})
	repo := NewContactRepository(db)

	contact, err := repo.FindByID(1)
	require.NoError(t, err)
	assert.Equal(t, "Fulano", contact.Name)

	_, err = repo.FindByID(999)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestContactRepositoryDeleteByIDs(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "A"})
	db.Create(&models.Contact{Name: "B"})
	db.Create(&models.Contact{Name: "C"})
	repo := NewContactRepository(db)

	require.NoError(t, repo.DeleteByIDs([]uint{1, 3}))

	var remaining []models.Contact
	require.NoError(t, db.Order("id").Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, "B", remaining[0].Name)
}

func TestContactRepositoryDeleteByIDsEmptyIsANoOp(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "A"})
	repo := NewContactRepository(db)

	require.NoError(t, repo.DeleteByIDs(nil))

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestContactRepositoryFindByNameNorm(t *testing.T) {
	db := newTestDB(t)
	repo := NewContactRepository(db)
	require.NoError(t, db.Create(&models.Contact{Name: "Fulano de Tal", NameNorm: "fulano de tal"}).Error)
	require.NoError(t, db.Create(&models.Contact{Name: "Fulano", NameNorm: "fulano"}).Error)

	contact, err := repo.FindByNameNorm("fulano de tal")
	require.NoError(t, err)
	assert.Equal(t, "Fulano de Tal", contact.Name)

	contact, err = repo.FindByNameNorm("fulano")
	require.NoError(t, err)
	assert.Equal(t, "Fulano", contact.Name)

	_, err = repo.FindByNameNorm("fulano de")
	assert.ErrorIs(t, err, ErrNotFound, "exact match only — FindByName's LIKE is the search flow")

	_, err = repo.FindByNameNorm("inexistente")
	assert.ErrorIs(t, err, ErrNotFound)
}
