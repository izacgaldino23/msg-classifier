package repository

import (
	"testing"
	"time"

	"msg-classifier/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newNoteTestDB mirrors the package newTestDB but migrates the notes schema.
func newNoteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "gorm.Open()")
	sqlDB, err := db.DB()
	require.NoError(t, err, "db.DB()")
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&models.Note{}, &models.TodoItem{}), "AutoMigrate()")
	return db
}

func datePtr(t time.Time) *time.Time { return &t }

func fixedDate() time.Time {
	return time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
}

func TestNotesRepositoryCreateWithoutItems(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)

	note := &models.Note{Type: models.NoteTypeNote, Content: "trocar o nome do projeto"}
	require.NoError(t, repo.Create(note, nil))
	assert.NotZero(t, note.ID)

	var count int64
	require.NoError(t, db.Model(&models.Note{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestNotesRepositoryCreateWithItems(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)

	items := []models.TodoItem{
		{Text: "comprar pão", Position: 0},
		{Text: "comprar leite", Position: 1},
	}
	note := &models.Note{Type: models.NoteTypeTodo, Content: "comprar pão e leite"}
	require.NoError(t, repo.Create(note, items))

	assert.NotZero(t, note.ID)
	require.Len(t, note.Items, 2)
	assert.Equal(t, note.ID, note.Items[0].NoteID)
	assert.Equal(t, "comprar leite", note.Items[1].Text)

	var count int64
	require.NoError(t, db.Model(&models.TodoItem{}).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

func TestNotesRepositoryCreateRollsBackOnItemFailure(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)

	// Dropping the items table makes the item insert fail inside the transaction.
	// The pool must stay open, otherwise the note insert fails first and nothing
	// is ever rolled back.
	require.NoError(t, db.Migrator().DropTable(&models.TodoItem{}))

	note := &models.Note{Type: models.NoteTypeTodo, Content: "comprar pão"}
	err := repo.Create(note, []models.TodoItem{{Text: "comprar pão", Position: 0}})
	require.Error(t, err)

	var count int64
	require.NoError(t, db.Model(&models.Note{}).Count(&count).Error)
	assert.Equal(t, int64(0), count, "the note insert must be rolled back")
}

func TestNotesRepositoryFindByDate(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)

	date := fixedDate()
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeReminder, Content: "pagar a conta", Date: &date}, nil))
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeNote, Content: "ideia solta"}, nil))

	notes, err := repo.FindByDate(date)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "pagar a conta", notes[0].Content)

	_, err = repo.FindByDate(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestNotesRepositoryFindUnfinished(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)

	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeTodo, Content: "lista a"}, []models.TodoItem{
		{Text: "feito", Done: true, Position: 0},
		{Text: "pendente", Position: 1},
	}))
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeTodo, Content: "lista b"}, []models.TodoItem{
		{Text: "tudo pronto", Done: true, Position: 0},
	}))
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeNote, Content: "sem itens"}, nil))

	notes, err := repo.FindUnfinished()
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "lista a", notes[0].Content)
	require.Len(t, notes[0].Items, 2)
	assert.Equal(t, "feito", notes[0].Items[0].Text, "items must be preloaded in position order")
	assert.Equal(t, "pendente", notes[0].Items[1].Text)
}

func TestNotesRepositoryFindUnfinishedEmpty(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)

	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeTodo, Content: "tudo pronto"}, []models.TodoItem{
		{Text: "feito", Done: true, Position: 0},
	}))

	_, err := repo.FindUnfinished()
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestNotesRepositoryFindByTerm(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)

	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeNote, Content: "renomear o Projeto Atlas"}, nil))
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeNote, Content: "comprar pão"}, nil))

	notes, err := repo.FindByTerm("projeto")
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "renomear o Projeto Atlas", notes[0].Content)

	_, err = repo.FindByTerm("inexistente")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestNotesRepositoryFindByTermLoadsItems(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)

	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeTodo, Content: "comprar pão"}, []models.TodoItem{
		{Text: "comprar pão", Position: 0},
	}))

	notes, err := repo.FindByTerm("comprar")
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Len(t, notes[0].Items, 1)
}

func TestNotesRepositoryClosedDB(t *testing.T) {
	db := newNoteTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	repo := NewNotesRepository(db)

	assert.Error(t, repo.Create(&models.Note{Type: models.NoteTypeNote, Content: "x"}, nil))
	_, err = repo.FindByTerm("x")
	assert.Error(t, err)
	_, err = repo.FindUnfinished()
	assert.Error(t, err)
	_, err = repo.FindByDate(fixedDate())
	assert.Error(t, err)
}