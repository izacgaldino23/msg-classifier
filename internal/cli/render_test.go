package cli

import (
	"testing"
	"time"

	"msg-classifier/internal/models"

	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string { return &s }

func classification() *models.Classification {
	return &models.Classification{
		Category: models.CategoryFinding{Choice: "contact", Confidence: 0.94},
		Kind:     models.KindFinding{Choice: "add", Confidence: 0.86},
	}
}

// TestRenderPerAction asserts the printed CONTENT of every branch, not that a
// status came back 200. A branch that prints the wrong sentence is exactly as
// broken as one that does not compile.
func TestRenderPerAction(t *testing.T) {
	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	clock := "14:30"

	tests := []struct {
		name    string
		outcome models.UseCaseOutcome
		want    []string
	}{
		{
			"contact add",
			models.UseCaseOutcome{
				Classification: classification(), Action: models.ActionContactAdd,
				Contact:  &models.Contact{ID: 7, Name: "Fulano Tal", Phone: strPtr("9292929290")},
				Segments: []models.SegmentScore{{Text: "Fulano", Score: 0.91, Included: true}, {Text: "de", Score: 0.08, Included: false}},
			},
			[]string{"Contato salvo · ID 7", "nome: Fulano Tal", "telefone: 9292929290", "extração:", "Fulano 0.91 ✓", "de 0.08 ✗"},
		},
		{
			"contact found",
			models.UseCaseOutcome{
				Classification: classification(), Action: models.ActionContactFound,
				Contact:    &models.Contact{Name: "Fulano Tal", Email: strPtr("f@t.io")},
				SearchTerm: "9292929290",
			},
			[]string{"Contato encontrado", "nome: Fulano Tal", "email: f@t.io"},
		},
		{
			"contact not found",
			models.UseCaseOutcome{Classification: classification(), Action: models.ActionContactNotFound, SearchTerm: "fulano tal"},
			[]string{"Nenhum contato encontrado para 'fulano tal'"},
		},
		{
			"contact duplicate",
			models.UseCaseOutcome{
				Classification: classification(), Action: models.ActionContactDuplicate,
				Contact: &models.Contact{Name: "Fulano", Phone: strPtr("9292929290")},
			},
			[]string{"Contato já existe: Fulano", "telefone: 9292929290"},
		},
		{
			"contact no data",
			models.UseCaseOutcome{Classification: classification(), Action: models.ActionContactNoData},
			[]string{"Nenhum dado de contato encontrado na mensagem."},
		},
		{
			"note add reminder",
			models.UseCaseOutcome{
				Classification: classification(), Action: models.ActionNoteAdd,
				Notes: []*models.Note{{ID: 3, Type: models.NoteTypeReminder, Content: "pagar a conta de luz", Date: &date, Time: &clock}},
			},
			[]string{"Lembrete salvo · ID 3", "conteúdo: pagar a conta de luz", "data: 10/05/2026", "horário: 14:30"},
		},
		{
			"note add todo",
			models.UseCaseOutcome{
				Classification: classification(), Action: models.ActionNoteAdd,
				Notes: []*models.Note{{ID: 4, Type: models.NoteTypeTodo, Content: "comprar", Items: []models.TodoItem{
					{Text: "pão"}, {Text: "leite", Done: true},
				}}},
			},
			[]string{"Lista de tarefas salva · ID 4", "○ pão", "✓ leite"},
		},
		{
			"note found",
			models.UseCaseOutcome{
				Classification: classification(), Action: models.ActionNoteFound, SearchTerm: "10/05/2026",
				Notes: []*models.Note{
					{ID: 1, Type: models.NoteTypeReminder, Content: "pagar a conta", Date: &date},
					{ID: 2, Type: models.NoteTypeNote, Content: "renomear o projeto"},
				},
			},
			[]string{"2 nota(s) encontrada(s) para '10/05/2026'", "[reminder] pagar a conta", "[note] renomear o projeto"},
		},
		{
			"note not found",
			models.UseCaseOutcome{Classification: classification(), Action: models.ActionNoteNotFound, SearchTerm: "projeto"},
			[]string{"Nenhuma nota encontrada para 'projeto'."},
		},
		{
			"note no data",
			models.UseCaseOutcome{Classification: classification(), Action: models.ActionNoteNoData},
			[]string{"Não consegui extrair os dados da nota. Para lembretes, informe a data."},
		},
		{
			"transaction add",
			models.UseCaseOutcome{
				Classification: classification(), Action: models.ActionTransactionAdd,
				Transactions: []*models.Transaction{{
					ID: 7, Type: models.TransactionTypePurchase, Amount: 1234.56,
					Date: date, Party: "supermercado", Content: "compras no supermercado",
				}},
			},
			[]string{"Transação salva · ID 7", "tipo: compra", "valor: R$ 1.234,56", "data: 10/05/2026", "estabelecimento: supermercado", "mensagem: compras no supermercado"},
		},
		{
			"transaction found with total",
			models.UseCaseOutcome{
				Classification: classification(), Action: models.ActionTransactionFound,
				SearchTerm: "supermercado", Total: 1300.5,
				Transactions: []*models.Transaction{
					{ID: 1, Type: models.TransactionTypePurchase, Amount: 50, Party: "supermercado", Date: date},
					{ID: 2, Type: models.TransactionTypePurchase, Amount: 1250.5, Party: "supermercado", Date: date},
				},
			},
			[]string{"2 transação(ões) para 'supermercado'", "total: R$ 1.300,50", "R$ 50,00 — supermercado", "R$ 1.250,50 — supermercado"},
		},
		{
			"transaction not found",
			models.UseCaseOutcome{Classification: classification(), Action: models.ActionTransactionNotFound, SearchTerm: "remedio"},
			[]string{"Nenhuma transação encontrada para 'remedio'."},
		},
		{
			"transaction no data",
			models.UseCaseOutcome{Classification: classification(), Action: models.ActionTransactionNoData, Missing: "o valor"},
			[]string{"Não consegui classificar a transação: o valor."},
		},
		{
			"none",
			models.UseCaseOutcome{Classification: classification(), Action: models.ActionNone},
			[]string{"[contact/add]", "categoria 94% · intenção 86%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := Render(&tt.outcome)
			for _, want := range tt.want {
				assert.Contains(t, out, want)
			}
		})
	}
}

// A nil payload must still print, not panic: the CLI runs in a request path too.
func TestRenderNeverPanics(t *testing.T) {
	for _, action := range []models.Action{
		models.ActionNone, models.ActionContactAdd, models.ActionContactFound,
		models.ActionContactDuplicate, models.ActionNoteAdd, models.ActionTransactionAdd,
		models.ActionTransactionFound,
	} {
		outcome := &models.UseCaseOutcome{Action: action, Classification: classification()}
		assert.NotPanics(t, func() { Render(outcome) }, "action %q with a nil payload", action)
	}
}

// The CLI must not render a Go pointer address into the terminal.
func TestRenderNeverLeaksPointers(t *testing.T) {
	out := Render(&models.UseCaseOutcome{
		Classification: classification(),
		Action:         models.ActionContactFound,
		Contact:        &models.Contact{Name: "Sem telefone"},
	})
	assert.NotContains(t, out, "0xc0", "a pointer address leaked into the output")
	assert.NotContains(t, out, "telefone:")
}