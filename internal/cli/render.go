// Package cli is the terminal presenter: a UseCaseOutcome becomes lines. It
// shares nothing with views or api but the outcome itself — no interface, no
// registry, just a switch on models.Action.
package cli

import (
	"fmt"
	"strings"

	"msg-classifier/internal/models"
	"msg-classifier/internal/ptbr"
)

// Render turns an outcome into the block of lines the terminal prints.
func Render(outcome *models.UseCaseOutcome) string {
	var out strings.Builder
	if c := outcome.Classification; c != nil {
		fmt.Fprintf(&out, "[%s/%s]  categoria %.0f%% · intenção %.0f%%\n",
			c.Category.Choice, c.Kind.Choice, c.Category.Confidence*100, c.Kind.Confidence*100)
	}
	out.WriteString(body(outcome))
	out.WriteString(segments(outcome.Segments))
	return out.String()
}

// body is the one switch every Action goes through. Wording mirrors result.html
// so the three surfaces say the same thing.
func body(outcome *models.UseCaseOutcome) string {
	var out strings.Builder
	switch outcome.Action {
	case models.ActionContactAdd:
		contact := outcome.Contact
		if contact == nil {
			contact = &models.Contact{}
		}
		fmt.Fprintf(&out, "contato salvo · id %d\n", contact.ID)
		writeContact(&out, contact, true)
	case models.ActionContactFound:
		fmt.Fprintln(&out, "contato encontrado")
		writeContact(&out, outcome.Contact, true)
	case models.ActionContactNotFound:
		fmt.Fprintf(&out, "nenhum contato encontrado para '%s'\n", outcome.SearchTerm)
	case models.ActionContactDuplicate:
		contact := outcome.Contact
		if contact == nil {
			contact = &models.Contact{}
		}
		fmt.Fprintf(&out, "contato já existe: %s\n", contact.Name)
		writeContact(&out, contact, false)
	case models.ActionContactNoData:
		fmt.Fprintln(&out, "nenhum dado de contato encontrado na mensagem")
	case models.ActionNoteAdd:
		for _, note := range outcome.Notes {
			fmt.Fprintf(&out, "%s salvo · id %d\n", noteKind(note.Type), note.ID)
			fmt.Fprintf(&out, "  conteúdo: %s\n", note.Content)
			if note.Date != nil {
				fmt.Fprintf(&out, "  data: %s\n", ptbr.DateBR(note.Date))
			}
			if note.Time != nil && *note.Time != "" {
				fmt.Fprintf(&out, "  horário: %s\n", *note.Time)
			}
			writeItems(&out, note.Items)
		}
	case models.ActionNoteFound:
		fmt.Fprintf(&out, "%d nota(s) encontrada(s) para '%s'\n", len(outcome.Notes), outcome.SearchTerm)
		for _, note := range outcome.Notes {
			fmt.Fprintf(&out, "  [%s] %s\n", note.Type, note.Content)
			if note.Date != nil {
				fmt.Fprintf(&out, "    data: %s\n", ptbr.DateBR(note.Date))
			}
			if note.Time != nil && *note.Time != "" {
				fmt.Fprintf(&out, "    horário: %s\n", *note.Time)
			}
			writeItems(&out, note.Items)
		}
	case models.ActionNoteNotFound:
		fmt.Fprintf(&out, "nenhuma nota encontrada para '%s'\n", outcome.SearchTerm)
	case models.ActionNoteNoData:
		fmt.Fprintln(&out, "não consegui extrair os dados da nota (lembretes precisam de uma data)")
	case models.ActionTransactionAdd:
		for _, transaction := range outcome.Transactions {
			fmt.Fprintf(&out, "transação salva · id %d\n", transaction.ID)
			writeTransaction(&out, transaction)
		}
	case models.ActionTransactionFound:
		fmt.Fprintf(&out, "%d transação(ões) para '%s'\n", len(outcome.Transactions), outcome.SearchTerm)
		if outcome.Total > 0 {
			fmt.Fprintf(&out, "  total: %s\n", ptbr.MoneyBRL(outcome.Total))
		}
		for _, transaction := range outcome.Transactions {
			fmt.Fprintf(&out, "  [%s] %s — %s\n", transaction.Type, ptbr.MoneyBRL(transaction.Amount), transaction.Party)
			fmt.Fprintf(&out, "    data: %s\n", ptbr.DateBR(&transaction.Date))
		}
	case models.ActionTransactionNotFound:
		fmt.Fprintf(&out, "nenhuma transação encontrada para '%s'\n", outcome.SearchTerm)
	case models.ActionTransactionNoData:
		fmt.Fprintf(&out, "não consegui classificar a transação: %s\n", outcome.Missing)
	}
	return out.String()
}

func writeContact(out *strings.Builder, contact *models.Contact, withName bool) {
	if contact == nil {
		return
	}
	if withName && contact.Name != "" {
		fmt.Fprintf(out, "  nome: %s\n", contact.Name)
	}
	if contact.Phone != nil && *contact.Phone != "" {
		fmt.Fprintf(out, "  telefone: %s\n", *contact.Phone)
	}
	if contact.Email != nil && *contact.Email != "" {
		fmt.Fprintf(out, "  email: %s\n", *contact.Email)
	}
}

func writeTransaction(out *strings.Builder, transaction *models.Transaction) {
	fmt.Fprintf(out, "  tipo: %s\n", transaction.Type)
	fmt.Fprintf(out, "  valor: %s\n", ptbr.MoneyBRL(transaction.Amount))
	fmt.Fprintf(out, "  data: %s\n", ptbr.DateBR(&transaction.Date))
	if transaction.Party != "" {
		fmt.Fprintf(out, "  estabelecimento: %s\n", transaction.Party)
	}
	if transaction.Content != "" {
		fmt.Fprintf(out, "  mensagem: %s\n", transaction.Content)
	}
}

func writeItems(out *strings.Builder, items []models.TodoItem) {
	for _, item := range items {
		mark := "○"
		if item.Done {
			mark = "✓"
		}
		fmt.Fprintf(out, "  %s %s\n", mark, item.Text)
	}
}

// segments is the extraction trace, on stderr-free stdout for now: the CLI has no
// UI to keep it out of the way of.
func segments(trace []models.SegmentScore) string {
	if len(trace) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("extração:\n")
	for _, segment := range trace {
		mark := "✗"
		if segment.Included {
			mark = "✓"
		}
		fmt.Fprintf(&out, "  %s %.2f %s\n", segment.Text, segment.Score, mark)
	}
	return out.String()
}

// noteKind is the PT-BR noun for a note sub-type.
func noteKind(noteType string) string {
	switch noteType {
	case models.NoteTypeReminder:
		return "lembrete"
	case models.NoteTypeTodo:
		return "lista de tarefas"
	default:
		return "nota"
	}
}