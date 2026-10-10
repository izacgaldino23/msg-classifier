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

// Transaction categories, mirroring the "transaction_category" choice question
// in the composite classification request. F2 asks Jev for the category in the
// same round trip as the type, so a transaction knows what it was for, not only
// what it was.
const (
	TransactionCategoryMarket    = "mercado"
	TransactionCategoryTransport = "transporte"
	TransactionCategoryHousing   = "moradia"
	TransactionCategoryHealth    = "saude"
	TransactionCategoryLeisure   = "lazer"
	TransactionCategoryService   = "servico"
	TransactionCategorySalary    = "salario"
	TransactionCategoryOther     = "outro"
)

// Transaction is a financial movement the user told us about. It is entered by
// hand: the app has no access to any bank or card, so there is no statement to
// reconcile against (DC-007).
type Transaction struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Type      string    `gorm:"index;size:20;not null" json:"type"`
	Category  string    `gorm:"index;size:20" json:"category"`
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

// IsTransactionCategory reports whether a raw category is one of the known
// criteria. The service drops any other Jev answer instead of persisting it.
func IsTransactionCategory(value string) bool {
	switch value {
	case TransactionCategoryMarket, TransactionCategoryTransport,
		TransactionCategoryHousing, TransactionCategoryHealth,
		TransactionCategoryLeisure, TransactionCategoryService,
		TransactionCategorySalary, TransactionCategoryOther:
		return true
	}
	return false
}