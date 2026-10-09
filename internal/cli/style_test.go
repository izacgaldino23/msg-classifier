package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A test writer is not a terminal, so nothing may carry escapes into it.
func TestStyleOffOnNonTerminal(t *testing.T) {
	plain := newStyle(&bytes.Buffer{})
	assert.False(t, plain.active(), "a buffer is not a terminal")
	assert.Equal(t, "x", plain.accent("x"))
	assert.Equal(t, "x", plain.dim("x"))
}

func TestStyleOffWhenNoColorIsSet(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	assert.False(t, newStyle(os.Stdout).active(), "NO_COLOR wins even on a terminal")
}

// The painting itself is testable without a terminal: force the style on.
func TestStylePaintsEachToken(t *testing.T) {
	on := style{on: true}
	assert.Equal(t, ansiAccent+"x"+ansiReset, on.accent("x"))
	assert.Equal(t, ansiDim+"x"+ansiReset, on.dim("x"))
	assert.Equal(t, ansiGreen+"x"+ansiReset, on.green("x"))
	assert.Equal(t, ansiRed+"x"+ansiReset, on.red("x"))
	assert.Equal(t, ansiYellow+"x"+ansiReset, on.yellow("x"))
	assert.Equal(t, "", on.accent(""), "nothing to paint, nothing to reset")
}

func TestTableBordersAndHeader(t *testing.T) {
	got := newStyle(&bytes.Buffer{}).table("2 contatos",
		[]string{"ID", "nome"}, [][]string{{"12", "Maria Silva"}, {"3", "Zé"}}, []string{"r", "l"})

	assert.Contains(t, got, "┌ 2 contatos ")
	assert.Contains(t, got, "├")
	assert.Contains(t, got, "└")
	assert.Contains(t, got, "│ ID")
	assert.Contains(t, got, "Maria Silva")
	// The numeric column is right-aligned: "3" carries the fill of "12".
	assert.Contains(t, got, "│  3   Zé")
}

func TestTableStretchesForALongTitle(t *testing.T) {
	title := strings.Repeat("t", 60)
	got := newStyle(&bytes.Buffer{}).table(title, []string{"a"}, [][]string{{"b"}}, nil)

	require.Contains(t, got, title)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	for _, line := range lines {
		assert.Equal(t, len([]rune(lines[0])), len([]rune(line)), "every line of the box has the top border's width")
	}
}

// An accented title has fewer runes than bytes: counting bytes there would leave the
// top border shorter than the rows it closes.
func TestTableTitleWithAccentsKeepsTheBorderAligned(t *testing.T) {
	got := newStyle(&bytes.Buffer{}).table("2 transação(ões) · total R$ 1.631,00",
		[]string{"ID", "estabelecimento"}, [][]string{{"2", "crédito"}}, []string{"r", "l"})

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	require.Len(t, lines, 5, "top border, header, separator, one row, bottom border")
	for _, line := range lines {
		assert.Equal(t, len([]rune(lines[0])), len([]rune(line)), "every line of the box has the top border's width")
	}
}

func TestCardDimsTheLabels(t *testing.T) {
	got := style{on: true}.card("Contato salvo · ID 7", [][2]string{{"nome", "Fulano"}, {"telefone", "123"}})
	assert.Contains(t, got, ansiDim+"nome", "the label carries the dim paint")
	assert.Contains(t, got, "Fulano")
	assert.Contains(t, got, "telefone", "the second label is a row of its own")
}

func TestAccentsCountAsOneColumn(t *testing.T) {
	assert.Equal(t, 4, textWidth("ação"), "runes, not bytes")
	assert.Equal(t, "ção  ", pad("ção", 5, "l"))
	assert.Equal(t, "  ção", pad("ção", 5, "r"))
}
