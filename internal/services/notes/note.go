package notes

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"msg-classifier/internal/models"
	"msg-classifier/internal/ptbr"
	"msg-classifier/internal/repository"
)

// NotesService handles the notes category use cases (notes, reminders, to-do lists).
type NotesService struct {
	extractor *NoteExtractor
	repo      *repository.NotesRepository
}

func NewService(extractor *NoteExtractor, repo *repository.NotesRepository) *NotesService {
	return &NotesService{extractor: extractor, repo: repo}
}

// Handle routes notes messages to the add or the get use case.
func (s *NotesService) Handle(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	if classification.Kind.Choice == "require" {
		return s.Get(request, classification)
	}
	return s.Add(request, classification)
}

// Add asks Jev for the sub-type and persists the note: a plain note keeps only the
// content, a reminder requires a date (time optional) and a to-do list is split
// into items. A missing reminder date or an unknown sub-type is a no-data outcome.
func (s *NotesService) Add(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	noteType, err := s.extractor.ExtractType(request)
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(request.Message)
	if content == "" {
		return noData(classification), nil
	}

	note := &models.Note{Type: strings.ToLower(strings.TrimSpace(noteType)), Content: content}
	items := make([]models.TodoItem, 0)

	switch note.Type {
	case models.NoteTypeNote:
	case models.NoteTypeReminder:
		now := time.Now()
		date, hasDate := ptbr.ParseDate(content, now)
		if !hasDate {
			return noData(classification), nil
		}
		note.Date = &date
		if clock, hasClock := ptbr.ParseTime(content); hasClock {
			note.Time = &clock
		}
	case models.NoteTypeTodo:
		for i, text := range SplitTodoItems(content) {
			items = append(items, models.TodoItem{Text: text, Position: i})
		}
		if len(items) == 0 {
			return noData(classification), nil
		}
	default:
		return noData(classification), nil
	}

	var existing *models.Note
	if request.DupAction != "new" {
		existing, err = s.repo.FindDuplicate(note.Type, note.Content, note.Date)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("failed to check duplicate note: %w", err)
		}
	}
	if existing != nil {
		if request.DupAction == "update" {
			// The key already pins type, content and date; only a time carried by
			// the pending payload can change anything.
			if note.Time != nil && *note.Time != "" {
				existing.Time = note.Time
			}
			if err := s.repo.Save(existing, items); err != nil {
				return nil, fmt.Errorf("failed to persist note: %w", err)
			}
			return &models.UseCaseOutcome{
				Classification: classification,
				Action:         models.ActionNoteAdd,
				Notes:          []*models.Note{existing},
			}, nil
		}
		return &models.UseCaseOutcome{
			Classification: classification,
			Action:         models.ActionNoteDuplicate,
			Notes:          []*models.Note{existing},
		}, nil
	}

	if err := s.repo.Create(note, items); err != nil {
		return nil, fmt.Errorf("failed to persist note: %w", err)
	}
	return &models.UseCaseOutcome{
		Classification: classification,
		Action:         models.ActionNoteAdd,
		Notes:          []*models.Note{note},
	}, nil
}

// noData is the shared "nothing extractable" outcome of the add path.
func noData(classification *models.Classification) *models.UseCaseOutcome {
	return &models.UseCaseOutcome{Classification: classification, Action: models.ActionNoteNoData}
}