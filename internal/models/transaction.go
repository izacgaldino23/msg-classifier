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
	ID        uint      `gorm:"primaryKey"`
	Type      string    `gorm:"index;size:20;not null"`
	Amount    float64   `gorm:"not null"`
	Date      time.Time `gorm:"index;not null"`
	Party     string    `gorm:"index;size:120"` // establishment or person involved
	Content   string    `gorm:"size:400"`       // the original message, kept verbatim
	CreatedAt time.Time
	UpdatedAt time.Time
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