package textutil

import "strings"

// FormatCurrencyAmount trucates empty decimals from values
func FormatCurrencyAmount(currencyAmount string) string {
	currencyAmount = strings.TrimSpace(currencyAmount)
	if strings.HasSuffix(currencyAmount, ".00") {
		idx := strings.IndexRune(currencyAmount, '.')
		if idx != -1 {
			return currencyAmount[:idx]
		}
	}
	return currencyAmount
}
