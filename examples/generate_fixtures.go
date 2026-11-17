package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

func main() {
	outDir := filepath.Join("examples", "messy_sheets")
	_ = os.MkdirAll(outDir, 0755)

	generateSupplierInvoice(filepath.Join(outDir, "messy_supplier_invoice.xlsx"))
	generateMultiTierPnL(filepath.Join(outDir, "multi_tier_pnl.xlsx"))
	generateBrokenFormulas(filepath.Join(outDir, "broken_formulas.xlsx"))

	fmt.Println("Successfully generated all sample test workbooks in examples/messy_sheets/")
}

func generateSupplierInvoice(path string) {
	sheet := &openxml.SheetData{
		Name:   "Invoice_Jan2025",
		MaxRow: 8,
		MaxCol: 4,
		Grid: [][]openxml.Cell{
			// Row 0: Merged title banner
			{
				{Ref: "A1", Resolved: "ACME LOGISTICS GLOBAL - SUPPLIER INVOICE", Type: "s", StyleID: 1},
				{Ref: "B1", Resolved: ""},
				{Ref: "C1", Resolved: ""},
				{Ref: "D1", Resolved: ""},
			},
			// Row 1: Merged header tier 1: "Product Details" over A-B, "Financials" over C-D
			{
				{Ref: "A2", Resolved: "Product Details", Type: "s", StyleID: 1},
				{Ref: "B2", Resolved: ""},
				{Ref: "C2", Resolved: "Financials", Type: "s", StyleID: 1},
				{Ref: "D2", Resolved: ""},
			},
			// Row 2: Sub-headers
			{
				{Ref: "A3", Resolved: "Item Code", Type: "s", StyleID: 1},
				{Ref: "B3", Resolved: "Delivery Date", Type: "s", StyleID: 1},
				{Ref: "C3", Resolved: "Quantity", Type: "s", StyleID: 1},
				{Ref: "D3", Resolved: "Unit Price", Type: "s", StyleID: 1},
			},
			// Row 3: Data 1
			{
				{Ref: "A4", Resolved: "SKU-9901", Type: "s"},
				{Ref: "B4", Resolved: "15/01/2025", Type: "s"}, // EU slash
				{Ref: "C4", Resolved: "10", Type: "n"},
				{Ref: "D4", Resolved: "$1,450.50", Type: "s"}, // US currency string
			},
			// Row 4: Data 2
			{
				{Ref: "A5", Resolved: "SKU-9902", Type: "s"},
				{Ref: "B5", Resolved: "2025-01-22", Type: "s"}, // ISO date
				{Ref: "C5", Resolved: "5", Type: "n"},
				{Ref: "D5", Resolved: "€ 2.100,00", Type: "s"}, // EU comma decimal
			},
			// Row 5: Data 3
			{
				{Ref: "A6", Resolved: "SKU-9903", Type: "s"},
				{Ref: "B6", Resolved: "28.01.2025", Type: "s"}, // EU dot date
				{Ref: "C6", Resolved: "20", Type: "n"},
				{Ref: "D6", Resolved: "750.25 RON", Type: "s"}, // Romanian Leu
			},
			// Row 6: Summary Row (hardcoded static values without formula)
			{
				{Ref: "A7", Resolved: "Total", Type: "s", StyleID: 1},
				{Ref: "B7", Resolved: ""},
				{Ref: "C7", Resolved: "35", Type: "n", StyleID: 1},
				{Ref: "D7", Resolved: "4300.75", Type: "n", StyleID: 1},
			},
			// Row 7: Footnote disclaimer
			{
				{Ref: "A8", Resolved: "* Confidential: For internal auditing and VAT reconciliation only.", Type: "s"},
				{Ref: "B8", Resolved: ""},
				{Ref: "C8", Resolved: ""},
				{Ref: "D8", Resolved: ""},
			},
		},
		MergeCells: []openxml.MergeCell{
			{Ref: "A1:D1", MinCol: 0, MinRow: 0, MaxCol: 3, MaxRow: 0},
			{Ref: "A2:B2", MinCol: 0, MinRow: 1, MaxCol: 1, MaxRow: 1},
			{Ref: "C2:D2", MinCol: 2, MinRow: 1, MaxCol: 3, MaxRow: 1},
		},
	}

	wb := &openxml.Workbook{
		Sheets:        []*openxml.SheetData{sheet},
		SharedStrings: openxml.NewSharedStringsTable(),
	}
	_ = openxml.WriteWorkbookFile(path, wb)
}

func generateMultiTierPnL(path string) {
	sheetBudget := &openxml.SheetData{
		Name:   "Budget_2025",
		MaxRow: 5,
		MaxCol: 2,
		Grid: [][]openxml.Cell{
			{{Ref: "A1", Resolved: "Cost Code", StyleID: 1}, {Ref: "B1", Resolved: "Allocated Budget", StyleID: 1}},
			{{Ref: "A2", Resolved: "CC-101 (Cloud Ops)"}, {Ref: "B2", Resolved: "12000.00", Type: "n"}},
			{{Ref: "A3", Resolved: "CC-102 (Engineering)"}, {Ref: "B3", Resolved: "45000.00", Type: "n"}},
			{{Ref: "A4", Resolved: "CC-103 (Security)"}, {Ref: "B4", Resolved: "8500.00", Type: "n"}},
			{{Ref: "A5", Resolved: "CC-104 (Marketing)"}, {Ref: "B5", Resolved: "15000.00", Type: "n"}},
		},
	}

	sheetActuals := &openxml.SheetData{
		Name:   "Actuals_2025",
		MaxRow: 5,
		MaxCol: 2,
		Grid: [][]openxml.Cell{
			{{Ref: "A1", Resolved: "Cost Code", StyleID: 1}, {Ref: "B1", Resolved: "Actual Spend", StyleID: 1}},
			{{Ref: "A2", Resolved: "CC-101 (Cloud Ops)"}, {Ref: "B2", Resolved: "12000.00", Type: "n"}}, // exact match
			{{Ref: "A3", Resolved: "CC-102 (Engineering)"}, {Ref: "B3", Resolved: "48200.00", Type: "n"}}, // +3200 variance
			{{Ref: "A4", Resolved: "CC-103 (Security)"}, {Ref: "B4", Resolved: "7900.00", Type: "n"}},  // -600 variance
			{{Ref: "A5", Resolved: "CC-105 (Compliance)"}, {Ref: "B5", Resolved: "5000.00", Type: "n"}}, // missing in budget
		},
	}

	wb := &openxml.Workbook{
		Sheets:        []*openxml.SheetData{sheetBudget, sheetActuals},
		SharedStrings: openxml.NewSharedStringsTable(),
	}
	_ = openxml.WriteWorkbookFile(path, wb)
}

func generateBrokenFormulas(path string) {
	sheet := &openxml.SheetData{
		Name:   "Calculations",
		MaxRow: 5,
		MaxCol: 3,
		Grid: [][]openxml.Cell{
			{{Ref: "A1", Resolved: "Region", StyleID: 1}, {Ref: "B1", Resolved: "Revenue", StyleID: 1}, {Ref: "C1", Resolved: "Growth", StyleID: 1}},
			{{Ref: "A2", Resolved: "North"}, {Ref: "B2", Resolved: "1500", Type: "n"}, {Ref: "C2", Resolved: "0.12", Type: "n"}},
			{{Ref: "A3", Resolved: "South"}, {Ref: "B3", Resolved: "2200", Type: "n"}, {Ref: "C3", Resolved: "0.08", Type: "n"}},
			{{Ref: "A4", Resolved: "East"}, {Ref: "B4", Resolved: "1800", Type: "n"}, {Ref: "C4", Resolved: "0.15", Type: "n"}},
			// Row 5 has broken formula references
			{
				{Ref: "A5", Resolved: "Total", StyleID: 1},
				{Ref: "B5", Formula: "SUM(#REF!)", Resolved: "#REF!", Type: "e", StyleID: 1},
				{Ref: "C5", Formula: "AVERAGE(C2:C4)", Resolved: "0.1167", Type: "n", StyleID: 1},
			},
		},
	}

	wb := &openxml.Workbook{
		Sheets:        []*openxml.SheetData{sheet},
		SharedStrings: openxml.NewSharedStringsTable(),
	}
	_ = openxml.WriteWorkbookFile(path, wb)
}
