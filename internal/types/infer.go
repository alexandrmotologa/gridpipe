package types

import (
	"strings"
)

// InferredColumnType represents schema analysis for a column.
type InferredColumnType struct {
	Type         string  `json:"type"`          // integer, float, currency, date, boolean, text
	CurrencyCode string  `json:"currency_code"` // if type is currency
	Confidence   float64 `json:"confidence"`
	NonNullCount int     `json:"non_null_count"`
	TotalCount   int     `json:"total_count"`
}

// InferColumnType evaluates an array of cell strings in a column and selects the primary type.
func InferColumnType(samples []string, localeHint string) InferredColumnType {
	totalCount := len(samples)
	if totalCount == 0 {
		return InferredColumnType{Type: "text", Confidence: 1.0}
	}

	dateCount := 0
	currCount := 0
	intCount := 0
	floatCount := 0
	boolCount := 0
	nonEmptyCount := 0

	currencyFreq := make(map[string]int)

	for _, raw := range samples {
		s := strings.TrimSpace(raw)
		if s == "" || s == "-" || s == "N/A" || s == "null" {
			continue
		}
		nonEmptyCount++

		// Check boolean
		lower := strings.ToLower(s)
		if lower == "true" || lower == "false" || lower == "yes" || lower == "no" {
			boolCount++
			continue
		}

		// Check currency
		currInfo := ExtractCurrency(s)
		if currInfo.HasCurrency {
			currCount++
			currencyFreq[currInfo.CurrencyCode]++
			numRes := ParseNumber(currInfo.CleanedText, localeHint)
			if numRes.Success {
				if numRes.Decimals == 0 {
					intCount++
				} else {
					floatCount++
				}
			}
			continue
		}

		// Check date
		dateRes := ParseDate(s)
		if dateRes.Success {
			dateCount++
			continue
		}

		// Check number
		numRes := ParseNumber(s, localeHint)
		if numRes.Success {
			if numRes.Decimals == 0 {
				intCount++
			} else {
				floatCount++
			}
			continue
		}
	}

	if nonEmptyCount == 0 {
		return InferredColumnType{
			Type:         "text",
			Confidence:   1.0,
			TotalCount:   totalCount,
			NonNullCount: 0,
		}
	}

	// Decision threshold: at least 70% of non-empty cells
	threshold := 0.70

	if float64(dateCount)/float64(nonEmptyCount) >= threshold {
		return InferredColumnType{
			Type:         "date",
			Confidence:   float64(dateCount) / float64(nonEmptyCount),
			TotalCount:   totalCount,
			NonNullCount: nonEmptyCount,
		}
	}

	if float64(currCount)/float64(nonEmptyCount) >= threshold {
		primaryCurr := "USD"
		maxFreq := 0
		for code, freq := range currencyFreq {
			if freq > maxFreq {
				maxFreq = freq
				primaryCurr = code
			}
		}
		return InferredColumnType{
			Type:         "currency",
			CurrencyCode: primaryCurr,
			Confidence:   float64(currCount) / float64(nonEmptyCount),
			TotalCount:   totalCount,
			NonNullCount: nonEmptyCount,
		}
	}

	if float64(intCount+floatCount)/float64(nonEmptyCount) >= threshold {
		if floatCount > 0 {
			return InferredColumnType{
				Type:         "float",
				Confidence:   float64(intCount+floatCount) / float64(nonEmptyCount),
				TotalCount:   totalCount,
				NonNullCount: nonEmptyCount,
			}
		}
		return InferredColumnType{
			Type:         "integer",
			Confidence:   float64(intCount) / float64(nonEmptyCount),
			TotalCount:   totalCount,
			NonNullCount: nonEmptyCount,
		}
	}

	if float64(boolCount)/float64(nonEmptyCount) >= threshold {
		return InferredColumnType{
			Type:         "boolean",
			Confidence:   float64(boolCount) / float64(nonEmptyCount),
			TotalCount:   totalCount,
			NonNullCount: nonEmptyCount,
		}
	}

	return InferredColumnType{
		Type:         "text",
		Confidence:   1.0,
		TotalCount:   totalCount,
		NonNullCount: nonEmptyCount,
	}
}
