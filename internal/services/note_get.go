package services

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"
)

// unfinishedMarkers select the "what is left to do" filter in the require flow.
// The list is compared against normalizeName output, so it is accent-free.
var unfinishedMarkers = []string{"falta", "faltam", "pendente", "pendentes", "nao fiz", "ainda nao"}

// noteStopwords are dropped from the message to build the LIKE search term.
var noteStopwords = map[string]bool{
	"a": true, "ai": true, "as": true, "com": true, "da": true, "das": true, "de": true, "do": true, "dos": true,
	"e": true, "eu": true, "mostra": true, "mostrar": true, "minha": true, "minhas": true, "meu": true, "meus": true,
	"o": true, "os": true, "anotacao": true, "anotacoes": true, "nota": true, "notas": true,
	"lembrete": true, "lembretes": true, "tarefa": true, "tarefas": true, "lista": true,
	"para": true, "por": true, "que": true, "qual": true, "quais": true, "quero": true, "sobre": true, "ver": true,
	"tem": true, "tinha": true, "tenho": true, "mostre": true, "me": true, "quando": true, "onde": true,
}

// Get searches stored notes (require flow). Filter priority: a date in the
// message (including "ontem"), then unfinished to-do items, then a content term.
func (s *NotesService) Get(request *models.ReceiveMessageRequest, classification *models.Classification) (*models.UseCaseOutcome, error) {
	normalized := normalizeName(request.Message)

	if date, ok := s.parser.ParseDate(request.Message, time.Now()); ok {
		notes, err := s.repo.FindByDate(date)
		return s.searchResult(classification, notes, err, date.Format(dateLayout))
	}

	if hasUnfinishedMarker(normalized) {
		notes, err := s.repo.FindUnfinished()
		return s.searchResult(classification, notes, err, "tarefas pendentes")
	}

	term := searchTerm(normalized)
	if term == "" {
		return noData(classification), nil
	}

	notes, err := s.repo.FindByTerm(term)
	if errors.Is(err, repository.ErrNotFound) {
		// The joined phrase misses (e.g. "comprar pao ovos"); retry with the last word.
		if last := lastWord(term); last != term {
			notes, err = s.repo.FindByTerm(last)
			term = last
		}
	}
	return s.searchResult(classification, notes, err, term)
}

// searchResult maps a repository lookup to a found/not-found outcome.
// repository.ErrNotFound is a "not found" outcome, not an error.
func (s *NotesService) searchResult(classification *models.Classification, notes []*models.Note, err error, searchTerm string) (*models.UseCaseOutcome, error) {
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("failed to search notes: %w", err)
	}
	if len(notes) > 0 {
		return &models.UseCaseOutcome{
			Classification: classification,
			Action:         models.ActionNoteFound,
			Notes:          notes,
			SearchTerm:     searchTerm,
		}, nil
	}
	return &models.UseCaseOutcome{
		Classification: classification,
		Action:         models.ActionNoteNotFound,
		SearchTerm:     searchTerm,
	}, nil
}

// hasUnfinishedMarker reports whether the normalized message asks for what is pending.
func hasUnfinishedMarker(normalized string) bool {
	for _, marker := range unfinishedMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

// searchTerm drops the punctuation and the PT-BR stopwords from the normalized
// message; the remainder is the LIKE term. It returns "" when nothing is left.
// Punctuation is stripped before the stopword check so "anotações?" is still
// recognized as the stopword "anotacoes" and cannot leak into the term.
func searchTerm(normalized string) string {
	words := strings.Fields(normalized)
	kept := make([]string, 0, len(words))
	for _, word := range words {
		word = stripPunctuation(word)
		if word == "" || noteStopwords[word] {
			continue
		}
		kept = append(kept, word)
	}
	return strings.Join(kept, " ")
}

// stripPunctuation keeps letters and digits only, so a trailing "?" cannot leak
// into the LIKE term and miss every row.
func stripPunctuation(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// lastWord returns the final word of a multi-word term.
func lastWord(term string) string {
	words := strings.Fields(term)
	if len(words) == 0 {
		return ""
	}
	return words[len(words)-1]
}