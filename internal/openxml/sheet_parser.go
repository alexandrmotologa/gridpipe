package openxml

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Cell represents an individual cell within a spreadsheet.
type Cell struct {
	Ref       string `json:"ref"`       // e.g. "B4"
	Col       int    `json:"col"`       // 0-based column index
	Row       int    `json:"row"`       // 0-based row index
	Type      string `json:"type"`      // s, n, b, str, inlineStr, e, d
	RawValue  string `json:"raw_value"` // Raw string from XML <v> node
	Formula   string `json:"formula"`   // Formula expression from XML <f> node
	StyleID   int    `json:"style_id"`
	Resolved  string `json:"resolved"`  // Decoded text value (from sharedStrings or raw)
}

// MergeCell represents a range of merged cells, e.g. "A1:C2".
type MergeCell struct {
	Ref    string `json:"ref"`
	MinCol int    `json:"min_col"`
	MinRow int    `json:"min_row"`
	MaxCol int    `json:"max_col"`
	MaxRow int    `json:"max_row"`
}

// SheetData holds the tabular grid and metadata for a single worksheet tab.
type SheetData struct {
	Name       string      `json:"name"`
	SheetID    string      `json:"sheet_id"`
	MaxRow     int         `json:"max_row"`
	MaxCol     int         `json:"max_col"`
	Grid       [][]Cell    `json:"grid"` // Grid[row][col]
	MergeCells []MergeCell `json:"merge_cells"`
}

// ColIndexToLetters converts a 0-based column index into Excel column letters (0 -> "A", 25 -> "Z", 26 -> "AA").
func ColIndexToLetters(col int) string {
	if col < 0 {
		return "A"
	}
	result := ""
	col++
	for col > 0 {
		rem := (col - 1) % 26
		result = string(rune('A'+rem)) + result
		col = (col - 1) / 26
	}
	return result
}

// ColLettersToIndex converts Excel column letters ("A" -> 0, "Z" -> 25, "AA" -> 26).
func ColLettersToIndex(letters string) int {
	letters = strings.ToUpper(strings.TrimSpace(letters))
	result := 0
	for _, r := range letters {
		if r < 'A' || r > 'Z' {
			continue
		}
		result = result*26 + int(r-'A'+1)
	}
	return result - 1
}

// CellRefToCoords parses "B4" into col 1, row 3 (0-based).
func CellRefToCoords(ref string) (int, int, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return 0, 0, fmt.Errorf("empty cell ref")
	}

	splitIdx := 0
	for i, r := range ref {
		if unicode.IsDigit(r) {
			splitIdx = i
			break
		}
	}

	if splitIdx == 0 {
		return 0, 0, fmt.Errorf("invalid cell ref: %s", ref)
	}

	colLetters := ref[:splitIdx]
	rowStr := ref[splitIdx:]

	col := ColLettersToIndex(colLetters)
	rowNum, err := strconv.Atoi(rowStr)
	if err != nil || rowNum < 1 {
		return 0, 0, fmt.Errorf("invalid row number in ref %s: %w", ref, err)
	}

	return col, rowNum - 1, nil
}

// CoordsToCellRef creates a cell reference from 0-based col and row (1, 3 -> "B4").
func CoordsToCellRef(col, row int) string {
	return fmt.Sprintf("%s%d", ColIndexToLetters(col), row+1)
}

// ParseMergeRef parses a range ref like "A1:C2" into a MergeCell struct with 0-based indices.
func ParseMergeRef(ref string) (MergeCell, error) {
	parts := strings.Split(ref, ":")
	if len(parts) != 2 {
		return MergeCell{}, fmt.Errorf("invalid merge range: %s", ref)
	}

	c1, r1, err := CellRefToCoords(parts[0])
	if err != nil {
		return MergeCell{}, err
	}
	c2, r2, err := CellRefToCoords(parts[1])
	if err != nil {
		return MergeCell{}, err
	}

	minCol, maxCol := c1, c2
	if minCol > maxCol {
		minCol, maxCol = maxCol, minCol
	}

	minRow, maxRow := r1, r2
	if minRow > maxRow {
		minRow, maxRow = maxRow, minRow
	}

	return MergeCell{
		Ref:    ref,
		MinCol: minCol,
		MinRow: minRow,
		MaxCol: maxCol,
		MaxRow: maxRow,
	}, nil
}
