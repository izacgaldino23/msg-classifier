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

// fakeData records the filters it was asked for, so a test can assert the PT-BR word
// reached the service as the token the service validates.
type fakeData struct {
	contacts       []models.Contact
	notes          []*models.Note
	transactions   []*models.Transaction
	contactFilters []string
	noteFilters    []string
	txFilters      []string
	txTerms        []string
}

func (f *fakeData) ListContacts(filter string) ([]models.Contact, error) {
	f.contactFilters = append(f.contactFilters, filter)
	return f.contacts, nil
}

func (f *fakeData) ListNotes(filter string) ([]*models.Note, error) {
	f.noteFilters = append(f.noteFilters, filter)
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
	require.NoError(t, commandRunner(&fakeData{}, &out).showHelp())

	for _, cmd := range commandTable() {
		assert.Contains(t, out.String(), cmd.name, "the help table must list %s", cmd.name)
		assert.Contains(t, out.String(), messages.HelpFor(cmd.help), "the help table must describe %s", cmd.name)
	}
	assert.Contains(t, out.String(), messages.HelpFooter())
}

func TestContactsSendsTheServiceToken(t *testing.T) {
	data := &fakeData{contacts: []models.Contact{{ID: 12, Name: "Maria Silva"}}}
	var out strings.Builder
	runner := commandRunner(data, &out)

	require.NoError(t, runner.showContacts(nil))
	require.NoError(t, runner.showContacts([]string{"telefone"}))
	require.NoError(t, runner.showContacts([]string{"nome"}))

	assert.Equal(t, []string{"all", "phone", "name"}, data.contactFilters)
	assert.Contains(t, out.String(), messages.ListCount("contacts", 1))
	assert.Contains(t, out.String(), "Maria Silva")
}

func TestUnknownFilterPrintsOneLineAndContinues(t *testing.T) {
	var out strings.Builder
	require.NoError(t, commandRunner(&fakeData{}, &out).showNotes([]string{"xyz"}))

	assert.Contains(t, out.String(), "✗ "+messages.CommandBadFilter("xyz"))
}

func TestExtraArgumentPrintsOneLine(t *testing.T) {
	var out strings.Builder
	require.NoError(t, commandRunner(&fakeData{}, &out).showContacts([]string{"todos", "telefone"}))

	assert.Contains(t, out.String(), "✗ "+messages.CommandBadArgs("telefone"))
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
