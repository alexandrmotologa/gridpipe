package layout

import (
	"strings"

	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

// TableBounds defines rectangular boundary indices for a data region.
type TableBounds struct {
	MinRow int `json:"min_row"`
	MaxRow int `json:"max_row"`
	MinCol int `json:"min_col"`
	MaxCol int `json:"max_col"`
}

// DetectTableBounds finds the tight bounding box containing data rows and active columns.
func DetectTableBounds(sheet *openxml.SheetData, dataStartRow int) TableBounds {
	if sheet.MaxRow == 0 || sheet.MaxCol == 0 {
		return TableBounds{0, 0, 0, 0}
	}

	minCol := sheet.MaxCol
	maxCol := -1
	minRow := dataStartRow
	maxRow := -1

	for r := dataStartRow; r < sheet.MaxRow; r++ {
		rowHasData := false
		for c := 0; c < sheet.MaxCol; c++ {
			val := strings.TrimSpace(sheet.Grid[r][c].Resolved)
			if val != "" {
				rowHasData = true
				if c < minCol {
					minCol = c
				}
				if c > maxCol {
					maxCol = c
				}
			}
		}
		if rowHasData {
			if minRow == -1 {
				minRow = r
			}
			maxRow = r
		}
	}

	if maxCol == -1 {
		// Sheet was empty in data region
		return TableBounds{
			MinRow: dataStartRow,
			MaxRow: dataStartRow,
			MinCol: 0,
			MaxCol: sheet.MaxCol - 1,
		}
	}

	return TableBounds{
		MinRow: minRow,
		MaxRow: maxRow,
		MinCol: minCol,
		MaxCol: maxCol,
	}
}

// IsRowEmpty returns true if all cells in a row are blank.
func IsRowEmpty(row []openxml.Cell) bool {
	for _, c := range row {
		if strings.TrimSpace(c.Resolved) != "" {
			return false
		}
	}
	return true
}

// IsColEmpty returns true if all cells in a column across the given row range are blank.
func IsColEmpty(grid [][]openxml.Cell, col, startRow, endRow int) bool {
	for r := startRow; r <= endRow && r < len(grid); r++ {
		if col < len(grid[r]) && strings.TrimSpace(grid[r][col].Resolved) != "" {
			return false
		}
	}
	return true
}
