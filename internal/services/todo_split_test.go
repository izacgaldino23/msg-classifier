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
			want:    []string{"comprar pão", "leite", "ovos"},
		},
		{
			name:    "conjunction chain with a comma",
			content: "Comprar: arroz, feijão, alho e 2 cenouras",
			want:    []string{"arroz", "feijão", "alho", "2 cenouras"},
		},
		{
			name:    "label alone is enough to split on the conjunction",
			content: "Lista do mercado: óleo, açúcar e café",
			want:    []string{"óleo", "açúcar", "café"},
		},
		{
			name:    "title case conjunction",
			content: "Mercado: Arroz, Feijão e Ovos",
			want:    []string{"Arroz", "Feijão", "Ovos"},
		},
		{
			name:    "label dropped when a list follows",
			content: "Preciso fazer: 1. revisar contrato 2. enviar relatório",
			want:    []string{"revisar contrato", "enviar relatório"},
		},
		{
			name:    "label dropped before a multiline list",
			content: "Tarefas:\n- comprar pão\n- lavar o carro",
			want:    []string{"comprar pão", "lavar o carro"},
		},
		{
			name:    "label kept when no list follows",
			content: "Revisar: contrato com o João às 10:00",
			want:    []string{"Revisar: contrato com o João às 10:00"},
		},
		{
			name:    "conjunction kept in a single statement",
			content: "comprar pão e leite",
			want:    []string{"comprar pão e leite"},
		},
		{
			name:    "conjunction kept in newline lists",
			content: "- comprar pão e leite\n- lavar o carro",
			want:    []string{"comprar pão e leite", "lavar o carro"},
		},
		{
			name:    "conjunction kept in numbered lists",
			content: "1. ligar para o cliente e confirmar\n2. enviar o relatório",
			want:    []string{"ligar para o cliente e confirmar", "enviar o relatório"},
		},
		{
			name:    "label alone yields no items",
			content: "Tarefas:",
			want:    []string{},
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