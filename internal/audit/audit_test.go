package audit

import (
	"testing"

	"github.com/alexandrmotologa/gridpipe/internal/layout"
	"github.com/alexandrmotologa/gridpipe/internal/openxml"
	"github.com/alexandrmotologa/gridpipe/internal/types"
)

func TestDetectAnomaliesAndScorecard(t *testing.T) {
	// Construct sheet with:
	// - 1 statistical outlier (row 9: 99999.00 among normal values ~100)
	// - 1 exact duplicate (row 4 duplicates row 3)
	// - 1 sign inversion (row 6 has -120 in an all-positive column)
	sheet := &openxml.SheetData{
		Name:   "AuditTest",
		MaxRow: 11,
		MaxCol: 2,
		Grid: [][]openxml.Cell{
			{{Resolved: "ID"}, {Resolved: "Amount"}},
			{{Resolved: "TX-01"}, {Resolved: "100.00"}},
			{{Resolved: "TX-02"}, {Resolved: "105.00"}},
			{{Resolved: "TX-03"}, {Resolved: "95.00"}},
			{{Resolved: "TX-03"}, {Resolved: "95.00"}}, // Duplicate of row 3
			{{Resolved: "TX-05"}, {Resolved: "110.00"}},
			{{Resolved: "TX-06"}, {Resolved: "-120.00"}}, // Sign inversion
			{{Resolved: "TX-07"}, {Resolved: "102.00"}},
			{{Resolved: "TX-08"}, {Resolved: "98.00"}},
			{{Resolved: "TX-09"}, {Resolved: "99999.00"}}, // Outlier
			{{Resolved: "TX-10"}, {Resolved: "104.00"}},
		},
	}

	hdr := layout.HeaderDetectionResult{
		HeaderStartRow: 0,
		HeaderEndRow:   0,
		DataStartRow:   1,
		ColumnNames:    []string{"ID", "Amount"},
	}

	bounds := layout.TableBounds{
		MinRow: 1,
		MaxRow: 10,
		MinCol: 0,
		MaxCol: 1,
	}

	colTypes := []types.InferredColumnType{
		{Type: "text"},
		{Type: "float"},
	}

	issues := DetectAnomalies(sheet, hdr, bounds, colTypes)
	if len(issues) < 2 {
		t.Fatalf("expected at least 2 issues detected, got %d", len(issues))
	}

	hasDuplicate := false
	hasOutlier := false
	hasSignInversion := false

	for _, issue := range issues {
		if issue.Category == "DUPLICATE" {
			hasDuplicate = true
		}
		if issue.Category == "OUTLIER" {
			hasOutlier = true
		}
		if issue.Category == "SIGN_INVERSION" {
			hasSignInversion = true
		}
	}

	if !hasDuplicate {
		t.Errorf("expected duplicate issue to be detected")
	}
	if !hasOutlier {
		t.Errorf("expected statistical outlier issue to be detected")
	}
	if !hasSignInversion {
		t.Errorf("expected sign inversion issue to be detected")
	}

	// Compute quality scorecard
	scorecard := ComputeScorecard(sheet, hdr, bounds, colTypes, nil)
	if scorecard.OverallScore <= 0 || scorecard.OverallScore > 100 {
		t.Errorf("invalid overall score: %.2f", scorecard.OverallScore)
	}
	if scorecard.Grade == "" {
		t.Errorf("expected grade to be computed")
	}
	if scorecard.DuplicateRowsCount != 1 {
		t.Errorf("scorecard duplicate count = %d, want 1", scorecard.DuplicateRowsCount)
	}
}
