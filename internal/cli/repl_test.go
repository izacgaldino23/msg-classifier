package cli

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"msg-classifier/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func okClassify(request *models.ReceiveMessageRequest) (*models.Classification, error) {
	return &models.Classification{
		Category: models.CategoryFinding{Choice: "contact", Confidence: 0.94},
		Kind:     models.KindFinding{Choice: "add", Confidence: 0.86},
	}, nil
}

func okDispatch(request *models.ReceiveMessageRequest, _ *models.Classification) (*models.UseCaseOutcome, error) {
	return &models.UseCaseOutcome{
		Classification: &models.Classification{
			Category: models.CategoryFinding{Choice: "contact", Confidence: 0.94},
			Kind:     models.KindFinding{Choice: "add", Confidence: 0.86},
		},
		Action:  models.ActionContactAdd,
		Contact: &models.Contact{ID: 7, Name: "Fulano Tal"},
	}, nil
}

func TestRunnerClassifiesAndPrintsEachLine(t *testing.T) {
	var out strings.Builder
	var seen []string
	runner := New(
		func(request *models.ReceiveMessageRequest) (*models.Classification, error) {
			seen = append(seen, request.Message)
			return okClassify(request)
		},
		okDispatch,
		strings.NewReader("salva o fulano\nexit\n"),
		&out,
	)

	require.NoError(t, runner.Run(), "Run()")

	assert.Equal(t, []string{"salva o fulano"}, seen, "each line is one message")
	body := out.String()
	assert.Contains(t, body, Prompt)
	assert.Contains(t, body, "contato salvo · id 7")
	assert.Contains(t, body, "nome: Fulano Tal")
}

// One failure must print one line and the loop must keep going: losing the session
// over a single upstream hiccup is the whole difference between a REPL and a script.
func TestRunnerContinuesAfterAnError(t *testing.T) {
	var out strings.Builder
	attempts := 0
	runner := New(
		func(request *models.ReceiveMessageRequest) (*models.Classification, error) {
			attempts++
			if attempts == 1 {
				return nil, errors.New("failed to call jev")
			}
			return okClassify(request)
		},
		okDispatch,
		strings.NewReader("primeira\nsegunda\nexit\n"),
		&out,
	)

	require.NoError(t, runner.Run(), "Run()")
	body := out.String()
	assert.Equal(t, 2, attempts, "the second line must reach the classifier")
	assert.Contains(t, body, "erro: failed to call jev")
	assert.Contains(t, body, "contato salvo · id 7")
}

func TestRunnerSkipsBlankLines(t *testing.T) {
	var out strings.Builder
	attempts := 0
	runner := New(
		func(request *models.ReceiveMessageRequest) (*models.Classification, error) {
			attempts++
			return okClassify(request)
		},
		okDispatch,
		strings.NewReader("\n   \n\nexit\n"),
		&out,
	)

	require.NoError(t, runner.Run(), "Run()")
	assert.Zero(t, attempts, "a blank line is not a message")
}

func TestRunnerExitsOnEOF(t *testing.T) {
	var out strings.Builder
	runner := New(okClassify, okDispatch, strings.NewReader("oi\n"), &out)

	require.NoError(t, runner.Run(), "EOF is a normal exit")
	assert.Contains(t, out.String(), "contato salvo · id 7")
}

func TestRunnerExitWords(t *testing.T) {
	for _, word := range []string{"exit", "sair", "quit", "  exit  "} {
		var out strings.Builder
		runner := New(okClassify, okDispatch, strings.NewReader(word+"\n"), &out)
		require.NoError(t, runner.Run(), "Run() with %q", word)
		assert.NotContains(t, out.String(), "contato salvo", "%q must stop before classifying", word)
	}
}

func TestRunnerSendsTheCLIUserID(t *testing.T) {
	var seen *models.ReceiveMessageRequest
	runner := New(
		func(request *models.ReceiveMessageRequest) (*models.Classification, error) {
			seen = request
			return okClassify(request)
		},
		okDispatch,
		strings.NewReader("oi\n"),
		&strings.Builder{},
	)

	require.NoError(t, runner.Run(), "Run()")
	require.NotNil(t, seen, "Classify() was never called")
	assert.Equal(t, userID, seen.UserID, "the terminal has no form, so it states its own identity")
}

func TestRunnerReturnsTheScannerError(t *testing.T) {
	failing := &erroringReader{}
	runner := New(okClassify, okDispatch, failing, &strings.Builder{})

	err := runner.Run()
	require.Error(t, err, "a read failure must surface")
	assert.Contains(t, err.Error(), fmt.Sprintf("%v", errRead), "the scanner error must pass through")
}

var errRead = errors.New("input exploded")

type erroringReader struct{}

func (r *erroringReader) Read([]byte) (int, error) { return 0, errRead }