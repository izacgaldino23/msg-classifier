package repository

import (
	"strings"
	"time"

	"msg-classifier/internal/models"

	"gorm.io/gorm"
)

// NotesRepository owns all gorm queries for the Note and TodoItem entities.
// It reuses the package's ErrNotFound sentinel (declared in contact.go).
type NotesRepository struct {
	db *gorm.DB
}

func NewNotesRepository(db *gorm.DB) *NotesRepository {
	return &NotesRepository{db: db}
}

// Create persists a note and its to-do items atomically; the note receives the
// generated id and the items are attached to it.
func (r *NotesRepository) Create(note *models.Note, items []models.TodoItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(note).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for i := range items {
			items[i].NoteID = note.ID
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		note.Items = items
		return nil
	})
}

// FindByDate returns the notes scheduled for the given date, or ErrNotFound.
func (r *NotesRepository) FindByDate(date time.Time) ([]*models.Note, error) {
	return r.find(func(db *gorm.DB) *gorm.DB {
		return db.Where("date = ?", date)
	})
}

// FindUnfinished returns the notes that still have an unfinished item, or ErrNotFound.
func (r *NotesRepository) FindUnfinished() ([]*models.Note, error) {
	return r.find(func(db *gorm.DB) *gorm.DB {
		pending := r.db.Model(&models.TodoItem{}).Select("note_id").Where("done = ?", false)
		return db.Where("id IN (?)", pending)
	})
}

// FindByTerm returns the notes whose content contains the term (case-insensitive),
// or ErrNotFound.
func (r *NotesRepository) FindByTerm(term string) ([]*models.Note, error) {
	return r.find(func(db *gorm.DB) *gorm.DB {
		return db.Where("LOWER(content) LIKE ?", "%"+strings.ToLower(term)+"%")
	})
}

// find runs a scoped query with the to-do items preloaded (ordered by position)
// and maps an empty result to ErrNotFound.
func (r *NotesRepository) find(scope func(*gorm.DB) *gorm.DB) ([]*models.Note, error) {
	var notes []*models.Note
	err := scope(r.db).
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("position") }).
		Order("id").
		Find(&notes).Error
	if err != nil {
		return nil, err
	}
	if len(notes) == 0 {
		return nil, ErrNotFound
	}
	return notes, nil
}

// List returns every note, newest first, with the to-do items preloaded in
// position order. An empty table is a valid state, so it never yields ErrNotFound.
func (r *NotesRepository) List() ([]*models.Note, error) {
	return r.listBy(func(db *gorm.DB) *gorm.DB { return db })
}

// ListByType returns the notes of a single sub-type, newest first.
func (r *NotesRepository) ListByType(noteType string) ([]*models.Note, error) {
	return r.listBy(func(db *gorm.DB) *gorm.DB { return db.Where("type = ?", noteType) })
}

// listBy runs a browse query with the items preloaded. Unlike find it does not
// map an empty result to ErrNotFound, which is correct for a search but wrong
// for a browse screen.
func (r *NotesRepository) listBy(scope func(*gorm.DB) *gorm.DB) ([]*models.Note, error) {
	var notes []*models.Note
	err := scope(r.db).
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("position") }).
		Order("id DESC").
		Find(&notes).Error
	if err != nil {
		return nil, err
	}
	return notes, nil
}

// FindByID returns the note with the given id (items preloaded), or ErrNotFound.
func (r *NotesRepository) FindByID(id uint) (*models.Note, error) {
	var note models.Note
	err := r.db.
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("position") }).
		First(&note, id).Error
	if err != nil {
		return nil, err
	}
	return &note, nil
}

// Save updates the note fields and replaces its to-do items atomically: the
// existing items are deleted and the new ones inserted with fresh ids. The
// Items association is omitted from the note write so a previously preloaded
// slice is never re-saved.
func (r *NotesRepository) Save(note *models.Note, items []models.TodoItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Items").Save(note).Error; err != nil {
			return err
		}
		if err := tx.Where("note_id = ?", note.ID).Delete(&models.TodoItem{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			note.Items = nil
			return nil
		}
		for i := range items {
			items[i].NoteID = note.ID
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		note.Items = items
		return nil
	})
}

// DeleteByIDs removes the given notes and their to-do items atomically. The
// items are deleted explicitly instead of leaning on the FK cascade, which would
// silently depend on the SQLite foreign_keys pragma being enabled.
func (r *NotesRepository) DeleteByIDs(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("note_id IN ?", ids).Delete(&models.TodoItem{}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", ids).Delete(&models.Note{}).Error
	})
}