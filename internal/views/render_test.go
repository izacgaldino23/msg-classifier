package views

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"msg-classifier/internal/models"

	"github.com/gin-gonic/gin"
)

func strPtr(s string) *string { return &s }

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.SetHTMLTemplate(template.Must(template.ParseGlob("../../web/templates/**/*.html")))
	return c, w
}

func TestRenderResultContactAdd(t *testing.T) {
	c, w := newTestContext()

	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "contact", Confidence: 0.9},
			Kind:     models.KindFinding{Choice: "add", Confidence: 0.8},
		},
		Action:  models.ActionContactAdd,
		Contact: &models.Contact{ID: 7, Name: "Fulano Tal", Phone: strPtr("9292929290")},
		Segments: []models.SegmentScore{
			{Text: "Fulano", Score: 0.91, Included: true},
			{Text: "de", Score: 0.08, Included: false},
			{Text: "Tal", Score: 0.88, Included: true},
		},
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"contact", "add", "Contato salvo", "ID 7", "Fulano Tal", "9292929290", "Fulano 0.91", "de 0.08", "Tal 0.88", "✓", "✗"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderResultContactNoData(t *testing.T) {
	c, w := newTestContext()

	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "contact", Confidence: 0.9},
			Kind:     models.KindFinding{Choice: "add", Confidence: 0.8},
		},
		Action: models.ActionContactNoData,
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"contact", "add", "Nenhum dado de contato"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderResultClassificationOnly(t *testing.T) {
	c, w := newTestContext()

	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "notes", Confidence: 0.7},
			Kind:     models.KindFinding{Choice: "add", Confidence: 0.6},
		},
		Action: models.ActionNone,
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"notes", "70.00%", "add"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}
