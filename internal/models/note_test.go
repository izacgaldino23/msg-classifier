package models

import (
	"reflect"
	"testing"
)

func TestNoteTypeConstants(t *testing.T) {
	for value, want := range map[string]string{
		NoteTypeNote:     "note",
		NoteTypeReminder: "reminder",
		NoteTypeTodo:     "todo",
	} {
		if value != want {
			t.Errorf("note type = %q, want %q", value, want)
		}
	}
}

func TestNotePrimaryKeyTag(t *testing.T) {
	f, ok := reflect.TypeOf(Note{}).FieldByName("ID")
	if !ok {
		t.Fatal("field ID missing")
	}
	if f.Tag.Get("gorm") != "primaryKey" {
		t.Errorf("ID gorm tag = %q, want %q", f.Tag.Get("gorm"), "primaryKey")
	}
}

func TestNoteNullableFields(t *testing.T) {
	typ := reflect.TypeOf(Note{})
	for _, field := range []string{"Date", "Time"} {
		f, ok := typ.FieldByName(field)
		if !ok {
			t.Fatalf("field %s missing", field)
		}
		if f.Type.Kind() != reflect.Ptr {
			t.Errorf("field %s type = %v, want pointer (nullable)", field, f.Type)
		}
	}
}

func TestNoteContentNotNull(t *testing.T) {
	f, ok := reflect.TypeOf(Note{}).FieldByName("Content")
	if !ok {
		t.Fatal("field Content missing")
	}
	if f.Tag.Get("gorm") != "not null" {
		t.Errorf("Content gorm tag = %q, want %q", f.Tag.Get("gorm"), "not null")
	}
}

func TestNoteIndexTags(t *testing.T) {
	typ := reflect.TypeOf(Note{})
	for _, field := range []string{"Type", "Date"} {
		f, ok := typ.FieldByName(field)
		if !ok {
			t.Fatalf("field %s missing", field)
		}
		if f.Tag.Get("gorm") != "index" {
			t.Errorf("%s gorm tag = %q, want %q", field, f.Tag.Get("gorm"), "index")
		}
	}
}

func TestNoteJSONTags(t *testing.T) {
	typ := reflect.TypeOf(Note{})
	for _, tc := range []struct{ field, tag string }{
		{"Type", "type"},
		{"Content", "content"},
		{"Date", "date"},
		{"Time", "time"},
		{"CreatedAt", "created_at"},
		{"UpdatedAt", "updated_at"},
	} {
		f, ok := typ.FieldByName(tc.field)
		if !ok {
			t.Fatalf("field %s missing", tc.field)
		}
		if f.Tag.Get("json") != tc.tag {
			t.Errorf("%s json tag = %q, want %q", tc.field, f.Tag.Get("json"), tc.tag)
		}
	}
}

func TestTodoItemFields(t *testing.T) {
	typ := reflect.TypeOf(TodoItem{})
	for _, tc := range []struct{ field, tag string }{
		{"ID", "id"},
		{"NoteID", "note_id"},
		{"Text", "text"},
		{"Done", "done"},
		{"Position", "position"},
	} {
		f, ok := typ.FieldByName(tc.field)
		if !ok {
			t.Fatalf("field %s missing", tc.field)
		}
		if f.Tag.Get("json") != tc.tag {
			t.Errorf("%s json tag = %q, want %q", tc.field, f.Tag.Get("json"), tc.tag)
		}
	}

	done, ok := typ.FieldByName("Done")
	if !ok {
		t.Fatal("field Done missing")
	}
	if done.Type.Kind() != reflect.Bool {
		t.Errorf("Done type = %v, want bool", done.Type.Kind())
	}
	if done.Tag.Get("gorm") != "index" {
		t.Errorf("Done gorm tag = %q, want %q", done.Tag.Get("gorm"), "index")
	}
}