package notes

import (
	"testing"

	"msg-classifier/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoteServiceAddDuplicateThenNew(t *testing.T) {
	service, db := noteService(t, models.NoteTypeNote)
	msg := &models.ReceiveMessageRequest{Message: "anota que o nome do projeto é atlas"}

	first, err := service.Add(msg, &models.Classification{})
	require.NoError(t, err)
	require.Equal(t, models.ActionNoteAdd, first.Action)

	dup, err := service.Add(msg, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteDuplicate, dup.Action)
	require.Len(t, dup.Notes, 1)
	assert.Equal(t, first.Notes[0].ID, dup.Notes[0].ID, "the duplicate outcome carries the existing row")
	assert.Equal(t, "anota que o nome do projeto é atlas", dup.Notes[0].Content)

	var count int64
	require.NoError(t, db.Model(&models.Note{}).Count(&count).Error)
	assert.Equal(t, int64(1), count, "a duplicate hit persists nothing")

	again, err := service.Add(&models.ReceiveMessageRequest{Message: msg.Message, DupAction: "new"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteAdd, again.Action)
	require.NoError(t, db.Model(&models.Note{}).Count(&count).Error)
	assert.Equal(t, int64(2), count, "dup_action=new inserts anyway")
}

func TestNoteServiceAddDuplicateReminderKeyIncludesDate(t *testing.T) {
	service, db := noteService(t, models.NoteTypeReminder)
	msg := &models.ReceiveMessageRequest{Message: "me lembra de pagar a conta dia 10"}

	first, err := service.Add(msg, &models.Classification{})
	require.NoError(t, err)
	require.Equal(t, models.ActionNoteAdd, first.Action)

	dup, err := service.Add(msg, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteDuplicate, dup.Action)

	var count int64
	require.NoError(t, db.Model(&models.Note{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestNoteServiceAddDuplicateUpdateKeepsID(t *testing.T) {
	service, db := noteService(t, models.NoteTypeNote)
	msg := &models.ReceiveMessageRequest{Message: "anota que o nome do projeto é atlas"}

	first, err := service.Add(msg, &models.Classification{})
	require.NoError(t, err)

	updated, err := service.Add(&models.ReceiveMessageRequest{Message: msg.Message, DupAction: "update"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteAdd, updated.Action)
	require.Len(t, updated.Notes, 1)
	assert.Equal(t, first.Notes[0].ID, updated.Notes[0].ID, "update keeps the existing row")

	var count int64
	require.NoError(t, db.Model(&models.Note{}).Count(&count).Error)
	assert.Equal(t, int64(1), count, "update must not insert a second row")
}

func TestNoteServiceAddUnknownDupActionChecksNormally(t *testing.T) {
	service, _ := noteService(t, models.NoteTypeNote)
	msg := &models.ReceiveMessageRequest{Message: "anota que o nome do projeto é atlas"}

	_, err := service.Add(msg, &models.Classification{})
	require.NoError(t, err)

	dup, err := service.Add(&models.ReceiveMessageRequest{Message: msg.Message, DupAction: "banana"}, &models.Classification{})
	require.NoError(t, err)
	assert.Equal(t, models.ActionNoteDuplicate, dup.Action, "an unknown dup_action is treated as empty")
}
