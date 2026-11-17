package types

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// NumberParseResult contains parsed numeric value and metadata.
type NumberParseResult struct {
	Value      float64 `json:"value"`
	Canonical  string  `json:"canonical"` // e.g. "1234.56"
	IsNegative bool    `json:"is_negative"`
	Decimals   int     `json:"decimals"`
	Success    bool    `json:"success"`
}

// LocaleSeparatorHeuristic determines whether a column uses comma or dot as decimal separator.
// Mode: "us" (1,234.56), "eu" (1.234,56), or "auto".
func DetermineColumnLocale(samples []string) string {
	commaAsDecimalScore := 0
	dotAsDecimalScore := 0

	for _, s := range samples {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		lastComma := strings.LastIndex(s, ",")
		lastDot := strings.LastIndex(s, ".")

		if lastComma > lastDot && lastComma != -1 {
			// e.g. 1.234,56 or 1234,56
			// If exactly 2 digits after comma, highly likely European decimal
			afterComma := s[lastComma+1:]
			digitsAfter := countTrailingDigits(afterComma)
			if digitsAfter == 2 || digitsAfter == 1 {
				commaAsDecimalScore += 2
			} else {
				commaAsDecimalScore++
			}
		} else if lastDot > lastComma && lastDot != -1 {
			// e.g. 1,234.56 or 1234.56
			afterDot := s[lastDot+1:]
			digitsAfter := countTrailingDigits(afterDot)
			if digitsAfter == 2 || digitsAfter == 1 {
				dotAsDecimalScore += 2
			} else {
				dotAsDecimalScore++
			}
		}
	}

	if commaAsDecimalScore > dotAsDecimalScore {
		return "eu"
	}
	return "us"
}

func countTrailingDigits(s string) int {
	count := 0
	for _, r := range s {
		if unicode.IsDigit(r) {
			count++
		} else {
			break
		}
	}
	return count
}

// ParseNumber parses a numeric string into float64 and canonical representation.
// Handles US (1,234.56), EU (1.234,56), accounting parentheses (1,234.56), trailing minus 1234.56-, and percent 15%.
func ParseNumber(s string, localeHint string) NumberParseResult {
	orig := strings.TrimSpace(s)
	// Replace non-breaking space
	orig = strings.ReplaceAll(orig, "\u00A0", "")
	orig = strings.ReplaceAll(orig, " ", "")

	if orig == "" || orig == "-" || orig == "N/A" || orig == "NaN" || orig == "null" {
		return NumberParseResult{Success: false}
	}

	isNegative := false
	isPercent := false

	// Check accounting parentheses: (1,234.56)
	if strings.HasPrefix(orig, "(") && strings.HasSuffix(orig, ")") {
		isNegative = true
		orig = orig[1 : len(orig)-1]
	}

	// Check trailing minus: 123.45-
	if strings.HasSuffix(orig, "-") {
		isNegative = true
		orig = orig[:len(orig)-1]
	} else if strings.HasPrefix(orig, "-") {
		isNegative = true
		orig = orig[1:]
	} else if strings.HasPrefix(orig, "+") {
		orig = orig[1:]
	}

	// Check percentage: 12.5%
	if strings.HasSuffix(orig, "%") {
		isPercent = true
		orig = orig[:len(orig)-1]
	}

	// Strip currency symbols if present
	currInfo := ExtractCurrency(orig)
	if currInfo.HasCurrency {
		orig = currInfo.CleanedText
	}

	lastComma := strings.LastIndex(orig, ",")
	lastDot := strings.LastIndex(orig, ".")

	var canonicalStr string
	decimals := 0

	if localeHint == "eu" || (localeHint == "auto" && lastComma > lastDot && lastComma != -1) {
		// European format: 1.234,56 -> remove dots, replace comma with dot
		cleaned := strings.ReplaceAll(orig, ".", "")
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
		canonicalStr = cleaned
		if lastComma != -1 {
			decimals = len(orig) - lastComma - 1
		}
	} else {
		// US format: 1,234.56 -> remove commas
		cleaned := strings.ReplaceAll(orig, ",", "")
		canonicalStr = cleaned
		if lastDot != -1 {
			decimals = len(orig) - lastDot - 1
		}
	}

	val, err := strconv.ParseFloat(canonicalStr, 64)
	if err != nil {
		return NumberParseResult{Success: false}
	}

	if isNegative {
		val = -val
	}
	if isPercent {
		val = val / 100.0
	}

	// Format canonical string
	var formatted string
	if math.Abs(val-math.Round(val)) < 1e-9 {
		formatted = fmt.Sprintf("%.0f", val)
	} else {
		formatted = strconv.FormatFloat(val, 'f', -1, 64)
	}

	return NumberParseResult{
		Value:      val,
		Canonical:  formatted,
		IsNegative: val < 0,
		Decimals:   decimals,
		Success:    true,
	}
}
