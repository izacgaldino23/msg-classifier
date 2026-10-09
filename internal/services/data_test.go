package services

import (
	"testing"
	"time"

	"msg-classifier/internal/models"
	"msg-classifier/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newDataService reuses the package newTestDB (Contact) and adds the notes and
// transactions schemas.
func newDataService(t *testing.T) (*DataService, *gorm.DB) {
	t.Helper()
	db := newTestDB(t)
	require.NoError(t, db.AutoMigrate(&models.Note{}, &models.TodoItem{}, &models.Transaction{}), "AutoMigrate(notes, transactions)")
	service := NewDataService(
		repository.NewContactRepository(db),
		repository.NewNotesRepository(db),
		repository.NewTransactionRepository(db),
	)
	return service, db
}

// seedDataNote persists a note through the repository: gorm's variadic Create
// conditions would otherwise swallow the items argument. Named seedDataNote to
// avoid clashing with the seedNote helper the note service tests already own.
func seedDataNote(t *testing.T, db *gorm.DB, note *models.Note, items []models.TodoItem) {
	t.Helper()
	require.NoError(t, repository.NewNotesRepository(db).Create(note, items), "seed note")
}

// noteTestDate is a fixed calendar date so assertions never depend on the day
// the suite runs.
func noteTestDate() time.Time { return time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC) }

func TestDataServiceListContacts(t *testing.T) {
	service, db := newDataService(t)
	db.Create(&models.Contact{Name: "Só telefone", NameNorm: "so telefone", Phone: strPtr("1111111111")})
	db.Create(&models.Contact{Name: "Só nome", NameNorm: "so nome"})

	all, err := service.ListContacts(ContactFilterAll, "")
	require.NoError(t, err)
	assert.Len(t, all, 2)

	empty, err := service.ListContacts("", "")
	require.NoError(t, err, "an empty filter means all")
	assert.Len(t, empty, 2)

	phone, err := service.ListContacts(ContactFilterPhone, "")
	require.NoError(t, err)
	require.Len(t, phone, 1)
	assert.Equal(t, "Só telefone", phone[0].Name)

	name, err := service.ListContacts(ContactFilterName, "")
	require.NoError(t, err)
	require.Len(t, name, 1)
	assert.Equal(t, "Só nome", name[0].Name)

	// The term is normalized before it reaches the column, so an accented
	// "só" still finds the accent-folded name_norm.
	byName, err := service.ListContacts(ContactFilterAll, "SÓ")
	require.NoError(t, err)
	assert.Len(t, byName, 2)

	byPhone, err := service.ListContacts(ContactFilterAll, "1111")
	require.NoError(t, err)
	require.Len(t, byPhone, 1)
	assert.Equal(t, "Só telefone", byPhone[0].Name)

	// Filter and term compose: the phone filter still excludes the other row.
	combined, err := service.ListContacts(ContactFilterPhone, "nome")
	require.NoError(t, err)
	assert.Len(t, combined, 0)

	_, err = service.ListContacts("bogus", "")
	assert.ErrorIs(t, err, ErrInvalidFilter)
}

func TestDataServiceListNotes(t *testing.T) {
	service, db := newDataService(t)
	seedDataNote(t, db, &models.Note{Type: models.NoteTypeNote, Content: "nota"}, nil)
	seedDataNote(t, db, &models.Note{Type: models.NoteTypeTodo, Content: "todo"}, []models.TodoItem{{Text: "pão", Position: 0}})

	all, err := service.ListNotes(NoteFilterAll, "")
	require.NoError(t, err)
	assert.Len(t, all, 2)

	todos, err := service.ListNotes(NoteFilterTodo, "")
	require.NoError(t, err)
	require.Len(t, todos, 1)
	assert.Equal(t, "todo", todos[0].Content)
	assert.Len(t, todos[0].Items, 1, "items are preloaded")

	notes, err := service.ListNotes(NoteFilterNote, "")
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "nota", notes[0].Content)

	byTerm, err := service.ListNotes(NoteFilterAll, "todo")
	require.NoError(t, err)
	require.Len(t, byTerm, 1)
	assert.Equal(t, "todo", byTerm[0].Content)

	combined, err := service.ListNotes(NoteFilterTodo, "nota")
	require.NoError(t, err)
	assert.Len(t, combined, 0, "the type filter still applies with a term")

	_, err = service.ListNotes("bogus", "")
	assert.ErrorIs(t, err, ErrInvalidFilter)
}

func TestDataServiceGetNotFound(t *testing.T) {
	service, _ := newDataService(t)

	_, err := service.GetContact(999)
	assert.ErrorIs(t, err, repository.ErrNotFound)

	_, err = service.GetNote(999)
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestDataServiceUpdateContact(t *testing.T) {
	service, db := newDataService(t)
	db.Create(&models.Contact{Name: "Antigo", NameNorm: "antigo", Phone: strPtr("1111111111")})

	contact, err := service.UpdateContact(1, "  Maria da Silva  ", " 2222222222 ", "   ")
	require.NoError(t, err)
	assert.Equal(t, "Maria da Silva", contact.Name)
	assert.Equal(t, "maria da silva", contact.NameNorm, "NameNorm is recomputed via normalizeName")
	require.NotNil(t, contact.Phone)
	assert.Equal(t, "2222222222", *contact.Phone)
	assert.Nil(t, contact.Email, "a blank field is stored as NULL")

	var reloaded models.Contact
	require.NoError(t, db.First(&reloaded, 1).Error)
	assert.Equal(t, "Maria da Silva", reloaded.Name)
	assert.Equal(t, "maria da silva", reloaded.NameNorm)
}

func TestDataServiceUpdateContactEmptyName(t *testing.T) {
	service, db := newDataService(t)
	db.Create(&models.Contact{Name: "Fulano"})

	_, err := service.UpdateContact(1, "   ", "x", "y")
	assert.ErrorIs(t, err, ErrInvalidData)
}

func TestDataServiceUpdateContactNotFound(t *testing.T) {
	service, _ := newDataService(t)

	_, err := service.UpdateContact(999, "Fulano", "", "")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestDataServiceUpdateNote(t *testing.T) {
	service, db := newDataService(t)
	seedDataNote(t, db, &models.Note{Type: models.NoteTypeReminder, Content: "pagar"}, nil)

	note, err := service.UpdateNote(1, "  pagar a conta de luz  ", "10/05/2026", "14h30", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "pagar a conta de luz", note.Content)
	require.NotNil(t, note.Date)
	assert.Equal(t, "10/05/2026", note.Date.Format("02/01/2006"))
	require.NotNil(t, note.Time)
	assert.Equal(t, "14:30", *note.Time, "the typed time is normalised to HH:MM")

	var reloaded models.Note
	require.NoError(t, db.First(&reloaded, 1).Error)
	require.NotNil(t, reloaded.Time)
	assert.Equal(t, "14:30", *reloaded.Time)
}

func TestDataServiceUpdateNoteClearsDateAndTime(t *testing.T) {
	service, db := newDataService(t)
	date := noteTestDate()
	clock := "09:00"
	seedDataNote(t, db, &models.Note{Type: models.NoteTypeReminder, Content: "pagar", Date: &date, Time: &clock}, nil)

	note, err := service.UpdateNote(1, "pagar", "", "", nil, nil)
	require.NoError(t, err)
	assert.Nil(t, note.Date, "an empty date field clears the date")
	assert.Nil(t, note.Time, "an empty time field clears the time")
}

func TestDataServiceUpdateNoteInvalidInput(t *testing.T) {
	service, db := newDataService(t)
	seedDataNote(t, db, &models.Note{Type: models.NoteTypeNote, Content: "x"}, nil)

	_, err := service.UpdateNote(1, "   ", "", "", nil, nil)
	assert.ErrorIs(t, err, ErrInvalidData, "empty content")

	_, err = service.UpdateNote(1, "ok", "31/02/2026", "", nil, nil)
	assert.ErrorIs(t, err, ErrInvalidData, "unparseable date")

	_, err = service.UpdateNote(1, "ok", "", "99h", nil, nil)
	assert.ErrorIs(t, err, ErrInvalidData, "unparseable time")
}

func TestDataServiceUpdateNoteRebuildsItems(t *testing.T) {
	service, db := newDataService(t)
	seedDataNote(t, db, &models.Note{Type: models.NoteTypeTodo, Content: "lista"}, []models.TodoItem{{Text: "antigo", Position: 0}})

	note, err := service.UpdateNote(1, "lista", "", "",
		[]string{"comprar pão", "   ", "leite", "ovos"},
		[]string{"1", "1", "0"})
	require.NoError(t, err)
	require.Len(t, note.Items, 3, "blank texts are dropped")
	assert.Equal(t, "comprar pão", note.Items[0].Text)
	assert.True(t, note.Items[0].Done)
	assert.Equal(t, 0, note.Items[0].Position)
	assert.Equal(t, "leite", note.Items[1].Text, "positions are re-sequenced after the drop")
	assert.Equal(t, 1, note.Items[1].Position)
	assert.False(t, note.Items[1].Done, "the done value follows the text index, not the kept index")
	assert.Equal(t, "ovos", note.Items[2].Text)
	assert.False(t, note.Items[2].Done, "a missing done value counts as not done")

	var items []models.TodoItem
	require.NoError(t, db.Where("note_id = ?", note.ID).Order("position").Find(&items).Error)
	require.Len(t, items, 3, "items are replaced, not merged")
	assert.Equal(t, "comprar pão", items[0].Text)
	assert.Equal(t, note.ID, items[0].NoteID)
}

func TestDataServiceUpdateNoteNotFound(t *testing.T) {
	service, _ := newDataService(t)

	_, err := service.UpdateNote(999, "x", "", "", nil, nil)
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestDataServiceListTransactions(t *testing.T) {
	service, db := newDataService(t)
	db.Create(&models.Transaction{Type: models.TransactionTypePurchase, Amount: 50, Date: noteTestDate(), Party: "Supermercado"})
	db.Create(&models.Transaction{Type: models.TransactionTypePayment, Amount: 120, Date: noteTestDate(), Party: "Conta"})

	// "all" is the gate's own vocabulary, not a type: the repository reads a
	// non-empty Type as an equality filter, so it must never reach the query.
	all, err := service.ListTransactions(TransactionFilterAll, "")
	require.NoError(t, err)
	assert.Len(t, all, 2)

	empty, err := service.ListTransactions("", "")
	require.NoError(t, err, "an empty filter means all")
	assert.Len(t, empty, 2)

	purchases, err := service.ListTransactions(models.TransactionTypePurchase, "")
	require.NoError(t, err)
	require.Len(t, purchases, 1)
	assert.Equal(t, models.TransactionTypePurchase, purchases[0].Type)

	search, err := service.ListTransactions(TransactionFilterAll, "supermercado")
	require.NoError(t, err)
	require.Len(t, search, 1)
	assert.Equal(t, 50.0, search[0].Amount)

	_, err = service.ListTransactions("bogus", "")
	assert.ErrorIs(t, err, ErrInvalidFilter)
}

func TestDataServiceGetTransaction(t *testing.T) {
	service, db := newDataService(t)
	transaction := &models.Transaction{Type: models.TransactionTypeSale, Amount: 30, Date: noteTestDate(), Party: "Fulano"}
	require.NoError(t, db.Create(transaction).Error)

	found, err := service.GetTransaction(transaction.ID)
	require.NoError(t, err)
	assert.Equal(t, "Fulano", found.Party)

	_, err = service.GetTransaction(transaction.ID + 99)
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestDataServiceUpdateTransaction(t *testing.T) {
	service, db := newDataService(t)
	transaction := &models.Transaction{Type: models.TransactionTypePurchase, Amount: 50, Date: noteTestDate(), Party: "Padaria"}
	require.NoError(t, db.Create(transaction).Error)

	// The amount and date go back through the finance parsers, so an edit can
	// never store a value the message flow would not have produced.
	saved, err := service.UpdateTransaction(transaction.ID, models.DataForm{
		Type: models.TransactionTypePayment, Amount: "1.234,56", Date: "12/07/2026", Party: "Farmácia",
	})
	require.NoError(t, err)

	updated, err := service.GetTransaction(saved.ID)
	require.NoError(t, err)
	assert.Equal(t, models.TransactionTypePayment, updated.Type)
	assert.InDelta(t, 1234.56, updated.Amount, 0.001)
	assert.Equal(t, "Farmácia", updated.Party)
	assert.Equal(t, time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC), updated.Date)
}

func TestDataServiceUpdateTransactionInvalidInput(t *testing.T) {
	service, db := newDataService(t)
	transaction := &models.Transaction{Type: models.TransactionTypePurchase, Amount: 50, Date: noteTestDate()}
	require.NoError(t, db.Create(transaction).Error)

	tests := []struct {
		name string
		form models.DataForm
	}{
		{"unknown type", models.DataForm{Type: "bogus", Amount: "10,00", Date: "12/07/2026"}},
		{"unparseable amount", models.DataForm{Type: models.TransactionTypePayment, Amount: "muito", Date: "12/07/2026"}},
		{"unparseable date", models.DataForm{Type: models.TransactionTypePayment, Amount: "10,00", Date: "quinta"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.UpdateTransaction(transaction.ID, tt.form)
			assert.ErrorIs(t, err, ErrInvalidData)
		})
	}

	_, err := service.UpdateTransaction(transaction.ID+99, models.DataForm{
		Type: models.TransactionTypePayment, Amount: "10,00", Date: "12/07/2026",
	})
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestDataServiceDeleteTransactions(t *testing.T) {
	service, db := newDataService(t)
	first := &models.Transaction{Type: models.TransactionTypePurchase, Amount: 10, Date: noteTestDate()}
	second := &models.Transaction{Type: models.TransactionTypeSale, Amount: 20, Date: noteTestDate()}
	require.NoError(t, db.Create(first).Error)
	require.NoError(t, db.Create(second).Error)

	require.NoError(t, service.DeleteTransactions([]uint{first.ID, second.ID}))

	remaining, err := service.ListTransactions(TransactionFilterAll, "")
	require.NoError(t, err)
	assert.Empty(t, remaining)
}

func TestDataServiceDelete(t *testing.T) {
	service, db := newDataService(t)
	db.Create(&models.Contact{Name: "A"})
	db.Create(&models.Contact{Name: "B"})
	seedDataNote(t, db, &models.Note{Type: models.NoteTypeTodo, Content: "c"}, []models.TodoItem{{Text: "x", Position: 0}})

	require.NoError(t, service.DeleteContacts([]uint{1}))
	var contacts int64
	require.NoError(t, db.Model(&models.Contact{}).Count(&contacts).Error)
	assert.Equal(t, int64(1), contacts)

	require.NoError(t, service.DeleteNotes([]uint{1}))
	var notes, items int64
	require.NoError(t, db.Model(&models.Note{}).Count(&notes).Error)
	require.NoError(t, db.Model(&models.TodoItem{}).Count(&items).Error)
	assert.Equal(t, int64(0), notes)
	assert.Equal(t, int64(0), items, "todo items go with the note")
}

func TestDataServiceDeleteEmptyIsANoOp(t *testing.T) {
	service, db := newDataService(t)
	db.Create(&models.Contact{Name: "A"})

	require.NoError(t, service.DeleteContacts(nil))
	require.NoError(t, service.DeleteNotes(nil))

	var count int64
	require.NoError(t, db.Model(&models.Contact{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
