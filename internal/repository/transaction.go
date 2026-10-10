package repository

import (
	"strings"
	"time"

	"msg-classifier/internal/models"

	"gorm.io/gorm"
)

// TransactionFilter narrows a transaction search. Every field is optional: a zero
// From means "any date", a zero Until leaves the range open, and an empty Type,
// Party or Term matches everything. One struct instead of one method per
// combination, so "quanto gastei com o mercado esse mês" is a single query.
type TransactionFilter struct {
	From  time.Time // inclusive
	Until time.Time // exclusive; zero means open-ended
	Type  string
	Party string // LIKE on the establishment or person
	Term  string // LIKE on the content or the party
}

// TransactionRepository owns all gorm queries for the Transaction entity. It
// reuses the package's ErrNotFound sentinel (declared in contact.go).
type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// Create persists a transaction; it receives the generated id.
func (r *TransactionRepository) Create(transaction *models.Transaction) error {
	return r.db.Create(transaction).Error
}

// Find returns the matching transactions, newest date first, or ErrNotFound when
// nothing matches — a search miss is not an error for the service to unwrap.
func (r *TransactionRepository) Find(filter TransactionFilter) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	err := scopeTransactions(r.db, filter).Order("date DESC, id DESC").Find(&transactions).Error
	if err != nil {
		return nil, err
	}
	if len(transactions) == 0 {
		return nil, ErrNotFound
	}
	return transactions, nil
}

// List returns the matching transactions for the browse screen, newest first. An
// empty table is a valid state, so it never yields ErrNotFound.
func (r *TransactionRepository) List(filter TransactionFilter) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	err := scopeTransactions(r.db, filter).Order("id DESC").Find(&transactions).Error
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

// Sum totals the amounts of the matching transactions ("quanto gastei"). A miss
// is zero, not ErrNotFound.
func (r *TransactionRepository) Sum(filter TransactionFilter) (float64, error) {
	var result struct {
		Total float64
	}
	err := scopeTransactions(r.db.Model(&models.Transaction{}), filter).
		Select("COALESCE(SUM(amount), 0) AS total").
		Scan(&result).Error
	if err != nil {
		return 0, err
	}
	return result.Total, nil
}

// FindByID returns the transaction with the given id, or ErrNotFound.
func (r *TransactionRepository) FindByID(id uint) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := r.db.First(&transaction, id).Error; err != nil {
		return nil, err
	}
	return &transaction, nil
}

// ListUncategorized returns the transactions that still have no category, oldest
// first — the rows the data screen's "Classificar pendentes" action works
// through.
func (r *TransactionRepository) ListUncategorized() ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	err := r.db.Where("category IS NULL OR category = ''").Order("id").Find(&transactions).Error
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

// UpdateCategory sets one transaction's category, leaving the other fields and
// the original message untouched.
func (r *TransactionRepository) UpdateCategory(id uint, category string) error {
	return r.db.Model(&models.Transaction{}).Where("id = ?", id).Update("category", category).Error
}

// FindDuplicate returns the first transaction matching the exact key
// (type + amount + date; party only when the key carries one — an empty pending
// party ignores party, a non-empty one matches equal or empty stored parties),
// or ErrNotFound.
func (r *TransactionRepository) FindDuplicate(txType string, amount float64, date time.Time, party string) (*models.Transaction, error) {
	query := r.db.Where("type = ? AND amount = ? AND date = ?", txType, amount, date)
	if party != "" {
		query = query.Where("(party IS NULL OR party = '' OR LOWER(party) = LOWER(?))", party)
	}
	var transaction models.Transaction
	if err := query.Order("id").First(&transaction).Error; err != nil {
		return nil, err
	}
	return &transaction, nil
}

// Save updates the transaction fields.
func (r *TransactionRepository) Save(transaction *models.Transaction) error {
	return r.db.Save(transaction).Error
}

// DeleteByIDs removes the given transactions. An empty id list is a no-op.
func (r *TransactionRepository) DeleteByIDs(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("id IN ?", ids).Delete(&models.Transaction{}).Error
}

// scopeTransactions applies the filter clauses. The OR is parenthesized on
// purpose: gorm does not wrap a raw OR string, so without it the term clause
// would swallow the other conditions.
func scopeTransactions(db *gorm.DB, filter TransactionFilter) *gorm.DB {
	if !filter.From.IsZero() {
		db = db.Where("date >= ?", filter.From)
	}
	if !filter.Until.IsZero() {
		db = db.Where("date < ?", filter.Until)
	}
	if filter.Type != "" {
		db = db.Where("type = ?", filter.Type)
	}
	if filter.Party != "" {
		db = db.Where("LOWER(party) LIKE ?", likeValue(filter.Party))
	}
	if filter.Term != "" {
		like := likeValue(filter.Term)
		db = db.Where("(LOWER(content) LIKE ? OR LOWER(party) LIKE ? OR LOWER(category) LIKE ?)", like, like, like)
	}
	return db
}

func likeValue(term string) string {
	return "%" + strings.ToLower(term) + "%"
}
