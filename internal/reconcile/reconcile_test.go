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
	if !strings.Contains(reconciledSheet.Grid[1][3].Formula, "C2-B2") {
		t.Errorf("row 1 delta formula = %s, want C2-B2", reconciledSheet.Grid[1][3].Formula)
	}
}
