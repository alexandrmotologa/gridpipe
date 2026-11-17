package layout

import (
	"testing"

	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

func TestHeaderDetectorMultiTier(t *testing.T) {
	// Construct a sheet with 2-tier header:
	// Row 0: "Q1" merged over col 0 and 1, "Q2" merged over col 2 and 3
	// Row 1: "Jan", "Feb", "Apr", "May"
	// Row 2: 100, 200, 300, 400
	sheet := &openxml.SheetData{
		Name:   "Sales",
		MaxRow: 4,
		MaxCol: 4,
		Grid: [][]openxml.Cell{
			{
				{Ref: "A1", Col: 0, Row: 0, Resolved: "Q1"},
				{Ref: "B1", Col: 1, Row: 0, Resolved: ""},
				{Ref: "C1", Col: 2, Row: 0, Resolved: "Q2"},
				{Ref: "D1", Col: 3, Row: 0, Resolved: ""},
			},
			{
				{Ref: "A2", Col: 0, Row: 1, Resolved: "Jan"},
				{Ref: "B2", Col: 1, Row: 1, Resolved: "Feb"},
				{Ref: "C2", Col: 2, Row: 1, Resolved: "Apr"},
				{Ref: "D2", Col: 3, Row: 1, Resolved: "May"},
			},
			{
				{Ref: "A3", Col: 0, Row: 2, Resolved: "100"},
				{Ref: "B3", Col: 1, Row: 2, Resolved: "200"},
				{Ref: "C3", Col: 2, Row: 2, Resolved: "300"},
				{Ref: "D3", Col: 3, Row: 2, Resolved: "400"},
			},
			{
				{Ref: "A4", Col: 0, Row: 3, Resolved: "* Confidential internal note"},
				{Ref: "B4", Col: 1, Row: 3, Resolved: ""},
				{Ref: "C4", Col: 2, Row: 3, Resolved: ""},
				{Ref: "D4", Col: 3, Row: 3, Resolved: ""},
			},
		},
		MergeCells: []openxml.MergeCell{
			{Ref: "A1:B1", MinCol: 0, MinRow: 0, MaxCol: 1, MaxRow: 0},
			{Ref: "C1:D1", MinCol: 2, MinRow: 0, MaxCol: 3, MaxRow: 0},
		},
	}

	headers := DetectHeaders(sheet, 5)
	if headers.HeaderEndRow != 1 {
		t.Errorf("HeaderEndRow = %d, want 1", headers.HeaderEndRow)
	}
	if headers.DataStartRow != 2 {
		t.Errorf("DataStartRow = %d, want 2", headers.DataStartRow)
	}

	expectedCols := []string{"Q1_Jan", "Q1_Feb", "Q2_Apr", "Q2_May"}
	for i, want := range expectedCols {
		if headers.ColumnNames[i] != want {
			t.Errorf("Col %d name = %s, want %s", i, headers.ColumnNames[i], want)
		}
	}

	// Test footnote detection
	footnoteAnalysis := AnalyzeFootnotes(sheet, headers.DataStartRow, sheet.MaxRow-1)
	if len(footnoteAnalysis.Footnotes) != 1 {
		t.Fatalf("expected 1 footnote, got %d", len(footnoteAnalysis.Footnotes))
	}
	if footnoteAnalysis.LastDataRow != 2 {
		t.Errorf("LastDataRow = %d, want 2", footnoteAnalysis.LastDataRow)
	}
}

func TestTableBoundsAndEmptyRows(t *testing.T) {
	sheet := &openxml.SheetData{
		Name:   "BoundsTest",
		MaxRow: 5,
		MaxCol: 4,
		Grid: [][]openxml.Cell{
			// Row 0 empty
			{{Resolved: ""}, {Resolved: ""}, {Resolved: ""}, {Resolved: ""}},
			// Row 1 header
			{{Resolved: "ID"}, {Resolved: "Name"}, {Resolved: ""}, {Resolved: ""}},
			// Row 2 data
			{{Resolved: "1"}, {Resolved: "Alice"}, {Resolved: ""}, {Resolved: ""}},
			// Row 3 data
			{{Resolved: "2"}, {Resolved: "Bob"}, {Resolved: ""}, {Resolved: ""}},
			// Row 4 empty
			{{Resolved: ""}, {Resolved: ""}, {Resolved: ""}, {Resolved: ""}},
		},
	}

	bounds := DetectTableBounds(sheet, 1)
	if bounds.MinRow != 1 || bounds.MaxRow != 3 {
		t.Errorf("row bounds = (%d, %d), want (1, 3)", bounds.MinRow, bounds.MaxRow)
	}
	if bounds.MinCol != 0 || bounds.MaxCol != 1 {
		t.Errorf("col bounds = (%d, %d), want (0, 1)", bounds.MinCol, bounds.MaxCol)
	}
}
