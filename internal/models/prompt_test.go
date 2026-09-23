package models

import (
	"reflect"
	"testing"
)

func TestFlowConstants(t *testing.T) {
	if FlowClassification != "classification" {
		t.Errorf("FlowClassification = %q, want %q", FlowClassification, "classification")
	}
	if FlowName != "name" {
		t.Errorf("FlowName = %q, want %q", FlowName, "name")
	}
}

func TestJevPromptPrimaryKeyTag(t *testing.T) {
	f, ok := reflect.TypeOf(JevPrompt{}).FieldByName("ID")
	if !ok {
		t.Fatal("field ID missing")
	}
	if f.Tag.Get("gorm") != "primaryKey" {
		t.Errorf("ID gorm tag = %q, want %q", f.Tag.Get("gorm"), "primaryKey")
	}
}

func TestJevPromptJSONTags(t *testing.T) {
	typ := reflect.TypeOf(JevPrompt{})
	for _, tc := range []struct{ field, tag string }{
		{"Flow", "flow"},
		{"Message", "message"},
		{"ExpectedResult", "expected_result"},
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

func TestEvaluationResultFields(t *testing.T) {
	result := EvaluationResult{PromptID: 1, Message: "msg", ExpectedResult: "contact:add", ObtainedResult: "contact:add", Match: true}
	if result.PromptID != 1 || result.Message != "msg" || result.ExpectedResult != "contact:add" || result.ObtainedResult != "contact:add" || !result.Match {
		t.Errorf("EvaluationResult = %+v", result)
	}
	if result.Segments != nil {
		t.Errorf("Segments = %v, want nil for classification", result.Segments)
	}
}