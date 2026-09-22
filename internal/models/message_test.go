package models

import (
	"reflect"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestContactNullableFields(t *testing.T) {
	typ := reflect.TypeOf(Contact{})

	for _, field := range []string{"Phone", "Email"} {
		f, ok := typ.FieldByName(field)
		if !ok {
			t.Fatalf("field %s missing", field)
		}
		if f.Type.Kind() != reflect.Ptr {
			t.Errorf("field %s type = %v, want pointer (nullable)", field, f.Type)
		}
	}
}

func TestContactPrimaryKeyTag(t *testing.T) {
	f, ok := reflect.TypeOf(Contact{}).FieldByName("ID")
	if !ok {
		t.Fatal("field ID missing")
	}
	if f.Tag.Get("gorm") != "primaryKey" {
		t.Errorf("ID gorm tag = %q, want %q", f.Tag.Get("gorm"), "primaryKey")
	}
}

func TestUseCaseOutcomeCarriesContact(t *testing.T) {
	contact := &Contact{Name: "Fulano", Phone: strPtr("9292929290")}
	outcome := &UseCaseOutcome{Action: ActionContactAdd, Contact: contact}

	if outcome.Contact != contact {
		t.Errorf("outcome.Contact = %v, want %v", outcome.Contact, contact)
	}
	if ActionContactNoData != "contact_no_data" {
		t.Errorf("ActionContactNoData = %q, want %q", ActionContactNoData, "contact_no_data")
	}
}

func TestSegmentScoreFields(t *testing.T) {
	segment := SegmentScore{Text: "Fulano", Score: 0.91, Included: true}

	if segment.Text != "Fulano" || segment.Score != 0.91 || !segment.Included {
		t.Errorf("SegmentScore = %+v, want {Fulano 0.91 true}", segment)
	}
}

func TestUseCaseOutcomeCarriesSegments(t *testing.T) {
	segments := []SegmentScore{{Text: "Fulano", Score: 0.91, Included: true}}
	outcome := &UseCaseOutcome{Action: ActionContactAdd, Segments: segments}

	if len(outcome.Segments) != 1 || outcome.Segments[0].Text != "Fulano" || !outcome.Segments[0].Included {
		t.Errorf("outcome.Segments = %v, want the segment trace", outcome.Segments)
	}
}
