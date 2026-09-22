package jev

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHttpResponseToJevResponseNoulAnswer(t *testing.T) {
	body := `{"model":"jev-latest","answers":{"segment_0":{"type":"noul","noul":0.99}}}`
	resp := &http.Response{Body: io.NopCloser(strings.NewReader(body))}

	jevResp, err := HttpResponseToJevResponse(resp)
	if err != nil {
		t.Fatalf("HttpResponseToJevResponse() error = %v", err)
	}

	answer, ok := jevResp.Answers["segment_0"].(*JevAnswerNoul)
	if !ok {
		t.Fatalf("answer type = %T, want *JevAnswerNoul", jevResp.Answers["segment_0"])
	}
	if answer.Noul != 0.99 {
		t.Errorf("Noul = %v, want 0.99", answer.Noul)
	}
}

func TestJevQuestionNoulMarshalNestedCriteria(t *testing.T) {
	question := &JevQuestionNoul{
		JevQuestion: JevQuestion{Type: NoulQuestionType, Instructions: "Is it part of the name?"},
		Criteria:    JevNoulCriteria{True: "yes", False: "no"},
	}

	data, err := json.Marshal(question)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	criteria, ok := raw["criteria"].(map[string]any)
	if !ok {
		t.Fatalf("criteria field missing or not an object: %v", raw)
	}
	if criteria["true"] != "yes" || criteria["false"] != "no" {
		t.Errorf("criteria = %v, want {true: yes, false: no}", criteria)
	}
	if _, ok := raw["true"]; ok {
		t.Errorf("flat true field should not be marshaled: %v", raw)
	}
	if _, ok := raw["false"]; ok {
		t.Errorf("flat false field should not be marshaled: %v", raw)
	}
}

func TestValidateJevRequestNoulRequiresDescriptions(t *testing.T) {
	request := &JevRequest{
		State: map[string]any{"message": "hi"},
		Model: "jev-latest",
		Questions: map[string]JevQuestionInterface{
			"segment_0": &JevQuestionNoul{
				JevQuestion: JevQuestion{Type: NoulQuestionType, Instructions: "Is it part of the name?"},
				Criteria:    JevNoulCriteria{True: "", False: "no"},
			},
		},
	}

	if err := validateJevRequest(request); err == nil {
		t.Error("validateJevRequest() = nil, want error for empty true description")
	}
}
