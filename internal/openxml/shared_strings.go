package openxml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// SharedStringsTable manages reading and writing the shared string table (xl/sharedStrings.xml).
type SharedStringsTable struct {
	strings []string
	index   map[string]int
}

// NewSharedStringsTable creates an empty shared strings table.
func NewSharedStringsTable() *SharedStringsTable {
	return &SharedStringsTable{
		strings: make([]string, 0),
		index:   make(map[string]int),
	}
}

// XML structures for reading sharedStrings.xml
type xmlSST struct {
	XMLName     xml.Name `xml:"sst"`
	Count       int      `xml:"count,attr"`
	UniqueCount int      `xml:"uniqueCount,attr"`
	SI          []xmlSI  `xml:"si"`
}

type xmlSI struct {
	T string   `xml:"t"`
	R []xmlRun `xml:"r"`
}

type xmlRun struct {
	T string `xml:"t"`
}

// ParseSharedStrings reads an XML stream and loads the string array.
func ParseSharedStrings(r io.Reader) (*SharedStringsTable, error) {
	sst := NewSharedStringsTable()

	decoder := xml.NewDecoder(r)
	var parsed xmlSST
	if err := decoder.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("failed to decode sharedStrings.xml: %w", err)
	}

	for _, si := range parsed.SI {
		text := si.T
		if text == "" && len(si.R) > 0 {
			var sb strings.Builder
			for _, run := range si.R {
				sb.WriteString(run.T)
			}
			text = sb.String()
		}
		sst.Add(text)
	}

	return sst, nil
}

// Add appends a string if not already present, returning its unique 0-based index.
func (s *SharedStringsTable) Add(str string) int {
	if idx, exists := s.index[str]; exists {
		return idx
	}
	idx := len(s.strings)
	s.strings = append(s.strings, str)
	s.index[str] = idx
	return idx
}

// Get returns the string at the given index.
func (s *SharedStringsTable) Get(idx int) (string, bool) {
	if idx < 0 || idx >= len(s.strings) {
		return "", false
	}
	return s.strings[idx], true
}

// Len returns the count of unique strings.
func (s *SharedStringsTable) Len() int {
	return len(s.strings)
}

// EncodeXML writes the shared strings table to an XML byte buffer.
func (s *SharedStringsTable) EncodeXML() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	buf.WriteString(fmt.Sprintf(`<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="%d" uniqueCount="%d">`, len(s.strings), len(s.strings)))

	for _, str := range s.strings {
		var escaped bytes.Buffer
		if err := xml.EscapeText(&escaped, []byte(str)); err != nil {
			return nil, err
		}
		buf.WriteString("<si><t>")
		buf.Write(escaped.Bytes())
		buf.WriteString("</t></si>")
	}

	buf.WriteString("</sst>")
	return buf.Bytes(), nil
}
