package notes

import (
	"testing"
	"time"

	"msg-classifier/internal/models"
	"msg-classifier/internal/ptbr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoteServiceHandleRequireRoutesToGet(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeNote)
	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	seedNote(t, service, &models.Note{Type: models.NoteTypeReminder, Content: "pagar a conta", Date: &date})

	classification := &models.Classification{Kind: models.KindFinding{Choice: "require"}}
	outcome, err := service.Handle(&models.ReceiveMessageRequest{Message: "o que tenho para 10/05/2026?"}, classification)
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteFound, outcome.Action)
	assert.Equal(t, classification, outcome.Classification)
}

func seedNote(t *testing.T, service *NotesService, note *models.Note, items ...models.TodoItem) {
	t.Helper()
	require.NoError(t, service.repo.Create(note, items))
}

func TestNoteServiceGetByDate(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeNote)
	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	seedNote(t, service, &models.Note{Type: models.NoteTypeReminder, Content: "pagar a conta", Date: &date})
	seedNote(t, service, &models.Note{Type: models.NoteTypeNote, Content: "ideia solta"})

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "o que tenho para 10/05/2026?"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteFound, outcome.Action)
	require.Len(t, outcome.Notes, 1)
	assert.Equal(t, "pagar a conta", outcome.Notes[0].Content)
	assert.Equal(t, "10/05/2026", outcome.SearchTerm)
}

func TestNoteServiceGetByYesterday(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeNote)
	date, ok := ptbr.ParseDate("ontem", time.Now())
	require.True(t, ok)
	seedNote(t, service, &models.Note{Type: models.NoteTypeNote, Content: "anotação de ontem", Date: &date})

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "o que eu anotei ontem?"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteFound, outcome.Action)
	assert.Len(t, outcome.Notes, 1)
}

func TestNoteServiceGetUnfinished(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeTodo)
	seedNote(t, service, &models.Note{Type: models.NoteTypeTodo, Content: "lista a"},
		models.TodoItem{Text: "feito", Done: true, Position: 0},
		models.TodoItem{Text: "pendente", Position: 1},
	)
	seedNote(t, service, &models.Note{Type: models.NoteTypeTodo, Content: "lista b"},
		models.TodoItem{Text: "pronto", Done: true, Position: 0},
	)

	for _, message := range []string{"o que falta?", "quais tarefas estão pendentes?", "o que eu ainda não fiz?"} {
		outcome, err := service.Get(&models.ReceiveMessageRequest{Message: message}, &models.Classification{})
		require.NoError(t, err, message)
		assert.Equal(t, models.ActionNoteFound, outcome.Action, message)
		require.Len(t, outcome.Notes, 1, message)
		assert.Equal(t, "lista a", outcome.Notes[0].Content, message)
		assert.Equal(t, "tarefas pendentes", outcome.SearchTerm, message)
		require.Len(t, outcome.Notes[0].Items, 2, "the items must come with the note")
	}
}

func TestNoteServiceGetByTerm(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeNote)
	seedNote(t, service, &models.Note{Type: models.NoteTypeNote, Content: "renomear o projeto atlas"})
	seedNote(t, service, &models.Note{Type: models.NoteTypeNote, Content: "comprar pão"})

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "me mostra minhas anotações sobre o projeto"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteFound, outcome.Action)
	require.Len(t, outcome.Notes, 1)
	assert.Equal(t, "renomear o projeto atlas", outcome.Notes[0].Content)
	assert.Equal(t, "projeto", outcome.SearchTerm)
}

func TestNoteServiceGetByTermFallsBackToLastWord(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeTodo)
	seedNote(t, service, &models.Note{Type: models.NoteTypeTodo, Content: "comprar pão, leite e ovos"},
		models.TodoItem{Text: "comprar pão", Position: 0},
		models.TodoItem{Text: "leite e ovos", Position: 1},
	)

	// The normalized message keeps "sao" (not a stopword), so the first term is
	// the two-word phrase "sao ovos", which misses; the retry with the last word
	// "ovos" is what matches. The content assertion pins that the retry branch
	// ran — without it the outcome would be ActionNoteNotFound.
	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "quais são minhas tarefas sobre ovos?"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteFound, outcome.Action)
	require.Len(t, outcome.Notes, 1)
	assert.Equal(t, "comprar pão, leite e ovos", outcome.Notes[0].Content)
	assert.Equal(t, "ovos", outcome.SearchTerm, "the outcome reports the term that actually matched")
}

func TestNoteServiceGetSingleWordTermDoesNotRetry(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeNote)
	seedNote(t, service, &models.Note{Type: models.NoteTypeNote, Content: "comprar pão"})

	// The term collapses to the single word "projeto" (lastWord == term), so a
	// miss must NOT retry: the outcome is not-found with the original term.
	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "me mostra minhas anotações sobre projeto"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteNotFound, outcome.Action)
	assert.Equal(t, "projeto", outcome.SearchTerm)
	assert.Empty(t, outcome.Notes)
}

func TestNoteServiceGetByTermStripsPunctuation(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeNote)
	seedNote(t, service, &models.Note{Type: models.NoteTypeNote, Content: "renomear o projeto atlas"})

	// The raw question keeps "?" and "anotações?" after normalization; the LIKE
	// term must be the clean word "projeto", not "anotacoes projeto".
	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "me mostra minhas anotações sobre o projeto?"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteFound, outcome.Action)
	require.Len(t, outcome.Notes, 1)
	assert.Equal(t, "projeto", outcome.SearchTerm)
}

func TestNoteServiceGetNotFound(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeNote)
	seedNote(t, service, &models.Note{Type: models.NoteTypeNote, Content: "comprar pão"})

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "me mostra minhas anotações sobre o projeto"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteNotFound, outcome.Action)
	assert.Equal(t, "projeto", outcome.SearchTerm)
	assert.Empty(t, outcome.Notes)
}

func TestNoteServiceGetDateNotFound(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeNote)

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "o que tenho para 10/05/2026?"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteNotFound, outcome.Action)
	assert.Equal(t, "10/05/2026", outcome.SearchTerm)
}

func TestNoteServiceGetNoData(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeNote)

	outcome, err := service.Get(&models.ReceiveMessageRequest{Message: "me mostra as minhas"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteNoData, outcome.Action)
	assert.Empty(t, outcome.Notes)
}

func TestNoteServiceGetDBFailure(t *testing.T) {
	service, db := noteService(t, models.NoteTypeNote)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	_, err = service.Get(&models.ReceiveMessageRequest{Message: "o que falta?"}, &models.Classification{})
	assert.ErrorContains(t, err, "failed to search notes")
}

func TestSearchTerm(t *testing.T) {
	tests := []struct {
		name       string
		normalized string
		want       string
	}{
		{"strips question words", "me mostra minhas anotacoes sobre o projeto", "projeto"},
		{"keeps the head noun", "quais tarefas sobre ovos", "ovos"},
		{"all stopwords", "me mostra as minhas", ""},
		{"empty", "", ""},
		{"no stopwords", "projeto", "projeto"},
		{"strips trailing punctuation", "me mostra minhas anotacoes sobre o projeto?", "projeto"},
		{"punctuation-only word dropped", "projeto?!", "projeto"},
		{"punctuation leaves empty term", "me mostra as minhas?", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, searchTerm(tt.normalized))
		})
	}
}

func TestLastWord(t *testing.T) {
	assert.Equal(t, "projeto", lastWord("sobre o projeto"))
	assert.Equal(t, "projeto", lastWord("projeto"))
	assert.Equal(t, "", lastWord("   "))
}

func TestHasUnfinishedMarker(t *testing.T) {
	for _, marker := range []string{"falta", "faltam", "pendente", "pendentes", "nao fiz", "ainda nao"} {
		assert.True(t, hasUnfinishedMarker("o que "+marker+"?"), marker)
	}
	assert.False(t, hasUnfinishedMarker("me mostra minhas anotacoes"))
}