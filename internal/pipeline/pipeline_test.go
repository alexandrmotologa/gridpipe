package pipeline

import (
	"strings"
	"testing"

	"github.com/alexandrmotologa/gridpipe/internal/config"
	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

func TestNormalizationPipeline(t *testing.T) {
	// Construct an unnormalized messy sheet:
	// Row 0: Merged "2025 Financials" spanning col 0-2
	// Row 1: Subheaders: "Item Code", "Invoice Date", "Gross Amount"
	// Row 2: "SKU-01", "15/01/2025", "$1,450.50"
	// Row 3: "SKU-02", "2025-02-10", "€ 2.100,00"
	// Row 4: "Total", "", "3550.50"
	// Row 5: "* Note: Preliminary audit"
	sheet := &openxml.SheetData{
		Name:   "Financials",
		MaxRow: 6,
		MaxCol: 3,
		Grid: [][]openxml.Cell{
			{
				{Ref: "A1", Resolved: "2025 Financials"},
				{Ref: "B1", Resolved: ""},
				{Ref: "C1", Resolved: ""},
			},
			{
				{Ref: "A2", Resolved: "Item Code"},
				{Ref: "B2", Resolved: "Invoice Date"},
				{Ref: "C2", Resolved: "Gross Amount"},
			},
			{
				{Ref: "A3", Resolved: "SKU-01"},
				{Ref: "B3", Resolved: "15/01/2025"},
				{Ref: "C3", Resolved: "$1,450.50", Type: "s"},
			},
			{
				{Ref: "A4", Resolved: "SKU-02"},
				{Ref: "B4", Resolved: "2025-02-10"},
				{Ref: "C4", Resolved: "€ 2.100,00", Type: "s"},
			},
			{
				{Ref: "A5", Resolved: "Total"},
				{Ref: "B5", Resolved: ""},
				{Ref: "C5", Resolved: "3550.50", Type: "n"},
			},
			{
				{Ref: "A6", Resolved: "* Note: Preliminary audit"},
				{Ref: "B6", Resolved: ""},
				{Ref: "C6", Resolved: ""},
			},
		},
		MergeCells: []openxml.MergeCell{
			{Ref: "A1:C1", MinCol: 0, MinRow: 0, MaxCol: 2, MaxRow: 0},
		},
	}

	wb := &openxml.Workbook{
		Sheets:        []*openxml.SheetData{sheet},
		SharedStrings: openxml.NewSharedStringsTable(),
	}

	cfg := config.DefaultPipelineConfig()
	pipe := NewPipeline(cfg)

	events := make([]string, 0)
	pipe.OnEvent = func(e PipelineEvent) {
		events = append(events, e.Stage)
	}

	results, err := pipe.ProcessWorkbook(wb)
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	res := results[0]
	if len(res.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(res.Columns))
	}

	// Check footnotes
	if len(res.Footnotes) != 1 {
		t.Errorf("expected 1 footnote extracted, got %d", len(res.Footnotes))
	}

	// Check date normalization on row 0 (which was row 2 in original)
	dateVal := res.Rows[0][1]
	if dateVal != "2025-01-15" {
		t.Errorf("expected date 2025-01-15, got %s", dateVal)
	}

	// Check currency normalization
	amt1 := res.Rows[0][2]
	if amt1 != "1450.5" && amt1 != "1450.50" {
		t.Errorf("expected numeric string 1450.5, got %s", amt1)
	}

	// Check JSONL export
	jsonlBytes, err := ExportJSONL(res)
	if err != nil {
		t.Fatalf("failed to export JSONL: %v", err)
	}
	if !strings.Contains(string(jsonlBytes), "2025-01-15") {
		t.Errorf("jsonl missing normalized date: %s", string(jsonlBytes))
	}

	// Check SQL export
	sqlText := ExportSQL(res, "test_table")
	if !strings.Contains(sqlText, "CREATE TABLE IF NOT EXISTS test_table") {
		t.Errorf("sql missing create table: %s", sqlText)
	}
}
