package openxml

import (
	"bytes"
	"testing"
)

func TestColCoordinates(t *testing.T) {
	cases := []struct {
		col     int
		letters string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
		{51, "AZ"},
		{52, "BA"},
		{701, "ZZ"},
		{702, "AAA"},
	}

	for _, tc := range cases {
		gotLetters := ColIndexToLetters(tc.col)
		if gotLetters != tc.letters {
			t.Errorf("ColIndexToLetters(%d) = %s, want %s", tc.col, gotLetters, tc.letters)
		}
		gotCol := ColLettersToIndex(tc.letters)
		if gotCol != tc.col {
			t.Errorf("ColLettersToIndex(%s) = %d, want %d", tc.letters, gotCol, tc.col)
		}
	}
}

func TestCellRefToCoords(t *testing.T) {
	col, row, err := CellRefToCoords("B4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if col != 1 || row != 3 {
		t.Errorf("CellRefToCoords(B4) = (%d, %d), want (1, 3)", col, row)
	}

	ref := CoordsToCellRef(1, 3)
	if ref != "B4" {
		t.Errorf("CoordsToCellRef(1, 3) = %s, want B4", ref)
	}

	col, row, err = CellRefToCoords("AA100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if col != 26 || row != 99 {
		t.Errorf("CellRefToCoords(AA100) = (%d, %d), want (26, 99)", col, row)
	}
}

func TestSharedStringsTable(t *testing.T) {
	sst := NewSharedStringsTable()
	idx1 := sst.Add("Hello")
	idx2 := sst.Add("World")
	idx3 := sst.Add("Hello")

	if idx1 != 0 {
		t.Errorf("expected idx1 = 0, got %d", idx1)
	}
	if idx2 != 1 {
		t.Errorf("expected idx2 = 1, got %d", idx2)
	}
	if idx3 != 0 {
		t.Errorf("expected idx3 = 0 for duplicate, got %d", idx3)
	}

	if sst.Len() != 2 {
		t.Errorf("expected len = 2, got %d", sst.Len())
	}

	val, ok := sst.Get(1)
	if !ok || val != "World" {
		t.Errorf("expected World, got %s", val)
	}

	xmlBytes, err := sst.EncodeXML()
	if err != nil {
		t.Fatalf("failed to encode XML: %v", err)
	}

	parsedSST, err := ParseSharedStrings(bytes.NewReader(xmlBytes))
	if err != nil {
		t.Fatalf("failed to parse generated sharedStrings XML: %v", err)
	}

	if parsedSST.Len() != 2 {
		t.Errorf("parsed SST length = %d, want 2", parsedSST.Len())
	}
}

func TestWorkbookRoundTrip(t *testing.T) {
	// Construct test sheet
	sheet := &SheetData{
		Name:    "TestSheet",
		SheetID: "1",
		MaxRow:  3,
		MaxCol:  3,
		Grid: [][]Cell{
			{
				{Ref: "A1", Col: 0, Row: 0, Resolved: "Item", Type: "s"},
				{Ref: "B1", Col: 1, Row: 0, Resolved: "Qty", Type: "s"},
				{Ref: "C1", Col: 2, Row: 0, Resolved: "Price", Type: "s"},
			},
			{
				{Ref: "A2", Col: 0, Row: 1, Resolved: "Widget A", Type: "s"},
				{Ref: "B2", Col: 1, Row: 1, Resolved: "10", Type: "n"},
				{Ref: "C2", Col: 2, Row: 1, Resolved: "25.5", Type: "n"},
			},
			{
				{Ref: "A3", Col: 0, Row: 2, Resolved: "Total", Type: "s"},
				{Ref: "B3", Col: 1, Row: 2, Resolved: "10", Formula: "SUM(B2:B2)", Type: "n"},
				{Ref: "C3", Col: 2, Row: 2, Resolved: "25.5", Formula: "SUM(C2:C2)", Type: "n"},
			},
		},
		MergeCells: []MergeCell{
			{Ref: "A1:B1", MinCol: 0, MinRow: 0, MaxCol: 1, MaxRow: 0},
		},
	}

	wb := &Workbook{
		Sheets:        []*SheetData{sheet},
		SharedStrings: NewSharedStringsTable(),
	}

	var buf bytes.Buffer
	err := WriteWorkbook(&buf, wb)
	if err != nil {
		t.Fatalf("failed to write workbook: %v", err)
	}

	readWb, err := ReadWorkbookFromBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("failed to read workbook: %v", err)
	}

	if len(readWb.Sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(readWb.Sheets))
	}

	rs := readWb.Sheets[0]
	if rs.Name != "TestSheet" {
		t.Errorf("sheet name = %s, want TestSheet", rs.Name)
	}
	if len(rs.MergeCells) != 1 || rs.MergeCells[0].Ref != "A1:B1" {
		t.Errorf("merge cells mismatch: %+v", rs.MergeCells)
	}

	if rs.Grid[0][0].Resolved != "Item" {
		t.Errorf("A1 resolved = %s, want Item", rs.Grid[0][0].Resolved)
	}
	if rs.Grid[1][1].Resolved != "10" {
		t.Errorf("B2 resolved = %s, want 10", rs.Grid[1][1].Resolved)
	}
	if rs.Grid[2][1].Formula != "SUM(B2:B2)" {
		t.Errorf("B3 formula = %s, want SUM(B2:B2)", rs.Grid[2][1].Formula)
	}
}

func TestCSVDelimiterDetection(t *testing.T) {
	semicolonCSV := []byte("ID;Name;Price\n1;Widget;12.50\n2;Gadget;19.99\n")
	delim := DetectDelimiter(semicolonCSV)
	if delim != ';' {
		t.Errorf("expected delimiter ';', got '%c'", delim)
	}

	commaCSV := []byte("ID,Name,Price\n1,Widget,12.50\n2,Gadget,19.99\n")
	delim = DetectDelimiter(commaCSV)
	if delim != ',' {
		t.Errorf("expected delimiter ',', got '%c'", delim)
	}

	sheet, err := ReadCSV(bytes.NewReader(semicolonCSV), "Sample")
	if err != nil {
		t.Fatalf("failed to read CSV: %v", err)
	}
	if sheet.MaxRow != 3 || sheet.MaxCol != 3 {
		t.Errorf("sheet dimensions = (%d, %d), want (3, 3)", sheet.MaxRow, sheet.MaxCol)
	}
	if sheet.Grid[0][1].Resolved != "Name" {
		t.Errorf("grid[0][1] = %s, want Name", sheet.Grid[0][1].Resolved)
	}
}
