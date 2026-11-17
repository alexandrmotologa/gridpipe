package types

import (
	"strconv"
	"strings"
	"time"
)

// DateParseResult contains normalized date information.
type DateParseResult struct {
	Time      time.Time `json:"time"`
	ISO8601   string    `json:"iso8601"` // "YYYY-MM-DD"
	Success   bool      `json:"success"`
	RawFormat string    `json:"raw_format"`
}

var commonDateLayouts = []string{
	// ISO
	"2006-01-02",
	"2006-01-02 15:04:05",
	time.RFC3339,
	// European (dot and slash)
	"02.01.2006",
	"02/01/2006",
	"02-01-2006",
	"02.01.06",
	"02/01/06",
	// US (slash and dash)
	"01/02/2006",
	"01-02-2006",
	"01/02/06",
	"1/2/2006",
	"1/2/06",
	// Textual Month
	"02-Jan-2006",
	"02 Jan 2006",
	"Jan 02, 2006",
	"January 02, 2006",
	"02 January 2006",
	"Jan-02-2006",
	"2006/01/02",
}

// ExcelEpoch is 1899-12-30, taking into account Excel's 1900 leap year bug.
var excelEpoch = time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)

// ParseDate converts diverse date expressions or Excel serial numbers into strict ISO-8601 ("YYYY-MM-DD").
func ParseDate(s string) DateParseResult {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || trimmed == "N/A" || trimmed == "null" {
		return DateParseResult{Success: false}
	}

	// 1. Check if string is an Excel serial number (e.g. 45000 to 55000)
	if serial, err := strconv.ParseFloat(trimmed, 64); err == nil && serial >= 1.0 && serial <= 100000.0 {
		days := int(serial)
		fraction := serial - float64(days)

		t := excelEpoch.AddDate(0, 0, days)
		if fraction > 0 {
			duration := time.Duration(fraction * float64(24*time.Hour))
			t = t.Add(duration)
		}

		// Only accept if year is between 1980 and 2100 to avoid false positives for regular numbers like 100
		if t.Year() >= 1980 && t.Year() <= 2100 {
			return DateParseResult{
				Time:      t,
				ISO8601:   t.Format("2006-01-02"),
				Success:   true,
				RawFormat: "excel_serial",
			}
		}
	}

	// 2. Try standard layouts
	for _, layout := range commonDateLayouts {
		if t, err := time.Parse(layout, trimmed); err == nil {
			// Normalise 2-digit years if needed
			if t.Year() < 100 {
				if t.Year() < 70 {
					t = t.AddDate(2000, 0, 0)
				} else {
					t = t.AddDate(1900, 0, 0)
				}
			}
			return DateParseResult{
				Time:      t,
				ISO8601:   t.Format("2006-01-02"),
				Success:   true,
				RawFormat: layout,
			}
		}
	}

	return DateParseResult{Success: false}
}

// FormatISO8601 converts a time.Time into YYYY-MM-DD.
func FormatISO8601(t time.Time) string {
	return t.Format("2006-01-02")
}

// IsLikelyDateColumn analyzes sample strings to determine if column contains dates.
func IsLikelyDateColumn(samples []string) bool {
	if len(samples) == 0 {
		return false
	}
	validDates := 0
	nonEmpty := 0
	for _, s := range samples {
		if strings.TrimSpace(s) == "" {
			continue
		}
		nonEmpty++
		res := ParseDate(s)
		if res.Success {
			validDates++
		}
	}
	if nonEmpty == 0 {
		return false
	}
	return float64(validDates)/float64(nonEmpty) >= 0.70
}
