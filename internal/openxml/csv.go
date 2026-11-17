package openxml

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// DetectDelimiter inspects the first few lines of CSV content to determine separator rune.
func DetectDelimiter(sample []byte) rune {
	// Strip UTF-8 BOM if present
	if len(sample) >= 3 && sample[0] == 0xEF && sample[1] == 0xBB && sample[2] == 0xBF {
		sample = sample[3:]
	}

	delimiters := []rune{',', ';', '\t', '|'}
	counts := make(map[rune]int)

	scanner := bufio.NewScanner(bytes.NewReader(sample))
	linesChecked := 0
	for scanner.Scan() && linesChecked < 5 {
		line := scanner.Text()
		for _, d := range delimiters {
			counts[d] += strings.Count(line, string(d))
		}
		linesChecked++
	}

	bestDelim := ','
	maxCount := 0
	for _, d := range delimiters {
		if counts[d] > maxCount {
			maxCount = counts[d]
			bestDelim = d
		}
	}

	return bestDelim
}

// ReadCSV reads CSV data from a reader and returns a SheetData representation.
func ReadCSV(r io.Reader, sheetName string) (*SheetData, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Remove UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	delim := DetectDelimiter(data)

	csvReader := csv.NewReader(bytes.NewReader(data))
	csvReader.Comma = delim
	csvReader.LazyQuotes = true
	csvReader.FieldsPerRecord = -1

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if sheetName == "" {
		sheetName = "Sheet1"
	}

	maxRow := len(records)
	maxCol := 0
	for _, row := range records {
		if len(row) > maxCol {
			maxCol = len(row)
		}
	}

	sheet := &SheetData{
		Name:       sheetName,
		SheetID:    "1",
		MaxRow:     maxRow,
		MaxCol:     maxCol,
		Grid:       make([][]Cell, maxRow),
		MergeCells: nil,
	}

	for r := 0; r < maxRow; r++ {
		sheet.Grid[r] = make([]Cell, maxCol)
		for c := 0; c < maxCol; c++ {
			val := ""
			if c < len(records[r]) {
				val = strings.TrimSpace(records[r][c])
			}
			cellType := "s"
			if isNumericString(val) {
				cellType = "n"
			}
			sheet.Grid[r][c] = Cell{
				Ref:      CoordsToCellRef(c, r),
				Col:      c,
				Row:      r,
				Type:     cellType,
				Resolved: val,
				RawValue: val,
			}
		}
	}

	return sheet, nil
}

// ReadCSVFile reads a CSV file from disk into a SheetData representation.
func ReadCSVFile(path string) (*SheetData, error) {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, err
	}
	base := filepath.Base(cleanPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return ReadCSV(bytes.NewReader(data), name)
}

// WriteCSV exports SheetData to a CSV writer.
func WriteCSV(w io.Writer, sheet *SheetData, delim rune) error {
	csvWriter := csv.NewWriter(w)
	if delim != 0 {
		csvWriter.Comma = delim
	}

	for _, row := range sheet.Grid {
		record := make([]string, len(row))
		for cIdx, cell := range row {
			record[cIdx] = cell.Resolved
		}
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

// CleanUTF8 returns a string with invalid UTF-8 bytes removed.
func CleanUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	v := make([]rune, 0, len(s))
	for i, r := range s {
		if r == utf8.RuneError {
			_, size := utf8.DecodeRuneInString(s[i:])
			if size == 1 {
				continue
			}
		}
		v = append(v, r)
	}
	return string(v)
}
