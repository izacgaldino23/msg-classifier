package ptbr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDateBR(t *testing.T) {
	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, "10/05/2026", DateBR(&date))
	assert.Equal(t, "", DateBR(nil))
}

// Whatever MoneyBRL prints must parse back through ParseAmount -- that round trip
// is why the transaction edit field is prefilled with it instead of "%.2f".
func TestMoneyBRL(t *testing.T) {
	tests := []struct {
		amount float64
		want   string
	}{
		{0, "R$ 0,00"},
		{50, "R$ 50,00"},
		{50.5, "R$ 50,50"},
		{1234.56, "R$ 1.234,56"},
		{1000, "R$ 1.000,00"},
		{1000000.99, "R$ 1.000.000,99"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, MoneyBRL(tt.amount), "MoneyBRL(%v)", tt.amount)
	}
}

func TestMoneyBRLRoundTripsThroughParseAmount(t *testing.T) {
	for _, amount := range []float64{50, 50.5, 1234.56, 1000, 1000000.99} {
		parsed, ok := ParseAmount(MoneyBRL(amount))
		assert.True(t, ok, "ParseAmount(%q) must accept the rendered form", MoneyBRL(amount))
		assert.InDelta(t, amount, parsed, 0.01, "round trip for %v", amount)
	}
}

func TestDateBRRoundTripsThroughParseDate(t *testing.T) {
	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	parsed, ok := ParseDate(DateBR(&date), date)
	assert.True(t, ok, "ParseDate must accept the rendered form")
	assert.Equal(t, date, parsed)
}