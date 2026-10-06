package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The REST presenter serialises entities directly, so a missing tag ships a
// Go field name on the wire instead of failing loudly.
func TestTransactionJSONTags(t *testing.T) {
	date := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(Transaction{
		ID: 7, Type: TransactionTypePurchase, Amount: 1234.56,
		Date: date, Party: "supermercado", Content: "compras no supermercado",
	})
	require.NoError(t, err, "json.Marshal()")

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(payload, &decoded), "json.Unmarshal()")

	for _, key := range []string{"id", "type", "amount", "date", "party", "content", "created_at", "updated_at"} {
		assert.Contains(t, decoded, key, "missing snake_case json tag")
	}
	for _, forbidden := range []string{"ID", "Type", "Amount", "Date", "Party", "Content", "CreatedAt"} {
		assert.NotContains(t, decoded, forbidden, "Go field name leaked onto the wire")
	}
}