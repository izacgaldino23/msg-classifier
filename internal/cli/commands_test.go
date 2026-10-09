package cli

import (
	"strings"
	"testing"
	"time"

	"msg-classifier/internal/messages"
	"msg-classifier/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeData records the filters and terms it was asked for, so a test can assert the
// PT-BR word reached the service as the token the service validates.
type fakeData struct {
	contacts       []models.Contact
	notes          []*models.Note
	transactions   []*models.Transaction
	contactFilters []string
	contactTerms   []string
	noteFilters    []string
	noteTerms      []string
	txFilters      []string
	txTerms        []string
}

func (f *fakeData) ListContacts(filter, term string) ([]models.Contact, error) {
	f.contactFilters = append(f.contactFilters, filter)
	f.contactTerms = append(f.contactTerms, term)
	return f.contacts, nil
}

func (f *fakeData) ListNotes(filter, term string) ([]*models.Note, error) {
	f.noteFilters = append(f.noteFilters, filter)
	f.noteTerms = append(f.noteTerms, term)
	return f.notes, nil
}

func (f *fakeData) ListTransactions(filter, term string) ([]*models.Transaction, error) {
	f.txFilters = append(f.txFilters, filter)
	f.txTerms = append(f.txTerms, term)
	return f.transactions, nil
}

// commandRunner is a Runner wired to a fake data surface and a buffer.
func commandRunner(data DataLister, out *strings.Builder) *Runner {
	return New(nil, nil, data, strings.NewReader(""), out)
}

func TestParseTellsCommandsFromMessages(t *testing.T) {
	tests := []struct {
		line      string
		isCommand bool
		known     bool
		name      string
	}{
		{"/contatos telefone", true, true, "/contatos"},
		{"/c", true, true, "/contatos"},
		{"  /NOTAS  ", true, true, "/notas"},
		{"exit", true, true, "/sair"},
		{"ajuda", true, true, "/ajuda"},
		{"?", true, true, "/ajuda"},
		{"--ajuda", true, true, "/ajuda"},
		{"-h", true, true, "/ajuda"},
		{"/xyz", true, false, "/xyz"},
		{"salva o fulano", false, false, ""},
		{"n", false, false, ""},
		{"u", false, false, ""},
		{"c", false, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			cmd, _, isCommand, known := parse(tt.line)
			assert.Equal(t, tt.isCommand, isCommand, "isCommand")
			assert.Equal(t, tt.known, known, "known")
			assert.Equal(t, tt.name, cmd.name, "name")
		})
	}
}

func TestParseKeepsTheArguments(t *testing.T) {
	_, args, _, _ := parse("/financas compra mercado extra")
	assert.Equal(t, []string{"compra", "mercado", "extra"}, args)
}

func TestHelpListsEveryCommand(t *testing.T) {
	var out strings.Builder
	require.NoError(t, commandRunner(&fakeData{}, &out).showHelp(nil))

	for _, cmd := range commandTable() {
		assert.Contains(t, out.String(), cmd.name, "the help table must list %s", cmd.name)
		assert.Contains(t, out.String(), messages.HelpFor(cmd.help), "the help table must describe %s", cmd.name)
	}
	assert.Contains(t, out.String(), messages.HelpFooter())
	assert.Contains(t, out.String(), messages.HelpHint(), "the table must point to the per-command help")
}

// The short table carries the description only; the filters live in the per-command
// help, so the table stays readable.
func TestHelpTableCarriesNoFilters(t *testing.T) {
	var out strings.Builder
	require.NoError(t, commandRunner(&fakeData{}, &out).showHelp(nil))

	assert.NotContains(t, out.String(), "telefone", "a filter word reached the short table")
	assert.Contains(t, out.String(), messages.HelpFor("contatos"))
}

// assertBoxLines asserts every non-empty line of a multi-line text reached the box:
// the box draws one row per line, so the whole text is never one contiguous substring.
func assertBoxLines(t *testing.T, got, text string) {
	t.Helper()
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) != "" {
			assert.Contains(t, got, line)
		}
	}
}

func TestHelpDetailPrintsTheFilters(t *testing.T) {
	var out strings.Builder
	require.NoError(t, commandRunner(&fakeData{}, &out).showHelp([]string{"contatos"}))

	assert.Contains(t, out.String(), "/contatos")
	assertBoxLines(t, out.String(), messages.HelpDetail("contatos"))
}

// "--ajuda" is a flag on every command, answered before the command runs.
func TestHelpFlagOnACommandPrintsItsDetail(t *testing.T) {
	var out strings.Builder
	runner := commandRunner(&fakeData{}, &out)

	require.True(t, isHelpFlag([]string{"--ajuda"}))
	require.True(t, isHelpFlag([]string{"silva", "-h"}))
	require.False(t, isHelpFlag([]string{"silva"}))

	runner.printCommandHelp("/notas")
	assertBoxLines(t, out.String(), messages.HelpDetail("notas"))
}

func TestHelpForAnUnknownWord(t *testing.T) {
	var out strings.Builder
	require.NoError(t, commandRunner(&fakeData{}, &out).showHelp([]string{"xyz"}))

	assert.Contains(t, out.String(), "✗ "+messages.CommandUnknown("xyz"))
}

// The long-help alias resolves the same way a command does: "contatos" and "/c"
// both land on the /contatos detail.
func TestCanonicalCommandResolvesAliases(t *testing.T) {
	assert.Equal(t, "/contatos", canonicalCommand("contatos"))
	assert.Equal(t, "/contatos", canonicalCommand("/c"))
	assert.Equal(t, "/ajuda", canonicalCommand("--ajuda"))
	assert.Equal(t, "", canonicalCommand("xyz"))
}

// The per-command help names every filter the parser accepts, so the text cannot
// drift away from the filter maps.
func TestHelpDetailNamesEveryFilter(t *testing.T) {
	cases := []struct {
		name    string
		filters map[string]string
	}{
		{"contatos", contactFilters},
		{"notas", noteFilters},
		{"financas", transactionFilters},
	}
	for _, tc := range cases {
		detail := messages.HelpDetail(tc.name)
		for word := range tc.filters {
			assert.Contains(t, detail, word, "the %s help must name the %q filter", tc.name, word)
		}
	}
}

func TestContactsSendsTheServiceToken(t *testing.T) {
	data := &fakeData{contacts: []models.Contact{{ID: 12, Name: "Maria Silva"}}}
	var out strings.Builder
	runner := commandRunner(data, &out)

	require.NoError(t, runner.showContacts(nil))
	require.NoError(t, runner.showContacts([]string{"telefone"}))
	require.NoError(t, runner.showContacts([]string{"nome"}))

	assert.Equal(t, []string{"all", "phone", "name"}, data.contactFilters)
	assert.Equal(t, []string{"", "", ""}, data.contactTerms)
	assert.Contains(t, out.String(), messages.ListCount("contacts", 1))
	assert.Contains(t, out.String(), "Maria Silva")
}

// A word nobody knows is a search term, not an error: the first argument is a filter
// only when it names one, so the listing can be searched without a filter.
func TestContactAndNoteSearchTerm(t *testing.T) {
	data := &fakeData{}
	var out strings.Builder
	runner := commandRunner(data, &out)

	require.NoError(t, runner.showContacts([]string{"silva"}))
	require.NoError(t, runner.showContacts([]string{"telefone", "silva"}))
	require.NoError(t, runner.showNotes([]string{"xyz"}))
	require.NoError(t, runner.showNotes([]string{"lembrete", "pagar", "conta"}))

	assert.Equal(t, []string{"all", "phone"}, data.contactFilters)
	assert.Equal(t, []string{"silva", "silva"}, data.contactTerms)
	assert.Equal(t, []string{"all", "reminder"}, data.noteFilters)
	assert.Equal(t, []string{"xyz", "pagar conta"}, data.noteTerms)
	assert.NotContains(t, out.String(), "✗", "a search term is not a bad command")
}

func TestEmptyListingIsNotAnError(t *testing.T) {
	var out strings.Builder
	require.NoError(t, commandRunner(&fakeData{}, &out).showTransactions(nil))

	assert.Contains(t, out.String(), messages.ListEmpty("transaction"))
	assert.NotContains(t, out.String(), "✗", "an empty listing is not a bad command")
}

func TestTransactionFilterWordIsAFilterOnlyWhenKnown(t *testing.T) {
	data := &fakeData{}
	var out strings.Builder
	runner := commandRunner(data, &out)

	require.NoError(t, runner.showTransactions([]string{"compra", "mercado", "extra"}))
	require.NoError(t, runner.showTransactions([]string{"mercado"}))

	assert.Equal(t, []string{"compra", "all"}, data.txFilters)
	assert.Equal(t, []string{"mercado extra", "mercado"}, data.txTerms)
}

func TestTransactionListingSumsTheRowsOnScreen(t *testing.T) {
	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	data := &fakeData{transactions: []*models.Transaction{
		{ID: 1, Type: models.TransactionTypePurchase, Amount: 50, Party: "Mercado", Date: date},
		{ID: 2, Type: models.TransactionTypePurchase, Amount: 1250.5, Party: "Mercado", Date: date},
	}}
	var out strings.Builder
	require.NoError(t, commandRunner(data, &out).showTransactions(nil))

	body := out.String()
	assert.Contains(t, body, messages.ListCount("transactions", 2))
	assert.Contains(t, body, messages.ListTotal("R$ 1.300,50"))
	assert.Contains(t, body, "Mercado")
}

func TestNoteListingShowsTheItemsOnAContinuationRow(t *testing.T) {
	data := &fakeData{notes: []*models.Note{{
		ID: 4, Type: models.NoteTypeTodo, Content: "comprar",
		Items: []models.TodoItem{{Text: "pão"}, {Text: "leite", Done: true}},
	}}}
	var out strings.Builder
	require.NoError(t, commandRunner(data, &out).showNotes([]string{"tarefa"}))

	body := out.String()
	assert.Equal(t, []string{"todo"}, data.noteFilters)
	assert.Equal(t, []string{""}, data.noteTerms)
	assert.Contains(t, body, messages.Label(models.NoteTypeTodo))
	assert.Contains(t, body, "○ pão · ✓ leite", "the done flag rides with the item")
}

func TestMissingOptionalFieldsRenderAsADash(t *testing.T) {
	data := &fakeData{contacts: []models.Contact{{ID: 3, Name: "Sem Telefone"}}}
	var out strings.Builder
	require.NoError(t, commandRunner(data, &out).showContacts(nil))

	assert.Contains(t, out.String(), "—")
	assert.NotContains(t, out.String(), "0xc", "a pointer address never reaches a table")
}

func TestClearScreenWritesNothingToAPipe(t *testing.T) {
	var out strings.Builder
	require.NoError(t, commandRunner(&fakeData{}, &out).clearScreen(nil))

	assert.Empty(t, out.String(), "no screen to clear, no escapes to write")
}

func TestClearScreenWritesTheEscapeOnATerminal(t *testing.T) {
	var out strings.Builder
	runner := commandRunner(&fakeData{}, &out)
	runner.style = style{on: true}

	require.NoError(t, runner.clearScreen(nil))
	assert.Equal(t, "\033[H\033[2J", out.String())
}

// The end-to-end route: the loop reads a command line and prints its table without
// classifying anything.
func TestRunnerRunsACommandWithoutClassifying(t *testing.T) {
	var out strings.Builder
	classified := 0
	runner := New(
		func(request *models.ReceiveMessageRequest) (*models.Classification, error) {
			classified++
			return okClassify(request)
		},
		okDispatch,
		&fakeData{contacts: []models.Contact{{ID: 1, Name: "Fulano"}}},
		strings.NewReader("/contatos\n/sair\n"),
		&out,
	)

	require.NoError(t, runner.Run())
	assert.Zero(t, classified, "a command is not a message")
	assert.Contains(t, out.String(), "Fulano")
}

// The "--ajuda" flag and the bare "/ajuda <comando>" both print the detail of a
// command, and neither reaches the classifier.
func TestRunnerAnswersTheHelpFlagWithoutClassifying(t *testing.T) {
	var out strings.Builder
	classified := 0
	runner := New(
		func(request *models.ReceiveMessageRequest) (*models.Classification, error) {
			classified++
			return okClassify(request)
		},
		okDispatch,
		&fakeData{},
		strings.NewReader("/contatos --ajuda\n/ajuda notas\n/sair\n"),
		&out,
	)

	require.NoError(t, runner.Run())
	assert.Zero(t, classified, "help is not a message")
	assertBoxLines(t, out.String(), messages.HelpDetail("contatos"))
	assertBoxLines(t, out.String(), messages.HelpDetail("notas"))
}

// The working indicator is written before the classification call and erased after it,
// but only where the terminal can erase it again.
func TestWorkingIndicatorIsErasedAfterTheAnswer(t *testing.T) {
	var out strings.Builder
	runner := New(okClassify, okDispatch, &fakeData{}, strings.NewReader(""), &out)
	runner.style = style{on: true}

	runner.answerMessage("salva o fulano", "")

	body := out.String()
	working := strings.Index(body, messages.Working())
	erased := strings.Index(body, "\r\033[K")
	require.NotEqual(t, -1, working, "the indicator must be printed")
	require.NotEqual(t, -1, erased, "the indicator must be erased")
	assert.Less(t, working, erased, "the indicator is erased after the answer arrives")
}
