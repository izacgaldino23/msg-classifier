package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitTodoItems(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "commas",
			content: "comprar pão, leite e ovos",
			want:    []string{"comprar pão", "leite e ovos"},
		},
		{
			name:    "semicolons",
			content: "pagar a conta; comprar café",
			want:    []string{"pagar a conta", "comprar café"},
		},
		{
			name:    "newlines win",
			content: "comprar pão\ncomprar leite\ncomprar ovos",
			want:    []string{"comprar pão", "comprar leite", "comprar ovos"},
		},
		{
			name:    "windows newlines",
			content: "primeiro\r\nsegundo",
			want:    []string{"primeiro", "segundo"},
		},
		{
			name:    "numbered on one line",
			content: "1. revisar contrato 2. enviar relatório",
			want:    []string{"revisar contrato", "enviar relatório"},
		},
		{
			name:    "numbered with parens",
			content: "1) revisar contrato\n2) enviar relatório",
			want:    []string{"revisar contrato", "enviar relatório"},
		},
		{
			name:    "dashes and trailing punctuation",
			content: "- comprar pão; - comprar leite.",
			want:    []string{"comprar pão", "comprar leite"},
		},
		{
			name:    "leading conjunction dropped",
			content: "comprar pão, e comprar leite, e comprar ovos",
			want:    []string{"comprar pão", "comprar leite", "comprar ovos"},
		},
		{
			name:    "empty items dropped",
			content: "comprar pão, , ; comprar leite",
			want:    []string{"comprar pão", "comprar leite"},
		},
		{
			name:    "single item keeps order",
			content: "  comprar pão  ",
			want:    []string{"comprar pão"},
		},
		{name: "empty", content: "", want: []string{}},
		{name: "blank", content: "   \n  ", want: []string{}},
		{name: "only punctuation", content: ".,;", want: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SplitTodoItems(tt.content))
		})
	}
}

func TestSplitTodoItemsNeverReturnsNil(t *testing.T) {
	assert.NotNil(t, SplitTodoItems(""))
}