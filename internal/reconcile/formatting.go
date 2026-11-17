package reconcile

import (
	"fmt"

	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

// GenerateReconciledSheet creates a new SheetData containing side-by-side comparison,
// formulas, and conditional styling IDs.
func GenerateReconciledSheet(summary ReconciliationSummary, sheetName string) *openxml.SheetData {
	if sheetName == "" {
		sheetName = "Reconciliation"
	}

	headers := []string{"Line Item Key", "Source (A)", "Comparison (B)", "Variance (B - A)", "% Delta", "Status", "Audit Note"}
	numRows := len(summary.Items) + 2 // header + data + totals row
	numCols := len(headers)

	sheet := &openxml.SheetData{
		Name:    sheetName,
		SheetID: "99",
		MaxRow:  numRows,
		MaxCol:  numCols,
		Grid:    make([][]openxml.Cell, numRows),
	}

	for r := 0; r < numRows; r++ {
		sheet.Grid[r] = make([]openxml.Cell, numCols)
		for c := 0; c < numCols; c++ {
			sheet.Grid[r][c] = openxml.Cell{
				Ref: openxml.CoordsToCellRef(c, r),
				Col: c,
				Row: r,
			}
		}
	}

	// 1. Header row
	for c, h := range headers {
		sheet.Grid[0][c].Resolved = h
		sheet.Grid[0][c].Type = "s"
		sheet.Grid[0][c].StyleID = 1 // Bold header
	}

	// 2. Data rows
	for i, item := range summary.Items {
		r := i + 1
		cellA := openxml.CoordsToCellRef(1, r)
		cellB := openxml.CoordsToCellRef(2, r)
		cellDelta := openxml.CoordsToCellRef(3, r)

		styleID := 0
		if item.Status == StatusMatch {
			styleID = 2 // Soft Green
		} else {
			styleID = 3 // Soft Red
		}

		// Col 0: Key
		sheet.Grid[r][0].Resolved = item.Key
		sheet.Grid[r][0].Type = "s"

		// Col 1: Value A
		sheet.Grid[r][1].Resolved = fmt.Sprintf("%.2f", item.ValueA)
		sheet.Grid[r][1].Type = "n"

		// Col 2: Value B
		sheet.Grid[r][2].Resolved = fmt.Sprintf("%.2f", item.ValueB)
		sheet.Grid[r][2].Type = "n"

		// Col 3: Delta formula = C2 - B2
		sheet.Grid[r][3].Formula = fmt.Sprintf("%s-%s", cellB, cellA)
		sheet.Grid[r][3].Resolved = fmt.Sprintf("%.2f", item.Delta)
		sheet.Grid[r][3].Type = "n"
		sheet.Grid[r][3].StyleID = styleID

		// Col 4: Pct formula = (Delta / ValueA)
		if item.ValueA != 0 {
			sheet.Grid[r][4].Formula = fmt.Sprintf("IFERROR(%s/%s*100, 0)", cellDelta, cellA)
			sheet.Grid[r][4].Resolved = fmt.Sprintf("%.1f%%", item.PercentDelta)
		} else {
			sheet.Grid[r][4].Resolved = "N/A"
		}
		sheet.Grid[r][4].Type = "n"
		sheet.Grid[r][4].StyleID = styleID

		// Col 5: Status
		sheet.Grid[r][5].Resolved = string(item.Status)
		sheet.Grid[r][5].Type = "s"
		sheet.Grid[r][5].StyleID = styleID

		// Col 6: Audit Note
		sheet.Grid[r][6].Resolved = item.AuditNote
		sheet.Grid[r][6].Type = "s"
	}

	// 3. Totals row at bottom
	totRow := numRows - 1
	startA := openxml.CoordsToCellRef(1, 1)
	endA := openxml.CoordsToCellRef(1, totRow-1)
	startB := openxml.CoordsToCellRef(2, 1)
	endB := openxml.CoordsToCellRef(2, totRow-1)
	startDelta := openxml.CoordsToCellRef(3, 1)
	endDelta := openxml.CoordsToCellRef(3, totRow-1)

	sheet.Grid[totRow][0].Resolved = "Grand Total"
	sheet.Grid[totRow][0].Type = "s"
	sheet.Grid[totRow][0].StyleID = 1

	sheet.Grid[totRow][1].Formula = fmt.Sprintf("SUM(%s:%s)", startA, endA)
	sheet.Grid[totRow][1].Resolved = fmt.Sprintf("%.2f", summary.SumValueA)
	sheet.Grid[totRow][1].Type = "n"
	sheet.Grid[totRow][1].StyleID = 1

	sheet.Grid[totRow][2].Formula = fmt.Sprintf("SUM(%s:%s)", startB, endB)
	sheet.Grid[totRow][2].Resolved = fmt.Sprintf("%.2f", summary.SumValueB)
	sheet.Grid[totRow][2].Type = "n"
	sheet.Grid[totRow][2].StyleID = 1

	sheet.Grid[totRow][3].Formula = fmt.Sprintf("SUM(%s:%s)", startDelta, endDelta)
	sheet.Grid[totRow][3].Resolved = fmt.Sprintf("%.2f", summary.NetDelta)
	sheet.Grid[totRow][3].Type = "n"
	sheet.Grid[totRow][3].StyleID = 1

	return sheet
}
