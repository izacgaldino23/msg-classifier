package services

import "testing"

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
			if got := normalizeName(tt.in); got != tt.want {
				t.Errorf("normalizeName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
