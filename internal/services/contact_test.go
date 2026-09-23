package services

import (
	"errors"
	"testing"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"

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

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"case", "FULANO", "fulano"},
		{"accent", "João", "joao"},
		{"cedilla", "José da Conceição", "jose da conceicao"},
		{"mixed accents", "MARIA CLÁUDIA", "maria claudia"},
		{"collapse whitespace", "  Fulano   de  Tal ", "fulano de tal"},
		{"tabs and newlines", "Fulano\tde\nTal", "fulano de tal"},
		{"already normalized", "fulano de tal", "fulano de tal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeName(tt.in))
		})
	}
}

func TestContactServiceAddPersistsContact(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{resp: noulResponse(0.99, 0.1, 0.98)}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "09292929290 Fulano de Tal"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactAdd, outcome.Action)
	require.NotNil(t, outcome.Contact)
	assert.Equal(t, "Fulano Tal", outcome.Contact.Name)
	assert.Equal(t, "fulano tal", outcome.Contact.NameNorm)
	require.NotNil(t, outcome.Contact.Phone)
	assert.Equal(t, "9292929290", *outcome.Contact.Phone)
	assert.Nil(t, outcome.Contact.Email)
	assert.NotZero(t, outcome.Contact.ID)

	wantSegments := []models.SegmentScore{
		{Text: "Fulano", Score: 0.99, Included: true},
		{Text: "de", Score: 0.1, Included: false},
		{Text: "Tal", Score: 0.98, Included: true},
	}
	assert.Equal(t, wantSegments, outcome.Segments)

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestContactServiceAddNoData(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "sem dados aqui"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactNoData, outcome.Action)
	assert.Nil(t, outcome.Contact)
	assert.Nil(t, outcome.Segments)
	assert.Nil(t, mock.got, "MakeJevRequest should not be called when no phone/email")

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestContactServiceAddEmailOnly(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{resp: noulResponse(0.1, 0.1, 0.1, 0.99, 0.1)}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "salva contato do fulano email x@y.com"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactAdd, outcome.Action)
	require.NotNil(t, outcome.Contact.Email)
	assert.Equal(t, "x@y.com", *outcome.Contact.Email)
	assert.Nil(t, outcome.Contact.Phone)
	assert.Equal(t, "fulano", outcome.Contact.Name)
	require.Len(t, outcome.Segments, 5)
	assert.Equal(t, "fulano", outcome.Segments[3].Text)
	assert.True(t, outcome.Segments[3].Included)
}

func TestContactServiceAddJevFailure(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{err: errors.New("boom")}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	_, err := service.Add(&models.ReceiveMessageRequest{Message: "09292929290 Fulano de Tal"}, &models.Classification{})
	assert.ErrorIs(t, err, ErrUpstream)
}

func TestContactServiceAddDBFailure(t *testing.T) {
	db := newTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	mock := &mockJevRequester{resp: noulResponse(0.99)}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	_, err = service.Add(&models.ReceiveMessageRequest{Message: "09292929290 Fulano"}, &models.Classification{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check duplicate contact")
}

func TestContactServiceAddDuplicatePhone(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano", Phone: strPtr("9292929290")})

	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "09292929290 Fulano de Tal"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactDuplicate, outcome.Action)
	require.NotNil(t, outcome.Contact)
	assert.Equal(t, "Fulano", outcome.Contact.Name)
	assert.Nil(t, outcome.Segments, "no Jev spent on duplicate")
	assert.Nil(t, mock.got, "MakeJevRequest should not be called on duplicate")

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Count(&count).Error)
	assert.Equal(t, int64(1), count, "no new row")
}

func TestContactServiceAddDuplicateEmail(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano", Email: strPtr("X@Y.COM")})

	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "salva fulano email x@y.com"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactDuplicate, outcome.Action)
	require.NotNil(t, outcome.Contact)
	assert.Equal(t, "Fulano", outcome.Contact.Name)
	assert.Nil(t, mock.got, "MakeJevRequest should not be called on duplicate")
}

func TestContactServiceBackfillNameNorm(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "João da Silva"})            // pre-migration row: empty NameNorm
	db.Create(&models.Contact{Name: "Maria", NameNorm: "maria"}) // already filled

	service := NewContactService(NewContactExtractor(&mockJevRequester{}), repository.NewContactRepository(db))
	require.NoError(t, service.BackfillNameNorm())

	var joao models.Contact
	require.NoError(t, db.Where("name = ?", "João da Silva").First(&joao).Error)
	assert.Equal(t, "joao da silva", joao.NameNorm)

	var maria models.Contact
	require.NoError(t, db.Where("name = ?", "Maria").First(&maria).Error)
	assert.Equal(t, "maria", maria.NameNorm)
}

func TestContactServiceHandleRequireRoutesToGet(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano", Phone: strPtr("9292929290")})

	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Handle(&models.ReceiveMessageRequest{Message: "09292929290"}, &models.Classification{Kind: models.KindFinding{Choice: "require"}})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactFound, outcome.Action)
	require.NotNil(t, outcome.Contact)
	assert.Equal(t, "Fulano", outcome.Contact.Name)
}