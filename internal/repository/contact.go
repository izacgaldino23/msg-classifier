package repository

import (
	"msg-classifier/internal/models"

	"gorm.io/gorm"
)

// ErrNotFound is the repository's not-found sentinel. It is the same value as
// gorm.ErrRecordNotFound, so services can detect not-found with errors.Is
// without importing gorm.
var ErrNotFound = gorm.ErrRecordNotFound

// ContactRepository owns all gorm queries for the Contact entity.
type ContactRepository struct {
	db *gorm.DB
}

func NewContactRepository(db *gorm.DB) *ContactRepository {
	return &ContactRepository{db: db}
}

// Create persists a new contact.
func (r *ContactRepository) Create(contact *models.Contact) error {
	return r.db.Create(contact).Error
}

// FindByPhone returns the contact with the given phone, or ErrNotFound.
func (r *ContactRepository) FindByPhone(phone string) (*models.Contact, error) {
	var contact models.Contact
	err := r.db.Where("phone = ?", phone).Order("id").First(&contact).Error
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

// FindByEmail returns the contact with the given email (case-insensitive), or ErrNotFound.
func (r *ContactRepository) FindByEmail(email string) (*models.Contact, error) {
	var contact models.Contact
	err := r.db.Where("LOWER(email) = LOWER(?)", email).Order("id").First(&contact).Error
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

// FindByName returns the first contact whose name_norm matches the term, or ErrNotFound.
func (r *ContactRepository) FindByName(term string) (*models.Contact, error) {
	var contact models.Contact
	err := r.db.Where("name_norm LIKE ?", "%"+term+"%").Order("id").First(&contact).Error
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

// FindByNameNorm returns the first contact whose normalized name equals the term
// exactly, or ErrNotFound. Unlike FindByName it does not LIKE-match substrings —
// it is the duplicate key (DC-008 name fallback).
func (r *ContactRepository) FindByNameNorm(term string) (*models.Contact, error) {
	var contact models.Contact
	if err := r.db.Where("name_norm = ?", term).Order("id").First(&contact).Error; err != nil {
		return nil, err
	}
	return &contact, nil
}

// ListNeedingNameNorm returns contacts whose NameNorm is empty or NULL (pre-migration rows).
func (r *ContactRepository) ListNeedingNameNorm() ([]models.Contact, error) {
	var contacts []models.Contact
	err := r.db.Where("name_norm = '' OR name_norm IS NULL").Find(&contacts).Error
	if err != nil {
		return nil, err
	}
	return contacts, nil
}

// Save persists changes to an existing contact (used by the backfill).
func (r *ContactRepository) Save(contact *models.Contact) error {
	return r.db.Save(contact).Error
}

// List returns the contacts matching the filter, newest first. Accepted filters:
// all, phone, email and name (phone and email both absent or empty). An unknown
// filter falls back to "all" — the service layer is the gate that rejects it.
func (r *ContactRepository) List(filter string) ([]models.Contact, error) {
	query := r.db.Order("id DESC")
	switch filter {
	case "phone":
		query = query.Where("phone IS NOT NULL AND phone <> ''")
	case "email":
		query = query.Where("email IS NOT NULL AND email <> ''")
	case "name":
		query = query.Where("(phone IS NULL OR phone = '') AND (email IS NULL OR email = '')")
	}
	var contacts []models.Contact
	if err := query.Find(&contacts).Error; err != nil {
		return nil, err
	}
	return contacts, nil
}

// FindByID returns the contact with the given id, or ErrNotFound.
func (r *ContactRepository) FindByID(id uint) (*models.Contact, error) {
	var contact models.Contact
	if err := r.db.First(&contact, id).Error; err != nil {
		return nil, err
	}
	return &contact, nil
}

// DeleteByIDs removes the given contacts in a single statement; an empty list is a no-op.
func (r *ContactRepository) DeleteByIDs(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("id IN ?", ids).Delete(&models.Contact{}).Error
}
