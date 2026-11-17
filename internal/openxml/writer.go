package openxml

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// WriteWorkbook writes a Workbook to an output stream as an .xlsx ZIP file.
func WriteWorkbook(w io.Writer, wb *Workbook) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	if wb.SharedStrings == nil {
		wb.SharedStrings = NewSharedStringsTable()
	}

	// 1. [Content_Types].xml
	ctBuf := bytes.NewBuffer(nil)
	ctBuf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
    <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
    <Default Extension="xml" ContentType="application/xml"/>
    <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
    <Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
    <Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
`)
	for i := range wb.Sheets {
		ctBuf.WriteString(fmt.Sprintf(`    <Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`+"\n", i+1))
	}
	ctBuf.WriteString(`</Types>`)
	if err := addZipEntry(zw, "[Content_Types].xml", ctBuf.Bytes()); err != nil {
		return err
	}

	// 2. _rels/.rels
	rootRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
    <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`
	if err := addZipEntry(zw, "_rels/.rels", []byte(rootRels)); err != nil {
		return err
	}

	// 3. xl/_rels/workbook.xml.rels
	wbRels := bytes.NewBuffer(nil)
	wbRels.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
    <Relationship Id="rIdStyles" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
    <Relationship Id="rIdSharedStrings" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>
`)
	for i := range wb.Sheets {
		wbRels.WriteString(fmt.Sprintf(`    <Relationship Id="rIdSheet%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`+"\n", i+1, i+1))
	}
	wbRels.WriteString(`</Relationships>`)
	if err := addZipEntry(zw, "xl/_rels/workbook.xml.rels", wbRels.Bytes()); err != nil {
		return err
	}

	// 4. xl/workbook.xml
	wbXML := bytes.NewBuffer(nil)
	wbXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
    <sheets>
`)
	for i, s := range wb.Sheets {
		name := s.Name
		if name == "" {
			name = fmt.Sprintf("Sheet%d", i+1)
		}
		var escapedName bytes.Buffer
		_ = xml.EscapeText(&escapedName, []byte(name))
		wbXML.WriteString(fmt.Sprintf(`        <sheet name="%s" sheetId="%d" r:id="rIdSheet%d"/>`+"\n", escapedName.String(), i+1, i+1))
	}
	wbXML.WriteString(`    </sheets>
</workbook>`)
	if err := addZipEntry(zw, "xl/workbook.xml", wbXML.Bytes()); err != nil {
		return err
	}

	// 5. xl/styles.xml
	if err := addZipEntry(zw, "xl/styles.xml", DefaultStylesXML()); err != nil {
		return err
	}

	// 6. Worksheets
	for i, s := range wb.Sheets {
		sheetBytes, err := encodeWorksheet(s, wb.SharedStrings)
		if err != nil {
			return fmt.Errorf("failed to encode sheet %d: %w", i+1, err)
		}
		if err := addZipEntry(zw, fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1), sheetBytes); err != nil {
			return err
		}
	}

	// 7. xl/sharedStrings.xml
	sstBytes, err := wb.SharedStrings.EncodeXML()
	if err != nil {
		return fmt.Errorf("failed to encode sharedStrings: %w", err)
	}
	if err := addZipEntry(zw, "xl/sharedStrings.xml", sstBytes); err != nil {
		return err
	}

	return nil
}

// WriteWorkbookFile writes a workbook directly to a file on disk.
func WriteWorkbookFile(path string, wb *Workbook) error {
	cleanPath := filepath.Clean(path)
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0755); err != nil {
		return err
	}
	f, err := os.Create(cleanPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return WriteWorkbook(f, wb)
}

func encodeWorksheet(sheet *SheetData, sst *SharedStringsTable) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	buf.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` + "\n")
	buf.WriteString(`    <sheetData>` + "\n")

	for rIdx, row := range sheet.Grid {
		// Check if row has any non-empty cells
		hasContent := false
		for _, cell := range row {
			if cell.Resolved != "" || cell.Formula != "" || cell.RawValue != "" {
				hasContent = true
				break
			}
		}
		if !hasContent {
			continue
		}

		buf.WriteString(fmt.Sprintf(`        <row r="%d">`+"\n", rIdx+1))
		for cIdx, cell := range row {
			if cell.Resolved == "" && cell.Formula == "" && cell.RawValue == "" {
				continue
			}

			cellRef := CoordsToCellRef(cIdx, rIdx)
			styleAttr := ""
			if cell.StyleID > 0 {
				styleAttr = fmt.Sprintf(` s="%d"`, cell.StyleID)
			}

			if cell.Formula != "" {
				// Formula cell
				var escFormula bytes.Buffer
				_ = xml.EscapeText(&escFormula, []byte(cell.Formula))
				var escVal bytes.Buffer
				_ = xml.EscapeText(&escVal, []byte(cell.Resolved))
				buf.WriteString(fmt.Sprintf(`            <c r="%s"%s><f>%s</f><v>%s</v></c>`+"\n", cellRef, styleAttr, escFormula.String(), escVal.String()))
			} else if cell.Type == "n" || isNumericString(cell.Resolved) {
				// Number cell
				buf.WriteString(fmt.Sprintf(`            <c r="%s"%s t="n"><v>%s</v></c>`+"\n", cellRef, styleAttr, cell.Resolved))
			} else {
				// Shared string cell
				strIdx := sst.Add(cell.Resolved)
				buf.WriteString(fmt.Sprintf(`            <c r="%s"%s t="s"><v>%d</v></c>`+"\n", cellRef, styleAttr, strIdx))
			}
		}
		buf.WriteString(`        </row>` + "\n")
	}

	buf.WriteString(`    </sheetData>` + "\n")

	// MergeCells
	if len(sheet.MergeCells) > 0 {
		buf.WriteString(fmt.Sprintf(`    <mergeCells count="%d">`+"\n", len(sheet.MergeCells)))
		for _, mc := range sheet.MergeCells {
			buf.WriteString(fmt.Sprintf(`        <mergeCell ref="%s"/>`+"\n", mc.Ref))
		}
		buf.WriteString(`    </mergeCells>` + "\n")
	}

	buf.WriteString(`</worksheet>`)
	return buf.Bytes(), nil
}

func isNumericString(s string) bool {
	if s == "" {
		return false
	}
	dotSeen := false
	for i, r := range s {
		if i == 0 && (r == '-' || r == '+') {
			continue
		}
		if r == '.' {
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

func addZipEntry(zw *zip.Writer, name string, content []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(content)
	return err
}
