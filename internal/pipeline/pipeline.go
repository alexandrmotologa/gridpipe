package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandrmotologa/gridpipe/internal/config"
	"github.com/alexandrmotologa/gridpipe/internal/formulas"
	"github.com/alexandrmotologa/gridpipe/internal/layout"
	"github.com/alexandrmotologa/gridpipe/internal/openxml"
	"github.com/alexandrmotologa/gridpipe/internal/types"
)

// PipelineEvent represents an execution step emitted over SSE or stdout.
type PipelineEvent struct {
	Stage       string `json:"stage"`
	ProgressPct int    `json:"progress_pct"`
	Message     string `json:"message"`
	Details     any    `json:"details,omitempty"`
}

// NormalizedSheet holds the cleaned relational dataset and audit metadata.
type NormalizedSheet struct {
	Name            string                       `json:"name"`
	Columns         []string                     `json:"columns"`
	InferredTypes   []types.InferredColumnType   `json:"inferred_types"`
	Rows            [][]string                   `json:"rows"`
	Footnotes       []layout.FootnoteRecord      `json:"footnotes"`
	Repairs         []formulas.FormulaRepairItem `json:"repairs"`
	FormulasCreated int                          `json:"formulas_created"`
	SourceSheet     *openxml.SheetData           `json:"-"`
	CleanSheet      *openxml.SheetData           `json:"-"`
}

// NormalizationPipeline coordinates end-to-end processing of a workbook.
type NormalizationPipeline struct {
	Config  config.PipelineConfig
	OnEvent func(event PipelineEvent)
}

// NewPipeline creates a new pipeline runner.
func NewPipeline(cfg config.PipelineConfig) *NormalizationPipeline {
	return &NormalizationPipeline{
		Config: cfg,
	}
}

// ProcessWorkbook cleans, normalizes, and synthesizes formulas across all sheets.
func (p *NormalizationPipeline) ProcessWorkbook(wb *openxml.Workbook) ([]*NormalizedSheet, error) {
	if wb == nil || len(wb.Sheets) == 0 {
		return nil, fmt.Errorf("empty workbook")
	}

	results := make([]*NormalizedSheet, 0, len(wb.Sheets))

	for sheetIdx, sheet := range wb.Sheets {
		p.emit("HEADER_SCAN", 10+(sheetIdx*20), fmt.Sprintf("Scanning headers and layout for sheet '%s'", sheet.Name), nil)

		// 1. Detect multi-tier headers and canonicalize names
		hdr := layout.DetectHeaders(sheet, p.Config.HeaderScanRows)
		bounds := layout.DetectTableBounds(sheet, hdr.DataStartRow)

		p.emit("BOUNDARY_DETECTION", 25+(sheetIdx*20), fmt.Sprintf("Table boundary detected: rows %d-%d, cols %d-%d", bounds.MinRow, bounds.MaxRow, bounds.MinCol, bounds.MaxCol), bounds)

		// 2. Separate footnotes and summary rows
		fnAnalysis := layout.AnalyzeFootnotes(sheet, bounds.MinRow, bounds.MaxRow)
		lastDataRow := fnAnalysis.LastDataRow
		if !p.Config.TrimFootnotes {
			lastDataRow = bounds.MaxRow
		}

		// 3. Inspect columns and canonicalize types
		activeCols := bounds.MaxCol - bounds.MinCol + 1
		colNames := make([]string, activeCols)
		for c := 0; c < activeCols; c++ {
			origCol := bounds.MinCol + c
			if origCol < len(hdr.ColumnNames) {
				colNames[c] = hdr.ColumnNames[origCol]
			} else {
				colNames[c] = fmt.Sprintf("column_%s", openxml.ColIndexToLetters(origCol))
			}
		}

		inferredTypes := make([]types.InferredColumnType, activeCols)
		for c := 0; c < activeCols; c++ {
			origCol := bounds.MinCol + c
			samples := make([]string, 0, lastDataRow-bounds.MinRow+1)
			for r := bounds.MinRow; r <= lastDataRow; r++ {
				samples = append(samples, sheet.Grid[r][origCol].Resolved)
			}
			inferredTypes[c] = types.InferColumnType(samples, p.Config.NumericLocale)
		}

		// 4. Synthesize formulas on totals
		formulasCreated := 0
		if p.Config.SynthesizeMath && len(fnAnalysis.TotalRows) > 0 {
			p.emit("FORMULA_SYNTHESIS", 60+(sheetIdx*20), "Synthesizing dynamic aggregate formulas on summary rows", fnAnalysis.TotalRows)
			opts := formulas.DefaultSynthesisOptions()
			formulasCreated = formulas.SynthesizeFormulas(sheet, bounds.MinRow, lastDataRow, fnAnalysis.TotalRows, opts)
		}

		// 5. Repair broken formulas if present
		repairs := make([]formulas.FormulaRepairItem, 0)
		if p.Config.RepairBrokenRefs {
			p.emit("FORMULA_REPAIR", 75+(sheetIdx*20), "Checking and repairing broken formulas (#REF!, #VALUE!)", nil)
			repReport := formulas.RepairWorkbookFormulas(&openxml.Workbook{Sheets: []*openxml.SheetData{sheet}})
			repairs = repReport.Repairs
		}

		// 6. Build clean relational output grid
		numDataRows := lastDataRow - bounds.MinRow + 1
		if numDataRows < 0 {
			numDataRows = 0
		}

		cleanRows := make([][]string, numDataRows)
		for r := 0; r < numDataRows; r++ {
			origRow := bounds.MinRow + r
			cleanRows[r] = make([]string, activeCols)
			for c := 0; c < activeCols; c++ {
				origCol := bounds.MinCol + c
				cellVal := sheet.Grid[origRow][origCol].Resolved

				// Apply type normalization
				colType := inferredTypes[c].Type
				switch colType {
				case "date":
					dateRes := types.ParseDate(cellVal)
					if dateRes.Success {
						cellVal = dateRes.ISO8601
					}
				case "currency":
					currInfo := types.ExtractCurrency(cellVal)
					numRes := types.ParseNumber(currInfo.CleanedText, p.Config.NumericLocale)
					if numRes.Success {
						cellVal = numRes.Canonical
					}
				case "integer", "float":
					numRes := types.ParseNumber(cellVal, p.Config.NumericLocale)
					if numRes.Success {
						cellVal = numRes.Canonical
					}
				}

				cleanRows[r][c] = cellVal
			}
		}

		// Create clean SheetData for export
		cleanSheet := &openxml.SheetData{
			Name:    sheet.Name,
			SheetID: sheet.SheetID,
			MaxRow:  numDataRows + 1, // header + data
			MaxCol:  activeCols,
			Grid:    make([][]openxml.Cell, numDataRows+1),
		}

		// Header row
		cleanSheet.Grid[0] = make([]openxml.Cell, activeCols)
		for c, name := range colNames {
			cleanSheet.Grid[0][c] = openxml.Cell{
				Ref:      openxml.CoordsToCellRef(c, 0),
				Col:      c,
				Row:      0,
				Resolved: name,
				Type:     "s",
				StyleID:  1,
			}
		}

		// Data rows
		for r := 0; r < numDataRows; r++ {
			cleanSheet.Grid[r+1] = make([]openxml.Cell, activeCols)
			for c := 0; c < activeCols; c++ {
				val := cleanRows[r][c]
				cType := "s"
				if inferredTypes[c].Type == "integer" || inferredTypes[c].Type == "float" || inferredTypes[c].Type == "currency" {
					cType = "n"
				}
				cleanSheet.Grid[r+1][c] = openxml.Cell{
					Ref:      openxml.CoordsToCellRef(c, r+1),
					Col:      c,
					Row:      r + 1,
					Resolved: val,
					Type:     cType,
				}
			}
		}

		results = append(results, &NormalizedSheet{
			Name:            sheet.Name,
			Columns:         colNames,
			InferredTypes:   inferredTypes,
			Rows:            cleanRows,
			Footnotes:       fnAnalysis.Footnotes,
			Repairs:         repairs,
			FormulasCreated: formulasCreated,
			SourceSheet:     sheet,
			CleanSheet:      cleanSheet,
		})
	}

	p.emit("COMPLETE", 100, "Workbook normalization completed successfully", nil)
	return results, nil
}

func (p *NormalizationPipeline) emit(stage string, pct int, msg string, details any) {
	if p.OnEvent != nil {
		p.OnEvent(PipelineEvent{
			Stage:       stage,
			ProgressPct: pct,
			Message:     msg,
			Details:     details,
		})
	}
}

// ExportJSONL writes normalized tabular rows as JSON Lines.
func ExportJSONL(sheet *NormalizedSheet) ([]byte, error) {
	var buf bytes.Buffer
	for _, row := range sheet.Rows {
		record := make(map[string]any)
		for c, colName := range sheet.Columns {
			if c < len(row) {
				record[colName] = row[c]
			}
		}
		data, err := json.Marshal(record)
		if err != nil {
			return nil, err
		}
		buf.Write(data)
		buf.WriteString("\n")
	}
	return buf.Bytes(), nil
}

// ExportSQL writes clean CREATE TABLE and INSERT statements.
func ExportSQL(sheet *NormalizedSheet, tableName string) string {
	if tableName == "" {
		tableName = layout.SanitizeColumnName(sheet.Name)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", tableName))
	for i, col := range sheet.Columns {
		sqlType := "TEXT"
		if i < len(sheet.InferredTypes) {
			switch sheet.InferredTypes[i].Type {
			case "integer":
				sqlType = "BIGINT"
			case "float", "currency":
				sqlType = "NUMERIC(15, 2)"
			case "date":
				sqlType = "DATE"
			case "boolean":
				sqlType = "BOOLEAN"
			}
		}
		comma := ","
		if i == len(sheet.Columns)-1 {
			comma = ""
		}
		sb.WriteString(fmt.Sprintf("    %s %s%s\n", col, sqlType, comma))
	}
	sb.WriteString(");\n\n")

	for _, row := range sheet.Rows {
		vals := make([]string, len(row))
		for i, v := range row {
			escaped := strings.ReplaceAll(v, "'", "''")
			vals[i] = fmt.Sprintf("'%s'", escaped)
		}
		sb.WriteString(fmt.Sprintf("INSERT INTO %s VALUES (%s);\n", tableName, strings.Join(vals, ", ")))
	}

	return sb.String()
}

// LoadFile reads either .xlsx or .csv from disk into a Workbook.
func LoadFile(path string) (*openxml.Workbook, error) {
	cleanPath := filepath.Clean(path)
	ext := strings.ToLower(filepath.Ext(cleanPath))

	if ext == ".csv" {
		sheet, err := openxml.ReadCSVFile(cleanPath)
		if err != nil {
			return nil, err
		}
		return &openxml.Workbook{
			Sheets:        []*openxml.SheetData{sheet},
			SharedStrings: openxml.NewSharedStringsTable(),
		}, nil
	}

	if ext == ".xlsx" {
		return openxml.ReadWorkbookFile(cleanPath)
	}

	return nil, fmt.Errorf("unsupported file extension: %s (must be .xlsx or .csv)", ext)
}

// SaveWorkbookFile saves a normalized workbook to disk as .xlsx or .csv.
func SaveWorkbookFile(path string, wb *openxml.Workbook) error {
	cleanPath := filepath.Clean(path)
	ext := strings.ToLower(filepath.Ext(cleanPath))

	if ext == ".csv" && len(wb.Sheets) > 0 {
		f, err := os.Create(cleanPath)
		if err != nil {
			return err
		}
		defer f.Close()
		return openxml.WriteCSV(f, wb.Sheets[0], ',')
	}

	return openxml.WriteWorkbookFile(cleanPath, wb)
}
