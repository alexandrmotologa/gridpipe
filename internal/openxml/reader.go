package openxml

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// Workbook represents a decoded Excel workbook with all sheets and shared strings.
type Workbook struct {
	Sheets        []*SheetData
	SharedStrings *SharedStringsTable
}

// XML structures for workbook.xml
type xmlWorkbook struct {
	XMLName xml.Name  `xml:"workbook"`
	Sheets  xmlSheets `xml:"sheets"`
}

type xmlSheets struct {
	Sheet []xmlSheet `xml:"sheet"`
}

type xmlSheet struct {
	Name    string `xml:"name,attr"`
	SheetID string `xml:"sheetId,attr"`
	RelID   string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
}

// XML structures for workbook.xml.rels
type xmlRelationships struct {
	XMLName       xml.Name          `xml:"Relationships"`
	Relationships []xmlRelationship `xml:"Relationship"`
}

type xmlRelationship struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
	Type   string `xml:"Type,attr"`
}

// XML structures for sheet*.xml
type xmlWorksheet struct {
	XMLName    xml.Name       `xml:"worksheet"`
	SheetData  xmlSheetData   `xml:"sheetData"`
	MergeCells *xmlMergeCells `xml:"mergeCells"`
}

type xmlMergeCells struct {
	MergeCell []xmlMergeCell `xml:"mergeCell"`
}

type xmlMergeCell struct {
	Ref string `xml:"ref,attr"`
}

type xmlSheetData struct {
	Row []xmlRow `xml:"row"`
}

type xmlRow struct {
	R int     `xml:"r,attr"`
	C []xmlCell `xml:"c"`
}

type xmlCell struct {
	R         string      `xml:"r,attr"`
	T         string      `xml:"t,attr"`
	S         int         `xml:"s,attr"`
	V         string      `xml:"v"`
	F         *xmlFormula `xml:"f"`
	InlineStr *xmlIS      `xml:"is"`
}

type xmlFormula struct {
	Content string `xml:",chardata"`
}

type xmlIS struct {
	T string `xml:"t"`
}

// ReadWorkbookFile opens an .xlsx file from a file path.
func ReadWorkbookFile(path string) (*Workbook, error) {
	cleanPath := filepath.Clean(path)
	r, err := zip.OpenReader(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file %s: %w", path, err)
	}
	defer r.Close()

	return ReadWorkbookFromZip(&r.Reader)
}

// ReadWorkbookFromBytes parses an .xlsx workbook from an in-memory byte slice.
func ReadWorkbookFromBytes(data []byte) (*Workbook, error) {
	readerAt := bytes.NewReader(data)
	zipReader, err := zip.NewReader(readerAt, int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip reader from bytes: %w", err)
	}
	return ReadWorkbookFromZip(zipReader)
}

// ReadWorkbookFromZip decodes an OpenXML zip archive.
func ReadWorkbookFromZip(zr *zip.Reader) (*Workbook, error) {
	// Index files inside zip archive
	files := make(map[string]*zip.File)
	for _, f := range zr.File {
		name := strings.TrimPrefix(filepath.ToSlash(f.Name), "/")
		files[name] = f
	}

	// 1. Read shared strings if available
	var sst *SharedStringsTable
	if sstFile, exists := files["xl/sharedStrings.xml"]; exists {
		rc, err := sstFile.Open()
		if err == nil {
			sst, _ = ParseSharedStrings(rc)
			rc.Close()
		}
	}
	if sst == nil {
		sst = NewSharedStringsTable()
	}

	// 2. Read workbook.xml to list sheets
	wbFile, exists := files["xl/workbook.xml"]
	if !exists {
		return nil, fmt.Errorf("missing xl/workbook.xml in archive")
	}

	wbRC, err := wbFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to read xl/workbook.xml: %w", err)
	}
	var wbXML xmlWorkbook
	err = xml.NewDecoder(wbRC).Decode(&wbXML)
	wbRC.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to parse xl/workbook.xml: %w", err)
	}

	// 3. Read xl/_rels/workbook.xml.rels to resolve targets
	relMap := make(map[string]string)
	if relsFile, exists := files["xl/_rels/workbook.xml.rels"]; exists {
		rc, err := relsFile.Open()
		if err == nil {
			var relsXML xmlRelationships
			if xml.NewDecoder(rc).Decode(&relsXML) == nil {
				for _, r := range relsXML.Relationships {
					target := r.Target
					if !strings.HasPrefix(target, "xl/") && !strings.HasPrefix(target, "/") {
						target = "xl/" + target
					}
					target = strings.TrimPrefix(target, "/")
					relMap[r.ID] = target
				}
			}
			rc.Close()
		}
	}

	workbook := &Workbook{
		Sheets:        make([]*SheetData, 0, len(wbXML.Sheets.Sheet)),
		SharedStrings: sst,
	}

	// 4. Parse each sheet
	for _, s := range wbXML.Sheets.Sheet {
		targetPath, exists := relMap[s.RelID]
		if !exists {
			// Fallback: standard naming convention
			targetPath = fmt.Sprintf("xl/worksheets/sheet%s.xml", s.SheetID)
		}

		sheetZipFile, ok := files[targetPath]
		if !ok {
			continue
		}

		rc, err := sheetZipFile.Open()
		if err != nil {
			continue
		}

		sheetData, err := parseWorksheet(s.Name, s.SheetID, rc, sst)
		rc.Close()
		if err == nil {
			workbook.Sheets = append(workbook.Sheets, sheetData)
		}
	}

	return workbook, nil
}

func parseWorksheet(name, id string, r io.Reader, sst *SharedStringsTable) (*SheetData, error) {
	var ws xmlWorksheet
	if err := xml.NewDecoder(r).Decode(&ws); err != nil {
		return nil, fmt.Errorf("failed to parse worksheet XML: %w", err)
	}

	sheet := &SheetData{
		Name:       name,
		SheetID:    id,
		Grid:       make([][]Cell, 0),
		MergeCells: make([]MergeCell, 0),
	}

	// Parse merge cells
	if ws.MergeCells != nil {
		for _, mc := range ws.MergeCells.MergeCell {
			parsedMerge, err := ParseMergeRef(mc.Ref)
			if err == nil {
				sheet.MergeCells = append(sheet.MergeCells, parsedMerge)
			}
		}
	}

	maxRow := 0
	maxCol := 0

	// First pass: find dimensions
	for _, row := range ws.SheetData.Row {
		rowIdx := row.R - 1
		if rowIdx > maxRow {
			maxRow = rowIdx
		}
		for _, c := range row.C {
			col, rIdx, err := CellRefToCoords(c.R)
			if err == nil {
				if col > maxCol {
					maxCol = col
				}
				if rIdx > maxRow {
					maxRow = rIdx
				}
			}
		}
	}

	if len(ws.SheetData.Row) > 0 {
		maxRow++
		maxCol++
	}

	sheet.MaxRow = maxRow
	sheet.MaxCol = maxCol

	// Initialize 2D grid
	sheet.Grid = make([][]Cell, maxRow)
	for r := 0; r < maxRow; r++ {
		sheet.Grid[r] = make([]Cell, maxCol)
		for c := 0; c < maxCol; c++ {
			sheet.Grid[r][c] = Cell{
				Ref: CoordsToCellRef(c, r),
				Col: c,
				Row: r,
			}
		}
	}

	// Populate cells
	for _, row := range ws.SheetData.Row {
		for _, c := range row.C {
			col, rIdx, err := CellRefToCoords(c.R)
			if err != nil || rIdx >= maxRow || col >= maxCol {
				continue
			}

			formulaStr := ""
			if c.F != nil {
				formulaStr = c.F.Content
			}

			resolved := c.V
			if c.T == "s" {
				var idx int
				if n, parseErr := fmt.Sscanf(c.V, "%d", &idx); n == 1 && parseErr == nil {
					if str, ok := sst.Get(idx); ok {
						resolved = str
					}
				}
			} else if c.T == "inlineStr" && c.InlineStr != nil {
				resolved = c.InlineStr.T
			}

			sheet.Grid[rIdx][col] = Cell{
				Ref:       c.R,
				Col:       col,
				Row:       rIdx,
				Type:      c.T,
				RawValue:  c.V,
				Formula:   formulaStr,
				StyleID:   c.S,
				Resolved:  resolved,
			}
		}
	}

	return sheet, nil
}
