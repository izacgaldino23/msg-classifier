package services

import (
	"errors"
	"strings"
	"testing"

	"msg-classifier/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&models.Contact{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	return db
}

func TestContactServiceAddPersistsContact(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{resp: noulResponse(0.99, 0.1, 0.98)}
	service := NewContactService(NewContactExtractor(mock), db)

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "09292929290 Fulano de Tal"}, &models.Classification{})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if outcome.Action != models.ActionContactAdd {
		t.Errorf("Action = %q, want %q", outcome.Action, models.ActionContactAdd)
	}
	if outcome.Contact == nil {
		t.Fatal("Contact is nil")
	}
	if outcome.Contact.Name != "Fulano Tal" {
		t.Errorf("Name = %q, want %q", outcome.Contact.Name, "Fulano Tal")
	}
	if outcome.Contact.Phone == nil || *outcome.Contact.Phone != "9292929290" {
		t.Errorf("Phone = %v, want %q", outcome.Contact.Phone, "9292929290")
	}
	if outcome.Contact.Email != nil {
		t.Errorf("Email = %v, want nil", *outcome.Contact.Email)
	}
	if outcome.Contact.ID == 0 {
		t.Error("Contact.ID = 0, want persisted id")
	}

	wantSegments := []models.SegmentScore{
		{Text: "Fulano", Score: 0.99, Included: true},
		{Text: "de", Score: 0.1, Included: false},
		{Text: "Tal", Score: 0.98, Included: true},
	}
	if len(outcome.Segments) != len(wantSegments) {
		t.Fatalf("Segments = %v, want %v", outcome.Segments, wantSegments)
	}
	for i := range wantSegments {
		if outcome.Segments[i] != wantSegments[i] {
			t.Errorf("Segments[%d] = %+v, want %+v", i, outcome.Segments[i], wantSegments[i])
		}
	}

	var count int64
	if err := db.Model(&models.Contact{}).Count(&count).Error; err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 1 {
		t.Errorf("contacts count = %d, want 1", count)
	}
}

func TestContactServiceAddNoData(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{}
	service := NewContactService(NewContactExtractor(mock), db)

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "sem dados aqui"}, &models.Classification{})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if outcome.Action != models.ActionContactNoData {
		t.Errorf("Action = %q, want %q", outcome.Action, models.ActionContactNoData)
	}
	if outcome.Contact != nil {
		t.Errorf("Contact = %v, want nil", outcome.Contact)
	}
	if outcome.Segments != nil {
		t.Errorf("Segments = %v, want nil", outcome.Segments)
	}
	if mock.got != nil {
		t.Error("MakeJevRequest should not be called when no phone/email")
	}

	var count int64
	if err := db.Model(&models.Contact{}).Count(&count).Error; err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("contacts count = %d, want 0", count)
	}
}

func TestContactServiceAddEmailOnly(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{resp: noulResponse(0.1, 0.1, 0.1, 0.99, 0.1)}
	service := NewContactService(NewContactExtractor(mock), db)

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "salva contato do fulano email x@y.com"}, &models.Classification{})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if outcome.Action != models.ActionContactAdd {
		t.Errorf("Action = %q, want %q", outcome.Action, models.ActionContactAdd)
	}
	if outcome.Contact.Email == nil || *outcome.Contact.Email != "x@y.com" {
		t.Errorf("Email = %v, want %q", outcome.Contact.Email, "x@y.com")
	}
	if outcome.Contact.Phone != nil {
		t.Errorf("Phone = %v, want nil", *outcome.Contact.Phone)
	}
	if outcome.Contact.Name != "fulano" {
		t.Errorf("Name = %q, want %q", outcome.Contact.Name, "fulano")
	}
	if len(outcome.Segments) != 5 {
		t.Fatalf("Segments len = %d, want 5", len(outcome.Segments))
	}
	if outcome.Segments[3].Text != "fulano" || !outcome.Segments[3].Included {
		t.Errorf("Segments[3] = %+v, want included fulano", outcome.Segments[3])
	}
}

func TestContactServiceAddJevFailure(t *testing.T) {
	db := newTestDB(t)
	mock := &mockJevRequester{err: errors.New("boom")}
	service := NewContactService(NewContactExtractor(mock), db)

	_, err := service.Add(&models.ReceiveMessageRequest{Message: "09292929290 Fulano de Tal"}, &models.Classification{})
	if !errors.Is(err, ErrUpstream) {
		t.Errorf("Add() error = %v, want wrapped ErrUpstream", err)
	}
}

func TestContactServiceAddDBFailure(t *testing.T) {
	db := newTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("sqlDB.Close() error = %v", err)
	}

	mock := &mockJevRequester{resp: noulResponse(0.99)}
	service := NewContactService(NewContactExtractor(mock), db)

	_, err = service.Add(&models.ReceiveMessageRequest{Message: "09292929290 Fulano"}, &models.Classification{})
	if err == nil {
		t.Fatal("Add() = nil, want db error")
	}
	if !strings.Contains(err.Error(), "failed to persist contact") {
		t.Errorf("Add() error = %v, want wrapped persist context", err)
	}
}
