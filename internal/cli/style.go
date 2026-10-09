package cli

import (
	"io"
	"os"
	"strings"
)

// The five colors the terminal uses, plus bold for a table header. An accent (cyan)
// marks the prompt and a card title; dim greys the echo, the classification line, the
// borders, the labels and the hints; green/red belong to the extraction marks; yellow
// to a "nothing found" line. Nothing else is painted, so the text the user typed keeps
// the terminal's own foreground.
const (
	ansiAccent = "\033[36m"
	ansiDim    = "\033[90m"
	ansiGreen  = "\033[32m"
	ansiRed    = "\033[31m"
	ansiYellow = "\033[33m"
	ansiBold   = "\033[1m"
	ansiReset  = "\033[0m"
)

// cellGap separates two columns inside a box.
const cellGap = "   "

// style paints text and draws boxes. Colors are dropped when NO_COLOR is set or the
// output is not a terminal, so a pipe or a file receives plain text with no escapes.
type style struct{ on bool }

// newStyle decides once, from the writer it will use, whether escapes may be written.
func newStyle(out io.Writer) style {
	if os.Getenv("NO_COLOR") != "" {
		return style{}
	}
	file, ok := out.(*os.File)
	if !ok {
		return style{}
	}
	info, err := file.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return style{}
	}
	return style{on: true}
}

// active reports whether escapes may be written: the in-flight indicator needs to know,
// because erasing a line means writing one.
func (s style) active() bool { return s.on }

func (s style) paint(code, text string) string {
	if !s.on || text == "" {
		return text
	}
	return code + text + ansiReset
}

func (s style) accent(text string) string { return s.paint(ansiAccent, text) }
func (s style) dim(text string) string    { return s.paint(ansiDim, text) }
func (s style) green(text string) string  { return s.paint(ansiGreen, text) }
func (s style) red(text string) string    { return s.paint(ansiRed, text) }
func (s style) yellow(text string) string { return s.paint(ansiYellow, text) }
func (s style) bold(text string) string   { return s.paint(ansiBold, text) }

// table draws a titled table: the header is bold, the columns are as wide as their
// content and aligns is one "l"/"r" per column ("r" for the numeric ones).
func (s style) table(title string, header []string, rows [][]string, aligns []string) string {
	return s.box(title, header, rows, aligns, nil)
}

// card draws a titled field list, the first column a dim label.
func (s style) card(title string, fields [][2]string) string {
	rows := make([][]string, 0, len(fields))
	for _, field := range fields {
		rows = append(rows, []string{field[0], field[1]})
	}
	return s.box(title, nil, rows, nil, []string{"dim", ""})
}

// cardLine draws a titled card whose whole body is one line of " · "-joined values,
// for the records that read better on a single line than as a field list.
func (s style) cardLine(title string, parts []string) string {
	return s.box(title, nil, [][]string{{strings.Join(parts, " · ")}}, nil, nil)
}

// box is the one drawing routine: border, optional title in the top border, optional
// bold header, cells padded to their column. colors names each column's paint
// ("", "dim", "accent" or "bold"), because which cell is a label is the caller's call.
func (s style) box(title string, header []string, rows [][]string, aligns, colors []string) string {
	columns := len(header)
	for _, row := range rows {
		if len(row) > columns {
			columns = len(row)
		}
	}
	if columns == 0 {
		return ""
	}

	widths := make([]int, columns)
	for i, cell := range header {
		widths[i] = textWidth(cell)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < columns && textWidth(cell) > widths[i] {
				widths[i] = textWidth(cell)
			}
		}
	}

	inner := 2 + len(cellGap)*(columns-1)
	for _, width := range widths {
		inner += width
	}
	// A title longer than the columns stretches the box instead of truncating it.
	// Widths are runes throughout: len() on "transação" would count the two accents
	// twice and leave the top border shorter than the rows it closes.
	if textWidth(title)+2 > inner {
		widths[columns-1] += textWidth(title) + 2 - inner
		inner = textWidth(title) + 2
	}

	titleSegment := ""
	if title != "" {
		titleSegment = " " + title + " "
	}

	var out strings.Builder
	out.WriteString(s.dim("┌"))
	out.WriteString(s.accent(titleSegment))
	out.WriteString(s.dim(strings.Repeat("─", inner-textWidth(titleSegment)) + "┐"))
	out.WriteString("\n")

	row := func(cells []string, isHeader bool) {
		out.WriteString(s.dim("│"))
		out.WriteString(" ")
		for i := 0; i < columns; i++ {
			if i > 0 {
				out.WriteString(cellGap)
			}
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			align := "l"
			if i < len(aligns) {
				align = aligns[i]
			}
			padded := pad(cell, widths[i], align)
			switch {
			case isHeader:
				padded = s.bold(padded)
			case i < len(colors):
				switch colors[i] {
				case "dim":
					padded = s.dim(padded)
				case "accent":
					padded = s.accent(padded)
				case "bold":
					padded = s.bold(padded)
				}
			}
			out.WriteString(padded)
		}
		out.WriteString(" ")
		out.WriteString(s.dim("│"))
		out.WriteString("\n")
	}

	if len(header) > 0 {
		row(header, true)
		out.WriteString(s.dim("├" + strings.Repeat("─", inner) + "┤"))
		out.WriteString("\n")
	}
	for _, cells := range rows {
		row(cells, false)
	}
	out.WriteString(s.dim("└" + strings.Repeat("─", inner) + "┘"))
	out.WriteString("\n")
	return out.String()
}

// pad pads a cell to width by its rune count, so an accent or a box character counts
// as one column.
func pad(cell string, width int, align string) string {
	fill := width - textWidth(cell)
	if fill <= 0 {
		return cell
	}
	if align == "r" {
		return strings.Repeat(" ", fill) + cell
	}
	return cell + strings.Repeat(" ", fill)
}

func textWidth(text string) int { return len([]rune(text)) }
