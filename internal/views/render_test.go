package views

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"msg-classifier/internal/messages"
	"msg-classifier/internal/models"

	"github.com/gin-gonic/gin"
)

func strPtr(s string) *string { return &s }

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	shared := template.Must(template.New("").Funcs(FuncMap).ParseGlob("../../web/templates/layouts/*.html"))
	shared = template.Must(shared.ParseGlob("../../web/templates/partial/*.html"))
	pr, err := NewPagesRenderer(shared, map[string]string{
		HomePage:    "../../web/templates/pages/index.html",
		PromptsPage: "../../web/templates/pages/prompts.html",
		DataPage:    "../../web/templates/pages/data.html",
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
	for _, want := range []string{
		`variant="brand">Contato`, `variant="success">Adicionar`,
		"Contato salvo", "ID 7", "Fulano Tal", "9292929290",
		`segment-text">Fulano`, `score-pill">0.91`,
		`segment-text">de`, `score-pill">0.08`,
		`segment-text">Tal`, `score-pill">0.88`,
		"segment-mark ok", "segment-mark no", "✓", "✗",
	} {
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
	for _, want := range []string{"Contato", "Adicionar", "Nenhum dado de contato"} {
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
	for _, want := range []string{"Notas", "70.00%", "Adicionar"} {
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
	for _, want := range []string{"Contato", "Consultar", "Contato encontrado", "Fulano Tal", "9292929290"} {
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
		Message: "09292929290 Fulano",
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	// DC-008: the duplicate screen carries the confirmation loop with the
	// original message re-posted through both dup_action forms.
	for _, want := range []string{
		"Contato já existe: Fulano", "9292929290",
		"dup_action", "value=\"update\"", "value=\"new\"",
		"value=\"09292929290 Fulano\"",
		"Atualizar existente", "Adicionar mesmo assim", "Cancelar",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderResultNoteDuplicate(t *testing.T) {
	c, w := newTestContext()

	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "notes", Confidence: 0.9},
			Kind:     models.KindFinding{Choice: "add", Confidence: 0.8},
		},
		Action: models.ActionNoteDuplicate,
		Notes: []*models.Note{
			{ID: 3, Type: models.NoteTypeReminder, Content: "pagar a conta", Date: &date},
		},
		Message: "me lembra de pagar a conta dia 10/05",
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"Nota já existe (lembrete)", "pagar a conta", "10/05/2026",
		"dup_action", "value=\"update\"", "value=\"new\"",
		"value=\"me lembra de pagar a conta dia 10/05\"",
		"Atualizar existente", "Adicionar mesmo assim", "Cancelar",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderResultTransactionDuplicate(t *testing.T) {
	c, w := newTestContext()

	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "finance", Confidence: 0.9},
			Kind:     models.KindFinding{Choice: "add", Confidence: 0.8},
		},
		Action: models.ActionTransactionDuplicate,
		Transactions: []*models.Transaction{
			{ID: 7, Type: models.TransactionTypePurchase, Amount: 20, Date: time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC), Party: "mercado", Content: "comprei 20 reais no mercado"},
		},
		Message: "comprei 20 reais no mercado",
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"Transação já existe", "R$ 20,00", "mercado",
		"dup_action", "value=\"update\"", "value=\"new\"",
		"comprei 20 reais no mercado",
		"Atualizar existente", "Adicionar mesmo assim", "Cancelar",
	} {
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
	RenderEvaluationResults(c, models.FlowClassification, results, "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"salva fulano", "contact:add", "finance:require", `variant="success">✓`, `variant="danger">✗`} {
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
	RenderEvaluationResults(c, models.FlowName, results, "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		`segment-text">João`, `score-pill">0.99`,
		`segment-text">da`, `score-pill">0.98`,
		`segment-text">Silva`, `score-pill">0.97`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderHomePage(t *testing.T) {
	c, w := newTestContext()

	RenderPage(c, HomePage, HomePageContent)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"Classificador de Mensagens",
		`hx-post="/api/message"`,
		// wa-input is a form-associated custom element, so it submits like a
		// native input — including Enter-to-submit.
		`<wa-input name="message"`,
		// The mocked user id is hidden, not a visible field.
		`<input type="hidden" name="user_id" value="12345">`,
		// wa-button defaults to type="button" (the opposite of a native
		// <button>), so the submit type has to be explicit.
		`<wa-button type="submit"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("home page missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "<textarea") {
		t.Errorf("home page still renders a textarea for the message: %s", body)
	}
	if strings.Contains(body, `id="user_id"`) {
		t.Errorf("home page still renders a visible user id field: %s", body)
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

func TestLabel(t *testing.T) {
	for input, want := range map[string]string{
		"contact": "Contato", "finance": "Finanças", "schedule": "Agenda",
		"notes": "Notas", "other": "Outro", "add": "Adicionar",
		"require": "Consultar", "both": "Ambos",
	} {
		if got := messages.Label(input); got != want {
			t.Errorf("messages.Label(%q) = %q, want %q", input, got, want)
		}
	}
	if got := messages.Label("unknown"); got != "unknown" {
		t.Errorf("messages.Label(%q) = %q, want fallback to input", "unknown", got)
	}
	if got := messages.Label(nil); got != "" {
		t.Errorf("messages.Label(nil) = %q, want empty string", got)
	}
}

func TestRenderResultNoteAddReminder(t *testing.T) {
	c, w := newTestContext()

	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	clock := "14:30"
	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "notes", Confidence: 0.9},
			Kind:     models.KindFinding{Choice: "add", Confidence: 0.8},
		},
		Action: models.ActionNoteAdd,
		Notes: []*models.Note{{
			ID:      3,
			Type:    models.NoteTypeReminder,
			Content: "pagar a conta de luz",
			Date:    &date,
			Time:    &clock,
		}},
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		`variant="brand">Notas`, "Lembrete salvo", "ID 3",
		`variant="warning">Lembrete`, "pagar a conta de luz", "10/05/2026", "14:30",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderResultNoteAddTodo(t *testing.T) {
	c, w := newTestContext()

	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "notes", Confidence: 0.88},
			Kind:     models.KindFinding{Choice: "add", Confidence: 0.91},
		},
		Action: models.ActionNoteAdd,
		Notes: []*models.Note{{
			ID:      4,
			Type:    models.NoteTypeTodo,
			Content: "comprar pão, leite e ovos",
			Items: []models.TodoItem{
				{ID: 1, NoteID: 4, Text: "comprar pão", Position: 0},
				{ID: 2, NoteID: 4, Text: "leite e ovos", Position: 1, Done: true},
			},
		}},
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"Lista de tarefas salva", "ID 4", `variant="success">Lista de tarefas`,
		"todo-list", "todo-item", "comprar pão", "leite e ovos", "todo-item done", "✓", "○",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderResultNoteFound(t *testing.T) {
	c, w := newTestContext()

	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "notes", Confidence: 0.9},
			Kind:     models.KindFinding{Choice: "require", Confidence: 0.9},
		},
		Action: models.ActionNoteFound,
		Notes: []*models.Note{
			{ID: 1, Type: models.NoteTypeReminder, Content: "pagar a conta", Date: &date},
			{ID: 2, Type: models.NoteTypeNote, Content: "renomear o projeto"},
		},
		SearchTerm: "10/05/2026",
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"2 nota(s) encontrada(s) para '10/05/2026'",
		"note-list", `variant="warning">Lembrete`, `variant="brand">Nota`,
		"pagar a conta", "renomear o projeto", "10/05/2026",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestRenderResultNoteNotFound(t *testing.T) {
	c, w := newTestContext()

	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "notes", Confidence: 0.9},
			Kind:     models.KindFinding{Choice: "require", Confidence: 0.9},
		},
		Action:     models.ActionNoteNotFound,
		SearchTerm: "projeto",
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "Nenhuma nota encontrada para 'projeto'") {
		t.Errorf("body missing not-found message with SearchTerm: %s", body)
	}
}

func TestRenderResultNoteNoData(t *testing.T) {
	c, w := newTestContext()

	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "notes", Confidence: 0.9},
			Kind:     models.KindFinding{Choice: "add", Confidence: 0.9},
		},
		Action: models.ActionNoteNoData,
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"Notas", "Não consegui extrair os dados da nota", "dia 10"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
}

func TestLabelNoteTypes(t *testing.T) {
	for input, want := range map[string]string{
		"note": "Nota", "reminder": "Lembrete", "todo": "Lista de tarefas",
	} {
		if got := messages.Label(input); got != want {
			t.Errorf("messages.Label(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRenderDataPage(t *testing.T) {
	c, w := newTestContext()
	RenderPage(c, DataPage, DataPageContent)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		">Dados<",
		`hx-get="/data/table?kind=contact&filter=all"`,
		`hx-get="/data/table?kind=contact&filter=name"`,
		`hx-get="/data/table?kind=notes&filter=todo"`,
		`hx-get="/data/table?kind=transaction&filter=all"`,
		`hx-get="/data/table?kind=transaction&filter=compra"`,
		`hx-get="/data/table?kind=transaction&filter=venda"`,
		`hx-get="/data/table?kind=transaction&filter=pagamento"`,
		`hx-get="/data/table?kind=transaction&filter=recebimento"`,
		`id="pane-finance"`,
		`id="txSearchForm"`,
		`id="txSearch"`,
		`id="contactSearchForm"`,
		`id="contactSearch"`,
		`id="noteSearchForm"`,
		`id="noteSearch"`,
		`id="dataPanel"`,
		`id="data-panel-body"`,
		`data-drawer="hide"`,
		`hx-trigger="dataChanged from:body"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("data page missing %q: %s", want, body)
		}
	}
	// Every search box posts the kind and the current filter, so a search never
	// escapes the selected type.
	for _, want := range []string{
		`<input type="hidden" name="kind" value="transaction">`,
		`<input type="hidden" name="kind" value="contact">`,
		`<input type="hidden" name="kind" value="notes">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("search box missing the kind hidden input %q: %s", want, body)
		}
	}
	// Each pane's search box carries a hidden filter that markFilterPill re-points
	// on a pill click, so a search and the filter on screen never disagree.
	if got := strings.Count(body, `class="search-filter"`); got != 3 {
		t.Errorf("search filter hidden inputs = %d, want 3 (one per pane)", got)
	}
	// Filter pill state is client-side: without these the "active" class stays
	// hardcoded on one pill and the highlight never follows the selection.
	// wa-tab-group replaces the old data-bs-toggle tabs, so the pane switch
	// rides the component's wa-tab-show event rather than a per-tab onclick.
	for _, want := range []string{
		`onclick="markFilterPill(this)"`,
		`@wa-tab-show="activateFilterPills('pane-' + $event.detail.name)"`,
		"function markFilterPill(pill)",
		"function activateFilterPills(paneId)",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("data page missing pill behaviour %q: %s", want, body)
		}
	}
	if got := strings.Count(body, "data-filter-pill"); got != 13 {
		t.Errorf("filter pills = %d, want 13 (4 contacts + 4 notes + 5 finance)", got)
	}
	if got := strings.Count(body, `variant="brand" data-filter-pill`); got != 3 {
		t.Errorf("pre-selected filter pills = %d, want 3 (one Todos per tab)", got)
	}
	// The contacts "Todos" pill boots the table on load but must stay clickable:
	// an explicit hx-trigger replaces htmx's default click trigger, so "load" alone
	// would make the pill a dead button (tabs would never re-load their sub-filter,
	// and clicking "Todos" would do nothing).
	if !strings.Contains(body, `hx-trigger="load, click"`) {
		t.Errorf("data page missing load+click pill trigger: %s", body)
	}
	if strings.Contains(body, `hx-trigger="load"`) {
		t.Errorf("data page has a bare load trigger that kills pill clicks: %s", body)
	}
	// activateFilterPills must load the pane's default filter unconditionally:
	// pane-contacts ships with class="... active" and wa-tab-group adds "active"
	// to whichever pane it shows, so a `contains('active') return` guard skips
	// the load exactly when you re-select the tab that is already showing.
	if strings.Contains(body, "classList.contains('active')) return") {
		t.Errorf("data page skips the default-filter load on an already-active pane: %s", body)
	}
	// "Validação Jev" is the shared sidebar nav link, so leak markers must come
	// from the other pages' own content.
	for _, leak := range []string{"Adicionar exemplo", "Classificador de Mensagens"} {
		if strings.Contains(body, leak) {
			t.Errorf("data page leaked %q: %s", leak, body)
		}
	}
}

func TestRenderContactsTable(t *testing.T) {
	c, w := newTestContext()
	contacts := []models.Contact{
		{ID: 1, Name: "Maria da Silva", NameNorm: "maria da silva", Phone: strPtr("9292929290")},
		{ID: 2, Name: "João sem contato"},
	}
	RenderContactsTable(c, "phone", "silva", contacts)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		`id="data-table-form"`, `name="kind" value="contact"`, `name="filter" value="phone"`,
		`name="search" value="silva"`,
		`class="data-check" name="ids" value="1"`, "Maria da Silva", "9292929290",
		"João sem contato", "Apagar selecionados", `hx-post="/data/delete"`,
		`hx-get="/data/contact/1"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("contacts table missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "0xc0") {
		t.Errorf("contacts table leaked a pointer address instead of dereferencing: %s", body)
	}
}

func TestRenderResultTransactionAdd(t *testing.T) {
	c, w := newTestContext()

	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "finance", Confidence: 0.94},
			Kind:     models.KindFinding{Choice: "add", Confidence: 0.86},
		},
		Action: models.ActionTransactionAdd,
		Transactions: []*models.Transaction{{
			ID:      7,
			Type:    models.TransactionTypePurchase,
			Amount:  1234.56,
			Date:    time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
			Party:   "supermercado",
			Content: "compras no supermercado",
		}},
	}

	RenderResult(c, outcome)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		`variant="success">Finanças`, "Transação salva", "ID 7", `variant="success">Compra`,
		"R$ 1.234,56", "10/05/2026", "supermercado", "compras no supermercado",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("transaction add missing %q: %s", want, body)
		}
	}
}

func TestRenderResultTransactionFoundWithTotal(t *testing.T) {
	c, w := newTestContext()

	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "finance", Confidence: 0.9},
			Kind:     models.KindFinding{Choice: "require", Confidence: 0.9},
		},
		Action:     models.ActionTransactionFound,
		SearchTerm: "supermercado",
		Total:      1300.5,
		Transactions: []*models.Transaction{
			{ID: 1, Type: models.TransactionTypePurchase, Amount: 50, Party: "supermercado", Date: time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)},
			{ID: 2, Type: models.TransactionTypePurchase, Amount: 1250.5, Party: "supermercado", Date: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)},
		},
	}

	RenderResult(c, outcome)

	body := w.Body.String()
	for _, want := range []string{
		"2 transações", "supermercado", "tx-total", "Total: R$ 1.300,50",
		"R$ 50,00", "R$ 1.250,50",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("transaction found missing %q: %s", want, body)
		}
	}
}

func TestRenderResultTransactionNoData(t *testing.T) {
	c, w := newTestContext()

	outcome := &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "finance", Confidence: 0.7},
			Kind:     models.KindFinding{Choice: "add", Confidence: 0.8},
		},
		Action:  models.ActionTransactionNoData,
		Missing: "o valor",
	}

	RenderResult(c, outcome)

	body := w.Body.String()
	for _, want := range []string{
		`variant="success">Finanças`, "Não consegui classificar", "o valor", "Acrescente à mensagem", "envie de novo",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("transaction no data missing %q: %s", want, body)
		}
	}
}

func TestRenderContactsTableEmpty(t *testing.T) {
	c, w := newTestContext()
	RenderContactsTable(c, "all", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "Nenhum contato cadastrado com este filtro.") {
		t.Errorf("contacts table missing empty state: %s", body)
	}
}

func TestRenderNotesTable(t *testing.T) {
	c, w := newTestContext()
	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	notes := []*models.Note{
		{ID: 1, Type: models.NoteTypeReminder, Content: "pagar a conta", Date: &date, Time: strPtr("14:30")},
		{ID: 2, Type: models.NoteTypeTodo, Content: "comprar", Items: []models.TodoItem{{Text: "pão"}}},
	}
	RenderNotesTable(c, "all", "pagar", notes)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		`name="filter" value="all"`, `name="search" value="pagar"`, "pagar a conta", "10/05/2026", "14:30",
		"comprar", `variant="warning">Lembrete`, `variant="success">Lista de tarefas`,
		`hx-get="/data/notes/1"`, "Apagar selecionados",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("notes table missing %q: %s", want, body)
		}
	}
}

func TestRenderNotesTableEmpty(t *testing.T) {
	c, w := newTestContext()
	RenderNotesTable(c, "todo", "", nil)
	if body := w.Body.String(); !strings.Contains(body, "Nenhuma nota cadastrada com este filtro.") {
		t.Errorf("notes table missing empty state: %s", body)
	}
}

func TestRenderDataDetailContactViewMode(t *testing.T) {
	c, w := newTestContext()
	RenderDataDetail(c, DataDetailData{
		Kind:    "contact",
		Mode:    "view",
		Contact: &models.Contact{ID: 7, Name: "Maria da Silva", Phone: strPtr("9292929290")},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"fieldset disabled", `name="name"`, `name="phone"`, `value="Maria da Silva"`,
		`hx-get="/data/contact/7?mode=edit"`, "Editar",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("contact detail missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "mode=view") {
		t.Errorf("view mode should not offer Cancelar: %s", body)
	}
}

func TestRenderDataDetailContactEditMode(t *testing.T) {
	c, w := newTestContext()
	RenderDataDetail(c, DataDetailData{
		Kind:    "contact",
		Mode:    "edit",
		Contact: &models.Contact{ID: 7, Name: "Maria da Silva"},
	})
	body := w.Body.String()
	if strings.Contains(body, "fieldset disabled") {
		t.Errorf("edit mode must enable the fieldset: %s", body)
	}
	for _, want := range []string{`hx-post="/data/contact/7"`, "Salvar", `hx-get="/data/contact/7?mode=view"`, "Cancelar"} {
		if !strings.Contains(body, want) {
			t.Errorf("contact edit detail missing %q: %s", want, body)
		}
	}
}

func TestRenderDataDetailNoteTodo(t *testing.T) {
	c, w := newTestContext()
	RenderDataDetail(c, DataDetailData{
		Kind: "notes",
		Mode: "edit",
		Note: &models.Note{
			ID:      3,
			Type:    models.NoteTypeTodo,
			Content: "comprar",
			Items: []models.TodoItem{
				{Text: "pão", Position: 0},
				{Text: "leite", Done: true, Position: 1},
			},
		},
	})
	body := w.Body.String()
	for _, want := range []string{
		`hx-post="/data/notes/3"`, `name="content"`, `id="todo-items"`,
		`value="pão"`, `value="leite"`, "todo-item-row",
		`hx-get="/data/item-row"`, `hx-swap="beforeend"`, "Adicionar item",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("todo detail missing %q: %s", want, body)
		}
	}
	if !strings.Contains(body, `name="item_done" value="1"`) {
		t.Errorf("done item should post item_done=1: %s", body)
	}
}

func TestRenderDataDetailNoteReminderHasDateAndTime(t *testing.T) {
	c, w := newTestContext()
	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	RenderDataDetail(c, DataDetailData{
		Kind: "notes",
		Mode: "view",
		Note: &models.Note{ID: 4, Type: models.NoteTypeReminder, Content: "pagar", Date: &date, Time: strPtr("09:00")},
	})
	body := w.Body.String()
	for _, want := range []string{`name="date"`, `value="10/05/2026"`, `name="time"`, `value="09:00"`} {
		if !strings.Contains(body, want) {
			t.Errorf("reminder detail missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "id=\"todo-items\"") {
		t.Errorf("a reminder should not render the to-do editor: %s", body)
	}
}

func TestRenderDataDetailTransactionViewMode(t *testing.T) {
	c, w := newTestContext()
	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	RenderDataDetail(c, DataDetailData{
		Kind: "transaction",
		Mode: "view",
		Transaction: &models.Transaction{
			ID: 7, Type: models.TransactionTypePayment, Amount: 1234.56,
			Date: date, Party: "Farmácia", Content: "Paguei a fatura do cartão",
		},
	})
	body := w.Body.String()
	// Every option must be rendered and the stored one selected: an empty select
	// is what the panel looked like when the transaction branch was truncated.
	for _, want := range []string{
		`name="type"`, `value="compra"`, `value="venda"`, `value="pagamento"`,
		`value="recebimento"`, `value="transferencia"`, `value="pagamento" selected`,
		`name="amount"`, `value="R$ 1.234,56"`, `name="date"`, `value="10/05/2026"`,
		`name="party"`, `value="Farmácia"`, `name="content"`, "Paguei a fatura do cartão",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("transaction detail missing %q: %s", want, body)
		}
	}
	if !strings.Contains(body, `<fieldset disabled class="wa-stack`) {
		t.Errorf("view mode must disable the fieldset: %s", body)
	}
}

func TestRenderDataDetailTransactionEditModeEnablesFieldset(t *testing.T) {
	c, w := newTestContext()
	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	RenderDataDetail(c, DataDetailData{
		Kind:        "transaction",
		Mode:        "edit",
		Transaction: &models.Transaction{ID: 7, Type: models.TransactionTypePurchase, Amount: 50, Date: date, Party: "Padaria"},
	})
	body := w.Body.String()
	if strings.Contains(body, "<fieldset disabled>") {
		t.Errorf("edit mode must enable the fieldset: %s", body)
	}
	if !strings.Contains(body, `value="compra" selected`) {
		t.Errorf("the stored type must be the selected option: %s", body)
	}
	// The amount must round-trip through ParseAmount, so it is prefilled in the
	// shape the parser reads ("50.00" would parse back as 5000).
	if !strings.Contains(body, `value="R$ 50,00"`) {
		t.Errorf("amount must be prefilled with money: %s", body)
	}
}

func TestRenderDataItemRow(t *testing.T) {
	c, w := newTestContext()
	RenderDataItemRow(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		`name="item_text"`, `value=""`, `name="item_done" value="0"`,
		"todo-item-row", `name="trash-can"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("item row missing %q: %s", want, body)
		}
	}
	// The checkbox carries no name: an unchecked wa-checkbox sends nothing, and
	// buildItems indexes item_done positionally against item_text, so a missing
	// value would shift the array and mark the wrong rows done.
	if strings.Contains(body, `<wa-checkbox name=`) {
		t.Errorf("wa-checkbox must not be named (it would send nothing when unchecked): %s", body)
	}
}

func TestDeref(t *testing.T) {
	if got := Deref(strPtr("abc")); got != "abc" {
		t.Errorf("Deref = %q, want abc", got)
	}
	if got := Deref(nil); got != "" {
		t.Errorf("Deref(nil) = %q, want empty", got)
	}
}

// Shoelace 2.x token names. Web Awesome 3.x renamed every one of them, and an
// undefined var() invalidates the whole declaration silently: no error, no log,
// the property just computes to unset. This is the failure that left the brand
// mark invisible and the active nav link unstyled. See IMP-006.md.
var shoelaceOnlyTokens = []string{
	"--wa-radius-",
	"--wa-font-family-mono",
	"--wa-font-weight-semi)",
	"--wa-color-text-muted",
	"--wa-color-surface-hover",
	"--wa-color-success-6",
	"--wa-color-brand-100",
	"--wa-color-brand-500",
	"--wa-color-brand-700",
	"--wa-color-brand-800",
	"--wa-color-cyan-500",
	"--wa-color-cyan-600",
}

func TestAppCSSUsesNoShoelaceTokens(t *testing.T) {
	css, err := os.ReadFile("../../web/static/css/app.css")
	if err != nil {
		t.Fatalf("failed to read app.css: %v", err)
	}
	for _, token := range shoelaceOnlyTokens {
		if strings.Contains(string(css), token) {
			t.Errorf("app.css references %q, a Shoelace 2.x name Web Awesome 3.x does not ship; the declaration silently does nothing", token)
		}
	}
}
