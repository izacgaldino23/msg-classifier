package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// historyLimit caps the session history; a longer session keeps the newest lines.
const historyLimit = 100

// The bytes a raw terminal delivers that are not text. Ctrl+C drops the line,
// Ctrl+D on an empty line ends the session, and a terminal sends DEL (or, on some
// configurations, Ctrl+H) for backspace.
const (
	ctrlC     = 0x03
	ctrlD     = 0x04
	ctrlH     = 0x08
	escape    = 0x1b
	backspace = 0x7f
)

// lineReader is all Run needs from a line: read it, echo it when the terminal
// allows, and answer io.EOF at the end of the input.
type lineReader interface {
	readLine(prompt string) (string, error)
}

// scannerReader is the plain path — a pipe, a file or a test. The terminal echoes
// what is typed there, and a pipe has nobody to echo to, so nothing is written back.
type scannerReader struct {
	scanner *bufio.Scanner
	out     io.Writer
}

func newScannerReader(in io.Reader, out io.Writer) *scannerReader {
	return &scannerReader{scanner: bufio.NewScanner(in), out: out}
}

func (s *scannerReader) readLine(prompt string) (string, error) {
	fmt.Fprint(s.out, prompt)
	if !s.scanner.Scan() {
		if err := s.scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return s.scanner.Text(), nil
}

// lineEditor reads one editable line: it echoes what is typed, walks the session
// history with ↑/↓, erases with backspace and redraws the whole line on every key,
// which is what keeps a recalled line editable without any cursor arithmetic.
//
// raw switches the terminal to raw input for the duration of one line and hands back
// the restore. Doing it per line — instead of for the whole session — keeps the
// console's own editing between reads, so Ctrl+C still interrupts a slow
// classification; a permanently raw console would swallow it into the input buffer.
type lineEditor struct {
	in  *bufio.Reader
	out io.Writer
	raw func() (restore func(), err error)

	// history is oldest → newest; the cursor walks it from the end.
	history []string
}

func newLineEditor(in io.Reader, out io.Writer, raw func() (func(), error)) *lineEditor {
	return &lineEditor{in: bufio.NewReader(in), out: out, raw: raw}
}

// readLine draws the prompt and returns the submitted line. Ctrl+C drops the line
// and answers an empty one, so the loop prints a fresh prompt; Ctrl+D on an empty
// line answers io.EOF, exactly like the end of a pipe.
func (e *lineEditor) readLine(prompt string) (string, error) {
	if e.raw != nil {
		restore, err := e.raw()
		if err != nil {
			return "", err
		}
		defer restore()
	}

	line := []rune{}
	draft := []rune{} // what was typed before the browsing started
	cursor := -1      // -1 = the fresh line, otherwise an index into history
	e.draw(prompt, line)

	for {
		r, _, err := e.in.ReadRune()
		if err != nil {
			return "", err // io.EOF: Run prints the newline the terminal did not
		}

		switch {
		case r == '\r' || r == '\n':
			fmt.Fprint(e.out, "\r\n")
			submitted := string(line)
			e.push(submitted)
			return submitted, nil

		case r == ctrlC:
			fmt.Fprint(e.out, "\r\033[K\r\n")
			return "", nil

		case r == ctrlD:
			if len(line) == 0 {
				return "", io.EOF
			}

		case r == backspace || r == ctrlH:
			if len(line) > 0 {
				line = line[:len(line)-1]
				draft, cursor = copyRunes(line), -1
				e.draw(prompt, line)
			}

		case r == escape:
			switch e.readEscape() {
			case 'A':
				line, draft, cursor = e.historyUp(line, draft, cursor)
				e.draw(prompt, line)
			case 'B':
				line, draft, cursor = e.historyDown(line, draft, cursor)
				e.draw(prompt, line)
			}

		case r >= ' ' && r != backspace:
			line = append(line, r)
			draft, cursor = copyRunes(line), -1
			e.draw(prompt, line)
		}
	}
}

// draw redraws the prompt and the line after erasing it, so the cursor always ends
// up where the text does.
//
// shortcut: it erases the current row only, so a line wider than the terminal that
// then shrinks can leave its wrapped rows above behind — erasing them needs the
// terminal width (term.GetSize) and a row count, the cursor arithmetic this redraw
// exists to avoid.
func (e *lineEditor) draw(prompt string, line []rune) {
	fmt.Fprintf(e.out, "\r\033[K%s%s", prompt, string(line))
}

// readEscape consumes the rest of an escape sequence and returns its final letter:
// ↑ and ↓ arrive as "ESC [ A" and "ESC [ B" (or "ESC O A"/"ESC O B" from some
// terminals), sometimes with parameters in between ("ESC [ 1 ; 5 A"), which are read
// and dropped. A lone ESC — or a sequence we do not map — is a no-op.
func (e *lineEditor) readEscape() rune {
	next, _, err := e.in.ReadRune()
	if err != nil || (next != '[' && next != 'O') {
		return 0
	}
	for range 8 {
		r, _, err := e.in.ReadRune()
		if err != nil {
			return 0
		}
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
			return r
		}
	}
	return 0
}

// historyUp walks to an older entry, keeping the typed line as the draft that ↓
// restores. At the oldest entry it stays put.
func (e *lineEditor) historyUp(line, draft []rune, cursor int) ([]rune, []rune, int) {
	if len(e.history) == 0 {
		return line, draft, cursor
	}
	switch {
	case cursor == -1:
		draft = copyRunes(line)
		cursor = len(e.history) - 1
	case cursor > 0:
		cursor--
	default:
		return line, draft, cursor
	}
	return []rune(e.history[cursor]), draft, cursor
}

// historyDown walks to a newer entry; past the newest it hands the draft back.
func (e *lineEditor) historyDown(line, draft []rune, cursor int) ([]rune, []rune, int) {
	if cursor == -1 {
		return line, draft, cursor
	}
	if cursor == len(e.history)-1 {
		return copyRunes(draft), draft, -1
	}
	return []rune(e.history[cursor+1]), draft, cursor + 1
}

// push stores the submitted line. A blank line and a repeat of the newest entry are
// not history — the same lines a shell drops.
func (e *lineEditor) push(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	if n := len(e.history); n > 0 && e.history[n-1] == line {
		return
	}
	e.history = append(e.history, line)
	if len(e.history) > historyLimit {
		e.history = e.history[len(e.history)-historyLimit:]
	}
}

func copyRunes(line []rune) []rune { return append([]rune(nil), line...) }
