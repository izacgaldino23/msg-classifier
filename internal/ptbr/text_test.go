package ptbr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"case", "FULANO", "fulano"},
		{"accent", "João", "joao"},
		{"cedilla", "José da Conceição", "jose da conceicao"},
		{"mixed accents", "MARIA CLÁUDIA", "maria claudia"},
		{"collapse whitespace", "  Fulano   de  Tal ", "fulano de tal"},
		{"tabs and newlines", "Fulano\tde\nTal", "fulano de tal"},
		{"already normalized", "fulano de tal", "fulano de tal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeName(tt.in))
		})
	}
}

func TestStripPunctuation(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"trailing question mark", "anotações?", "anotações"},
		{"drops inner spaces", "10 dias", "10dias"},
		{"drops punctuation only", "!!!", ""},
		{"keeps digits, drops currency", "r$50", "r50"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, StripPunctuation(tt.in))
		})
	}
}

func TestTrimSegment(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"leading and trailing quotes", `"João",`, "João"},
		{"keeps the period of a Jr.", "Jr.", "Jr."},
		{"inner punctuation untouched", "d,o", "d,o"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, TrimSegment(tt.in))
		})
	}
}

func TestIsCapitalized(t *testing.T) {
	assert.True(t, IsCapitalized("João"))
	assert.False(t, IsCapitalized("joão"))
	assert.False(t, IsCapitalized(""))
	assert.False(t, IsCapitalized("1 dez"))
}