package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"msg-classifier/internal/messages"
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

// Runner is the interactive loop: prompt, read a line, classify, dispatch, print. A
// line that names a command goes to commands.go instead of the classifier.
type Runner struct {
	classify ClassifyFunc
	dispatch DispatchFunc
	data     DataLister
	in       io.Reader
	out      io.Writer
	style    style

	// pending is the message awaiting a duplicate answer (u/n/c, DC-008); empty
	// means the loop is closed and the next line is a fresh message.
	pending string
}

// New wires the loop. data is read only by the /commands.
func New(classify ClassifyFunc, dispatch DispatchFunc, data DataLister, in io.Reader, out io.Writer) *Runner {
	return &Runner{classify: classify, dispatch: dispatch, data: data, in: in, out: out, style: newStyle(out)}
}

// Run reads lines until the exit command, "exit"/"sair"/"quit", or EOF. A failed
// message prints one line and the loop continues, because a hiccup in one Jev call
// should not end the session.
//
// ponytail: Ctrl+C needs no handler — SIGINT already ends the process, so a
// signal.Notify would be code that changes nothing.
func (r *Runner) Run() error {
	scanner := bufio.NewScanner(r.in)
	for {
		fmt.Fprint(r.out, r.style.accent(Prompt))

		if !scanner.Scan() {
			fmt.Fprintln(r.out)
			return scanner.Err()
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		cmd, args, isCommand, known := parse(line)
		// The help flag wins over everything, including /sair: "--ajuda" is always a
		// request to read, never the action itself.
		if isCommand && known && isHelpFlag(args) {
			fmt.Fprintln(r.out)
			r.printCommandHelp(cmd.name)
			continue
		}
		if isCommand && known && cmd.name == "/sair" {
			return nil
		}

		// Duplicate confirmation loop (DC-008): only the first line after a
		// duplicate is read as an answer; a command or any other line drops the
		// pending and is handled on its own below.
		if r.pending != "" {
			switch answer := confirmAction(line); answer {
			case "cancel":
				r.pending = ""
				fmt.Fprintln(r.out, messages.Cancelled())
				continue
			case "update", "new":
				message := r.pending
				r.pending = ""
				r.answerMessage(message, answer)
				continue
			}
			r.pending = ""
		}

		if isCommand {
			// Separate the answer from the prompt, the way the message block does:
			// the table would otherwise start on the prompt line.
			fmt.Fprintln(r.out)
			if !known {
				r.reportBad(messages.CommandUnknown(cmd.name))
				continue
			}
			// "--ajuda" (or "-h") on any command prints that command's long
			// help instead of running it — handled above, before /sair.
			if err := cmd.run(r, args); err != nil {
				if errors.Is(err, errExitSession) {
					return nil
				}
				fmt.Fprintln(r.out, messages.ErrorLine(err))
			}
			continue
		}

		r.answerMessage(line, "")
	}
}

// answerMessage classifies one message and prints its block. The in-flight indicator
// is written only when the terminal can erase it again.
func (r *Runner) answerMessage(message, dupAction string) {
	if r.style.active() {
		fmt.Fprint(r.out, r.style.dim(messages.Working()))
	}
	outcome, err := r.handle(message, dupAction)
	if r.style.active() {
		fmt.Fprint(r.out, "\r\033[K")
	}
	if err != nil {
		fmt.Fprintln(r.out, messages.ErrorLine(err))
		return
	}
	fmt.Fprint(r.out, r.style.render(outcome))
	r.remember(outcome)
}

// remember opens the confirmation loop when the outcome is a duplicate.
func (r *Runner) remember(outcome *models.UseCaseOutcome) {
	if isDuplicate(outcome.Action) {
		fmt.Fprintln(r.out, r.style.dim(messages.DupOptions()))
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
