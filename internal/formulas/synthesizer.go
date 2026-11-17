package formulas

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/gridpipe/internal/layout"
	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

// SynthesisOptions configures automatic formula generation.
type SynthesisOptions struct {
	SynthesizeSums     bool `json:"synthesize_sums"`
	SynthesizeAverages bool `json:"synthesize_averages"`
	UseXLookup         bool `json:"use_xlookup"`
	WrapWithIfError    bool `json:"wrap_with_if_error"`
}

// DefaultSynthesisOptions returns standard options.
func DefaultSynthesisOptions() SynthesisOptions {
	return SynthesisOptions{
		SynthesizeSums:     true,
		SynthesizeAverages: true,
		UseXLookup:         true,
		WrapWithIfError:    true,
	}
}

// SynthesizeFormulas scans a worksheet's summary rows and injects native Excel formulas
// into OpenXML cell formula definitions.
func SynthesizeFormulas(sheet *openxml.SheetData, dataStartRow, dataEndRow int, totalRows []int, opts SynthesisOptions) int {
	if !opts.SynthesizeSums && !opts.SynthesizeAverages {
		return 0
	}

	formulasCreated := 0

	for _, totRow := range totalRows {
		if totRow < 0 || totRow >= sheet.MaxRow {
			continue
		}

		// Check first cell in this row to see if it's Sum or Average
		rowLabel := strings.ToLower(sheet.Grid[totRow][0].Resolved)
		isAverage := strings.Contains(rowLabel, "average") || strings.Contains(rowLabel, "avg")

		// Iterate through numeric columns
		for c := 0; c < sheet.MaxCol; c++ {
			// Skip column 0 if it holds the "Total" label
			if c == 0 && strings.TrimSpace(sheet.Grid[totRow][c].Resolved) != "" {
				continue
			}

			// Check if column above has numeric data
			numCount := 0
			for r := dataStartRow; r <= dataEndRow; r++ {
				if r == totRow {
					continue
				}
				val := strings.TrimSpace(sheet.Grid[r][c].Resolved)
				if val != "" && sheet.Grid[r][c].Type == "n" {
					numCount++
				}
			}

			// If majority of column data rows are numeric, synthesize formula
			if numCount > 0 {
				startRef := openxml.CoordsToCellRef(c, dataStartRow)
				endRef := openxml.CoordsToCellRef(c, dataEndRow)

				var formula string
				if isAverage && opts.SynthesizeAverages {
					formula = fmt.Sprintf("AVERAGE(%s:%s)", startRef, endRef)
				} else if opts.SynthesizeSums {
					formula = fmt.Sprintf("SUM(%s:%s)", startRef, endRef)
				}

				if formula != "" {
					if opts.WrapWithIfError {
						formula = fmt.Sprintf("IFERROR(%s, 0)", formula)
					}
					sheet.Grid[totRow][c].Formula = formula
					formulasCreated++
				}
			}
		}
	}

	return formulasCreated
}

// SynthesizeXLookup generates an XLOOKUP formula for cross-sheet lookup.
// e.g. =XLOOKUP(A2, 'Vendors'!$A$2:$A$500, 'Vendors'!$C$2:$C$500, "Not Found")
func SynthesizeXLookup(keyCell string, targetSheet string, keyColLetters string, valColLetters string, startRow, endRow int) string {
	lookupRange := fmt.Sprintf("'%s'!$%s$%d:$%s$%d", targetSheet, keyColLetters, startRow+1, keyColLetters, endRow+1)
	returnRange := fmt.Sprintf("'%s'!$%s$%d:$%s$%d", targetSheet, valColLetters, startRow+1, valColLetters, endRow+1)
	return fmt.Sprintf("IFERROR(XLOOKUP(%s, %s, %s, \"Not Found\"), \"Not Found\")", keyCell, lookupRange, returnRange)
}

// AutoSynthesizeGrid detects table geometry and injects formulas on summary rows.
func AutoSynthesizeGrid(sheet *openxml.SheetData, hdr layout.HeaderDetectionResult, opts SynthesisOptions) int {
	bounds := layout.DetectTableBounds(sheet, hdr.DataStartRow)
	footnoteInfo := layout.AnalyzeFootnotes(sheet, bounds.MinRow, bounds.MaxRow)
	return SynthesizeFormulas(sheet, bounds.MinRow, footnoteInfo.LastDataRow, footnoteInfo.TotalRows, opts)
}
