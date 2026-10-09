package cli

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"msg-classifier/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// editorOver builds an editor over a test buffer. raw is nil: a buffer is not a
// terminal, so the editor reads bytes without touching any console.
func editorOver(input string, history ...string) (*lineEditor, *strings.Builder) {
	var out strings.Builder
	editor := newLineEditor(strings.NewReader(input), &out, nil)
	editor.history = history
	return editor, &out
}

func TestEditorRecallsHistoryWithArrows(t *testing.T) {
	editor, out := editorOver("\x1b[A\r\x1b[A\x1b[A\r", "primeira", "segunda")

	first, err := editor.readLine(Prompt)
	require.NoError(t, err, "readLine()")
	assert.Equal(t, "segunda", first, "↑ recalls the newest line")

	second, err := editor.readLine(Prompt)
	require.NoError(t, err, "readLine()")
	assert.Equal(t, "primeira", second, "↑↑ walks to the older line")

	assert.Contains(t, out.String(), "segunda", "the recalled line is drawn")
}

func TestEditorDownRestoresTheDraft(t *testing.T) {
	editor, _ := editorOver("ras\x1b[A\x1b[B\r", "antiga")

	line, err := editor.readLine(Prompt)
	require.NoError(t, err, "readLine()")
	assert.Equal(t, "ras", line, "↓ past the newest gives back what was typed")
	assert.Equal(t, []string{"antiga", "ras"}, editor.history, "the submitted draft is history")
}

func TestEditorBackspaceDropsOneRuneNotOneByte(t *testing.T) {
	editor, _ := editorOver("ação\x7f\r")

	line, err := editor.readLine(Prompt)
	require.NoError(t, err, "readLine()")
	assert.Equal(t, "açã", line, "backspace erases a rune")
}

func TestEditorCtrlCAnswersAnEmptyLine(t *testing.T) {
	editor, _ := editorOver("abc\x03ok\r")

	dropped, err := editor.readLine(Prompt)
	require.NoError(t, err, "readLine()")
	assert.Equal(t, "", dropped, "Ctrl+C drops the line")

	line, err := editor.readLine(Prompt)
	require.NoError(t, err, "readLine()")
	assert.Equal(t, "ok", line)
	assert.Equal(t, []string{"ok"}, editor.history, "the dropped line is not history")
}

func TestEditorCtrlDIsEOFOnAnEmptyLineOnly(t *testing.T) {
	editor, _ := editorOver("\x04", "antiga")
	line, err := editor.readLine(Prompt)
	assert.ErrorIs(t, err, io.EOF, "Ctrl+D on an empty line ends the input")
	assert.Equal(t, "", line)

	editor, _ = editorOver("ab\x04c\r")
	line, err = editor.readLine(Prompt)
	require.NoError(t, err, "readLine()")
	assert.Equal(t, "abc", line, "Ctrl+D mid-line is ignored")
}

func TestEditorConsumesEscapeSequencesWithParameters(t *testing.T) {
	// Ctrl+↑ is "ESC [ 1 ; 5 A": the parameters must be swallowed, the final letter
	// still maps — here to a plain ↑.
	editor, out := editorOver("\x1b[1;5A\r", "antiga")
	line, err := editor.readLine(Prompt)
	require.NoError(t, err, "readLine()")
	assert.Equal(t, "antiga", line, "the sequence maps by its final letter")
	assert.NotContains(t, out.String(), "1;5", "the parameters never become text")

	// A sequence we do not map (Ctrl+→) is a no-op, not text.
	editor, _ = editorOver("\x1b[1;5Cx\r")
	line, err = editor.readLine(Prompt)
	require.NoError(t, err, "readLine()")
	assert.Equal(t, "x", line, "an unmapped sequence is dropped")
}

func TestEditorDropsBlankLinesAndRepeatsFromHistory(t *testing.T) {
	editor, _ := editorOver("")
	editor.push("x")
	editor.push("x")
	editor.push("   ")
	editor.push("y")

	assert.Equal(t, []string{"x", "y"}, editor.history)
}

func TestEditorHistoryKeepsTheNewestLines(t *testing.T) {
	editor, _ := editorOver("")
	for i := range historyLimit + 5 {
		editor.push(fmt.Sprintf("linha %d", i))
	}

	require.Len(t, editor.history, historyLimit, "the cap holds")
	assert.Equal(t, "linha 5", editor.history[0], "the oldest lines were dropped")
	assert.Equal(t, "linha 104", editor.history[len(editor.history)-1], "the newest line stays")
}

// The loop must run through the same editor the terminal gets: an injected reader is
// how a test drives ↑ without a TTY.
func TestRunnerReadsThroughTheInjectedEditor(t *testing.T) {
	var out strings.Builder
	var seen []string
	runner := New(
		func(request *models.ReceiveMessageRequest) (*models.Classification, error) {
			seen = append(seen, request.Message)
			return okClassify(request)
		},
		okDispatch,
		&fakeData{},
		strings.NewReader(""),
		&out,
	)
	runner.reader = newLineEditor(strings.NewReader("salva o fulano\r\x1b[A\r"), &out, nil)

	require.NoError(t, runner.Run(), "Run()")

	assert.Equal(t, []string{"salva o fulano", "salva o fulano"}, seen,
		"↑ brings the previous line back as a new message")
}
