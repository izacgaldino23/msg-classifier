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

	// pending is the message awaiting a duplicate answer (u/n/c, DC-008); empty
	// means the loop is closed and the next line is a fresh message.
	pending string
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
		line := strings.TrimSpace(scanner.Text())
		switch line {
		case "":
			continue
		case "exit", "sair", "quit":
			return nil
		}

		// Duplicate confirmation loop (DC-008): only the first line after a
		// duplicate is read as an answer; anything else cancels and starts over.
		if r.pending != "" {
			switch answer := confirmAction(line); answer {
			case "cancel":
				r.pending = ""
				fmt.Fprintln(r.out, "cancelado")
				continue
			case "update", "new":
				message := r.pending
				r.pending = ""
				outcome, err := r.handle(message, answer)
				if err != nil {
					fmt.Fprintf(r.out, "erro: %v\n", err)
				} else {
					fmt.Fprint(r.out, Render(outcome))
					r.remember(outcome)
				}
				continue
			default:
				// Not an answer: the pending loop is dropped and this line is a
				// fresh message.
				r.pending = ""
			}
		}

		outcome, err := r.handle(line, "")
		if err != nil {
			fmt.Fprintf(r.out, "erro: %v\n", err)
			continue
		}
		fmt.Fprint(r.out, Render(outcome))
		r.remember(outcome)
	}
}

// remember opens the confirmation loop when the outcome is a duplicate.
func (r *Runner) remember(outcome *models.UseCaseOutcome) {
	if isDuplicate(outcome.Action) {
		fmt.Fprint(r.out, Options)
		r.pending = outcome.Message
		return
	}
	r.pending = ""
}

// confirmAction maps the answer line to a dup_action; "" means not an answer.
func confirmAction(line string) string {
	switch line {
	case "u", "atualizar":
		return "update"
	case "n", "novo", "sim":
		return "new"
	case "c", "cancelar", "não", "nao":
		return "cancel"
	default:
		return ""
	}
}

func (r *Runner) handle(message, dupAction string) (*models.UseCaseOutcome, error) {
	request := &models.ReceiveMessageRequest{Message: message, UserID: userID, DupAction: dupAction}
	classification, err := r.classify(request)
	if err != nil {
		return nil, err
	}
	return r.dispatch(request, classification)
}