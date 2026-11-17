package layout

import (
	"strings"

	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

// TableSegment represents an isolated rectangular table region in a worksheet.
type TableSegment struct {
	SegmentID int         `json:"segment_id"`
	Bounds    TableBounds `json:"bounds"`
	RowCount  int         `json:"row_count"`
	ColCount  int         `json:"col_count"`
}

// SegmentSheet identifies separate table segments separated by 2 or more consecutive blank rows.
func SegmentSheet(sheet *openxml.SheetData) []TableSegment {
	if sheet.MaxRow == 0 || sheet.MaxCol == 0 {
		return nil
	}

	segments := make([]TableSegment, 0)
	currentStart := -1
	consecutiveBlank := 0

	for r := 0; r < sheet.MaxRow; r++ {
		rowBlank := true
		for c := 0; c < sheet.MaxCol; c++ {
			if strings.TrimSpace(sheet.Grid[r][c].Resolved) != "" {
				rowBlank = false
				break
			}
		}

		if !rowBlank {
			if currentStart == -1 {
				currentStart = r
			}
			consecutiveBlank = 0
		} else {
			if currentStart != -1 {
				consecutiveBlank++
				if consecutiveBlank >= 2 {
					// Segment ended
					endRow := r - consecutiveBlank
					segBounds := DetectTableBounds(sheet, currentStart)
					segBounds.MaxRow = endRow
					segments = append(segments, TableSegment{
						SegmentID: len(segments) + 1,
						Bounds:    segBounds,
						RowCount:  segBounds.MaxRow - segBounds.MinRow + 1,
						ColCount:  segBounds.MaxCol - segBounds.MinCol + 1,
					})
					currentStart = -1
					consecutiveBlank = 0
				}
			}
		}
	}

	if currentStart != -1 {
		segBounds := DetectTableBounds(sheet, currentStart)
		segments = append(segments, TableSegment{
			SegmentID: len(segments) + 1,
			Bounds:    segBounds,
			RowCount:  segBounds.MaxRow - segBounds.MinRow + 1,
			ColCount:  segBounds.MaxCol - segBounds.MinCol + 1,
		})
	}

	return segments
}
