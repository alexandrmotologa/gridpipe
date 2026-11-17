package layout

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

// HeaderDetectionResult stores the resolved column definitions and header boundary.
type HeaderDetectionResult struct {
	HeaderStartRow int      `json:"header_start_row"`
	HeaderEndRow   int      `json:"header_end_row"` // Last header row
	DataStartRow   int      `json:"data_start_row"`   // First data row
	ColumnNames    []string `json:"column_names"`     // Compounded canonical names
	Tiers          [][]string `json:"tiers"`          // Tiers[tierIdx][colIdx]
}

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func isNumericString(s string) bool {
	if s == "" {
		return false
	}
	dotSeen := false
	for i, r := range s {
		if i == 0 && (r == '-' || r == '+') {
			continue
		}
		if r == '.' || r == ',' {
			if dotSeen {
				return false
			}
			dotSeen = true
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// isLikelyDataCell returns true if a cell string represents data (currency, date, serial ID) rather than a header label.
func isLikelyDataCell(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// Direct numeric check
	if isNumericString(s) {
		return true
	}
	// Currency check
	if strings.ContainsAny(s, "$€£¥₹") || strings.Contains(s, "RON") || strings.Contains(s, "EUR") || strings.Contains(s, "USD") {
		return true
	}
	// Date check: DD/MM/YYYY, YYYY-MM-DD, DD.MM.YYYY
	if strings.Count(s, "/") == 2 || strings.Count(s, "-") == 2 || strings.Count(s, ".") == 2 {
		// If it has digits, it is likely a date
		digitCount := 0
		for _, r := range s {
			if unicode.IsDigit(r) {
				digitCount++
			}
		}
		if digitCount >= 4 {
			return true
		}
	}
	return false
}

// SanitizeColumnName converts arbitrary header text into a canonical snake_case or clean identifier.
func SanitizeColumnName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "unnamed"
	}
	// Replace non-alphanumeric with underscore
	cleaned := nonAlphanumericRegex.ReplaceAllString(name, "_")
	cleaned = strings.Trim(cleaned, "_")
	if cleaned == "" {
		return "col"
	}
	return cleaned
}

// DetectHeaders scans the top rows of a worksheet, unmerges multi-tier cells,
// and synthesizes flat canonical column names.
func DetectHeaders(sheet *openxml.SheetData, scanMaxRows int) HeaderDetectionResult {
	if scanMaxRows <= 0 {
		scanMaxRows = 10
	}
	if scanMaxRows > sheet.MaxRow {
		scanMaxRows = sheet.MaxRow
	}

	if sheet.MaxRow == 0 || sheet.MaxCol == 0 {
		return HeaderDetectionResult{}
	}

	// 1. Identify which top rows constitute the header block.
	// We skip decorative title banners that span across columns with only 1 non-empty cell.
	startRow := -1
	for r := 0; r < scanMaxRows; r++ {
		nonEmpty := 0
		firstVal := ""
		for c := 0; c < sheet.MaxCol; c++ {
			v := strings.TrimSpace(sheet.Grid[r][c].Resolved)
			if v != "" {
				nonEmpty++
				if firstVal == "" {
					firstVal = v
				}
			}
		}

		if nonEmpty == 1 && sheet.MaxCol >= 3 && r < scanMaxRows-1 {
			// Single cell spanning a table with 3+ columns is a title banner, skip it
			continue
		}

		if nonEmpty > 0 {
			startRow = r
			break
		}
	}

	if startRow == -1 {
		startRow = 0
	}

	// Find the end row of the header.
	// 1. Merged cells extending vertically
	endRow := startRow
	for _, mc := range sheet.MergeCells {
		if mc.MinRow <= endRow && mc.MaxRow < scanMaxRows {
			if mc.MaxRow > endRow {
				endRow = mc.MaxRow
			}
		}
	}

	// 2. If row startRow has horizontal merged cells (e.g. A1:B1 for Q1),
	// or if the following row has sub-headers before data begins,
	// extend endRow (at most 3 header tiers total).
	for r := endRow + 1; r < scanMaxRows && (r-startRow < 3); r++ {
		textCount := 0
		dataCount := 0
		for c := 0; c < sheet.MaxCol; c++ {
			val := strings.TrimSpace(sheet.Grid[r][c].Resolved)
			if val == "" {
				continue
			}
			if isLikelyDataCell(val) {
				dataCount++
			} else {
				textCount++
			}
		}

		// If this row contains data (numbers, dates, currency), headers have ended!
		if dataCount > 0 {
			break
		}

		// If this row has text labels without data, it is a sub-header tier
		if textCount > 0 {
			endRow = r
			continue
		}

		break
	}
	numHeaderRows := endRow - startRow + 1
	if numHeaderRows < 1 {
		numHeaderRows = 1
		endRow = startRow
	}

	// 2. Build unmerged tier matrix: matrix[tier][col]
	tiers := make([][]string, numHeaderRows)
	for t := 0; t < numHeaderRows; t++ {
		tiers[t] = make([]string, sheet.MaxCol)
		r := startRow + t
		for c := 0; c < sheet.MaxCol; c++ {
			tiers[t][c] = strings.TrimSpace(sheet.Grid[r][c].Resolved)
		}
	}

	// Propagate merged cells across the tier matrix
	for _, mc := range sheet.MergeCells {
		if mc.MinRow >= startRow && mc.MaxRow <= endRow {
			// Find root cell value
			rootVal := ""
			tRoot := mc.MinRow - startRow
			if tRoot >= 0 && tRoot < len(tiers) && mc.MinCol < sheet.MaxCol {
				rootVal = tiers[tRoot][mc.MinCol]
			}
			if rootVal == "" {
				continue
			}

			// Fill all cells within the merge range
			for r := mc.MinRow; r <= mc.MaxRow; r++ {
				t := r - startRow
				for c := mc.MinCol; c <= mc.MaxCol; c++ {
					if c < sheet.MaxCol && t < len(tiers) {
						tiers[t][c] = rootVal
					}
				}
			}
		}
	}

	// 3. Forward fill empty cells horizontally on parent tiers if followed by subheaders
	for t := 0; t < numHeaderRows-1; t++ {
		lastVal := ""
		for c := 0; c < sheet.MaxCol; c++ {
			if tiers[t][c] != "" {
				lastVal = tiers[t][c]
			} else if lastVal != "" && tiers[t+1][c] != "" {
				// Child has a value, so parent likely spans here
				tiers[t][c] = lastVal
			}
		}
	}

	// 4. Synthesize compounded canonical column names
	colNames := make([]string, sheet.MaxCol)
	nameCounts := make(map[string]int)

	for c := 0; c < sheet.MaxCol; c++ {
		parts := make([]string, 0, numHeaderRows)
		for t := 0; t < numHeaderRows; t++ {
			val := strings.TrimSpace(tiers[t][c])
			if val != "" {
				// Avoid repeating identical adjacent tokens
				if len(parts) == 0 || parts[len(parts)-1] != val {
					parts = append(parts, val)
				}
			}
		}

		rawCompound := strings.Join(parts, "_")
		clean := SanitizeColumnName(rawCompound)
		if clean == "" || clean == "unnamed" || clean == "col" {
			clean = "column_" + openxml.ColIndexToLetters(c)
		}

		// Ensure uniqueness across columns
		nameCounts[clean]++
		finalName := clean
		if nameCounts[clean] > 1 {
			finalName = strings.TrimSpace(clean) + "_" + strings.TrimSpace(string(rune('0'+nameCounts[clean]-1)))
		}

		colNames[c] = finalName
	}

	return HeaderDetectionResult{
		HeaderStartRow: startRow,
		HeaderEndRow:   endRow,
		DataStartRow:   endRow + 1,
		ColumnNames:    colNames,
		Tiers:          tiers,
	}
}
