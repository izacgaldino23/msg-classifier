package contact

import (
	"testing"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContactServiceNameFallbackDuplicate(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.Create(&models.Contact{Name: "Fulano de Tal", NameNorm: "fulano de tal"}).Error)

	mock := &mockJevRequester{resp: noulResponse(0.1, 0.1, 0.1, 0.1, 0.99, 0.99, 0.99)}
	service := NewService(NewExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "salva o contato do Fulano de Tal"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactDuplicate, outcome.Action)
	require.NotNil(t, outcome.Contact)
	assert.Equal(t, "Fulano de Tal", outcome.Contact.Name)
	require.NotNil(t, mock.got, "the name fallback extracts before checking")

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Count(&count).Error)
	assert.Equal(t, int64(1), count, "no new row on a name hit")
}

func TestContactServiceNameFallbackMissCreates(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{resp: noulResponse(0.1, 0.1, 0.1, 0.1, 0.99, 0.99, 0.99)}
	service := NewService(NewExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "salva o contato do Fulano de Tal"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactAdd, outcome.Action)
	require.NotNil(t, outcome.Contact)
	assert.Equal(t, "fulano de tal", outcome.Contact.NameNorm)

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestContactServiceDuplicateNewSkipsCheck(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.Create(&models.Contact{Name: "Fulano", Phone: strPtr("9292929290")}).Error)

	mock := &mockJevRequester{resp: noulResponse(0.99, 0.1, 0.98)}
	service := NewService(NewExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Add(&models.ReceiveMessageRequest{
		Message: "09292929290 Fulano de Tal", DupAction: "new",
	}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactAdd, outcome.Action)
	require.NotNil(t, mock.got, "the name extraction re-runs on the save-anyway path")

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Count(&count).Error)
	assert.Equal(t, int64(2), count, "dup_action=new inserts a second row")
}

func TestContactServiceDuplicateUpdateMerges(t *testing.T) {
	db := newTestDB(t)
	seeded := &models.Contact{Name: "Fulano", NameNorm: "fulano", Phone: strPtr("9292929290")}
	require.NoError(t, db.Create(seeded).Error)

	mock := &mockJevRequester{resp: noulResponse(0.99, 0.1, 0.98)}
	service := NewService(NewExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Add(&models.ReceiveMessageRequest{
		Message: "09292929290 Fulano de Tal", DupAction: "update",
	}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactAdd, outcome.Action)
	require.NotNil(t, outcome.Contact)
	assert.Equal(t, seeded.ID, outcome.Contact.ID, "update keeps the existing row")
	assert.Equal(t, "Fulano de Tal", outcome.Contact.Name, "the extracted name overwrites")
	assert.Equal(t, "fulano de tal", outcome.Contact.NameNorm)
	require.NotNil(t, outcome.Contact.Phone)
	assert.Equal(t, "9292929290", *outcome.Contact.Phone)

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Count(&count).Error)
	assert.Equal(t, int64(1), count, "update must not insert a second row")
}

func TestContactServiceUnknownDupActionChecksNormally(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.Create(&models.Contact{Name: "Fulano", Phone: strPtr("9292929290")}).Error)

	service := NewService(NewExtractor(&mockJevRequester{}), repository.NewContactRepository(db))

	outcome, err := service.Add(&models.ReceiveMessageRequest{
		Message: "09292929290", DupAction: "banana",
	}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactDuplicate, outcome.Action, "an unknown dup_action is treated as empty")
}
