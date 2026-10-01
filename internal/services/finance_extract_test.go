package services

import (
	"errors"
	"testing"

	"msg-classifier/internal/models"
	"msg-classifier/pkg/jev"
)

// financeResponse serves the type choice plus one Noul verdict per segment.
func financeResponse(choice string, nouls ...float64) *jev.JevResponse {
	answers := map[string]any{
		typeAnswerKey: &jev.JevAnswerChoice{Choice: choice, Confidence: 0.9},
	}
	for i, noul := range nouls {
		answers[segmentKey(i)] = &jev.JevAnswerNoul{Noul: noul}
	}
	return &jev.JevResponse{Model: "jev-latest", Answers: answers}
}

func TestFinanceExtractorExtract(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		resp      *jev.JevResponse
		wantType  string
		wantParty string
	}{
		{
			name:      "shop from the last preposition",
			message:   "Comprei pão, leite e ovos no supermercado. O total foi de 50 reais.",
			resp:      financeResponse(models.TransactionTypePurchase, 0.95),
			wantType:  models.TransactionTypePurchase,
			wantParty: "supermercado",
		},
		{
			name:      "person as receiver",
			message:   "Recebi pix de 100 reais hoje de Fulano",
			resp:      financeResponse(models.TransactionTypeReceipt, 0.9),
			wantType:  models.TransactionTypeReceipt,
			wantParty: "Fulano",
		},
		{
			name:      "payment method kept when it is the last candidate",
			message:   "Cabo de celular 20 reais hoje no cartão",
			resp:      financeResponse(models.TransactionTypePurchase, 0.7),
			wantType:  models.TransactionTypePurchase,
			wantParty: "cartão",
		},
		{
			name:      "amount candidate is skipped, the shop wins",
			message:   "Paguei o aluguel de 1200 reais no mercado",
			resp:      financeResponse(models.TransactionTypePayment, 0.9),
			wantType:  models.TransactionTypePayment,
			wantParty: "mercado",
		},
		{
			name:      "no candidate means no party",
			message:   "Paguei 50 reais",
			resp:      financeResponse(models.TransactionTypePayment),
			wantType:  models.TransactionTypePayment,
			wantParty: "",
		},
		{
			name:      "junk candidate is dropped",
			message:   "Paguei o valor de 50 reais",
			resp:      financeResponse(models.TransactionTypePayment),
			wantType:  models.TransactionTypePayment,
			wantParty: "",
		},
		{
			name:      "unknown type falls back to purchase",
			message:   "gastei 20 reais no mercado",
			resp:      financeResponse("investimento", 0.9),
			wantType:  models.TransactionTypePurchase,
			wantParty: "mercado",
		},
		{
			name:      "raw candidate when jev keeps nothing",
			message:   "comprei na padaria do bairro por 30 reais",
			resp:      financeResponse(models.TransactionTypePurchase, 0.1, 0.1, 0.1, 0.1),
			wantType:  models.TransactionTypePurchase,
			wantParty: "padaria do bairro",
		},
		{
			name:      "multi word party keeps only the kept segments",
			message:   "almoço no restaurante do tio por 80 reais",
			resp:      financeResponse(models.TransactionTypePurchase, 0.9, 0.1, 0.9),
			wantType:  models.TransactionTypePurchase,
			wantParty: "restaurante tio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockJevRequester{resp: tt.resp}
			result, err := NewFinanceExtractor(mock).Extract(tt.message)
			if err != nil {
				t.Fatalf("Extract(%q) error = %v", tt.message, err)
			}
			if result.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", result.Type, tt.wantType)
			}
			if result.Party != tt.wantParty {
				t.Errorf("Party = %q, want %q", result.Party, tt.wantParty)
			}
		})
	}
}

// The type question and the party fan-out must travel in one request, so a finance
// message costs a single extra Jev call like the notes and contacts flows.
func TestFinanceExtractorAsksTypeAndPartyInOneCall(t *testing.T) {
	mock := &mockJevRequester{resp: financeResponse(models.TransactionTypePurchase, 0.9)}

	if _, err := NewFinanceExtractor(mock).Extract("comprei no supermercado por 50 reais"); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if mock.calls != 1 {
		t.Fatalf("Jev calls = %d, want 1", mock.calls)
	}
	if _, ok := mock.got.Questions[typeAnswerKey].(*jev.JevQuestionChoice); !ok {
		t.Errorf("missing the %q choice question, got %T", typeAnswerKey, mock.got.Questions[typeAnswerKey])
	}
	if _, ok := mock.got.Questions[segmentKey(0)].(*jev.JevQuestionNoul); !ok {
		t.Errorf("missing the segment noul question, got %T", mock.got.Questions[segmentKey(0)])
	}
}

// A rejected request must not cost the transaction: the type is mandatory, the
// party is not, so the extractor retries without the fan-out.
func TestFinanceExtractorFallsBackToTypeOnly(t *testing.T) {
	mock := &mockJevRequester{failFirst: true, resp: financeResponse(models.TransactionTypePayment)}

	result, err := NewFinanceExtractor(mock).Extract("paguei 1200 reais do aluguel")
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if result.Type != models.TransactionTypePayment {
		t.Errorf("Type = %q, want %q", result.Type, models.TransactionTypePayment)
	}
	if result.Party != "" {
		t.Errorf("Party = %q, want empty after the degraded retry", result.Party)
	}
	if mock.calls != 2 {
		t.Fatalf("Jev calls = %d, want 2", mock.calls)
	}
	if _, ok := mock.got.Questions[segmentKey(0)]; ok {
		t.Error("the retry must not carry the party fan-out")
	}
}

func TestFinanceExtractorUpstreamErrors(t *testing.T) {
	mock := &mockJevRequester{err: errors.New("boom")}

	_, err := NewFinanceExtractor(mock).Extract("comprei por 50 reais no mercado")
	if !errors.Is(err, ErrUpstream) {
		t.Fatalf("error = %v, want ErrUpstream", err)
	}
}

// A response without the type answer is an upstream failure, not a default.
func TestFinanceExtractorMissingTypeAnswer(t *testing.T) {
	mock := &mockJevRequester{resp: &jev.JevResponse{Answers: map[string]any{}}}

	_, err := NewFinanceExtractor(mock).Extract("comprei por 50 reais no mercado")
	if !errors.Is(err, ErrUpstream) {
		t.Fatalf("error = %v, want ErrUpstream", err)
	}
}

func TestPartyCandidateFrom(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		wantParty string
	}{
		{"shop", "comprei no supermercado", "supermercado"},
		{"last one wins", "fui na padaria e depois no mercado", "mercado"},
		{"amount is not a party", "gastei 50 reais", ""},
		{"no preposition", "padaria, 30 reais", ""},
		{"junk only", "paguei o total de 100 reais", ""},
		{"trailing punctuation", "almoço no restaurante.", "restaurante"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := partyCandidateFrom(tt.message)
			if got.Raw != tt.wantParty {
				t.Errorf("Raw = %q, want %q", got.Raw, tt.wantParty)
			}
		})
	}
}

func TestIsTransactionType(t *testing.T) {
	for _, valid := range []string{"compra", "venda", "transferencia", "recebimento", "pagamento"} {
		if !models.IsTransactionType(valid) {
			t.Errorf("IsTransactionType(%q) = false, want true", valid)
		}
	}
	for _, invalid := range []string{"", "Compra", "investimento", "outro"} {
		if models.IsTransactionType(invalid) {
			t.Errorf("IsTransactionType(%q) = true, want false", invalid)
		}
	}
}
