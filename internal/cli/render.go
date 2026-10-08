// Package cli is the terminal presenter: a UseCaseOutcome becomes lines. It
// shares nothing with views or api but the outcome itself — no interface, no
// registry, just a switch on models.Action. The wording comes from
// internal/messages, so this file owns layout only.
package cli

import (
	"fmt"
	"strings"

	"msg-classifier/internal/messages"
	"msg-classifier/internal/models"
	"msg-classifier/internal/ptbr"
)

// Render turns an outcome into the block of lines the terminal prints.
func Render(outcome *models.UseCaseOutcome) string {
	var out strings.Builder
	if c := outcome.Classification; c != nil {
		fmt.Fprintln(&out, messages.Classification(c.Category.Choice, c.Kind.Choice, c.Category.Confidence, c.Kind.Confidence))
	}
	out.WriteString(body(outcome))
	out.WriteString(segments(outcome.Segments))
	return out.String()
}

// isDuplicate tells the REPL the loop is open: the next line answers u/n/c.
func isDuplicate(action models.Action) bool {
	switch action {
	case models.ActionContactDuplicate, models.ActionNoteDuplicate, models.ActionTransactionDuplicate:
		return true
	default:
		return false
	}
}

// body is the one switch every Action goes through. Wording mirrors result.html
// because both read the same locale.
func body(outcome *models.UseCaseOutcome) string {
	var out strings.Builder
	switch outcome.Action {
	case models.ActionContactAdd:
		contact := outcome.Contact
		if contact == nil {
			contact = &models.Contact{}
		}
		fmt.Fprintln(&out, messages.ContactSaved(contact.ID))
		writeContact(&out, contact, true)
	case models.ActionContactFound:
		fmt.Fprintln(&out, messages.ContactFound())
		writeContact(&out, outcome.Contact, true)
	case models.ActionContactNotFound:
		fmt.Fprintln(&out, messages.ContactNotFound(outcome.SearchTerm))
	case models.ActionContactDuplicate:
		contact := outcome.Contact
		if contact == nil {
			contact = &models.Contact{}
		}
		fmt.Fprintln(&out, messages.ContactDuplicate(contact.Name))
		writeContact(&out, contact, false)
	case models.ActionContactNoData:
		fmt.Fprintln(&out, messages.ContactNoData())
	case models.ActionNoteAdd:
		for _, note := range outcome.Notes {
			fmt.Fprintln(&out, messages.NoteSaved(note.Type, note.ID))
			writeNote(&out, note, "  ")
		}
	case models.ActionNoteFound:
		fmt.Fprintln(&out, messages.NoteFound(len(outcome.Notes), outcome.SearchTerm))
		for _, note := range outcome.Notes {
			fmt.Fprintf(&out, "  [%s] %s\n", note.Type, note.Content)
			writeNote(&out, note, "    ")
		}
	case models.ActionNoteNotFound:
		fmt.Fprintln(&out, messages.NoteNotFound(outcome.SearchTerm))
	case models.ActionNoteNoData:
		fmt.Fprintln(&out, messages.NoteNoData())
	case models.ActionNoteDuplicate:
		for _, note := range outcome.Notes {
			fmt.Fprintln(&out, messages.NoteDuplicate(note.Content))
			writeNote(&out, note, "  ")
		}
	case models.ActionTransactionAdd:
		for _, transaction := range outcome.Transactions {
			fmt.Fprintln(&out, messages.TransactionSaved(transaction.ID))
			writeTransaction(&out, transaction, "  ")
		}
	case models.ActionTransactionFound:
		fmt.Fprintln(&out, messages.TransactionFound(len(outcome.Transactions), outcome.SearchTerm))
		if outcome.Total > 0 {
			fmt.Fprintf(&out, "  %s: %s\n", messages.Field("total"), ptbr.MoneyBRL(outcome.Total))
		}
		for _, transaction := range outcome.Transactions {
			fmt.Fprintf(&out, "  [%s] %s — %s\n", transaction.Type, ptbr.MoneyBRL(transaction.Amount), transaction.Party)
			fmt.Fprintf(&out, "    %s: %s\n", messages.Field("date"), ptbr.DateBR(&transaction.Date))
		}
	case models.ActionTransactionNotFound:
		fmt.Fprintln(&out, messages.TransactionNotFound(outcome.SearchTerm))
	case models.ActionTransactionNoData:
		fmt.Fprintln(&out, messages.TransactionNoData(outcome.Missing))
	case models.ActionTransactionDuplicate:
		for _, transaction := range outcome.Transactions {
			fmt.Fprintln(&out, messages.TransactionDuplicate())
			writeTransaction(&out, transaction, "  ")
		}
	}
	return out.String()
}

func writeContact(out *strings.Builder, contact *models.Contact, withName bool) {
	if contact == nil {
		return
	}
	if withName && contact.Name != "" {
		fmt.Fprintf(out, "  %s: %s\n", messages.Field("name"), contact.Name)
	}
	if contact.Phone != nil && *contact.Phone != "" {
		fmt.Fprintf(out, "  %s: %s\n", messages.Field("phone"), *contact.Phone)
	}
	if contact.Email != nil && *contact.Email != "" {
		fmt.Fprintf(out, "  %s: %s\n", messages.Field("email"), *contact.Email)
	}
}

// writeNote prints the fields a note may carry, indented by the caller: the add block
// indents two spaces, the found list four.
func writeNote(out *strings.Builder, note *models.Note, indent string) {
	fmt.Fprintf(out, "%s%s: %s\n", indent, messages.Field("content"), note.Content)
	if note.Date != nil {
		fmt.Fprintf(out, "%s%s: %s\n", indent, messages.Field("date"), ptbr.DateBR(note.Date))
	}
	if note.Time != nil && *note.Time != "" {
		fmt.Fprintf(out, "%s%s: %s\n", indent, messages.Field("time"), *note.Time)
	}
	writeItems(out, note.Items, indent)
}

func writeTransaction(out *strings.Builder, transaction *models.Transaction, indent string) {
	fmt.Fprintf(out, "%s%s: %s\n", indent, messages.Field("type"), transaction.Type)
	fmt.Fprintf(out, "%s%s: %s\n", indent, messages.Field("amount"), ptbr.MoneyBRL(transaction.Amount))
	fmt.Fprintf(out, "%s%s: %s\n", indent, messages.Field("date"), ptbr.DateBR(&transaction.Date))
	if transaction.Party != "" {
		fmt.Fprintf(out, "%s%s: %s\n", indent, messages.Field("party"), transaction.Party)
	}
	if transaction.Content != "" {
		fmt.Fprintf(out, "%s%s: %s\n", indent, messages.Field("message"), transaction.Content)
	}
}

func writeItems(out *strings.Builder, items []models.TodoItem, indent string) {
	for _, item := range items {
		mark := "○"
		if item.Done {
			mark = "✓"
		}
		fmt.Fprintf(out, "%s%s %s\n", indent, mark, item.Text)
	}
}

// segments is the extraction trace, on stderr-free stdout for now: the CLI has no
// UI to keep it out of the way of.
func segments(trace []models.SegmentScore) string {
	if len(trace) == 0 {
		return ""
	}
	var out strings.Builder
	fmt.Fprintln(&out, messages.SectionExtraction())
	for _, segment := range trace {
		mark := "✗"
		if segment.Included {
			mark = "✓"
		}
		fmt.Fprintf(&out, "  %s %.2f %s\n", segment.Text, segment.Score, mark)
	}
	return out.String()
}