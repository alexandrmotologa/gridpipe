package types

import (
	"strings"
)

// CurrencyInfo contains extracted currency code and cleaned numeric text.
type CurrencyInfo struct {
	CurrencyCode string `json:"currency_code"`
	CleanedText  string `json:"cleaned_text"`
	HasCurrency  bool   `json:"has_currency"`
}

var currencySymbolMap = map[string]string{
	"$":   "USD",
	"€":   "EUR",
	"£":   "GBP",
	"¥":   "JPY",
	"₹":   "INR",
	"CHF": "CHF",
	"RON": "RON",
	"LEI": "RON",
	"lei": "RON",
	"MDL": "MDL",
	"USD": "USD",
	"EUR": "EUR",
	"GBP": "GBP",
}

// ExtractCurrency inspects a string for currency symbols and codes.
func ExtractCurrency(s string) CurrencyInfo {
	trimmed := strings.TrimSpace(s)
	detectedCode := ""

	for symbol, code := range currencySymbolMap {
		if strings.Contains(trimmed, symbol) {
			detectedCode = code
			trimmed = strings.ReplaceAll(trimmed, symbol, "")
			break
		}
	}

	trimmed = strings.TrimSpace(trimmed)
	return CurrencyInfo{
		CurrencyCode: detectedCode,
		CleanedText:  trimmed,
		HasCurrency:  detectedCode != "",
	}
}
