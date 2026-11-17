package reconcile

import (
	"strings"
	"testing"

	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

func TestReconcileSheets(t *testing.T) {
	// Sheet A (Budget)
	sheetA := &openxml.SheetData{
		Name:   "Budget",
		MaxRow: 4,
		MaxCol: 2,
		Grid: [][]openxml.Cell{
			{{Resolved: "SKU"}, {Resolved: "BudgetAmount"}},
			{{Resolved: "A100"}, {Resolved: "1000.00"}},
			{{Resolved: "B200"}, {Resolved: "2500.00"}},
			{{Resolved: "C300"}, {Resolved: "500.00"}},
		},
	}

	// Sheet B (Actuals)
	// A100 is exact match, B200 has discrepancy, C300 is missing, D400 is new
	sheetB := &openxml.SheetData{
		Name:   "Actuals",
		MaxRow: 4,
		MaxCol: 2,
		Grid: [][]openxml.Cell{
			{{Resolved: "SKU"}, {Resolved: "ActualAmount"}},
			{{Resolved: "A100"}, {Resolved: "1000.00"}},
			{{Resolved: "B200"}, {Resolved: "2750.00"}}, // +250 variance
			{{Resolved: "D400"}, {Resolved: "400.00"}},  // Missing in A
		},
	}

	summary := ReconcileSheets(
		sheetA, 0, 1, 1,
		sheetB, 0, 1, 1,
		0.01,
	)

	if summary.MatchedRows != 1 {
		t.Errorf("matched rows = %d, want 1", summary.MatchedRows)
	}
	if summary.Discrepancies != 1 {
		t.Errorf("discrepancies = %d, want 1", summary.Discrepancies)
	}
	if summary.MissingInB != 1 {
		t.Errorf("missing in B = %d, want 1 (C300)", summary.MissingInB)
	}
	if summary.MissingInA != 1 {
		t.Errorf("missing in A = %d, want 1 (D400)", summary.MissingInA)
	}

	reconciledSheet := GenerateReconciledSheet(summary, "AuditReport")
	if reconciledSheet.MaxRow != len(summary.Items)+2 {
		t.Errorf("expected max row %d, got %d", len(summary.Items)+2, reconciledSheet.MaxRow)
	}

	// Check delta formula on row 1
	if !strings.Contains(reconciledSheet.Grid[1][6].Formula, "F2-E2") {
		t.Errorf("row 1 delta formula = %s, want F2-E2", reconciledSheet.Grid[1][6].Formula)
	}
}

func TestFuzzyReconciliationAndCompositeKeys(t *testing.T) {
	// Sheet A (Vendor Invoice)
	// Has slight typo: "ACME-WIDGET-99" instead of "ACME_WIDGET_99"
	sheetA := &openxml.SheetData{
		Name:   "Vendor",
		MaxRow: 3,
		MaxCol: 3,
		Grid: [][]openxml.Cell{
			{{Resolved: "Category"}, {Resolved: "PartNumber"}, {Resolved: "Total"}},
			{{Resolved: "Hardware"}, {Resolved: "ACME-WIDGET-99"}, {Resolved: "150.00"}},
			{{Resolved: "Tools"}, {Resolved: "DRILL-X1"}, {Resolved: "80.00"}},
		},
	}

	// Sheet B (Internal PO)
	// Uses underscore: "ACME_WIDGET_99"
	sheetB := &openxml.SheetData{
		Name:   "InternalPO",
		MaxRow: 3,
		MaxCol: 3,
		Grid: [][]openxml.Cell{
			{{Resolved: "Category"}, {Resolved: "PartNumber"}, {Resolved: "Total"}},
			{{Resolved: "Hardware"}, {Resolved: "ACME_WIDGET_99"}, {Resolved: "150.00"}},
			{{Resolved: "Tools"}, {Resolved: "DRILL-X1"}, {Resolved: "80.00"}},
		},
	}

	// Test with composite keys: Col 0 (Category) + Col 1 (PartNumber) and Fuzzy matching enabled
	opts := ReconcileOptions{
		KeyColsA:       []int{0, 1},
		ValColA:        2,
		StartRowA:      1,
		KeyColsB:       []int{0, 1},
		ValColB:        2,
		StartRowB:      1,
		Epsilon:        0.01,
		EnableFuzzy:    true,
		FuzzyThreshold: 0.85,
	}

	summary := ReconcileSheetsWithOptions(sheetA, sheetB, opts)

	if summary.MatchedRows != 2 {
		t.Errorf("matched rows = %d, want 2 (1 exact + 1 fuzzy)", summary.MatchedRows)
	}
	if summary.FuzzyMatches != 1 {
		t.Errorf("fuzzy matches = %d, want 1", summary.FuzzyMatches)
	}
	if summary.Discrepancies != 0 {
		t.Errorf("discrepancies = %d, want 0", summary.Discrepancies)
	}
}
