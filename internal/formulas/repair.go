package formulas

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

// FormulaRepairItem documents a repaired formula cell.
type FormulaRepairItem struct {
	SheetName   string `json:"sheet_name"`
	CellRef     string `json:"cell_ref"`
	Original    string `json:"original"`
	Repaired    string `json:"repaired"`
	ErrorReason string `json:"error_reason"`
}

// RepairReport contains all healed formulas and statistics.
type RepairReport struct {
	TotalScanned  int                 `json:"total_scanned"`
	TotalRepaired int                 `json:"total_repaired"`
	Repairs       []FormulaRepairItem `json:"repairs"`
}

// RepairWorkbookFormulas scans all worksheets and heals broken formula syntax.
func RepairWorkbookFormulas(wb *openxml.Workbook) RepairReport {
	report := RepairReport{
		Repairs: make([]FormulaRepairItem, 0),
	}

	for _, sheet := range wb.Sheets {
		for r := 0; r < sheet.MaxRow; r++ {
			for c := 0; c < sheet.MaxCol; c++ {
				cell := &sheet.Grid[r][c]
				if cell.Formula == "" {
					// Check if resolved value itself is an error string like #REF!
					if HasFormulaError(cell.Resolved) {
						report.TotalScanned++
						// Synthesize fallback formula or clear error
						if r > 1 {
							// Try to reconstruct SUM of rows above
							startRef := openxml.CoordsToCellRef(c, 1)
							endRef := openxml.CoordsToCellRef(c, r-1)
							newFormula := fmt.Sprintf("IFERROR(SUM(%s:%s), 0)", startRef, endRef)
							report.Repairs = append(report.Repairs, FormulaRepairItem{
								SheetName:   sheet.Name,
								CellRef:     cell.Ref,
								Original:    cell.Resolved,
								Repaired:    newFormula,
								ErrorReason: "Raw error literal in cell value",
							})
							cell.Formula = newFormula
							report.TotalRepaired++
						}
					}
					continue
				}

				report.TotalScanned++

				if HasFormulaError(cell.Formula) {
					orig := cell.Formula
					repaired := HealFormula(cell.Formula, c, r)
					if repaired != orig {
						report.Repairs = append(report.Repairs, FormulaRepairItem{
							SheetName:   sheet.Name,
							CellRef:     cell.Ref,
							Original:    orig,
							Repaired:    repaired,
							ErrorReason: "Broken formula reference error",
						})
						cell.Formula = repaired
						report.TotalRepaired++
					}
				}
			}
		}
	}

	return report
}

// HealFormula fixes broken tokens in a formula string.
func HealFormula(formula string, col, row int) string {
	clean := CleanFormula(formula)

	// Case 1: SUM(#REF!) or SUM(B2:#REF!)
	if strings.Contains(clean, "#REF!") {
		// Reconstruct range from row 1 to row-1
		if row > 1 {
			startRef := openxml.CoordsToCellRef(col, 1)
			endRef := openxml.CoordsToCellRef(col, row-1)
			validRange := fmt.Sprintf("%s:%s", startRef, endRef)
			clean = strings.ReplaceAll(clean, "#REF!:#REF!", validRange)
			clean = strings.ReplaceAll(clean, ":#REF!", ":"+endRef)
			clean = strings.ReplaceAll(clean, "#REF!:", startRef+":")
			clean = strings.ReplaceAll(clean, "#REF!", validRange)
		} else {
			clean = strings.ReplaceAll(clean, "#REF!", "0")
		}
	}

	// Case 2: Wrap in IFERROR if not already wrapped
	if !strings.HasPrefix(strings.ToUpper(clean), "IFERROR(") {
		clean = fmt.Sprintf("IFERROR(%s, 0)", clean)
	}

	return clean
}
