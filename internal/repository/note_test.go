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

func TestNotesRepositoryListNewestFirstWithItems(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeNote, Content: "primeira"}, nil))
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeTodo, Content: "segunda"}, []models.TodoItem{
		{Text: "item a", Position: 1},
		{Text: "item b", Position: 0},
	}))

	notes, err := repo.List()
	require.NoError(t, err)
	require.Len(t, notes, 2)
	assert.Equal(t, "segunda", notes[0].Content, "newest first (id DESC)")
	require.Len(t, notes[0].Items, 2)
	assert.Equal(t, "item b", notes[0].Items[0].Text, "items preloaded in position order")
	assert.Equal(t, "item a", notes[0].Items[1].Text)
}

func TestNotesRepositoryListEmptyIsNotAnError(t *testing.T) {
	repo := NewNotesRepository(newNoteTestDB(t))

	notes, err := repo.List()
	require.NoError(t, err, "an empty table is a valid state, not a not-found")
	assert.Empty(t, notes)
}

func TestNotesRepositoryListByType(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeNote, Content: "nota"}, nil))
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeReminder, Content: "lembrete", Date: datePtr(fixedDate())}, nil))
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeReminder, Content: "outro lembrete"}, nil))

	reminders, err := repo.ListByType(models.NoteTypeReminder)
	require.NoError(t, err)
	require.Len(t, reminders, 2)
	assert.Equal(t, "outro lembrete", reminders[0].Content)

	notes, err := repo.ListByType(models.NoteTypeNote)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "nota", notes[0].Content)

	todos, err := repo.ListByType(models.NoteTypeTodo)
	require.NoError(t, err)
	assert.Empty(t, todos)
}

func TestNotesRepositoryFindByID(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeTodo, Content: "comprar"}, []models.TodoItem{
		{Text: "pão", Position: 0},
		{Text: "leite", Position: 1, Done: true},
	}))

	note, err := repo.FindByID(1)
	require.NoError(t, err)
	assert.Equal(t, "comprar", note.Content)
	require.Len(t, note.Items, 2)
	assert.Equal(t, "pão", note.Items[0].Text)
	assert.True(t, note.Items[1].Done)

	_, err = repo.FindByID(999)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestNotesRepositorySaveReplacesItems(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeTodo, Content: "antes"}, []models.TodoItem{
		{Text: "antigo 1", Position: 0},
		{Text: "antigo 2", Position: 1},
	}))

	note, err := repo.FindByID(1)
	require.NoError(t, err)
	note.Content = "depois"
	require.NoError(t, repo.Save(note, []models.TodoItem{{Text: "novo", Position: 0, Done: true}}))

	var items []models.TodoItem
	require.NoError(t, db.Where("note_id = ?", note.ID).Order("position").Find(&items).Error)
	require.Len(t, items, 1, "old items are deleted, not merged")
	assert.Equal(t, "novo", items[0].Text)
	assert.True(t, items[0].Done)
	assert.Equal(t, note.ID, items[0].NoteID)

	reloaded, err := repo.FindByID(note.ID)
	require.NoError(t, err)
	assert.Equal(t, "depois", reloaded.Content)
}

func TestNotesRepositorySaveWithoutItems(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeTodo, Content: "antes"}, []models.TodoItem{
		{Text: "antigo", Position: 0},
	}))

	note, err := repo.FindByID(1)
	require.NoError(t, err)
	require.NoError(t, repo.Save(note, nil))

	var count int64
	require.NoError(t, db.Model(&models.TodoItem{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestNotesRepositorySaveDoesNotResurrectLoadedItems(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeTodo, Content: "antes"}, []models.TodoItem{
		{Text: "antigo", Position: 0},
	}))

	// note.Items is populated by FindByID; Save must ignore the association and
	// write only the fields, otherwise the stale items are re-saved.
	note, err := repo.FindByID(1)
	require.NoError(t, err)
	require.Len(t, note.Items, 1)
	note.Content = "depois"
	require.NoError(t, repo.Save(note, nil))

	var count int64
	require.NoError(t, db.Model(&models.TodoItem{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestNotesRepositoryDeleteByIDsDropsItems(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeTodo, Content: "a"}, []models.TodoItem{{Text: "x", Position: 0}}))
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeNote, Content: "b"}, nil))
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeTodo, Content: "c"}, []models.TodoItem{{Text: "y", Position: 0}}))

	require.NoError(t, repo.DeleteByIDs([]uint{1, 3}))

	var notes int64
	require.NoError(t, db.Model(&models.Note{}).Count(&notes).Error)
	assert.Equal(t, int64(1), notes)

	var items int64
	require.NoError(t, db.Model(&models.TodoItem{}).Count(&items).Error)
	assert.Equal(t, int64(0), items, "todo items are deleted with their notes")
}

func TestNotesRepositoryDeleteByIDsEmptyIsANoOp(t *testing.T) {
	db := newNoteTestDB(t)
	repo := NewNotesRepository(db)
	require.NoError(t, repo.Create(&models.Note{Type: models.NoteTypeNote, Content: "a"}, nil))

	require.NoError(t, repo.DeleteByIDs(nil))

	var count int64
	require.NoError(t, db.Model(&models.Note{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}