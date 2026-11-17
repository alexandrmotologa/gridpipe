# gridpipe

Autonomous spreadsheet normalization, cross-tab reconciliation, and formula synthesis engine.

## Overview

gridpipe processes irregular enterprise spreadsheets in Excel (.xlsx) and CSV formats. It flattens multi-tier merged headers, canonicalizes inconsistent date and numeric encodings, and synthesizes native OpenXML formulas (such as SUM and XLOOKUP) instead of static numbers. It also performs key-based cross-tab reconciliation with conditional formatting and includes an embedded React 19 Sheet Studio on port 7575.

## Key Capabilities

1. Pure Go OpenXML Engine: Ingests and generates .xlsx zip containers directly without CGO, Python runtimes, or Microsoft Office automation.
2. Layout Normalization: Unmerges multi-level header hierarchies into flat canonical keys, detects rectangular table boundaries, and extracts trailing footnote disclaimers into metadata records.
3. Type Canonicalization: Disambiguates European (1.234,56) vs US (1,234.56) decimal separators using column distribution statistics, strips currency glyphs, and standardizes dates to ISO-8601 (YYYY-MM-DD).
4. Formula Synthesis: Generates dynamic Excel formulas (=SUM, =AVERAGE, =XLOOKUP, =IFERROR) directly into OpenXML calculation nodes for summary totals.
5. Reference Repair: Scans workbooks for corrupted references (#REF!, #VALUE!, #DIV/0!) and reconstructs valid range bounds.
6. Calculation DAG: Builds a directed acyclic graph of cell dependencies, checks for circular references, and computes topological evaluation order.
7. Cross-Tab Reconciliation: Compares corresponding line items across two worksheets, calculates item variances and percentage deltas, and injects soft green/red conditional styling with audit notes.
8. React 19 Sheet Studio: In-browser workbench with virtualized data grid, schema alignment mapper, interactive formula DAG visualizer, variance heatmaps, and real-time execution stream via Server-Sent Events (SSE).

## Installation

### Prerequisites

- Go 1.22 or higher
- Node.js 20 or higher (for building frontend assets)

### Build from Source

```bash
# Clone the repository
git clone https://github.com/alexandrmotologa/gridpipe.git
cd gridpipe

# Build React 19 web assets
cd web
npm install
npm run build
cd ..

# Copy web dist to server embed folder
cp -r web/dist/* internal/server/dist/

# Compile the single binary
go build -o bin/gridpipe.exe ./cmd/gridpipe
```

## CLI Usage

### 1. Clean and Normalize a Messy Workbook

```bash
gridpipe clean -i examples/messy_sheets/messy_supplier_invoice.xlsx -o invoice_clean.xlsx --jsonl invoice.jsonl --sql invoice.sql
```

Output:
- Flattens multi-tier headers into snake_case column names.
- Trims title banners and extracts footnote disclaimers into metadata.
- Converts dates to ISO-8601 and numbers to clean floats.
- Synthesizes `=IFERROR(SUM(...), 0)` on the total row.
- Exports clean Excel, JSONL, and SQL table schemas.

### 2. Reconcile Cross-Tab Variances

```bash
gridpipe reconcile -i examples/messy_sheets/multi_tier_pnl.xlsx --sheet-a Budget_2025 --sheet-b Actuals_2025 -k 0 -v 1 -o pnl_reconciled.xlsx
```

Output:
- Aligns line items between Budget and Actuals by Cost Code.
- Generates a `Reconciliation_Audit` worksheet with delta formulas `=C2-B2` and percentage deltas.
- Applies conditional formatting: Soft green for matches, soft red for variances exceeding threshold.

### 3. Repair Broken Formulas and Validate DAG

```bash
gridpipe formula -i examples/messy_sheets/broken_formulas.xlsx --repair --dag -o healed.xlsx
```

Output:
- Heals `#REF!` errors by recalculating table column boundaries.
- Wraps arithmetic in `IFERROR(..., 0)` to prevent division errors.
- Validates the calculation DAG and confirms acyclicity.

### 4. Infer Canonical Schema

```bash
gridpipe schema -i examples/messy_sheets/messy_supplier_invoice.xlsx
```

### 5. Launch Interactive React 19 Sheet Studio

```bash
gridpipe studio --port 7575
```

Open your browser to `http://localhost:7575` to interact with the Sheet Studio interface.

## Performance Benchmark

Tested on a 50,000-row spreadsheet with 4 mixed columns (dates, numbers, currency symbols):
- Processing Time: 412ms
- Memory Allocation Delta: under 20MB baseline
- Formula Extraction: zero #REF! or #VALUE! errors

Run the benchmark suite:

```bash
go test -v -run TestPerformanceAndMemoryBenchmark ./internal/pipeline/...
```

## Project Structure

```
cmd/gridpipe/          CLI commands: clean, reconcile, formula, schema, studio
internal/
  config/              Runtime pipeline options and schema definitions
  openxml/             Pure Go OpenXML zip archive and XML stream processor
  layout/              Header hierarchy flattener and boundary detector
  types/               Statistical type parser (numbers, currencies, dates)
  formulas/            Formula synthesizer, reference repair, and DAG engine
  reconcile/           Cross-tab line item matcher and variance auditor
  pipeline/            End-to-end normalization orchestrator
  server/              REST API, SSE broadcast hub, and embedded studio server
web/                   React 19 Sheet Studio (Vite, TypeScript, Tailwind CSS)
examples/              Messy test workbooks and sample configurations
docs/                  Architecture, OpenXML specifications, and reconciliation guide
```

## Running Tests

```bash
go test -v ./...
```
