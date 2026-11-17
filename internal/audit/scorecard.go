package audit

import (
	"strings"

	"github.com/alexandrmotologa/gridpipe/internal/formulas"
	"github.com/alexandrmotologa/gridpipe/internal/layout"
	"github.com/alexandrmotologa/gridpipe/internal/openxml"
	"github.com/alexandrmotologa/gridpipe/internal/types"
)

// DataQualityScorecard provides a quantitative readiness assessment for enterprise pipelines.
type DataQualityScorecard struct {
	OverallScore          float64        `json:"overall_score"` // 0.0 - 100.0
	Grade                 string         `json:"grade"`         // A, B, C, D, F
	CompletenessScore     float64        `json:"completeness_score"`
	ConformanceScore      float64        `json:"conformance_score"`
	FormulaIntegrityScore float64        `json:"formula_integrity_score"`
	UniquenessScore       float64        `json:"uniqueness_score"`
	TotalRows             int            `json:"total_rows"`
	TotalColumns          int            `json:"total_columns"`
	TotalCells            int            `json:"total_cells"`
	EmptyCells            int            `json:"empty_cells"`
	DuplicateRowsCount    int            `json:"duplicate_rows_count"`
	AnomaliesCount        int            `json:"anomalies_count"`
	Issues                []AnomalyIssue `json:"issues"`
}

// ComputeScorecard evaluates a sheet and calculation graph, generating an auditable quality scorecard.
func ComputeScorecard(sheet *openxml.SheetData, hdr layout.HeaderDetectionResult, bounds layout.TableBounds, colTypes []types.InferredColumnType, dag *formulas.DependencyDAG) DataQualityScorecard {
	startRow := bounds.MinRow
	endRow := bounds.MaxRow
	numRows := endRow - startRow + 1
	numCols := bounds.MaxCol - bounds.MinCol + 1
	if numRows <= 0 || numCols <= 0 {
		return DataQualityScorecard{Grade: "F"}
	}

	totalCells := numRows * numCols
	emptyCells := 0
	nonEmptyCells := 0
	conformantCells := 0

	for r := startRow; r <= endRow; r++ {
		for c := 0; c < len(hdr.ColumnNames); c++ {
			origCol := bounds.MinCol + c
			val := ""
			if origCol < len(sheet.Grid[r]) {
				val = strings.TrimSpace(sheet.Grid[r][origCol].Resolved)
			}
			if val == "" {
				emptyCells++
				continue
			}
			nonEmptyCells++

			// Check conformance with inferred type
			colType := "text"
			if c < len(colTypes) {
				colType = colTypes[c].Type
			}

			switch colType {
			case "date":
				if types.ParseDate(val).Success {
					conformantCells++
				}
			case "integer", "float", "currency":
				if types.ParseNumber(val, "auto").Success {
					conformantCells++
				}
			default:
				conformantCells++
			}
		}
	}

	// 1. Completeness Score (non-empty / total)
	completenessScore := 100.0
	if totalCells > 0 {
		completenessScore = (float64(nonEmptyCells) / float64(totalCells)) * 100.0
	}

	// 2. Conformance Score (conformant / non-empty)
	conformanceScore := 100.0
	if nonEmptyCells > 0 {
		conformanceScore = (float64(conformantCells) / float64(nonEmptyCells)) * 100.0
	}

	// 3. Formula Integrity Score
	formulaScore := 100.0
	if dag != nil {
		if dag.HasCycles {
			formulaScore -= 40.0
		}
		// Deduct for formula errors if any
		for _, n := range dag.Nodes {
			if formulas.HasFormulaError(n.Formula) {
				formulaScore -= 10.0
			}
		}
		if formulaScore < 0 {
			formulaScore = 0
		}
	}

	// 4. Uniqueness & Anomalies
	issues := DetectAnomalies(sheet, hdr, bounds, colTypes)
	dupCount := 0
	for _, issue := range issues {
		if issue.Category == "DUPLICATE" {
			dupCount++
		}
	}

	uniquenessScore := 100.0
	if numRows > 0 {
		uniquenessScore = (float64(numRows-dupCount) / float64(numRows)) * 100.0
	}

	// Weighted Overall Score: 30% Completeness + 30% Conformance + 20% Formula + 20% Uniqueness
	overall := (completenessScore * 0.30) + (conformanceScore * 0.30) + (formulaScore * 0.20) + (uniquenessScore * 0.20)
	if overall > 100.0 {
		overall = 100.0
	}
	if overall < 0.0 {
		overall = 0.0
	}

	grade := "F"
	switch {
	case overall >= 90.0:
		grade = "A"
	case overall >= 80.0:
		grade = "B"
	case overall >= 70.0:
		grade = "C"
	case overall >= 60.0:
		grade = "D"
	}

	return DataQualityScorecard{
		OverallScore:          overall,
		Grade:                 grade,
		CompletenessScore:     completenessScore,
		ConformanceScore:      conformanceScore,
		FormulaIntegrityScore: formulaScore,
		UniquenessScore:       uniquenessScore,
		TotalRows:             numRows,
		TotalColumns:          numCols,
		TotalCells:            totalCells,
		EmptyCells:            emptyCells,
		DuplicateRowsCount:    dupCount,
		AnomaliesCount:        len(issues),
		Issues:                issues,
	}
}
