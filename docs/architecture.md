# Architecture and Component Design

gridpipe is an autonomous spreadsheet normalization and reconciliation engine built in Go with an embedded React 19 Sheet Studio. It converts irregular multi-tier Excel and CSV workbooks into canonical relational models while synthesizing native OpenXML formulas.

## Core Design Goals

1. Self Contained Binary: Runs without external C libraries, Python dependencies, or Microsoft Office automation runtimes.
2. Low Memory Baseline: Operates under a 20MB baseline footprint using streaming XML processing and indexed shared string lookup tables.
3. Computational Auditability: Rather than overwriting calculated totals with dead static numbers, gridpipe synthesizes native Excel formulas (=SUM, =XLOOKUP, =IFERROR) directly into the output workbook.
4. Relational Integrity: Detects and unmerges hierarchical headers into structured snake_case keys while trimming decorative banners and separating footnote disclaimers into metadata records.

## Data Flow Pipeline

```
Raw Spreadsheet (.xlsx / .csv)
             |
             v
[OpenXML Package Decompressor]
             |
             v
[Layout & Boundary Detector]
  - Skips decorative title banners
  - Unmerges multi-level headers into canonical column keys
  - Identifies rectangular data boundary
  - Separates trailing footnotes and summary rows
             |
             v
[Type Canonicalizer]
  - Disambiguates comma vs dot decimal separators via column distribution
  - Extracts currency symbols and preserves ISO currency codes
  - Converts mixed date strings and Excel serials to ISO-8601 (YYYY-MM-DD)
             |
             v
[Formula Synthesis & DAG Engine]
  - Injects native OpenXML formula tags (<f>) on summary rows
  - Heals broken cell references (#REF!, #VALUE!)
  - Validates calculation DAG to verify absence of circular references
             |
             v
[Cross-Tab Reconciliation Auditor]
  - Aligns line items across tabs via primary keys
  - Computes deltas and percentage variances
  - Injects conditional formatting (soft green / soft red)
             |
             v
Audited Output: .xlsx / .csv / .jsonl / .sql / React 19 Studio (:7575)
```

## Component Overview

### internal/openxml
Handles zip archive decompression, shared string table indexing, style definitions, and cell coordinate translations. The writer reconstructs compliant OpenXML spreadsheets containing calculation formulas and conditional styles.

### internal/layout
Determines the structural geometry of worksheets. It handles horizontal and vertical cell merges, builds compounded header identifiers, identifies rectangular data boundaries, and separates disclaimers and legal footnotes from the relational records.

### internal/types
Applies statistical heuristics to infer and canonicalize column data types. It resolves European (1.234,56) versus US (1,234.56) numeric conventions, parses accounting negative formats, strips currency symbols, and normalizes date expressions to ISO-8601.

### internal/formulas
Manages formula syntax, extraction, healing, and graph validation. It constructs a directed acyclic graph (DAG) representing formula cell dependencies, flags circular references, and synthesizes dynamic aggregation formulas for total rows.

### internal/reconcile
Compares line items across two worksheets using exact or composite keys. It computes numeric variances, percentage deltas, identifies missing rows in either source or target, and compiles an audited report with conditional formatting.

### internal/server
Provides an HTTP server on port 7575 supporting REST endpoints for file upload, sheet inspection, cleaning, formula healing, DAG query, and export. It also includes an event hub for Server-Sent Events (SSE) broadcasting real-time progress.

### web (React 19 Sheet Studio)
A single-page application built with React 19, TypeScript, and Tailwind CSS. It features a virtualized grid canvas with formula bar, schema mapping canvas, interactive formula DAG visualizer, and reconciliation heatmaps.
