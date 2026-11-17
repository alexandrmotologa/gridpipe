package layout

import (
	"strings"

	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

// FootnoteRecord stores metadata extracted from non-tabular trailing rows.
type FootnoteRecord struct {
	RowIndex int    `json:"row_index"`
	Content  string `json:"content"`
	Kind     string `json:"kind"` // footnote, disclaimer, summary_total, metadata
}

// FootnoteAnalysis contains trimmed data row limits and extracted notes.
type FootnoteAnalysis struct {
	LastDataRow int              `json:"last_data_row"`
	Footnotes   []FootnoteRecord `json:"footnotes"`
	TotalRows   []int            `json:"total_rows"`
}

// Common keywords indicating footnote or legal disclaimer rows.
var footnoteKeywords = []string{
	"*",
	"note:",
	"notes:",
	"confidential",
	"disclaimer",
	"source:",
	"prepared by",
	"internal use only",
	"all rights reserved",
}

// Common keywords indicating total or subtotal summary rows.
var totalKeywords = []string{
	"total",
	"subtotal",
	"grand total",
	"sum",
	"average",
	"overall",
}

// AnalyzeFootnotes inspects rows from the bottom of the active data region upwards,
// separating legal footnotes, disclaimers, and total rows from raw relational data.
func AnalyzeFootnotes(sheet *openxml.SheetData, dataStartRow, dataEndRow int) FootnoteAnalysis {
	if dataEndRow < dataStartRow || dataEndRow >= sheet.MaxRow {
		return FootnoteAnalysis{LastDataRow: dataEndRow}
	}

	footnotes := make([]FootnoteRecord, 0)
	totalRows := make([]int, 0)
	lastDataRow := dataEndRow

	// Traverse from bottom up
	for r := dataEndRow; r >= dataStartRow; r-- {
		row := sheet.Grid[r]
		nonEmptyCells := 0
		rowText := make([]string, 0)
		for _, c := range row {
			val := strings.TrimSpace(c.Resolved)
			if val != "" {
				nonEmptyCells++
				rowText = append(rowText, val)
			}
		}

		if nonEmptyCells == 0 {
			// Blank separator row at bottom
			lastDataRow = r - 1
			continue
		}

		fullLine := strings.ToLower(strings.Join(rowText, " "))

		// Check for total/summary row
		isTotal := false
		for _, kw := range totalKeywords {
			if strings.HasPrefix(fullLine, kw) || strings.Contains(fullLine, " "+kw) {
				isTotal = true
				break
			}
		}

		if isTotal {
			totalRows = append(totalRows, r)
			// Total row is preserved in data or metadata according to user pipeline
			continue
		}

		// Check if it's a footnote (sparse row with disclaimer keywords or single long text cell)
		isFootnote := false
		for _, kw := range footnoteKeywords {
			if strings.Contains(fullLine, kw) {
				isFootnote = true
				break
			}
		}

		// A single string cell spanning a table with 4+ columns is almost certainly a footnote or disclaimer
		if !isFootnote && nonEmptyCells == 1 && sheet.MaxCol >= 4 && len(fullLine) > 20 {
			isFootnote = true
		}

		if isFootnote {
			footnotes = append(footnotes, FootnoteRecord{
				RowIndex: r,
				Content:  strings.Join(rowText, " | "),
				Kind:     "footnote",
			})
			lastDataRow = r - 1
		} else {
			// We encountered a normal multi-cell data row, so bottom footnotes end here
			break
		}
	}

	return FootnoteAnalysis{
		LastDataRow: lastDataRow,
		Footnotes:   footnotes,
		TotalRows:   totalRows,
	}
}
