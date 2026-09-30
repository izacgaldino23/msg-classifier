package services

import (
	"errors"
	"testing"
	"time"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newNoteService builds the service over an in-memory DB with the notes schema.
func newNoteService(t *testing.T, extractor *NoteExtractor) (*NotesService, *gorm.DB) {
	t.Helper()
	db := newTestDB(t)
	require.NoError(t, db.AutoMigrate(&models.Note{}, &models.TodoItem{}), "AutoMigrate(notes)")
	return NewNotesService(extractor, NewDateParser(), repository.NewNotesRepository(db)), db
}

func noteService(t *testing.T, noteType string) (*NotesService, *gorm.DB) {
	t.Helper()
	return newNoteService(t, NewNoteExtractor(&mockJevClient{resp: noteTypeResponse(noteType)}))
}

func TestNoteServiceAddNote(t *testing.T) {
	service, db := noteService(t, models.NoteTypeNote)

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "anota que o nome do projeto é atlas"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteAdd, outcome.Action)
	require.Len(t, outcome.Notes, 1)
	assert.Nil(t, outcome.Contact, "the notes flow never carries a contact")

	note := outcome.Notes[0]
	assert.NotZero(t, note.ID)
	assert.Equal(t, models.NoteTypeNote, note.Type)
	assert.Equal(t, "anota que o nome do projeto é atlas", note.Content)
	assert.Nil(t, note.Date)
	assert.Nil(t, note.Time)
	assert.Empty(t, note.Items)

	var count int64
	require.NoError(t, db.Model(&models.Note{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestNoteServiceAddReminderWithDayOfMonth(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeReminder)
	now := time.Now()

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "me lembra de pagar a conta dia 10"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteAdd, outcome.Action)
	require.Len(t, outcome.Notes, 1)

	note := outcome.Notes[0]
	assert.Equal(t, models.NoteTypeReminder, note.Type)
	require.NotNil(t, note.Date)
	assert.Equal(t, time.Date(now.Year(), now.Month(), 10, 0, 0, 0, 0, time.UTC), *note.Date)
	assert.Nil(t, note.Time, "the time is optional for a reminder")
	assert.Empty(t, note.Items)
}

func TestNoteServiceAddReminderWithFullDateAndTime(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeReminder)

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "consulta com o dentista 10/05/2026 às 14h30"}, &models.Classification{})
	require.NoError(t, err)
	require.Len(t, outcome.Notes, 1)

	note := outcome.Notes[0]
	require.NotNil(t, note.Date)
	assert.Equal(t, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), *note.Date)
	require.NotNil(t, note.Time)
	assert.Equal(t, "14:30", *note.Time)
}

func TestNoteServiceAddReminderWithoutDate(t *testing.T) {
	service, db := noteService(t, models.NoteTypeReminder)

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "me lembra de pagar a conta"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteNoData, outcome.Action)
	assert.Empty(t, outcome.Notes)

	var count int64
	require.NoError(t, db.Model(&models.Note{}).Count(&count).Error)
	assert.Zero(t, count, "a reminder without a date must not be persisted")
}

func TestNoteServiceAddTodoSplitsItems(t *testing.T) {
	service, db := noteService(t, models.NoteTypeTodo)

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "comprar pão, comprar leite e ovos"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteAdd, outcome.Action)
	require.Len(t, outcome.Notes, 1)

	note := outcome.Notes[0]
	assert.Equal(t, models.NoteTypeTodo, note.Type)
	assert.Equal(t, "comprar pão, comprar leite e ovos", note.Content)
	assert.Nil(t, note.Date)
	require.Len(t, note.Items, 2)
	assert.NotZero(t, note.Items[0].ID, "the item is persisted")
	assert.Equal(t, note.ID, note.Items[0].NoteID, "the item points back to the note")
	assert.Equal(t, "comprar pão", note.Items[0].Text)
	assert.Zero(t, note.Items[0].Position)
	assert.Equal(t, "comprar leite e ovos", note.Items[1].Text)
	assert.Equal(t, 1, note.Items[1].Position)

	var count int64
	require.NoError(t, db.Model(&models.TodoItem{}).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

// A leading "Label:" is not stripped from the first item (known limitation).
func TestNoteServiceAddTodoKeepsLeadingLabelOnFirstItem(t *testing.T) {
	service, db := noteService(t, models.NoteTypeTodo)

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "Tarefas: comprar pão, leite e ovos"}, &models.Classification{})
	require.NoError(t, err)
	require.Len(t, outcome.Notes, 1)

	note := outcome.Notes[0]
	require.Len(t, note.Items, 2)
	assert.Equal(t, "Tarefas: comprar pão", note.Items[0].Text, "the leading label is kept as-is - see docs/todos.md")
	assert.Equal(t, "leite e ovos", note.Items[1].Text)

	var count int64
	require.NoError(t, db.Model(&models.TodoItem{}).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

func TestNoteServiceAddTodoWithoutItems(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeTodo)

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "  ,.  "}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteNoData, outcome.Action)
}

func TestNoteServiceAddUnknownType(t *testing.T) {
	service, _ := noteService(t, "agenda")

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "algo qualquer"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteNoData, outcome.Action)
}

func TestNoteServiceAddNormalizesType(t *testing.T) {
	service, _ := noteService(t, "  TODO ")

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "comprar pão, leite"}, &models.Classification{})
	require.NoError(t, err)
	require.Len(t, outcome.Notes, 1)
	assert.Equal(t, models.NoteTypeTodo, outcome.Notes[0].Type)
}

func TestNoteServiceAddEmptyMessage(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeNote)

	outcome, err := service.Add(&models.ReceiveMessageRequest{Message: "   "}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteNoData, outcome.Action)
}

func TestNoteServiceAddJevFailure(t *testing.T) {
	service, _ := newNoteService(t, NewNoteExtractor(&mockJevClient{err: errors.New("boom")}))

	_, err := service.Add(&models.ReceiveMessageRequest{Message: "anota isso"}, &models.Classification{})
	assert.ErrorIs(t, err, ErrUpstream)
}

func TestNoteServiceAddDBFailure(t *testing.T) {
	service, db := noteService(t, models.NoteTypeNote)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	_, err = service.Add(&models.ReceiveMessageRequest{Message: "anota isso"}, &models.Classification{})
	assert.ErrorContains(t, err, "failed to persist note")
}