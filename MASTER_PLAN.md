# gridpipe: Architecture and Engineering Master Plan
> Autonomous Unstructured Spreadsheet Normalization, Reconciliation and Formula Synthesis Engine
> Target: Single-Binary Go Engine (< 20MB RAM) with Embedded React 19 Sheet Studio and Native OpenXML Spreadsheet Engine

---

## 1. Executive Summary and Problem Statement

### 1.1 The Operational Problem in Enterprise Spreadsheets
Operations, financial planning, logistics, and procurement teams exchange thousands of spreadsheets monthly. These spreadsheets arrive from disparate enterprise resource planning systems, legacy database exports, external vendors, and regional branch offices. They consistently exhibit structural chaos:
1. Multi-level merged title cells and split column headers that break relational schema ingestion.
2. Mixed date formatting conventions (such as DD/MM/YYYY alongside MM/DD/YYYY and Excel numeric serial dates) within identical columns.
3. Inconsistent numeric encodings with mixed decimal and thousands separators (European 1.234,56 versus US 1,234.56) and currency glyphs.
4. Overwritten calculation cells containing static numbers instead of audit-ready formulas.
5. Pervasive calculation reference errors (#REF!, #VALUE!, #NAME!, #DIV/0!) caused by row deletions and cross-sheet column movements.

### 1.2 The gridpipe Engine
gridpipe solves these operational breakdowns as a standalone, high-performance Go application (< 20MB baseline memory footprint) paired with an embedded React 19 Sheet Studio web interface:
- Native Pure-Go OpenXML Processor: Directly reads and writes .xlsx zip archives, sheet XML streams, shared string tables, styles, and calculation chains without external Office runtimes or Python dependencies.
- Layout Disentanglement Engine: Automatically detects merged header zones, unmerges cell matrices, compounds hierarchical headers into canonical keys, and identifies table boundaries while stripping decorative title banners and footer disclaimer notes.
- Statistical Type Canonicalizer: Determines numeric locale separators using column-level distribution statistics, standardizes currency representations, and outputs ISO-8601 dates.
- Formula Synthesis and Repair: Injects native OpenXML formula tags (<f>) for summary totals, reconciliations, and cross-sheet lookups instead of dead static values. Evaluates reference validity and replaces broken references.
- Directed Acyclic Graph (DAG) Dependency Auditor: Validates workbook calculation graphs, detects circular dependencies, and exposes topological order over REST and Server-Sent Events.
- Automated Cross-Tab Reconciliation: Compares source and target sheets (for example Budget versus Actuals), computes item-level absolute and percentage variances, and applies OpenXML conditional formatting rules with audit notes.
- Embedded React 19 Sheet Studio: Operates locally on port 7575, providing a visual virtualized data grid, schema mapping canvas, formula DAG inspector, and variance heatmaps.

---

## 2. System Architecture

```
       +-------------------------------------------------------------+
       |             Raw Unstructured Spreadsheets & CSVs            |
       |  (Supplier Price Lists, Multi-Tab P&L Sheets, Branch Drops) |
       +-------------------------------------------------------------+
                                      |
                         Native OpenXML Stream Reader
                                      v
       +-------------------------------------------------------------+
       |                      gridpipe Core Engine                   |
       |                    (High-Performance Go CLI)                |
       |                                                             |
       |  +---------------------+        +------------------------+  |
       |  | OpenXML & Zip Stream|------->| Structural Grid Layout |  |
       |  | Decompressor        |        | Analyzer (Header/Data) |  |
       |  +---------------------+        +------------------------+  |
       |                                              |              |
       |                                              v              |
       |  +-------------------------------------------------------+  |
       |  |             Data Cleansing & Type Normalizer          |  |
       |  |  - Strip Currency Symbols ($ / EUR / RON / GBP)       |  |
       |  |  - Statistical Comma vs Dot Locale Disambiguation     |  |
       |  |  - Convert Mixed Dates to Strict ISO-8601             |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       |                               v                             |
       |  +-------------------------------------------------------+  |
       |  |          Formula Synthesis & Dependency Engine        |  |
       |  |  - Synthesizes Dynamic Excel Formulas (SUM, XLOOKUP)  |  |
       |  |  - Eliminates Broken References (#REF!, #VALUE!)      |  |
       |  |  - Directed Acyclic Graph (DAG) Cycle Validator       |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       |                               v                             |
       |  +-------------------------------------------------------+  |
       |  |             Cross-Tab Reconciliation Auditor          |  |
       |  |  - Keyed Line-Item Variance Calculation               |  |
       |  |  - Injects Conditional Formatting & Audit Notes       |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       +-------------------------------|-----------------------------+
                                       |
                                       v
       +-------------------------------------------------------------+
       |                 Audited Spreadsheet Output                  |
       |  1. Clean, Formula-Driven Microsoft Excel (.xlsx) File      |
       |  2. Downstream Database Ingestion (JSONL / Parquet / CSV)   |
       |  3. Embedded React 19 Sheet Studio (Local HTTP :7575)       |
       +-------------------------------------------------------------+
```

---

## 3. Detailed Component Specifications

### 3.1 OpenXML Package Engine (`internal/openxml`)
- `reader.go`: Decodes ZIP archives in memory or stream mode. Parses `[Content_Types].xml`, `xl/workbook.xml`, `xl/_rels/workbook.xml.rels`, and `xl/worksheets/sheet*.xml`.
- `shared_strings.go`: High-speed memory-optimized string table indexer with slice lookups. Handles string escaping and rich text runs.
- `sheet_parser.go`: Translates column-row coordinate strings (such as `AA42`) to 0-based indices and vice versa. Extracts raw values, cell types (`s`, `n`, `b`, `str`, `inlineStr`), styles, and raw formula strings (`<f>`).
- `writer.go`: Constructs valid OpenXML `.xlsx` zip archives with worksheet definitions, shared strings, cell styles (`xl/styles.xml`), and formula elements.
- `csv.go`: Streaming CSV reader and writer with delimiter auto-detection (comma, semicolon, tab, pipe).

### 3.2 Layout & Grid Flattening Engine (`internal/layout`)
- `header_detector.go`: Scans initial rows for merged ranges (`<mergeCells>`). Unmerges multi-tier structures and builds compounded keys (e.g. `FY2025_Q1_Revenue`). Handles blank parent cells via forward-fill.
- `boundary.go`: Computes actual data table bounding boxes `(minRow, minCol, maxRow, maxCol)`. Filters out empty margin columns and top banners.
- `footnote.go`: Detects trailing summary rows, non-tabular metadata, and legal text blocks based on key column vacancy and text density. Preserves footnotes in metadata records.
- `segmenter.go`: Detects side-by-side or vertically stacked separate tables within a single worksheet tab.

### 3.3 Type Canonicalization Engine (`internal/types`)
- `numbers.go`: Evaluates column character frequencies. If comma occurs as decimal separator (single comma near end, dots as thousand separators), normalizes to IEEE-754 standard representation. Handles accounting parenthesis formatting `(500.00)` as negative `-500.00`.
- `currency.go`: Detects currency symbols (`$`, `€`, `£`, `¥`, `lei`, `RON`, `MDL`, `USD`, `EUR`) and extracts the numeric value alongside ISO currency code metadata.
- `dates.go`: Robust multi-pattern date parser supporting ISO-8601 (`YYYY-MM-DD`), US (`MM/DD/YYYY`), European (`DD.MM.YYYY`, `DD/MM/YYYY`), text dates (`15-Jan-2025`), and Excel serial numbers (offset from 1899-12-30). Emits standardized ISO-8601 strings.
- `schema.go`: Type inference per column: `integer`, `float`, `currency`, `date`, `boolean`, `text`.

### 3.4 Formula Synthesis and DAG Verifier (`internal/formulas`)
- `synthesizer.go`: Synthesizes dynamic Excel formulas into OpenXML:
  - Aggregate formulas: `=SUM(C2:C50)`, `=AVERAGE(D2:D50)`.
  - Conditional lookups: `=XLOOKUP(A2, Catalog!$A$2:$A$500, Catalog!$B$2:$B$500, "N/A")` or `=VLOOKUP(...)`.
  - Safe error wrappers: `=IFERROR(formula, fallback)`.
- `dependency_dag.go`: Constructs a directed graph of cell dependencies. Detects circular references using Kahn's algorithm or Tarjan's strongly connected components algorithm. Calculates evaluation order.
- `repair.go`: Locates `#REF!`, `#VALUE!`, `#NAME?`, `#DIV/0!` errors. Repairs orphaned ranges by reconstructing bounds from table geometry.

### 3.5 Cross-Tab Reconciliation Auditor (`internal/reconcile`)
- `variance.go`: Matches rows between two sheets using single or composite primary keys. Computes delta $\Delta = V_{actual} - V_{baseline}$ and percentage variance.
- `formatting.go`: Injects OpenXML conditional formatting rules:
  - Fills cell green when variance is within tolerance ($\le 0$).
  - Fills cell soft red when variance exceeds tolerance.
  - Adds OpenXML cell comments with audit discrepancy timestamps and details.

### 3.6 REST API and Server-Sent Events (`internal/server`)
- `api.go`: HTTP endpoints:
  - `POST /api/upload`: Multipart file upload.
  - `GET /api/sheets`: Returns sheet metadata and tab listing.
  - `POST /api/clean`: Executes normalization pipeline with options.
  - `POST /api/reconcile`: Executes cross-tab reconciliation.
  - `POST /api/formulas/repair`: Analyzes and heals formula errors.
  - `GET /api/dag`: Returns JSON representation of the formula dependency DAG.
  - `POST /api/export`: Generates downloadable `.xlsx`, `.csv`, or `.jsonl`.
- `sse.go`: Event stream on `/api/stream` broadcasting pipeline progress steps in real time.
- `embed.go`: Serves built React 19 single-page application from `web/dist` via Go `embed.FS`.

### 3.7 React 19 Sheet Studio (`web/`)
- Responsive dark-themed operational UI:
  - `GridCanvas.tsx`: High-performance virtualized spreadsheet grid displaying formula bar, coordinates, cell types, and calculation states.
  - `SchemaMapper.tsx`: Visual column transformation canvas with type overriding and compounding preview.
  - `FormulaDagView.tsx`: Interactive SVG/canvas graph rendering formula nodes, dependencies, and reference chains.
  - `VarianceHeatmap.tsx`: Reconciliation side-by-side view with color-coded discrepancy highlights.
  - `ExportControls.tsx`: Export buttons for .xlsx, JSONL, and CSV.

---

## 4. Repository Structure

```
gridpipe/
├── cmd/
│   └── gridpipe/
│       └── main.go                 # CLI entry point: clean, reconcile, formula, schema, studio
├── internal/
│   ├── config/                     # Configuration and schema definitions
│   │   ├── config.go
│   │   └── schema.go
│   ├── openxml/                    # Pure-Go OpenXML parser and generator
│   │   ├── reader.go
│   │   ├── shared_strings.go
│   │   ├── sheet_parser.go
│   │   ├── writer.go
│   │   ├── styles.go
│   │   └── csv.go
│   ├── layout/                     # Layout, header, and boundary detection
│   │   ├── header_detector.go
│   │   ├── boundary.go
│   │   ├── footnote.go
│   │   └── segmenter.go
│   ├── types/                      # Syntactic normalization and type parsing
│   │   ├── numbers.go
│   │   ├── currency.go
│   │   ├── dates.go
│   │   └── infer.go
│   ├── formulas/                   # Formula synthesis, repair, and DAG
│   │   ├── synthesizer.go
│   │   ├── dependency_dag.go
│   │   ├── repair.go
│   │   └── parser.go
│   ├── reconcile/                  # Cross-tab reconciliation engine
│   │   ├── variance.go
│   │   └── formatting.go
│   └── server/                     # REST API, SSE, and embedded studio server
│       ├── server.go
│       ├── api.go
│       ├── sse.go
│       └── embed.go
├── web/                            # React 19 Sheet Studio frontend
│   ├── src/
│   │   ├── App.tsx
│   │   ├── index.css
│   │   ├── components/
│   │   │   ├── GridCanvas.tsx
│   │   │   ├── SchemaMapper.tsx
│   │   │   ├── FormulaDagView.tsx
│   │   │   ├── VarianceHeatmap.tsx
│   │   │   └── PipelineStream.tsx
│   │   └── types.ts
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── examples/
│   ├── messy_sheets/               # Test fixtures
│   │   ├── messy_supplier_invoice.xlsx
│   │   ├── multi_tier_pnl.xlsx
│   │   └── broken_formulas.xlsx
│   └── gridpipe.yaml
├── docs/
│   ├── architecture.md
│   ├── openxml_spec.md
│   └── reconciliation_guide.md
├── Makefile
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 5. Execution Plan and Milestones

1. Milestone 1: Project Setup and Pure-Go OpenXML Engine
   - Initialize Go module, CLI structure with subcommands.
   - Implement OpenXML reader, shared strings index, worksheet XML parser, and OpenXML writer with style and formula support.
   - Implement CSV parser with auto-delimiter detection.
   - Unit tests for OpenXML serialization and deserialization.

2. Milestone 2: Layout Flattening and Boundary Trimming
   - Implement multi-tier header unmerging and compounding.
   - Implement table boundary detection (start row/col, end row/col).
   - Implement footnote detection and extraction.
   - Test suite verifying messy header grids flatten into clean canonical column names.

3. Milestone 3: Syntactic Type Normalization
   - Implement statistical decimal comma vs dot separator resolution.
   - Implement currency glyph removal and code preservation.
   - Implement multi-format date parser generating strict ISO-8601 strings.
   - Automated type inference per column.

4. Milestone 4: Formula Synthesis, DAG Verification, and Reference Repair
   - Implement formula synthesizer emitting native OpenXML `<f>` nodes.
   - Build DAG dependency graph to detect circular dependencies and compute calculation order.
   - Implement formula repair for broken `#REF!` and `#VALUE!` coordinates.

5. Milestone 5: Cross-Tab Reconciliation Engine
   - Implement key-based line item reconciliation across worksheets.
   - Compute variance deltas and percentage shifts.
   - Inject OpenXML conditional formatting rules and audit cell comments.

6. Milestone 6: REST API and Embedded React 19 Sheet Studio
   - Build REST API and Server-Sent Events stream in Go.
   - Build React 19 Sheet Studio with virtualized grid, DAG visualization, schema mapper, and variance heatmap.
   - Embed web bundle into Go binary using `embed.FS`.

7. Milestone 7: Test Fixtures, Verification, and Documentation
   - Generate realistic messy spreadsheets in `examples/messy_sheets`.
   - Comprehensive benchmarks validating memory usage (< 20MB) and processing time (< 500ms).
   - Author clear humanized README.md and documentation without AI clichés or em dashes.
