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

func TestContactNameNormField(t *testing.T) {
	f, ok := reflect.TypeOf(Contact{}).FieldByName("NameNorm")
	if !ok {
		t.Fatal("field NameNorm missing")
	}
	if f.Type.Kind() != reflect.String {
		t.Errorf("field NameNorm type = %v, want string", f.Type.Kind())
	}
	if f.Tag.Get("json") != "name_norm" {
		t.Errorf("NameNorm json tag = %q, want %q", f.Tag.Get("json"), "name_norm")
	}
}

func TestContactSearchActionConstants(t *testing.T) {
	if ActionContactFound != "contact_found" {
		t.Errorf("ActionContactFound = %q, want %q", ActionContactFound, "contact_found")
	}
	if ActionContactNotFound != "contact_not_found" {
		t.Errorf("ActionContactNotFound = %q, want %q", ActionContactNotFound, "contact_not_found")
	}
	if ActionContactDuplicate != "contact_duplicate" {
		t.Errorf("ActionContactDuplicate = %q, want %q", ActionContactDuplicate, "contact_duplicate")
	}
}

func TestUseCaseOutcomeCarriesSearchTerm(t *testing.T) {
	outcome := &UseCaseOutcome{Action: ActionContactNotFound, SearchTerm: "fulano tal"}
	if outcome.SearchTerm != "fulano tal" {
		t.Errorf("outcome.SearchTerm = %q, want %q", outcome.SearchTerm, "fulano tal")
	}
}

func TestNoteActionConstants(t *testing.T) {
	for value, want := range map[Action]string{
		ActionNoteAdd:      "note_add",
		ActionNoteNoData:   "note_no_data",
		ActionNoteFound:    "note_found",
		ActionNoteNotFound: "note_not_found",
	} {
		if string(value) != want {
			t.Errorf("action = %q, want %q", value, want)
		}
	}
}

func TestUseCaseOutcomeCarriesNotes(t *testing.T) {
	notes := []*Note{{ID: 1, Type: NoteTypeTodo, Content: "comprar pão"}}
	outcome := &UseCaseOutcome{Action: ActionNoteAdd, Notes: notes}

	if len(outcome.Notes) != 1 {
		t.Fatalf("outcome.Notes = %v, want 1 note", outcome.Notes)
	}
	if outcome.Notes[0].Type != NoteTypeTodo || outcome.Notes[0].Content != "comprar pão" {
		t.Errorf("outcome.Notes[0] = %+v", outcome.Notes[0])
	}
	if outcome.Contact != nil {
		t.Errorf("outcome.Contact = %v, want nil for the notes flow", outcome.Contact)
	}
}

func TestDuplicateActionConstants(t *testing.T) {
	for value, want := range map[Action]string{
		ActionNoteDuplicate:        "note_duplicate",
		ActionTransactionDuplicate: "transaction_duplicate",
	} {
		if string(value) != want {
			t.Errorf("action = %q, want %q", value, want)
		}
	}
}

func TestReceiveMessageRequestDupActionTags(t *testing.T) {
	f, ok := reflect.TypeOf(ReceiveMessageRequest{}).FieldByName("DupAction")
	if !ok {
		t.Fatal("field DupAction missing")
	}
	if f.Tag.Get("json") != "dup_action" || f.Tag.Get("form") != "dup_action" {
		t.Errorf("DupAction tags = json:%q form:%q, want dup_action on both", f.Tag.Get("json"), f.Tag.Get("form"))
	}
}

func TestUseCaseOutcomeCarriesMessage(t *testing.T) {
	outcome := &UseCaseOutcome{Action: ActionNoteDuplicate, Message: "anota que o projeto é atlas"}
	if outcome.Message != "anota que o projeto é atlas" {
		t.Errorf("outcome.Message = %q, want the original message", outcome.Message)
	}
}
