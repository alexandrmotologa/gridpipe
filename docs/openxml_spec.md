# OpenXML Packaging and Processing Specification

This document details how gridpipe processes and generates Microsoft Excel OpenXML (.xlsx) containers natively in pure Go.

## Package Architecture

An OpenXML workbook is a ZIP archive conforming to ECMA-376 and ISO/IEC 29500 standards. The internal directory structure is organized as follows:

```
[Content_Types].xml               Package-level content type mappings
_rels/.rels                       Root relationships linking to workbook.xml
xl/
  workbook.xml                    Workbook structure and sheet definitions
  _rels/workbook.xml.rels         Relationships linking to sheets, styles, and shared strings
  styles.xml                      Cell formatting, fonts, fills, and borders
  sharedStrings.xml               Central string repository for memory deduplication
  worksheets/
    sheet1.xml                    Worksheet cell grid, values, formulas, and merge ranges
    sheet2.xml
```

## Shared Strings Table (`xl/sharedStrings.xml`)

OpenXML stores repeated text strings in a centralized table to reduce archive size. Cell elements of type `s` reference string indices:

```xml
<c r="A2" t="s">
    <v>0</v>
</c>
```

The corresponding shared string entry:

```xml
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="1" uniqueCount="1">
    <si>
        <t>Item Description</t>
    </si>
</sst>
```

gridpipe reads `<si><t>` text nodes as well as rich text runs (`<si><r><t>`) and flattens them into string values during ingestion.

## Native Formula Synthesis (`<f>` and `<v>`)

Rather than writing only static values (`<v>`), gridpipe emits dynamic formula definitions using the `<f>` tag:

```xml
<c r="D7" s="1" t="n">
    <f>SUM(D4:D6)</f>
    <v>4300.75</v>
</c>
```

When opened in Excel or downstream analytics tools, the formula recalculates automatically if input numbers are altered by users.

## Conditional Formatting and Cell Styles (`xl/styles.xml`)

gridpipe injects specific style definitions in `cellXfs`:
- Style 0: Standard tabular cell with light border
- Style 1: Bold header text
- Style 2: Soft green background fill (#D4EDDA) for variance within tolerance
- Style 3: Soft red background fill (#F8D7DA) for variance exceeding tolerance or missing records
