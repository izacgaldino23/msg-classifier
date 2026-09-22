package services

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"
)

type mockJevRequester struct {
	resp *jev.JevResponse
	err  error
	got  *jev.JevRequest
}

func (m *mockJevRequester) MakeJevRequest(request *jev.JevRequest) (*jev.JevResponse, error) {
	m.got = request
	return m.resp, m.err
}

func noulResponse(nouls ...float64) *jev.JevResponse {
	answers := make(map[string]jev.JevAnswer, len(nouls))
	for i, n := range nouls {
		answers[fmt.Sprintf("segment_%d", i)] = &jev.JevAnswerNoul{Type: jev.NoulQuestionType, Noul: n}
	}
	return &jev.JevResponse{Model: "jev-latest", Answers: answers}
}

func TestExtractPhone(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
		wantOK  bool
	}{
		{"mobile formatted", "(11) 91234-5678", "11912345678", true},
		{"mobile spaces", "11 91234-5678", "11912345678", true},
		{"mobile country code", "+55 11 91234-5678", "11912345678", true},
		{"mobile country code no plus", "55 11 91234-5678", "11912345678", true},
		{"mobile unformatted", "11987654321", "11987654321", true},
		{"landline unformatted", "1112345678", "1112345678", true},
		{"landline trunk prefix", "09292929290", "9292929290", true},
		{"landline trunk prefix spaced", "0 92 92929290", "9292929290", true},
		{"ddd 55 mobile", "55912345678", "55912345678", true},
		{"embedded in sentence", "me liga no 11987654321 por favor", "11987654321", true},
		{"too short", "91234-5678", "", false},
		{"short digits", "12345", "", false},
		{"no phone", "sem telefone aqui", "", false},
		{"invalid ddd", "0123456789", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewContactExtractor(&mockJevRequester{})
			got, span, ok := e.ExtractPhone(tt.message)
			if ok != tt.wantOK {
				t.Fatalf("ExtractPhone(%q) ok = %v, want %v", tt.message, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("ExtractPhone(%q) = %q, want %q", tt.message, got, tt.want)
			}
			if ok && (span.Start < 0 || span.End <= span.Start || span.End > len(tt.message)) {
				t.Errorf("ExtractPhone(%q) span = %+v, out of bounds", tt.message, span)
			}
		})
	}
}

func TestExtractEmail(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
		wantOK  bool
	}{
		{"simple", "x@y.com", "x@y.com", true},
		{"embedded", "salva contato do fulano, email x@y.com", "x@y.com", true},
		{"br domain", "fulano@exemplo.com.br", "fulano@exemplo.com.br", true},
		{"no email", "sem email aqui", "", false},
		{"no at sign", "fulano exemplo.com", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewContactExtractor(&mockJevRequester{})
			got, span, ok := e.ExtractEmail(tt.message)
			if ok != tt.wantOK {
				t.Fatalf("ExtractEmail(%q) ok = %v, want %v", tt.message, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("ExtractEmail(%q) = %q, want %q", tt.message, got, tt.want)
			}
			if ok && (span.Start < 0 || span.End <= span.Start || span.End > len(tt.message)) {
				t.Errorf("ExtractEmail(%q) span = %+v, out of bounds", tt.message, span)
			}
		})
	}
}

func TestExtractName(t *testing.T) {
	t.Run("builds noul fan-out and joins above threshold", func(t *testing.T) {
		mock := &mockJevRequester{resp: noulResponse(0.99, 0.1, 0.98)}
		e := NewContactExtractor(mock)

		result, err := e.ExtractName("09292929290 Fulano de Tal", []Span{{Start: 0, End: 11}})
		if err != nil {
			t.Fatalf("ExtractName() error = %v", err)
		}
		if result.Name != "Fulano Tal" {
			t.Errorf("Name = %q, want %q", result.Name, "Fulano Tal")
		}

		wantTrace := []models.SegmentScore{
			{Text: "Fulano", Score: 0.99, Included: true},
			{Text: "de", Score: 0.1, Included: false},
			{Text: "Tal", Score: 0.98, Included: true},
		}
		if len(result.Segments) != len(wantTrace) {
			t.Fatalf("Segments = %v, want %v", result.Segments, wantTrace)
		}
		for i := range wantTrace {
			if result.Segments[i] != wantTrace[i] {
				t.Errorf("Segments[%d] = %+v, want %+v", i, result.Segments[i], wantTrace[i])
			}
		}

		if mock.got == nil {
			t.Fatal("MakeJevRequest was not called")
		}
		state, ok := mock.got.State.(map[string]any)
		if !ok {
			t.Fatalf("state type = %T, want map[string]any", mock.got.State)
		}
		if state["message"] != "09292929290 Fulano de Tal" {
			t.Errorf("state message = %v, want original message", state["message"])
		}
		segments, ok := state["segments"].([]string)
		if !ok {
			t.Fatalf("state segments type = %T, want []string", state["segments"])
		}
		wantSegments := []string{"Fulano", "de", "Tal"}
		if len(segments) != len(wantSegments) {
			t.Fatalf("segments = %v, want %v", segments, wantSegments)
		}
		for i := range wantSegments {
			if segments[i] != wantSegments[i] {
				t.Errorf("segments[%d] = %q, want %q", i, segments[i], wantSegments[i])
			}
		}
		if len(mock.got.Questions) != 3 {
			t.Fatalf("questions count = %d, want 3", len(mock.got.Questions))
		}
		for i := 0; i < 3; i++ {
			key := fmt.Sprintf("segment_%d", i)
			q, ok := mock.got.Questions[key].(*jev.JevQuestionNoul)
			if !ok {
				t.Fatalf("question %q type = %T, want *jev.JevQuestionNoul", key, mock.got.Questions[key])
			}
			if q.Criteria.True == "" || q.Criteria.False == "" {
				t.Errorf("question %q criteria descriptions are empty: %+v", key, q.Criteria)
			}
			if !strings.Contains(q.Instructions, fmt.Sprintf("segments[%d]", i)) {
				t.Errorf("question %q instructions %q missing segment reference", key, q.Instructions)
			}
		}
	})

	t.Run("preserves segment order", func(t *testing.T) {
		mock := &mockJevRequester{resp: noulResponse(0.1, 0.99, 0.98)}
		e := NewContactExtractor(mock)

		result, err := e.ExtractName("Fulano de Tal", nil)
		if err != nil {
			t.Fatalf("ExtractName() error = %v", err)
		}
		if result.Name != "de Tal" {
			t.Errorf("Name = %q, want %q", result.Name, "de Tal")
		}
		wantTexts := []string{"Fulano", "de", "Tal"}
		if len(result.Segments) != len(wantTexts) {
			t.Fatalf("Segments = %v, want %v", result.Segments, wantTexts)
		}
		for i := range wantTexts {
			if result.Segments[i].Text != wantTexts[i] {
				t.Errorf("Segments[%d].Text = %q, want %q", i, result.Segments[i].Text, wantTexts[i])
			}
		}
	})

	t.Run("no segments skips jev call", func(t *testing.T) {
		mock := &mockJevRequester{}
		e := NewContactExtractor(mock)

		result, err := e.ExtractName("09292929290", []Span{{Start: 0, End: 11}})
		if err != nil {
			t.Fatalf("ExtractName() error = %v", err)
		}
		if result.Name != "" {
			t.Errorf("Name = %q, want empty", result.Name)
		}
		if result.Segments != nil {
			t.Errorf("Segments = %v, want nil", result.Segments)
		}
		if mock.got != nil {
			t.Error("MakeJevRequest should not be called with no segments")
		}
	})

	t.Run("jev failure wraps ErrUpstream", func(t *testing.T) {
		mock := &mockJevRequester{err: errors.New("boom")}
		e := NewContactExtractor(mock)

		_, err := e.ExtractName("Fulano de Tal", nil)
		if !errors.Is(err, ErrUpstream) {
			t.Errorf("ExtractName() error = %v, want wrapped ErrUpstream", err)
		}
	})

	t.Run("missing answer wraps ErrUpstream", func(t *testing.T) {
		mock := &mockJevRequester{resp: &jev.JevResponse{Model: "jev-latest", Answers: map[string]jev.JevAnswer{}}}
		e := NewContactExtractor(mock)

		_, err := e.ExtractName("Fulano de Tal", nil)
		if !errors.Is(err, ErrUpstream) {
			t.Errorf("ExtractName() error = %v, want wrapped ErrUpstream", err)
		}
	})
}
