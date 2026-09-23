package services

import (
	"errors"
	"testing"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContactServiceGetByPhone(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano Tal", Phone: strPtr("9292929290")})

	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "09292929290"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactFound, outcome.Action)
	require.NotNil(t, outcome.Contact)
	assert.Equal(t, "Fulano Tal", outcome.Contact.Name)
	assert.Equal(t, "9292929290", outcome.SearchTerm)
	assert.Nil(t, outcome.Segments, "no segments for phone search")
	assert.Nil(t, mock.got, "MakeJevRequest should not be called for phone search")
}

func TestContactServiceGetByPhoneNotFound(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "09292929290"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactNotFound, outcome.Action)
	assert.Nil(t, outcome.Contact)
	assert.Equal(t, "9292929290", outcome.SearchTerm)
}

func TestContactServiceGetByEmail(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano", Email: strPtr("X@Y.COM")})

	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "email x@y.com"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactFound, outcome.Action)
	require.NotNil(t, outcome.Contact)
	assert.Equal(t, "Fulano", outcome.Contact.Name)
	assert.Equal(t, "x@y.com", outcome.SearchTerm)
	assert.Nil(t, mock.got, "MakeJevRequest should not be called for email search")
}

func TestContactServiceGetByEmailNotFound(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "email x@y.com"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactNotFound, outcome.Action)
	assert.Equal(t, "x@y.com", outcome.SearchTerm)
}

func TestContactServiceGetByName(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano Tal", NameNorm: "fulano tal"})

	mock := &mockJevRequester{resp: noulResponse(0.1, 0.1, 0.99, 0.1, 0.99)}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "Número de fulano de tal"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactFound, outcome.Action)
	require.NotNil(t, outcome.Contact)
	assert.Equal(t, "Fulano Tal", outcome.Contact.Name)
	assert.Equal(t, "fulano tal", outcome.SearchTerm)
	assert.Len(t, outcome.Segments, 5, "trace kept")
}

func TestContactServiceGetByNameNotFound(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano Tal", NameNorm: "fulano tal"})

	mock := &mockJevRequester{resp: noulResponse(0.99, 0.99, 0.99, 0.99, 0.99)}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "Número de fulano de tal"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactNotFound, outcome.Action)
	assert.Equal(t, "numero de fulano de tal", outcome.SearchTerm)
}

func TestContactServiceGetNoData(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{resp: noulResponse(0.1, 0.1)}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "qualquer coisa"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionContactNoData, outcome.Action)
	assert.Equal(t, "", outcome.SearchTerm)
}

func TestContactServiceGetJevFailure(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{err: errors.New("boom")}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	_, err := service.Get(&models.ReceiveMessageRequest{Message: "Número de fulano de tal"}, &models.Classification{})
	assert.ErrorIs(t, err, ErrUpstream)
}

func TestContactServiceGetDBFailure(t *testing.T) {
	db := newTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), repository.NewContactRepository(db))

	_, err = service.Get(&models.ReceiveMessageRequest{Message: "09292929290"}, &models.Classification{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to search contact")
}