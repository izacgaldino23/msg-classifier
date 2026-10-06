// Package api presents a use case outcome as JSON. It is one of three presenters
// over the same core (views renders HTML, cli prints text) and shares no interface
// with them — each is a switch on models.Action, because that is all a presenter is.
package api

import (
	"fmt"

	"msg-classifier/internal/models"
)

// MessageResponse is the JSON body of POST /api/v1/messages. `action` is the
// discriminant: the enum states a fact, so a client reads one field instead of
// inferring the outcome from the payload. Absent payload fields are omitted rather
// than sent as null — a contact outcome has no notes to speak of.
type MessageResponse struct {
	Action         string                 `json:"action"`
	Message        string                 `json:"message"`
	Classification *models.Classification `json:"classification,omitempty"`
	SearchTerm     string                 `json:"search_term,omitempty"`
	Missing        string                 `json:"missing,omitempty"`
	Total          float64                `json:"total,omitempty"`
	Contact        *models.Contact        `json:"contact,omitempty"`
	Notes          []*models.Note         `json:"notes,omitempty"`
	Transactions   []*models.Transaction  `json:"transactions,omitempty"`
	Segments       []models.SegmentScore  `json:"segments,omitempty"`
}

// Outcome presents a use case outcome as JSON.
func Outcome(outcome *models.UseCaseOutcome) MessageResponse {
	return MessageResponse{
		Action:         string(outcome.Action),
		Message:        summary(outcome),
		Classification: outcome.Classification,
		SearchTerm:     outcome.SearchTerm,
		Missing:        outcome.Missing,
		Total:          outcome.Total,
		Contact:        outcome.Contact,
		Notes:          outcome.Notes,
		Transactions:   outcome.Transactions,
		Segments:       outcome.Segments,
	}
}

// summary is the PT-BR one-liner for the action, worded like result.html so the
// two surfaces say the same thing. The contact guard is load-bearing: never
// panic in the request path, and a malformed outcome must still answer.
func summary(outcome *models.UseCaseOutcome) string {
	contact := outcome.Contact
	if contact == nil {
		contact = &models.Contact{}
	}
	switch outcome.Action {
	case models.ActionContactAdd:
		return fmt.Sprintf("Contato salvo · ID %d", contact.ID)
	case models.ActionContactFound:
		return "Contato encontrado"
	case models.ActionContactNotFound:
		return fmt.Sprintf("Nenhum contato encontrado para '%s'", outcome.SearchTerm)
	case models.ActionContactDuplicate:
		return fmt.Sprintf("Contato já existe: %s", contact.Name)
	case models.ActionContactNoData:
		return "Nenhum dado de contato encontrado na mensagem."
	case models.ActionNoteAdd:
		if note := firstNote(outcome.Notes); note != nil {
			return fmt.Sprintf("%s · ID %d", noteKind(note.Type), note.ID)
		}
		return "Nota salva"
	case models.ActionNoteFound:
		return fmt.Sprintf("%d nota(s) encontrada(s) para '%s'", len(outcome.Notes), outcome.SearchTerm)
	case models.ActionNoteNotFound:
		return fmt.Sprintf("Nenhuma nota encontrada para '%s'.", outcome.SearchTerm)
	case models.ActionNoteNoData:
		return "Não consegui extrair os dados da nota."
	case models.ActionTransactionAdd:
		if transaction := firstTransaction(outcome.Transactions); transaction != nil {
			return fmt.Sprintf("Transação salva · ID %d", transaction.ID)
		}
		return "Transação salva"
	case models.ActionTransactionFound:
		return fmt.Sprintf("%d transação(ões) para '%s'", len(outcome.Transactions), outcome.SearchTerm)
	case models.ActionTransactionNotFound:
		return fmt.Sprintf("Nenhuma transação encontrada para '%s'.", outcome.SearchTerm)
	case models.ActionTransactionNoData:
		return fmt.Sprintf("Não consegui classificar a transação: %s.", outcome.Missing)
	default:
		return ""
	}
}

// noteKind is the PT-BR saved-form label for a note sub-type, worded as
// result.html says it. The participle travels with the noun because the two
// genders disagree — "Lembrete salvo", but "Nota salva" and "Lista de tarefas
// salva"; a shared `"%s salvo"` would print the feminine ones wrong.
// views.Label owns the badge map, but importing views here would pull
// html/template into the JSON binary.
func noteKind(noteType string) string {
	switch noteType {
	case models.NoteTypeReminder:
		return "Lembrete salvo"
	case models.NoteTypeTodo:
		return "Lista de tarefas salva"
	default:
		return "Nota salva"
	}
}

func firstNote(notes []*models.Note) *models.Note {
	if len(notes) == 0 {
		return nil
	}
	return notes[0]
}

func firstTransaction(transactions []*models.Transaction) *models.Transaction {
	if len(transactions) == 0 {
		return nil
	}
	return transactions[0]
}
