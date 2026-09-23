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
	shared := template.Must(template.ParseGlob("../../web/templates/layouts/*.html"))
	shared = template.Must(shared.ParseGlob("../../web/templates/partial/*.html"))
	pr, err := NewPagesRenderer(shared, map[string]string{
		HomePage:    "../../web/templates/pages/index.html",
		PromptsPage: "../../web/templates/pages/prompts.html",
	})
	if err != nil {
		panic("failed to build pages renderer: " + err.Error())
	}
	engine.HTMLRender = pr
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

func TestRenderResultContactFound(t *testing.T) {
	c, w := newTestContext()

	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "contact", Confidence: 0.9},
			Kind:     models.KindFinding{Choice: "require", Confidence: 0.8},
		},
		Action:     models.ActionContactFound,
		Contact:    &models.Contact{Name: "Fulano Tal", Phone: strPtr("9292929290")},
		SearchTerm: "9292929290",
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"contact", "require", "Contato encontrado", "Fulano Tal", "9292929290"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderResultContactNotFound(t *testing.T) {
	c, w := newTestContext()

	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "contact", Confidence: 0.9},
			Kind:     models.KindFinding{Choice: "require", Confidence: 0.8},
		},
		Action:     models.ActionContactNotFound,
		SearchTerm: "fulano tal",
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Nenhum contato encontrado para 'fulano tal'") {
		t.Errorf("body missing not-found message with SearchTerm: %s", body)
	}
}

func TestRenderResultContactDuplicate(t *testing.T) {
	c, w := newTestContext()

	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "contact", Confidence: 0.9},
			Kind:     models.KindFinding{Choice: "add", Confidence: 0.8},
		},
		Action:  models.ActionContactDuplicate,
		Contact: &models.Contact{Name: "Fulano", Phone: strPtr("9292929290")},
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"Contato já existe: Fulano", "9292929290"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderPromptTable(t *testing.T) {
	c, w := newTestContext()

	prompts := []models.JevPrompt{
		{ID: 1, Flow: models.FlowClassification, Message: "salva fulano", ExpectedResult: "contact:add"},
		{ID: 2, Flow: models.FlowClassification, Message: "quanto gastei?", ExpectedResult: "finance:require"},
	}
	RenderPromptTable(c, models.FlowClassification, prompts)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"prompt-form", "salva fulano", "contact:add", "quanto gastei?", "finance:require", `name="ids"`, `value="1"`, `value="2"`, "Avaliar", "Exportar CSV"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderEvaluationResults(t *testing.T) {
	c, w := newTestContext()

	results := []models.EvaluationResult{
		{PromptID: 1, Message: "salva fulano", ExpectedResult: "contact:add", ObtainedResult: "contact:add", Match: true},
		{PromptID: 2, Message: "quanto gastei?", ExpectedResult: "contact:add", ObtainedResult: "finance:require", Match: false},
	}
	RenderEvaluationResults(c, models.FlowClassification, results)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"salva fulano", "contact:add", "finance:require", "✓", "✗"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderEvaluationResultsNameSegments(t *testing.T) {
	c, w := newTestContext()

	results := []models.EvaluationResult{
		{
			PromptID: 3, Message: "João da Silva", ExpectedResult: "João da Silva",
			ObtainedResult: "João da Silva", Match: true,
			Segments: []models.SegmentScore{
				{Text: "João", Score: 0.99, Included: true},
				{Text: "da", Score: 0.98, Included: true},
				{Text: "Silva", Score: 0.97, Included: true},
			},
		},
	}
	RenderEvaluationResults(c, models.FlowName, results)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"João 0.99", "da 0.98", "Silva 0.97"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderExportResult(t *testing.T) {
	c, w := newTestContext()

	RenderExportResult(c, "exports/classification-20260923-101530.csv")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "exports/classification-20260923-101530.csv") {
		t.Errorf("body missing export path: %s", body)
	}
}

func TestRenderHomePage(t *testing.T) {
	c, w := newTestContext()

	RenderPage(c, HomePage, HomePageContent)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"Classificador de Mensagens", `hx-post="/api/message"`} {
		if !strings.Contains(body, want) {
			t.Errorf("home page missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, `hx-get="/prompts/table"`) {
		t.Errorf("home page leaked prompts content: %s", body)
	}
}

func TestRenderPromptsPage(t *testing.T) {
	c, w := newTestContext()

	RenderPage(c, PromptsPage, PromptsPageContent)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"Validação Jev", `hx-get="/prompts/table"`, "Adicionar exemplo"} {
		if !strings.Contains(body, want) {
			t.Errorf("prompts page missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "Classificador de Mensagens") {
		t.Errorf("prompts page leaked home content: %s", body)
	}
}
