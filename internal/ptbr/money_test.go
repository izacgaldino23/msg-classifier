package ptbr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    float64
		wantOK  bool
	}{
		{"currency symbol with space", "paguei R$ 50,00 no cartão", 50, true},
		{"currency symbol no space", "R$50,00", 50, true},
		{"reais suffix", "o total foi de 50 reais", 50, true},
		{"real singular", "custou 20 real", 20, true},
		{"decimal only", "gastei 50,00 ontem", 50, true},
		{"thousands and decimals", "R$ 1.234,56 na farmácia", 1234.56, true},
		{"thousands without decimals", "deu 1.234 reais", 1234, true},
		{"installments skip the count", "3 vezes de 300 reais no cartão", 300, true},
		{"amount after the shop", "comprei no supermercado. o total foi de 50 reais", 50, true},
		{"uppercase currency", "PAGUEI r$ 99,90", 99.9, true},
		{"date is not money", "10/10/2023", 0, false},
		{"dash date is not money", "10-10-2023", 0, false},
		{"no amount", "comprei pão e leite", 0, false},
		{"empty", "", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseAmount(tt.message)
			assert.Equal(t, tt.wantOK, ok)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestParseAmountIsAlwaysPositive(t *testing.T) {
	// The transaction type carries the direction, so a receipt is not a negative
	// amount.
	got, ok := ParseAmount("recebi 100 reais hoje de Fulano")
	assert.True(t, ok)
	assert.Equal(t, 100.0, got)
}
