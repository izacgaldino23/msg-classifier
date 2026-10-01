package models

import (
	"reflect"
	"testing"
)

func TestDataFormTags(t *testing.T) {
	typ := reflect.TypeOf(DataForm{})
	for _, tc := range []struct{ field, tag string }{
		{"Name", "name"}, {"Phone", "phone"}, {"Email", "email"},
		{"Content", "content"}, {"Date", "date"}, {"Time", "time"},
		{"ItemText", "item_text"}, {"ItemDone", "item_done"},
	} {
		f, ok := typ.FieldByName(tc.field)
		if !ok {
			t.Fatalf("DataForm field %s missing", tc.field)
		}
		if got := f.Tag.Get("form"); got != tc.tag {
			t.Errorf("DataForm.%s form tag = %q, want %q", tc.field, got, tc.tag)
		}
		if got := f.Tag.Get("json"); got != tc.tag {
			t.Errorf("DataForm.%s json tag = %q, want %q", tc.field, got, tc.tag)
		}
	}
}

func TestDataFormItemSlicesAreParallel(t *testing.T) {
	form := DataForm{ItemText: []string{"pão", "leite"}, ItemDone: []string{"1", "0"}}
	if len(form.ItemText) != 2 || len(form.ItemDone) != 2 {
		t.Fatalf("DataForm = %+v", form)
	}
}

func TestDataDeleteFormTags(t *testing.T) {
	typ := reflect.TypeOf(DataDeleteForm{})
	for _, tc := range []struct{ field, tag string }{
		{"Kind", "kind"}, {"Filter", "filter"}, {"IDs", "ids"},
	} {
		f, ok := typ.FieldByName(tc.field)
		if !ok {
			t.Fatalf("DataDeleteForm field %s missing", tc.field)
		}
		if got := f.Tag.Get("form"); got != tc.tag {
			t.Errorf("DataDeleteForm.%s form tag = %q, want %q", tc.field, got, tc.tag)
		}
	}
}
