// Package api presents a use case outcome as JSON. It is one of three presenters
// over the same core (views renders HTML, cli prints text) and shares no interface
// with them — each is a switch on models.Action, because that is all a presenter is.
package api

import (
	"msg-classifier/internal/messages"
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
// surfaces say the same thing — the text lives in internal/messages, this switch only
// picks which message. The contact guard is load-bearing: never panic in the request
// path, and a malformed outcome must still answer.
func summary(outcome *models.UseCaseOutcome) string {
	contact := outcome.Contact
	if contact == nil {
		contact = &models.Contact{}
	}
	switch outcome.Action {
	case models.ActionContactAdd:
		return messages.ContactSaved(contact.ID)
	case models.ActionContactFound:
		return messages.ContactFound()
	case models.ActionContactNotFound:
		return messages.ContactNotFound(outcome.SearchTerm)
	case models.ActionContactDuplicate:
		return messages.ContactDuplicate(contact.Name)
	case models.ActionContactNoData:
		return messages.ContactNoData()
	case models.ActionNoteAdd:
		if note := firstNote(outcome.Notes); note != nil {
			return messages.NoteSaved(note.Type, note.ID)
		}
		return messages.NoteSavedNoID()
	case models.ActionNoteFound:
		return messages.NoteFound(len(outcome.Notes), outcome.SearchTerm)
	case models.ActionNoteNotFound:
		return messages.NoteNotFound(outcome.SearchTerm)
	case models.ActionNoteNoData:
		return messages.NoteNoData()
	case models.ActionNoteDuplicate:
		if note := firstNote(outcome.Notes); note != nil {
			return messages.NoteDuplicate(note.Content)
		}
		return messages.NoteDuplicateNoContent()
	case models.ActionTransactionAdd:
		if transaction := firstTransaction(outcome.Transactions); transaction != nil {
			return messages.TransactionSaved(transaction.ID)
		}
		return messages.TransactionSavedNoID()
	case models.ActionTransactionFound:
		return messages.TransactionFound(len(outcome.Transactions), outcome.SearchTerm)
	case models.ActionTransactionNotFound:
		return messages.TransactionNotFound(outcome.SearchTerm)
	case models.ActionTransactionNoData:
		return messages.TransactionNoData(outcome.Missing)
	case models.ActionTransactionDuplicate:
		return messages.TransactionDuplicate()
	default:
		return ""
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
