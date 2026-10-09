package cli

import (
	"strings"
	"testing"

	"msg-classifier/internal/messages"
	"msg-classifier/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dupDispatch returns a duplicate outcome on the first call and an add outcome on
// a confirmed re-post, recording the dup_action it received.
func dupDispatch(action *models.Action, dupAction *string, message *string) DispatchFunc {
	return func(request *models.ReceiveMessageRequest, _ *models.Classification) (*models.UseCaseOutcome, error) {
		*dupAction = request.DupAction
		*message = request.Message
		if request.DupAction == "" {
			*action = models.ActionContactDuplicate
			return &models.UseCaseOutcome{
				Classification: okClassifyResult(),
				Action:         models.ActionContactDuplicate,
				Contact:        &models.Contact{Name: "Fulano"},
				Message:        request.Message,
			}, nil
		}
		*action = models.ActionContactAdd
		return &models.UseCaseOutcome{
			Classification: okClassifyResult(),
			Action:         models.ActionContactAdd,
			Contact:        &models.Contact{ID: 7, Name: "Fulano"},
			Message:        request.Message,
		}, nil
	}
}

func okClassifyResult() *models.Classification {
	return &models.Classification{
		Category: models.CategoryFinding{Choice: "contact", Confidence: 0.94},
		Kind:     models.KindFinding{Choice: "add", Confidence: 0.86},
	}
}

func TestRunnerDuplicateConfirmUpdate(t *testing.T) {
	var out strings.Builder
	var action models.Action
	var dupAction, message string

	runner := New(okClassify, dupDispatch(&action, &dupAction, &message), &fakeData{},
		strings.NewReader("09292929290 Fulano\nu\nexit\n"), &out)

	require.NoError(t, runner.Run(), "Run()")

	assert.Equal(t, "update", dupAction, "u confirms with dup_action=update")
	assert.Equal(t, "09292929290 Fulano", message, "the original message is re-posted")
	body := out.String()
	assert.Contains(t, body, "Contato já existe")
	assert.Contains(t, body, messages.DupOptions())
	assert.Contains(t, body, "Contato salvo · ID 7")
}

func TestRunnerDuplicateConfirmNew(t *testing.T) {
	var out strings.Builder
	var action models.Action
	var dupAction, message string

	runner := New(okClassify, dupDispatch(&action, &dupAction, &message), &fakeData{},
		strings.NewReader("09292929290 Fulano\nn\nexit\n"), &out)

	require.NoError(t, runner.Run(), "Run()")

	assert.Equal(t, "new", dupAction, "n confirms with dup_action=new")
	assert.Contains(t, out.String(), "Contato salvo · ID 7")
}

func TestRunnerDuplicateCancel(t *testing.T) {
	var out strings.Builder
	var action models.Action
	var dupAction, message string
	calls := 0

	runner := New(okClassify,
		func(request *models.ReceiveMessageRequest, c *models.Classification) (*models.UseCaseOutcome, error) {
			calls++
			return dupDispatch(&action, &dupAction, &message)(request, c)
		},
		&fakeData{},
		strings.NewReader("09292929290 Fulano\nc\nexit\n"), &out)

	require.NoError(t, runner.Run(), "Run()")

	assert.Equal(t, 1, calls, "cancel sends nothing to dispatch")
	assert.Contains(t, out.String(), messages.Cancelled())
}

func TestRunnerDuplicateUnknownLineCancelsAndTreatsAsNewMessage(t *testing.T) {
	var out strings.Builder
	var action models.Action
	var dupAction, message string

	runner := New(okClassify, dupDispatch(&action, &dupAction, &message), &fakeData{},
		strings.NewReader("09292929290 Fulano\noutro texto qualquer\nexit\n"), &out)

	require.NoError(t, runner.Run(), "Run()")

	assert.Equal(t, "outro texto qualquer", message, "a non-answer becomes the next message")
	assert.Equal(t, "", dupAction, "the fresh message carries no dup_action")
	assert.NotContains(t, out.String(), messages.Cancelled())
}

// A command is not a duplicate answer: it drops the pending confirmation and runs,
// which keeps the two loops from swallowing each other's lines.
func TestRunnerCommandDuringPendingDuplicateCancelsIt(t *testing.T) {
	var out strings.Builder
	var action models.Action
	var dupAction, message string
	calls := 0

	runner := New(okClassify,
		func(request *models.ReceiveMessageRequest, c *models.Classification) (*models.UseCaseOutcome, error) {
			calls++
			return dupDispatch(&action, &dupAction, &message)(request, c)
		},
		&fakeData{contacts: []models.Contact{{ID: 3, Name: "Maria"}}},
		strings.NewReader("09292929290 Fulano\n/contatos\nexit\n"), &out)

	require.NoError(t, runner.Run(), "Run()")

	assert.Equal(t, 1, calls, "the command is not re-posted as a confirmation")
	body := out.String()
	assert.Contains(t, body, "Maria", "the command ran")
	assert.NotContains(t, body, "Contato salvo", "the pending confirmation was dropped")
}

func TestRenderResultDuplicateNoteBlock(t *testing.T) {
	body := Render(&models.UseCaseOutcome{
		Classification: okClassifyResult(),
		Action:         models.ActionNoteDuplicate,
		Notes:          []*models.Note{{Type: models.NoteTypeReminder, Content: "pagar conta"}},
		Message:        "lembra de pagar conta",
	})
	assert.Contains(t, body, "Nota já existe: pagar conta")
	assert.Contains(t, body, "pagar conta")
	assert.Contains(t, body, "▸ lembra de pagar conta")
}

func TestRenderResultDuplicateTransactionBlock(t *testing.T) {
	body := Render(&models.UseCaseOutcome{
		Classification: okClassifyResult(),
		Action:         models.ActionTransactionDuplicate,
		Transactions: []*models.Transaction{
			{Type: models.TransactionTypePurchase, Amount: 20, Party: "mercado", Content: "comprei 20"},
		},
		Message: "comprei 20",
	})
	assert.Contains(t, body, "Transação já existe")
	assert.Contains(t, body, "mercado")
}
