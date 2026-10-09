// Package cli is the terminal presenter: a UseCaseOutcome becomes lines. It
// shares nothing with views or api but the outcome itself — no interface, no
// registry, just a switch on models.Action. The wording comes from
// internal/messages and the colors and boxes from style.go, so this file owns the
// shape of the block.
package cli

import (
	"fmt"
	"strings"

	"msg-classifier/internal/messages"
	"msg-classifier/internal/models"
	"msg-classifier/internal/ptbr"
)

// Render turns an outcome into the plain-text block, with no escapes: it is the entry
// point for a caller that has no terminal to write to. The REPL uses style.render.
func Render(outcome *models.UseCaseOutcome) string {
	return style{}.render(outcome)
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

// render is the whole block: a blank line, the echoed message, the classification
// line, the action body and the extraction trace. The echo of the message is what
// separates what the user typed from what the system answered.
func (s style) render(outcome *models.UseCaseOutcome) string {
	var out strings.Builder
	out.WriteString("\n")
	if outcome.Message != "" {
		fmt.Fprintln(&out, s.dim("▸ "+outcome.Message))
	}
	if c := outcome.Classification; c != nil {
		fmt.Fprintln(&out, s.dim(messages.Classification(c.Category.Choice, c.Kind.Choice, c.Category.Confidence, c.Kind.Confidence)))
	}
	out.WriteString(s.body(outcome))
	out.WriteString(s.segments(outcome.Segments))
	return out.String()
}

// body is the one switch every Action goes through. Wording mirrors result.html
// because both read the same locale.
func (s style) body(outcome *models.UseCaseOutcome) string {
	var out strings.Builder
	switch outcome.Action {
	case models.ActionContactAdd:
		contact := contactOrEmpty(outcome.Contact)
		out.WriteString(s.card(messages.ContactSaved(contact.ID), contactFields(contact, true)))
	case models.ActionContactFound:
		out.WriteString(s.card(messages.ContactFound(), contactFields(contactOrEmpty(outcome.Contact), true)))
	case models.ActionContactNotFound:
		fmt.Fprintln(&out, s.yellow(messages.ContactNotFound(outcome.SearchTerm)))
	case models.ActionContactDuplicate:
		contact := contactOrEmpty(outcome.Contact)
		out.WriteString(s.cardLine(messages.ContactDuplicate(contact.Name), contactParts(contact)))
	case models.ActionContactNoData:
		fmt.Fprintln(&out, s.yellow(messages.ContactNoData()))
	case models.ActionNoteAdd:
		for _, note := range outcome.Notes {
			out.WriteString(s.noteCard(messages.NoteSaved(note.Type, note.ID), note))
		}
	case models.ActionNoteFound:
		fmt.Fprintln(&out, messages.NoteFound(len(outcome.Notes), outcome.SearchTerm))
		for _, note := range outcome.Notes {
			out.WriteString(s.noteCard(noteTitle(note), note))
		}
	case models.ActionNoteNotFound:
		fmt.Fprintln(&out, s.yellow(messages.NoteNotFound(outcome.SearchTerm)))
	case models.ActionNoteNoData:
		fmt.Fprintln(&out, s.yellow(messages.NoteNoData()))
	case models.ActionNoteDuplicate:
		for _, note := range outcome.Notes {
			out.WriteString(s.noteCard(messages.NoteDuplicate(note.Content), note))
		}
	case models.ActionTransactionAdd:
		for _, transaction := range outcome.Transactions {
			out.WriteString(s.transactionCard(messages.TransactionSaved(transaction.ID), transaction))
		}
	case models.ActionTransactionFound:
		fmt.Fprintln(&out, messages.TransactionFound(len(outcome.Transactions), outcome.SearchTerm))
		if outcome.Total > 0 {
			fmt.Fprintf(&out, "  %s: %s\n", messages.Field("total"), ptbr.MoneyBRL(outcome.Total))
		}
		for _, transaction := range outcome.Transactions {
			out.WriteString(s.transactionCard(transactionTitle(transaction), transaction))
		}
	case models.ActionTransactionNotFound:
		fmt.Fprintln(&out, s.yellow(messages.TransactionNotFound(outcome.SearchTerm)))
	case models.ActionTransactionNoData:
		fmt.Fprintln(&out, s.yellow(messages.TransactionNoData(outcome.Missing)))
	case models.ActionTransactionDuplicate:
		for _, transaction := range outcome.Transactions {
			out.WriteString(s.transactionCard(messages.TransactionDuplicate(), transaction))
		}
	}
	return out.String()
}

func contactOrEmpty(contact *models.Contact) *models.Contact {
	if contact == nil {
		return &models.Contact{}
	}
	return contact
}

// contactFields are a contact card's rows, the name first for a record just saved or
// found; a duplicate carries its name in the title instead.
func contactFields(contact *models.Contact, withName bool) [][2]string {
	fields := make([][2]string, 0, 3)
	if withName && contact.Name != "" {
		fields = append(fields, [2]string{messages.Field("name"), contact.Name})
	}
	if contact.Phone != nil && *contact.Phone != "" {
		fields = append(fields, [2]string{messages.Field("phone"), *contact.Phone})
	}
	if contact.Email != nil && *contact.Email != "" {
		fields = append(fields, [2]string{messages.Field("email"), *contact.Email})
	}
	return fields
}

// contactParts is the duplicate one-liner: the existing record reads better on one
// line than as a field list, since the title already carries the name.
func contactParts(contact *models.Contact) []string {
	parts := make([]string, 0, 4)
	if contact.ID > 0 {
		parts = append(parts, fmt.Sprintf("%s %d", messages.Field("id"), contact.ID))
	}
	if contact.Name != "" {
		parts = append(parts, contact.Name)
	}
	if contact.Phone != nil && *contact.Phone != "" {
		parts = append(parts, *contact.Phone)
	}
	if contact.Email != nil && *contact.Email != "" {
		parts = append(parts, *contact.Email)
	}
	return parts
}

// noteCard is a note's card with its to-do items underneath it.
func (s style) noteCard(title string, note *models.Note) string {
	var out strings.Builder
	fields := [][2]string{{messages.Field("content"), note.Content}}
	if note.Date != nil {
		fields = append(fields, [2]string{messages.Field("date"), ptbr.DateBR(note.Date)})
	}
	if note.Time != nil && *note.Time != "" {
		fields = append(fields, [2]string{messages.Field("time"), *note.Time})
	}
	out.WriteString(s.card(title, fields))
	out.WriteString(s.items(note.Items))
	return out.String()
}

func (s style) transactionCard(title string, transaction *models.Transaction) string {
	fields := [][2]string{
		{messages.Field("type"), transaction.Type},
		{messages.Field("amount"), ptbr.MoneyBRL(transaction.Amount)},
		{messages.Field("date"), ptbr.DateBR(&transaction.Date)},
	}
	if transaction.Party != "" {
		fields = append(fields, [2]string{messages.Field("party"), transaction.Party})
	}
	if transaction.Content != "" {
		fields = append(fields, [2]string{messages.Field("message"), transaction.Content})
	}
	return s.card(title, fields)
}

// noteTitle names a note in a found list: "Lembrete · ID 3".
func noteTitle(note *models.Note) string {
	title := messages.Label(note.Type)
	if note.ID > 0 {
		title += fmt.Sprintf(" · %s %d", messages.Field("id"), note.ID)
	}
	return title
}

// transactionTitle names a transaction in a found list.
func transactionTitle(transaction *models.Transaction) string {
	if transaction.ID == 0 {
		return messages.Label(transaction.Type)
	}
	return fmt.Sprintf("%s · %s %d", messages.Label(transaction.Type), messages.Field("id"), transaction.ID)
}

// items prints a to-do list under its card: one line per item, the done mark green.
func (s style) items(items []models.TodoItem) string {
	var out strings.Builder
	for _, item := range items {
		mark := "○"
		if item.Done {
			mark = s.green("✓")
		}
		fmt.Fprintf(&out, "  %s %s\n", mark, item.Text)
	}
	return out.String()
}

// segments is the extraction trace, on stdout because the CLI has no second pane to
// keep it in: the header is dim and only the marks carry color.
func (s style) segments(trace []models.SegmentScore) string {
	if len(trace) == 0 {
		return ""
	}
	var out strings.Builder
	fmt.Fprintln(&out, s.dim(messages.SectionExtraction()))
	for _, segment := range trace {
		mark := s.red("✗")
		if segment.Included {
			mark = s.green("✓")
		}
		fmt.Fprintf(&out, "  %s %.2f %s\n", segment.Text, segment.Score, mark)
	}
	return out.String()
}
