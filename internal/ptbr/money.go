package ptbr

import (
	"regexp"
	"strconv"
	"strings"
)

// amountPatterns match a monetary amount in the PT-BR formats we accept, tried in
// order: a currency-marked amount, an amount with the "reais" suffix, then a bare
// number that carries a decimal or thousands separator. Requiring one of those
// markers is what keeps "3 vezes de 300 reais" from reading the installment count
// as the amount, and a date like "10/10/2023" from reading as money.
var amountPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)r\$\s*(\d{1,3}(?:\.\d{3})*(?:,\d+)?|\d+(?:,\d+)?)`),
	regexp.MustCompile(`(?i)\b(\d{1,3}(?:\.\d{3})+(?:,\d+)?|\d+(?:,\d+)?)\s*(?:reais|real)\b`),
	regexp.MustCompile(`\b(\d{1,3}(?:\.\d{3})+(?:,\d+)?|\d+,\d+)\b`),
}

// ParseAmount returns the first monetary amount in the message and true, or
// false when there is none. The result is always positive: the transaction type
// carries the direction, so a "recebimento" is not a negative amount.
// PT-BR format — "R$ 50,00", "1.234,56", "50 reais" — dots are thousands
// separators and the comma is the decimal mark.
func ParseAmount(message string) (float64, bool) {
	for _, pattern := range amountPatterns {
		match := pattern.FindStringSubmatch(message)
		if match == nil {
			continue
		}
		if amount, ok := parseMoney(match[1]); ok {
			return amount, true
		}
	}
	return 0, false
}

// parseMoney turns a PT-BR amount string into a float, dropping the thousands
// separators and turning the comma into a dot.
func parseMoney(value string) (float64, bool) {
	digits := strings.ReplaceAll(value, ".", "")
	digits = strings.ReplaceAll(digits, ",", ".")
	amount, err := strconv.ParseFloat(digits, 64)
	if err != nil {
		return 0, false
	}
	return amount, true
}
