package audit

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/alexandrmotologa/gridpipe/internal/layout"
	"github.com/alexandrmotologa/gridpipe/internal/openxml"
	"github.com/alexandrmotologa/gridpipe/internal/types"
)

// AnomalySeverity indicates urgency level.
type AnomalySeverity string

const (
	SeverityHigh   AnomalySeverity = "HIGH"
	SeverityMedium AnomalySeverity = "MEDIUM"
	SeverityLow    AnomalySeverity = "LOW"
)

// AnomalyIssue represents a single data quality or business logic violation.
type AnomalyIssue struct {
	SheetName   string          `json:"sheet_name"`
	CellRef     string          `json:"cell_ref"`
	Row         int             `json:"row"`
	Col         int             `json:"col"`
	ColumnName  string          `json:"column_name"`
	Value       string          `json:"value"`
	Category    string          `json:"category"` // OUTLIER, DUPLICATE, SIGN_INVERSION, TYPE_VIOLATION
	Severity    AnomalySeverity `json:"severity"`
	Description string          `json:"description"`
	Suggested   string          `json:"suggested,omitempty"`
}

// DetectAnomalies runs comprehensive quality and anomaly checks across a worksheet.
func DetectAnomalies(sheet *openxml.SheetData, hdr layout.HeaderDetectionResult, bounds layout.TableBounds, colTypes []types.InferredColumnType) []AnomalyIssue {
	issues := make([]AnomalyIssue, 0)
	if sheet.MaxRow == 0 || sheet.MaxCol == 0 {
		return issues
	}

	startRow := bounds.MinRow
	endRow := bounds.MaxRow
	numRows := endRow - startRow + 1
	if numRows <= 0 {
		return issues
	}

	// 1. Check Duplicates across entire row
	seenRows := make(map[string][]int) // hash -> row indices
	for r := startRow; r <= endRow; r++ {
		rowVals := make([]string, len(hdr.ColumnNames))
		for c := 0; c < len(hdr.ColumnNames); c++ {
			origCol := bounds.MinCol + c
			if origCol < len(sheet.Grid[r]) {
				rowVals[c] = strings.TrimSpace(sheet.Grid[r][origCol].Resolved)
			}
		}
		hash := strings.Join(rowVals, "||")
		seenRows[hash] = append(seenRows[hash], r)
	}

	for _, rowIndices := range seenRows {
		if len(rowIndices) > 1 {
			for i := 1; i < len(rowIndices); i++ {
				dupRow := rowIndices[i]
				issues = append(issues, AnomalyIssue{
					SheetName:   sheet.Name,
					CellRef:     openxml.CoordsToCellRef(bounds.MinCol, dupRow),
					Row:         dupRow,
					Col:         bounds.MinCol,
					ColumnName:  hdr.ColumnNames[0],
					Category:    "DUPLICATE",
					Severity:    SeverityMedium,
					Description: fmt.Sprintf("Row %d is an exact duplicate of row %d", dupRow+1, rowIndices[0]+1),
				})
			}
		}
	}

	// 2. Check each column for type violations, outliers, and sign inversions
	for c, colName := range hdr.ColumnNames {
		origCol := bounds.MinCol + c
		colType := "text"
		if c < len(colTypes) {
			colType = colTypes[c].Type
		}

		numericValues := make([]float64, 0, numRows)
		numericRows := make([]int, 0, numRows)
		positiveCount := 0
		negativeCount := 0

		for r := startRow; r <= endRow; r++ {
			if origCol >= len(sheet.Grid[r]) {
				continue
			}
			cell := sheet.Grid[r][origCol]
			rawVal := strings.TrimSpace(cell.Resolved)
			if rawVal == "" {
				continue
			}

			// Type checking
			switch colType {
			case "date":
				dRes := types.ParseDate(rawVal)
				if !dRes.Success {
					issues = append(issues, AnomalyIssue{
						SheetName:   sheet.Name,
						CellRef:     cell.Ref,
						Row:         r,
						Col:         origCol,
						ColumnName:  colName,
						Value:       rawVal,
						Category:    "TYPE_VIOLATION",
						Severity:    SeverityHigh,
						Description: fmt.Sprintf("Value '%s' does not conform to date format", rawVal),
					})
				}
			case "integer", "float", "currency":
				numRes := types.ParseNumber(rawVal, "auto")
				if !numRes.Success {
					issues = append(issues, AnomalyIssue{
						SheetName:   sheet.Name,
						CellRef:     cell.Ref,
						Row:         r,
						Col:         origCol,
						ColumnName:  colName,
						Value:       rawVal,
						Category:    "TYPE_VIOLATION",
						Severity:    SeverityHigh,
						Description: fmt.Sprintf("Value '%s' is not a valid number", rawVal),
					})
				} else {
					numericValues = append(numericValues, numRes.Value)
					numericRows = append(numericRows, r)
					if numRes.Value > 0 {
						positiveCount++
					} else if numRes.Value < 0 {
						negativeCount++
					}
				}
			}
		}

		// 3. Statistical Outliers via IQR
		if len(numericValues) >= 8 {
			sortedVals := make([]float64, len(numericValues))
			copy(sortedVals, numericValues)
			sort.Float64s(sortedVals)

			q1 := sortedVals[len(sortedVals)/4]
			q3 := sortedVals[(len(sortedVals)*3)/4]
			iqr := q3 - q1
			lowerBound := q1 - 1.5*iqr
			upperBound := q3 + 1.5*iqr

			for idx, v := range numericValues {
				if v < lowerBound || v > upperBound {
					r := numericRows[idx]
					cellRef := openxml.CoordsToCellRef(origCol, r)
					issues = append(issues, AnomalyIssue{
						SheetName:   sheet.Name,
						CellRef:     cellRef,
						Row:         r,
						Col:         origCol,
						ColumnName:  colName,
						Value:       fmt.Sprintf("%.2f", v),
						Category:    "OUTLIER",
						Severity:    SeverityLow,
						Description: fmt.Sprintf("Statistical outlier: value %.2f is outside IQR expected range [%.2f, %.2f]", v, lowerBound, upperBound),
					})
				}
			}
		}

		// 4. Sign Inversion Check (e.g. >= 85% positive and rare negative, or vice versa)
		totNumeric := positiveCount + negativeCount
		if totNumeric >= 5 {
			if positiveCount > 0 && negativeCount > 0 {
				ratioNeg := float64(negativeCount) / float64(totNumeric)
				ratioPos := float64(positiveCount) / float64(totNumeric)

				if (ratioNeg <= 0.15 || negativeCount == 1) && positiveCount >= 4 {
					// Rare negative numbers in overwhelmingly positive column
					for idx, v := range numericValues {
						if v < 0 {
							r := numericRows[idx]
							issues = append(issues, AnomalyIssue{
								SheetName:   sheet.Name,
								CellRef:     openxml.CoordsToCellRef(origCol, r),
								Row:         r,
								Col:         origCol,
								ColumnName:  colName,
								Value:       fmt.Sprintf("%.2f", v),
								Category:    "SIGN_INVERSION",
								Severity:    SeverityMedium,
								Description: fmt.Sprintf("Potential sign inversion: negative value %.2f in predominantly positive column (%.0f%%)", v, ratioPos*100),
								Suggested:   fmt.Sprintf("%.2f", math.Abs(v)),
							})
						}
					}
				} else if (ratioPos <= 0.15 || positiveCount == 1) && negativeCount >= 4 {
					// Rare positive numbers in overwhelmingly negative column
					for idx, v := range numericValues {
						if v > 0 {
							r := numericRows[idx]
							issues = append(issues, AnomalyIssue{
								SheetName:   sheet.Name,
								CellRef:     openxml.CoordsToCellRef(origCol, r),
								Row:         r,
								Col:         origCol,
								ColumnName:  colName,
								Value:       fmt.Sprintf("%.2f", v),
								Category:    "SIGN_INVERSION",
								Severity:    SeverityMedium,
								Description: fmt.Sprintf("Potential sign inversion: positive value %.2f in predominantly negative column (%.0f%%)", v, ratioNeg*100),
								Suggested:   fmt.Sprintf("-%.2f", v),
							})
						}
					}
				}
			}
		}
	}

	return issues
}
