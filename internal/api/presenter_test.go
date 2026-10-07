package api

import (
	"encoding/json"
	"testing"
	"time"

	"msg-classifier/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }

func classification() *models.Classification {
	return &models.Classification{
		Category: models.CategoryFinding{Choice: "contact", Confidence: 0.9},
		Kind:     models.KindFinding{Choice: "add", Confidence: 0.8},
	}
}

// TestSummaryPerAction asserts the printed content of every Action branch. A
// branch that returns a wrong sentence is as broken as one that does not compile,
// and a status-only test would not notice: the gap that let sixteen dead Shoelace
// tokens through IMP-006 while the whole suite stayed green.
func TestSummaryPerAction(t *testing.T) {
	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	clock := "14:30"

	tests := []struct {
		action  models.Action
		outcome models.UseCaseOutcome
		want    string
	}{
		{models.ActionNone, models.UseCaseOutcome{}, ""},
		{
			models.ActionContactAdd,
			models.UseCaseOutcome{Contact: &models.Contact{ID: 7, Name: "Fulano Tal"}},
			"Contato salvo · ID 7",
		},
		{
			models.ActionContactAdd,
			models.UseCaseOutcome{},
			"Contato salvo · ID 0",
		},
		{
			models.ActionContactFound,
			models.UseCaseOutcome{Contact: &models.Contact{Name: "Fulano Tal"}},
			"Contato encontrado",
		},
		{
			models.ActionContactNotFound,
			models.UseCaseOutcome{SearchTerm: "fulano tal"},
			"Nenhum contato encontrado para 'fulano tal'",
		},
		{
			models.ActionContactDuplicate,
			models.UseCaseOutcome{Contact: &models.Contact{Name: "Fulano"}},
			"Contato já existe: Fulano",
		},
		{
			models.ActionContactNoData,
			models.UseCaseOutcome{},
			"Nenhum dado de contato encontrado na mensagem.",
		},
		{
			models.ActionNoteAdd,
			models.UseCaseOutcome{Notes: []*models.Note{{ID: 3, Type: models.NoteTypeReminder, Date: &date, Time: &clock}}},
			"Lembrete salvo · ID 3",
		},
		{
			models.ActionNoteAdd,
			models.UseCaseOutcome{Notes: []*models.Note{{ID: 4, Type: models.NoteTypeTodo}}},
			"Lista de tarefas salva · ID 4",
		},
		{
			models.ActionNoteAdd,
			models.UseCaseOutcome{Notes: []*models.Note{{ID: 5, Type: models.NoteTypeNote}}},
			"Nota salva · ID 5",
		},
		{
			models.ActionNoteAdd,
			models.UseCaseOutcome{},
			"Nota salva",
		},
		{
			models.ActionNoteFound,
			models.UseCaseOutcome{Notes: []*models.Note{{ID: 1}, {ID: 2}}, SearchTerm: "10/05/2026"},
			"2 nota(s) encontrada(s) para '10/05/2026'",
		},
		{
			models.ActionNoteNotFound,
			models.UseCaseOutcome{SearchTerm: "projeto"},
			"Nenhuma nota encontrada para 'projeto'.",
		},
		{
			models.ActionNoteNoData,
			models.UseCaseOutcome{},
			"Não consegui extrair os dados da nota.",
		},
		{
			models.ActionNoteDuplicate,
			models.UseCaseOutcome{Notes: []*models.Note{{ID: 9, Type: models.NoteTypeNote, Content: "pagar a conta"}}},
			"Nota já existe: pagar a conta",
		},
		{
			models.ActionNoteDuplicate,
			models.UseCaseOutcome{},
			"Nota já existe",
		},
		{
			models.ActionTransactionAdd,
			models.UseCaseOutcome{Transactions: []*models.Transaction{{ID: 7, Type: models.TransactionTypePurchase}}},
			"Transação salva · ID 7",
		},
		{
			models.ActionTransactionAdd,
			models.UseCaseOutcome{},
			"Transação salva",
		},
		{
			models.ActionTransactionFound,
			models.UseCaseOutcome{Transactions: []*models.Transaction{{ID: 1}, {ID: 2}}, SearchTerm: "supermercado"},
			"2 transação(ões) para 'supermercado'",
		},
		{
			models.ActionTransactionNotFound,
			models.UseCaseOutcome{SearchTerm: "remedio"},
			"Nenhuma transação encontrada para 'remedio'.",
		},
		{
			models.ActionTransactionNoData,
			models.UseCaseOutcome{Missing: "o valor"},
			"Não consegui classificar a transação: o valor.",
		},
		{
			models.ActionTransactionDuplicate,
			models.UseCaseOutcome{Transactions: []*models.Transaction{{ID: 7}}},
			"Transação já existe",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			outcome := tt.outcome
			outcome.Action = tt.action
			assert.Equal(t, tt.want, summary(&outcome))
			assert.Equal(t, string(tt.action), Outcome(&outcome).Action, "action must be echoed verbatim")
		})
	}
}

// TestSummaryNeverPanics — a malformed outcome still has to answer. The contact
// guard exists for exactly this, so keep it under test.
func TestSummaryNeverPanics(t *testing.T) {
	for _, action := range []models.Action{
		models.ActionContactAdd, models.ActionContactFound, models.ActionContactDuplicate,
		models.ActionNoteAdd, models.ActionNoteDuplicate,
		models.ActionTransactionAdd, models.ActionTransactionFound, models.ActionTransactionDuplicate,
	} {
		outcome := &models.UseCaseOutcome{Action: action, Classification: classification()}
		assert.NotPanics(t, func() { Outcome(outcome) }, "action %q with a nil payload", action)
	}
}

func TestOutcomeCarriesPayloadAndOmitsTheRest(t *testing.T) {
	outcome := &models.UseCaseOutcome{
		Classification: classification(),
		Action:         models.ActionTransactionFound,
		SearchTerm:     "supermercado",
		Total:          1300.5,
		Transactions: []*models.Transaction{{
			ID: 7, Type: models.TransactionTypePurchase, Amount: 1234.56,
			Date: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), Party: "supermercado",
		}},
		Segments: []models.SegmentScore{{Text: "supermercado", Score: 0.91, Included: true}},
	}

	payload, err := json.Marshal(Outcome(outcome))
	require.NoError(t, err, "json.Marshal()")
	body := string(payload)

	for _, want := range []string{
		`"action":"transaction_found"`,
		`"search_term":"supermercado"`,
		`"total":1300.5`,
		`"amount":1234.56`,
		`"segments":[{"text":"supermercado","score":0.91,"included":true}]`,
	} {
		assert.Contains(t, body, want)
	}
	// A contact outcome carries no notes, so the key must be absent — not null.
	// The needle is the key plus its colon: a bare `"contact"` would also match
	// the classification's category choice value.
	assert.NotContains(t, body, `"notes":`)
	assert.NotContains(t, body, `"contact":`)
}
