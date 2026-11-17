package formulas

import (
	"regexp"
	"strings"

	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

// CellRefMatch represents a cell or range reference extracted from a formula.
type CellRefMatch struct {
	SheetName  string `json:"sheet_name,omitempty"`
	RawRef     string `json:"raw_ref"`
	IsRange    bool   `json:"is_range"`
	StartCol   int    `json:"start_col"`
	StartRow   int    `json:"start_row"`
	EndCol     int    `json:"end_col"`
	EndRow     int    `json:"end_row"`
}

// Regex to capture cell references like A1, $A$1, Sheet1!A1, A1:B10, Sheet1!$A$1:$B$10, #REF!
var cellRefRegex = regexp.MustCompile(`(?i)(?:'([^']+)'!|([A-Za-z0-9_]+)!)?\$?([A-Za-z]+)\$?([0-9]+)(?::\$?([A-Za-z]+)\$?([0-9]+))?`)
var refErrorRegex = regexp.MustCompile(`(?i)#REF!|#VALUE!|#NAME\?|#DIV/0!|#N/A`)

// ExtractReferences parses an Excel formula and extracts all target cells and ranges it depends on.
func ExtractReferences(formula string) []CellRefMatch {
	matches := cellRefRegex.FindAllStringSubmatch(formula, -1)
	results := make([]CellRefMatch, 0, len(matches))

	for _, m := range matches {
		sheet := m[1]
		if sheet == "" {
			sheet = m[2]
		}
		col1Str := m[3]
		row1Str := m[4]
		col2Str := m[5]
		row2Str := m[6]

		col1, row1, err1 := openxml.CellRefToCoords(col1Str + row1Str)
		if err1 != nil {
			continue
		}

		if col2Str != "" && row2Str != "" {
			col2, row2, err2 := openxml.CellRefToCoords(col2Str + row2Str)
			if err2 == nil {
				results = append(results, CellRefMatch{
					SheetName: sheet,
					RawRef:    m[0],
					IsRange:   true,
					StartCol:  col1,
					StartRow:  row1,
					EndCol:    col2,
					EndRow:    row2,
				})
				continue
			}
		}

		results = append(results, CellRefMatch{
			SheetName: sheet,
			RawRef:    m[0],
			IsRange:   false,
			StartCol:  col1,
			StartRow:  row1,
			EndCol:    col1,
			EndRow:    row1,
		})
	}

	return results
}

// HasFormulaError checks if formula contains corrupted tokens like #REF! or #VALUE!.
func HasFormulaError(formula string) bool {
	return refErrorRegex.MatchString(formula)
}

// CleanFormula removes leading '=' if present and trims spaces.
func CleanFormula(formula string) string {
	formula = strings.TrimSpace(formula)
	formula = strings.TrimPrefix(formula, "=")
	return strings.TrimSpace(formula)
}
