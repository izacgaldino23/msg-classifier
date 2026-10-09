package cli

import (
	"errors"
	"fmt"
	"strings"

	"msg-classifier/internal/messages"
	"msg-classifier/internal/models"
	"msg-classifier/internal/ptbr"
)

// DataLister is the browse surface the /commands read. *services.DataService
// satisfies it, and this interface is all the CLI knows of the service layer — the
// same reason classify and dispatch arrive as plain functions.
type DataLister interface {
	ListContacts(filter string) ([]models.Contact, error)
	ListNotes(filter string) ([]*models.Note, error)
	ListTransactions(filter, term string) ([]*models.Transaction, error)
}

// errExitSession is the one reason a command stops the loop.
var errExitSession = errors.New("exit the session")

// command is one row of the help table and one branch of the parser. The usage is
// syntax, not prose, so it stays here; the description is a key into the locale.
type command struct {
	name  string
	usage string
	help  string
	run   func(r *Runner, args []string) error
}

// commandTable is the one list the parser reads and the help table prints. It is a
// function, not a variable, so the help command can range over it without a package
// initialization cycle.
func commandTable() []command {
	return []command{
		{"/ajuda", "", "ajuda", func(r *Runner, _ []string) error { return r.showHelp() }},
		{"/contatos", "[filtro]", "contatos", (*Runner).showContacts},
		{"/notas", "[filtro]", "notas", (*Runner).showNotes},
		{"/financas", "[filtro] [termo]", "financas", (*Runner).showTransactions},
		{"/limpar", "", "limpar", (*Runner).clearScreen},
		{"/sair", "", "sair", func(*Runner, []string) error { return errExitSession }},
	}
}

// aliases resolve a typed word to a canonical command: the short forms and the bare
// words the terminal has always accepted for exit. The bare letters u/n/c are NOT
// here — they answer the duplicate confirmation loop (DC-008).
var aliases = map[string]string{
	"/ajuda": "/ajuda", "/?": "/ajuda", "/help": "/ajuda", "ajuda": "/ajuda", "?": "/ajuda",
	"/contatos": "/contatos", "/c": "/contatos",
	"/notas": "/notas", "/n": "/notas",
	"/financas": "/financas", "/f": "/financas",
	"/limpar": "/limpar",
	"/sair":   "/sair", "sair": "/sair", "exit": "/sair", "quit": "/sair",
}

// The PT-BR filter words, mapped to the tokens the data service validates. The
// transaction words are also the models constants, spelled without the accent.
var (
	contactFilters = map[string]string{"todos": "all", "telefone": "phone", "email": "email", "nome": "name"}
	noteFilters    = map[string]string{"todas": "all", "nota": "note", "lembrete": "reminder", "tarefa": "todo"}
	transactionFilters = map[string]string{
		"todas": "all", "compra": "compra", "venda": "venda", "pagamento": "pagamento",
		"recebimento": "recebimento", "transferencia": "transferencia",
	}
)

// parse resolves a line into a command. isCommand tells a line that names a command
// (a leading "/" or a bare alias) from a message, and known separates an existing
// command from a typo, so the loop can answer each with its own line.
func parse(line string) (cmd command, args []string, isCommand, known bool) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return command{}, nil, false, false
	}
	word := strings.ToLower(fields[0])
	name, aliased := aliases[word]
	if !aliased {
		if !strings.HasPrefix(word, "/") {
			return command{}, nil, false, false
		}
		name = word
	}
	for _, candidate := range commandTable() {
		if candidate.name == name {
			return candidate, fields[1:], true, true
		}
	}
	return command{name: name}, nil, true, false
}

// showHelp prints the command table: the command in the accent color, the description dim.
func (r *Runner) showHelp() error {
	rows := make([][]string, 0, len(commandTable()))
	for _, cmd := range commandTable() {
		line := cmd.name
		if cmd.usage != "" {
			line += " " + cmd.usage
		}
		rows = append(rows, []string{line, messages.HelpFor(cmd.help)})
	}
	fmt.Fprint(r.out, r.style.box(messages.HelpHeader(), nil, rows, nil, []string{"accent", "dim"}))
	fmt.Fprintln(r.out, "  "+r.style.dim(messages.HelpFooter()))
	return nil
}

func (r *Runner) showContacts(args []string) error {
	filter, bad := oneFilter(args, contactFilters)
	if bad != "" {
		return r.reportBad(bad)
	}
	contacts, err := r.data.ListContacts(filter)
	if err != nil {
		return err
	}
	if len(contacts) == 0 {
		fmt.Fprintln(r.out, r.style.yellow(messages.ListEmpty("contact")))
		return nil
	}
	rows := make([][]string, 0, len(contacts))
	for _, contact := range contacts {
		rows = append(rows, []string{
			fmt.Sprintf("%d", contact.ID),
			contact.Name,
			optional(contact.Phone),
			optional(contact.Email),
		})
	}
	headers := []string{messages.Field("id"), messages.Field("name"), messages.Field("phone"), messages.Field("email")}
	fmt.Fprint(r.out, r.style.table(messages.ListCount("contacts", len(contacts)), headers, rows, []string{"r"}))
	return nil
}

func (r *Runner) showNotes(args []string) error {
	filter, bad := oneFilter(args, noteFilters)
	if bad != "" {
		return r.reportBad(bad)
	}
	notes, err := r.data.ListNotes(filter)
	if err != nil {
		return err
	}
	if len(notes) == 0 {
		fmt.Fprintln(r.out, r.style.yellow(messages.ListEmpty("note")))
		return nil
	}
	rows := make([][]string, 0, len(notes))
	for _, note := range notes {
		date, clock := "", ""
		if note.Date != nil {
			date = ptbr.DateBR(note.Date)
		}
		if note.Time != nil {
			clock = *note.Time
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", note.ID), messages.Label(note.Type), date, clock, note.Content,
		})
		// shortcut: the items ride in a continuation row of the content column; a note
		// with a long list is easier to read in the data screen's drawer.
		if len(note.Items) > 0 {
			rows = append(rows, []string{"", "", "", "", itemLine(note.Items)})
		}
	}
	headers := []string{messages.Field("id"), messages.Field("type"), messages.Field("date"), messages.Field("time"), messages.Field("content")}
	fmt.Fprint(r.out, r.style.table(messages.ListCount("notes", len(notes)), headers, rows, []string{"r"}))
	return nil
}

func (r *Runner) showTransactions(args []string) error {
	filter, term := transactionFilter(args)
	transactions, err := r.data.ListTransactions(filter, term)
	if err != nil {
		return err
	}
	if len(transactions) == 0 {
		fmt.Fprintln(r.out, r.style.yellow(messages.ListEmpty("transaction")))
		return nil
	}
	rows := make([][]string, 0, len(transactions))
	total := 0.0
	for _, transaction := range transactions {
		total += transaction.Amount
		party := transaction.Party
		if party == "" {
			party = "—"
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", transaction.ID), ptbr.DateBR(&transaction.Date), transaction.Type,
			ptbr.MoneyBRL(transaction.Amount), party,
		})
	}
	// The total sums what is on screen, not what is in the database: the header says
	// which rows it covers.
	title := messages.ListCount("transactions", len(transactions))
	if total > 0 {
		title += " · " + messages.ListTotal(ptbr.MoneyBRL(total))
	}
	headers := []string{messages.Field("id"), messages.Field("date"), messages.Field("type"), messages.Field("amount"), messages.Field("party")}
	fmt.Fprint(r.out, r.style.table(title, headers, rows, []string{"r", "l", "l", "r", "l"}))
	return nil
}

// clearScreen wipes the terminal; a piped session gets nothing, because there is no
// screen to clear and no escapes to write.
func (r *Runner) clearScreen(_ []string) error {
	if r.style.active() {
		fmt.Fprint(r.out, "\033[H\033[2J")
	}
	return nil
}

// transactionFilter reads the optional filter word and the search term: the first
// argument is a filter only when it names a type, so "/financas mercado" searches for
// "mercado" across every type.
func transactionFilter(args []string) (filter, term string) {
	filter = "all"
	if len(args) == 0 {
		return filter, ""
	}
	if token, ok := transactionFilters[strings.ToLower(args[0])]; ok {
		filter, args = token, args[1:]
	}
	return filter, strings.Join(args, " ")
}

// oneFilter maps the optional PT-BR filter word to the service token. A word nobody
// knows and a second argument both come back as the text of one line to print.
func oneFilter(args []string, filters map[string]string) (filter, bad string) {
	switch len(args) {
	case 0:
		return "all", ""
	case 1:
		if token, ok := filters[strings.ToLower(args[0])]; ok {
			return token, ""
		}
		return "", messages.CommandBadFilter(args[0])
	default:
		return "", messages.CommandBadArgs(args[1])
	}
}

// reportBad prints an error line a command produced from user input: expected, so it
// does not end the session and it is not the "erro:" line a failed call gets.
func (r *Runner) reportBad(text string) error {
	if text == "" {
		return nil
	}
	fmt.Fprintln(r.out, r.style.red("✗")+" "+text)
	return nil
}

// optional renders an optional column value; a pointer never reaches the terminal.
func optional(value *string) string {
	if value == nil || *value == "" {
		return "—"
	}
	return *value
}

// itemLine renders a to-do list on one line: "✓ pão · ○ leite".
func itemLine(items []models.TodoItem) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		mark := "○"
		if item.Done {
			mark = "✓"
		}
		parts = append(parts, mark+" "+item.Text)
	}
	return strings.Join(parts, " · ")
}
