package models

import "time"

// Transaction type criteria, mirroring the "transaction_type" choice question in
// the Jev finance request. A type is semantic, so it comes from Jev; money and
// dates stay deterministic in the service.
const (
	TransactionTypePurchase = "compra"
	TransactionTypeSale     = "venda"
	TransactionTypeTransfer = "transferencia"
	TransactionTypeReceipt  = "recebimento"
	TransactionTypePayment  = "pagamento"
)

// Transaction is a financial movement the user told us about. It is entered by
// hand: the app has no access to any bank or card, so there is no statement to
// reconcile against (DC-007).
type Transaction struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Type      string    `gorm:"index;size:20;not null" json:"type"`
	Amount    float64   `gorm:"not null" json:"amount"`
	Date      time.Time `gorm:"index;not null" json:"date"`
	Party     string    `gorm:"index;size:120" json:"party"` // establishment or person involved
	Content   string    `gorm:"size:400" json:"content"`     // the original message, kept verbatim
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsTransactionType reports whether a raw type is one of the known criteria.
func IsTransactionType(value string) bool {
	switch value {
	case TransactionTypePurchase, TransactionTypeSale, TransactionTypeTransfer,
		TransactionTypeReceipt, TransactionTypePayment:
		return true
	}
	return false
}