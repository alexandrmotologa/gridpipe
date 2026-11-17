package types

import (
	"testing"
)

func TestParseNumberUSandEU(t *testing.T) {
	// Test US format
	resUS := ParseNumber("1,234.56", "us")
	if !resUS.Success || resUS.Value != 1234.56 || resUS.Canonical != "1234.56" {
		t.Errorf("US 1,234.56 failed: %+v", resUS)
	}

	// Test EU format
	resEU := ParseNumber("1.234,56", "eu")
	if !resEU.Success || resEU.Value != 1234.56 || resEU.Canonical != "1234.56" {
		t.Errorf("EU 1.234,56 failed: %+v", resEU)
	}

	// Test Accounting parentheses
	resParen := ParseNumber("(450.75)", "us")
	if !resParen.Success || resParen.Value != -450.75 || !resParen.IsNegative {
		t.Errorf("Accounting (450.75) failed: %+v", resParen)
	}

	// Test Percentage
	resPct := ParseNumber("15.5%", "us")
	if !resPct.Success || resPct.Value != 0.155 {
		t.Errorf("Percentage 15.5%% failed: %+v", resPct)
	}

	// Auto locale detection
	euSamples := []string{"1.500,20", "2.100,50", "300,10"}
	if loc := DetermineColumnLocale(euSamples); loc != "eu" {
		t.Errorf("expected eu locale, got %s", loc)
	}

	usSamples := []string{"1,500.20", "2,100.50", "300.10"}
	if loc := DetermineColumnLocale(usSamples); loc != "us" {
		t.Errorf("expected us locale, got %s", loc)
	}
}

func TestExtractCurrency(t *testing.T) {
	cases := []struct {
		input       string
		wantCode    string
		wantCleaned string
	}{
		{"$1,200.00", "USD", "1,200.00"},
		{"€ 450,50", "EUR", "450,50"},
		{"120.00 RON", "RON", "120.00"},
		{"£99.99", "GBP", "99.99"},
	}

	for _, tc := range cases {
		info := ExtractCurrency(tc.input)
		if info.CurrencyCode != tc.wantCode {
			t.Errorf("ExtractCurrency(%s) code = %s, want %s", tc.input, info.CurrencyCode, tc.wantCode)
		}
		if info.CleanedText != tc.wantCleaned {
			t.Errorf("ExtractCurrency(%s) cleaned = %s, want %s", tc.input, info.CleanedText, tc.wantCleaned)
		}
	}
}

func TestParseDate(t *testing.T) {
	// ISO
	d1 := ParseDate("2025-05-18")
	if !d1.Success || d1.ISO8601 != "2025-05-18" {
		t.Errorf("ISO date failed: %+v", d1)
	}

	// EU dot
	d2 := ParseDate("18.05.2025")
	if !d2.Success || d2.ISO8601 != "2025-05-18" {
		t.Errorf("EU dot date failed: %+v", d2)
	}

	// US slash
	d3 := ParseDate("05/18/2025")
	if !d3.Success || d3.ISO8601 != "2025-05-18" {
		t.Errorf("US slash date failed: %+v", d3)
	}

	// Excel serial number (45672 is 2025-01-15)
	d4 := ParseDate("45672")
	if !d4.Success || d4.ISO8601 != "2025-01-15" {
		t.Errorf("Excel serial failed: %+v (got %s)", d4, d4.ISO8601)
	}
}

func TestInferColumnType(t *testing.T) {
	// Date column
	dateSamples := []string{"2025-01-01", "2025-01-02", "15.01.2025", "N/A"}
	inf := InferColumnType(dateSamples, "auto")
	if inf.Type != "date" {
		t.Errorf("expected date, got %s", inf.Type)
	}

	// Currency column
	currSamples := []string{"$12.50", "$15.00", "$99.90"}
	inf = InferColumnType(currSamples, "us")
	if inf.Type != "currency" || inf.CurrencyCode != "USD" {
		t.Errorf("expected USD currency, got %+v", inf)
	}

	// Integer column
	intSamples := []string{"10", "25", "100", "42"}
	inf = InferColumnType(intSamples, "auto")
	if inf.Type != "integer" {
		t.Errorf("expected integer, got %s", inf.Type)
	}
}
