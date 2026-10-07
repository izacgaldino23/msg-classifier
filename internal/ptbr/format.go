// Package ptbr provides PT-BR specific formatting and parsing utilities.
// It owns both the parsing (read) and formatting (write) sides of
// monetary values and dates to ensure round-trip compatibility.
package ptbr

import (
	"fmt"
	"strings"
	"time"
)

// DateBR formats a date as dd/mm/aaaa; nil renders "".
// It is the write side of ParseDate: whatever this prints, ParseDate reads back.
func DateBR(d *time.Time) string {
	if d == nil {
		return ""
	}
	return d.Format("02/01/2006")
}

// MoneyBRL formats an amount as PT-BR currency: "R$ 1.234,56".
// It is the write side of ParseAmount, so dots group thousands and the comma
// is the decimal mark -- the same shape the parser expects.
// Amounts are always positive: the transaction type carries the direction.
func MoneyBRL(amount float64) string {
	// Format with exactly two decimal places
	wholePart, decimals, _ := strings.Cut(fmt.Sprintf("%.2f", amount), ".")
	
	// Group thousands with dots
	var grouped []byte
	for i, digit := range []byte(wholePart) {
		if i > 0 && (len(wholePart)-i)%3 == 0 {
			grouped = append(grouped, '.')
		}
		grouped = append(grouped, digit)
	}
	
	return "R$ " + string(grouped) + "," + decimals
}