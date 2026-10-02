package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"msg-classifier/internal/models"
	"msg-classifier/internal/ptbr"
	"msg-classifier/internal/repository"
)

// Data screen kinds — the :kind route segment.
const (
	DataKindContact      = "contact"
	DataKindNotes        = "notes"
	DataKindTransactions = "transaction"
)

// Filters accepted by ListContacts.
const (
	ContactFilterAll   = "all"
	ContactFilterPhone = "phone"
	ContactFilterEmail = "email"
	ContactFilterName  = "name"
)

// Filters accepted by ListNotes.
const (
	NoteFilterAll      = "all"
	NoteFilterNote     = "note"
	NoteFilterReminder = "reminder"
	NoteFilterTodo     = "todo"
)

// Filters accepted by ListTransactions on top of the transaction types.
const (
	TransactionFilterAll = "all"
)

// ListTransactions returns the transactions matching the filter and the optional
// search term, newest first. The filter is one of the transaction types or "all";
// an empty filter means "all" and an unknown one is ErrInvalidFilter.
func (s *DataService) ListTransactions(filter, term string) ([]*models.Transaction, error) {
	switch filter {
	case "", TransactionFilterAll, models.TransactionTypePurchase, models.TransactionTypeSale,
		models.TransactionTypePayment, models.TransactionTypeReceipt, models.TransactionTypeTransfer:
	default:
		return nil, fmt.Errorf("%w: transaction filter %q", ErrInvalidFilter, filter)
	}
	// "all" is this gate's own vocabulary, not a transaction type: the repository
	// reads a non-empty Type as an equality filter, so "all" would match nothing.
	if filter == "" || filter == TransactionFilterAll {
		filter = ""
	}
	transactions, err := s.transactions.List(repository.TransactionFilter{Type: filter, Term: strings.TrimSpace(term)})
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	return transactions, nil
}

// GetTransaction returns a transaction by id; repository.ErrNotFound propagates so
// the controller can answer 404.
func (s *DataService) GetTransaction(id uint) (*models.Transaction, error) {
	transaction, err := s.transactions.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load transaction: %w", err)
	}
	return transaction, nil
}

// UpdateTransaction validates and persists the editable transaction fields. The
// amount is re-parsed from the typed text with the same parser the add path uses,
// and the date through the same event-date parser, so an edit can never store a
// value the message flow would not have produced.
func (s *DataService) UpdateTransaction(id uint, form models.DataForm) (*models.Transaction, error) {
	transaction, err := s.GetTransaction(id)
	if err != nil {
		return nil, err
	}
	if !models.IsTransactionType(form.Type) {
		return nil, fmt.Errorf("%w: transaction type %q", ErrInvalidData, form.Type)
	}
	amount, ok := ptbr.ParseAmount(form.Amount)
	if !ok || amount <= 0 {
		return nil, fmt.Errorf("%w: transaction amount %q", ErrInvalidData, form.Amount)
	}
	date, ok := ptbr.ParseEventDate(form.Date, time.Now())
	if !ok {
		return nil, fmt.Errorf("%w: unparseable date %q", ErrInvalidData, form.Date)
	}
	transaction.Type = form.Type
	transaction.Amount = amount
	transaction.Date = date
	transaction.Party = strings.TrimSpace(form.Party)
	if content := strings.TrimSpace(form.Content); content != "" {
		transaction.Content = content
	}
	if err := s.transactions.Save(transaction); err != nil {
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}
	return transaction, nil
}

// DeleteTransactions removes the given transactions in a single statement.
func (s *DataService) DeleteTransactions(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.transactions.DeleteByIDs(ids); err != nil {
		return fmt.Errorf("failed to delete transactions: %w", err)
	}
	return nil
}

var (
	// ErrInvalidFilter marks an unknown kind or filter; controllers map it to 400.
	ErrInvalidFilter = errors.New("invalid data filter")
	// ErrInvalidData marks invalid edit input (empty name/content, bad date or
	// time); controllers map it to 400.
	ErrInvalidData = errors.New("invalid data")
)

// DataService serves the data browsing screen. It composes both repositories and
// owns the edit rules, keeping the browse/admin CRUD out of the classification
// flow. It reuses the package's unexported normalizeName so NameNorm stays
// consistent with the add path.
type DataService struct {
	contacts     *repository.ContactRepository
	notes        *repository.NotesRepository
	transactions *repository.TransactionRepository
}

func NewDataService(contacts *repository.ContactRepository, notes *repository.NotesRepository, transactions *repository.TransactionRepository) *DataService {
	return &DataService{contacts: contacts, notes: notes, transactions: transactions}
}

// ListContacts returns the contacts matching the filter, newest first. An empty
// filter means "all"; an unknown one is ErrInvalidFilter.
func (s *DataService) ListContacts(filter string) ([]models.Contact, error) {
	switch filter {
	case "", ContactFilterAll, ContactFilterPhone, ContactFilterEmail, ContactFilterName:
	default:
		return nil, fmt.Errorf("%w: contact filter %q", ErrInvalidFilter, filter)
	}
	contacts, err := s.contacts.List(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list contacts: %w", err)
	}
	return contacts, nil
}

// ListNotes returns the notes matching the filter, newest first. An empty filter
// means "all"; an unknown one is ErrInvalidFilter.
func (s *DataService) ListNotes(filter string) ([]*models.Note, error) {
	switch filter {
	case "", NoteFilterAll, NoteFilterNote, NoteFilterReminder, NoteFilterTodo:
	default:
		return nil, fmt.Errorf("%w: note filter %q", ErrInvalidFilter, filter)
	}

	var notes []*models.Note
	var err error
	if filter == NoteFilterNote || filter == NoteFilterReminder || filter == NoteFilterTodo {
		notes, err = s.notes.ListByType(filter)
	} else {
		notes, err = s.notes.List()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}
	return notes, nil
}

// GetContact returns a contact by id; repository.ErrNotFound propagates so the
// controller can answer 404.
func (s *DataService) GetContact(id uint) (*models.Contact, error) {
	contact, err := s.contacts.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load contact: %w", err)
	}
	return contact, nil
}

// GetNote returns a note by id with its items preloaded.
func (s *DataService) GetNote(id uint) (*models.Note, error) {
	note, err := s.notes.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load note: %w", err)
	}
	return note, nil
}

// UpdateContact validates and persists the editable contact fields. The name is
// trimmed and required; NameNorm is always recomputed from the stored name.
func (s *DataService) UpdateContact(id uint, name, phone, email string) (*models.Contact, error) {
	contact, err := s.GetContact(id)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: contact name is required", ErrInvalidData)
	}
	contact.Name = trimmed
	contact.NameNorm = ptbr.NormalizeName(trimmed)
	contact.Phone = optionalString(phone)
	contact.Email = optionalString(email)
	if err := s.contacts.Save(contact); err != nil {
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}
	return contact, nil
}

// UpdateNote validates and persists the editable note fields. Type is read-only
// on the screen, so it is never changed. Date and time are re-parsed from the
// text the user typed, and the to-do list is rebuilt from the parallel
// text/done arrays.
func (s *DataService) UpdateNote(id uint, content, dateText, timeText string, itemTexts, itemDones []string) (*models.Note, error) {
	note, err := s.GetNote(id)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: note content is required", ErrInvalidData)
	}
	note.Content = trimmed

	date, err := s.parseDate(dateText)
	if err != nil {
		return nil, err
	}
	note.Date = date

	clock, err := s.parseTime(timeText)
	if err != nil {
		return nil, err
	}
	note.Time = clock

	if err := s.notes.Save(note, buildItems(itemTexts, itemDones)); err != nil {
		return nil, fmt.Errorf("failed to update note: %w", err)
	}
	return note, nil
}

// DeleteContacts removes the given contacts in a single statement.
func (s *DataService) DeleteContacts(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.contacts.DeleteByIDs(ids); err != nil {
		return fmt.Errorf("failed to delete contacts: %w", err)
	}
	return nil
}

// DeleteNotes removes the given notes and their to-do items.
func (s *DataService) DeleteNotes(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.notes.DeleteByIDs(ids); err != nil {
		return fmt.Errorf("failed to delete notes: %w", err)
	}
	return nil
}

// parseDate turns the typed date text into a stored UTC midnight; an empty field
// clears the date and an unparseable one is invalid input.
func (s *DataService) parseDate(text string) (*time.Time, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	date, ok := ptbr.ParseDate(text, time.Now())
	if !ok {
		return nil, fmt.Errorf("%w: unparseable date %q", ErrInvalidData, text)
	}
	return &date, nil
}

// parseTime normalises the typed time to HH:MM; an empty field clears it.
func (s *DataService) parseTime(text string) (*string, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	clock, ok := ptbr.ParseTime(text)
	if !ok {
		return nil, fmt.Errorf("%w: unparseable time %q", ErrInvalidData, text)
	}
	return &clock, nil
}

// buildItems zips the parallel text/done arrays by index, drops blank texts and
// re-sequences Position from the kept rows. A shorter done array (or a non-"1"
// value) counts as not done.
func buildItems(texts, dones []string) []models.TodoItem {
	items := make([]models.TodoItem, 0, len(texts))
	for i, raw := range texts {
		text := strings.TrimSpace(raw)
		if text == "" {
			continue
		}
		done := i < len(dones) && dones[i] == "1"
		items = append(items, models.TodoItem{Text: text, Done: done, Position: len(items)})
	}
	return items
}

// optionalString trims the field and returns nil when it is empty, so a cleared
// phone/email is stored as NULL instead of an empty string.
func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
