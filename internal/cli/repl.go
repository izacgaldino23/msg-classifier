package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"msg-classifier/internal/models"
)

// Prompt is what the loop prints before each line.
const Prompt = "msg> "

// userID is the Jev state identity for the terminal. The web sends a hidden field;
// the CLI has no form, so it states its own.
const userID = "cli"

// ClassifyFunc classifies a request; DispatchFunc routes it to a use case. They are
// plain functions so a test can drive the loop without a Jev call — the same
// reason services define jevClient at its own boundary.
type ClassifyFunc func(request *models.ReceiveMessageRequest) (*models.Classification, error)
type DispatchFunc func(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error)

// Runner is the interactive loop: prompt, read a line, classify, dispatch, print.
type Runner struct {
	classify ClassifyFunc
	dispatch DispatchFunc
	in       io.Reader
	out      io.Writer
}

func New(classify ClassifyFunc, dispatch DispatchFunc, in io.Reader, out io.Writer) *Runner {
	return &Runner{classify: classify, dispatch: dispatch, in: in, out: out}
}

// Run reads lines until "exit" or EOF. A failed message prints one line and the
// loop continues, because a hiccup in one Jev call should not end the session.
//
// ponytail: Ctrl+C needs no handler — SIGINT already ends the process, so a
// signal.Notify would be code that changes nothing.
func (r *Runner) Run() error {
	scanner := bufio.NewScanner(r.in)
	for {
		fmt.Fprint(r.out, Prompt)

		if !scanner.Scan() {
			fmt.Fprintln(r.out)
			return scanner.Err()
		}
		message := strings.TrimSpace(scanner.Text())
		switch message {
		case "":
			continue
		case "exit", "sair", "quit":
			return nil
		}

		outcome, err := r.handle(message)
		if err != nil {
			fmt.Fprintf(r.out, "erro: %v\n", err)
			continue
		}
		fmt.Fprint(r.out, Render(outcome))
	}
}

func (r *Runner) handle(message string) (*models.UseCaseOutcome, error) {
	request := &models.ReceiveMessageRequest{Message: message, UserID: userID}
	classification, err := r.classify(request)
	if err != nil {
		return nil, err
	}
	return r.dispatch(request, classification)
}