package services

import (
	"errors"
	"strings"
	"testing"

	"msg-classifier/internal/models"
)

func TestContactServiceGetByPhone(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano Tal", Phone: strPtr("9292929290")})

	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), db)

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "09292929290"}, &models.Classification{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if outcome.Action != models.ActionContactFound {
		t.Errorf("Action = %q, want %q", outcome.Action, models.ActionContactFound)
	}
	if outcome.Contact == nil || outcome.Contact.Name != "Fulano Tal" {
		t.Errorf("Contact = %+v, want Fulano Tal", outcome.Contact)
	}
	if outcome.SearchTerm != "9292929290" {
		t.Errorf("SearchTerm = %q, want %q", outcome.SearchTerm, "9292929290")
	}
	if outcome.Segments != nil {
		t.Errorf("Segments = %v, want nil for phone search", outcome.Segments)
	}
	if mock.got != nil {
		t.Error("MakeJevRequest should not be called for phone search")
	}
}

func TestContactServiceGetByPhoneNotFound(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), db)

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "09292929290"}, &models.Classification{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if outcome.Action != models.ActionContactNotFound {
		t.Errorf("Action = %q, want %q", outcome.Action, models.ActionContactNotFound)
	}
	if outcome.Contact != nil {
		t.Errorf("Contact = %v, want nil", outcome.Contact)
	}
	if outcome.SearchTerm != "9292929290" {
		t.Errorf("SearchTerm = %q, want %q", outcome.SearchTerm, "9292929290")
	}
}

func TestContactServiceGetByEmail(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano", Email: strPtr("X@Y.COM")})

	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), db)

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "email x@y.com"}, &models.Classification{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if outcome.Action != models.ActionContactFound {
		t.Errorf("Action = %q, want %q", outcome.Action, models.ActionContactFound)
	}
	if outcome.Contact == nil || outcome.Contact.Name != "Fulano" {
		t.Errorf("Contact = %+v, want Fulano", outcome.Contact)
	}
	if outcome.SearchTerm != "x@y.com" {
		t.Errorf("SearchTerm = %q, want %q", outcome.SearchTerm, "x@y.com")
	}
	if mock.got != nil {
		t.Error("MakeJevRequest should not be called for email search")
	}
}

func TestContactServiceGetByEmailNotFound(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), db)

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "email x@y.com"}, &models.Classification{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if outcome.Action != models.ActionContactNotFound {
		t.Errorf("Action = %q, want %q", outcome.Action, models.ActionContactNotFound)
	}
	if outcome.SearchTerm != "x@y.com" {
		t.Errorf("SearchTerm = %q, want %q", outcome.SearchTerm, "x@y.com")
	}
}

func TestContactServiceGetByName(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano Tal", NameNorm: "fulano tal"})

	mock := &mockJevRequester{resp: noulResponse(0.1, 0.1, 0.99, 0.1, 0.99)}
	service := NewContactService(NewContactExtractor(mock), db)

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "Número de fulano de tal"}, &models.Classification{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if outcome.Action != models.ActionContactFound {
		t.Errorf("Action = %q, want %q", outcome.Action, models.ActionContactFound)
	}
	if outcome.Contact == nil || outcome.Contact.Name != "Fulano Tal" {
		t.Errorf("Contact = %+v, want Fulano Tal", outcome.Contact)
	}
	if outcome.SearchTerm != "fulano tal" {
		t.Errorf("SearchTerm = %q, want %q", outcome.SearchTerm, "fulano tal")
	}
	if len(outcome.Segments) != 5 {
		t.Errorf("Segments len = %d, want 5 (trace kept)", len(outcome.Segments))
	}
}

func TestContactServiceGetByNameNotFound(t *testing.T) {
	db := newTestDB(t)
	db.Create(&models.Contact{Name: "Fulano Tal", NameNorm: "fulano tal"})

	mock := &mockJevRequester{resp: noulResponse(0.99, 0.99, 0.99, 0.99, 0.99)}
	service := NewContactService(NewContactExtractor(mock), db)

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "Número de fulano de tal"}, &models.Classification{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if outcome.Action != models.ActionContactNotFound {
		t.Errorf("Action = %q, want %q", outcome.Action, models.ActionContactNotFound)
	}
	if outcome.SearchTerm != "numero de fulano de tal" {
		t.Errorf("SearchTerm = %q, want %q", outcome.SearchTerm, "numero de fulano de tal")
	}
}

func TestContactServiceGetNoData(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{resp: noulResponse(0.1, 0.1)}
	service := NewContactService(NewContactExtractor(mock), db)

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "qualquer coisa"}, &models.Classification{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if outcome.Action != models.ActionContactNoData {
		t.Errorf("Action = %q, want %q", outcome.Action, models.ActionContactNoData)
	}
	if outcome.SearchTerm != "" {
		t.Errorf("SearchTerm = %q, want empty", outcome.SearchTerm)
	}
}

func TestContactServiceGetJevFailure(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{err: errors.New("boom")}
	service := NewContactService(NewContactExtractor(mock), db)

	_, err := service.Get(&models.ReceiveMessageRequest{Message: "Número de fulano de tal"}, &models.Classification{})
	if !errors.Is(err, ErrUpstream) {
		t.Errorf("Get() error = %v, want wrapped ErrUpstream", err)
	}
}

func TestContactServiceGetDBFailure(t *testing.T) {
	db := newTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("sqlDB.Close() error = %v", err)
	}

	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), db)

	_, err = service.Get(&models.ReceiveMessageRequest{Message: "09292929290"}, &models.Classification{})
	if err == nil {
		t.Fatal("Get() = nil, want db error")
	}
	if !strings.Contains(err.Error(), "failed to search contact") {
		t.Errorf("Get() error = %v, want wrapped search context", err)
	}
}
