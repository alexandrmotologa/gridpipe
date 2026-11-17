package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/alexandrmotologa/gridpipe/internal/audit"
	"github.com/alexandrmotologa/gridpipe/internal/config"
	"github.com/alexandrmotologa/gridpipe/internal/formulas"
	"github.com/alexandrmotologa/gridpipe/internal/layout"
	"github.com/alexandrmotologa/gridpipe/internal/openxml"
	"github.com/alexandrmotologa/gridpipe/internal/pipeline"
	"github.com/alexandrmotologa/gridpipe/internal/reconcile"
	"github.com/alexandrmotologa/gridpipe/internal/server"
	"github.com/alexandrmotologa/gridpipe/internal/types"
	"github.com/alexandrmotologa/gridpipe/internal/watcher"
)

const version = "1.1.0"

func printUsage() {
	fmt.Printf(`gridpipe %s - Autonomous Spreadsheet Normalization, Reconciliation & Formula Synthesis Engine

Usage:
  gridpipe <command> [arguments]

Available Commands:
  clean       Normalize messy multi-tier headers, dates, numbers, and synthesize formulas
  reconcile   Cross-tab variance reconciliation with composite keys & fuzzy matching
  formula     Scan and repair broken formula references (#REF!, #VALUE!) and audit DAG
  audit       Compute quantitative Data Quality Scorecard (0-100) and detect anomalies
  watch       Headless directory watcher for automated spreadsheet ingestion
  schema      Infer canonical tabular schema from workbook columns
  studio      Launch embedded React 19 Sheet Studio web interface
  version     Display engine version

Run 'gridpipe <command> -h' for more information on a command.
`, version)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "clean":
		handleCleanCmd(os.Args[2:])
	case "reconcile":
		handleReconcileCmd(os.Args[2:])
	case "formula":
		handleFormulaCmd(os.Args[2:])
	case "audit":
		handleAuditCmd(os.Args[2:])
	case "watch":
		handleWatchCmd(os.Args[2:])
	case "schema":
		handleSchemaCmd(os.Args[2:])
	case "studio":
		handleStudioCmd(os.Args[2:])
	case "version":
		fmt.Printf("gridpipe engine v%s\n", version)
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command '%s'\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleCleanCmd(args []string) {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	input := fs.String("i", "", "Input spreadsheet path (.xlsx or .csv)")
	output := fs.String("o", "", "Output normalized spreadsheet path (.xlsx or .csv)")
	jsonlOut := fs.String("jsonl", "", "Optional output path for JSON Lines export")
	sqlOut := fs.String("sql", "", "Optional output path for SQL DDL/DML export")
	locale := fs.String("locale", "auto", "Numeric locale hint: auto, us, eu")
	fs.Parse(args)

	if *input == "" {
		fmt.Fprintln(os.Stderr, "Error: -i (input file) is required")
		fs.Usage()
		os.Exit(1)
	}

	wb, err := pipeline.LoadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading input: %v\n", err)
		os.Exit(1)
	}

	cfg := config.DefaultPipelineConfig()
	cfg.NumericLocale = *locale
	pipe := pipeline.NewPipeline(cfg)

	fmt.Printf("[gridpipe] Ingesting '%s' (%d sheets)...\n", *input, len(wb.Sheets))

	results, err := pipe.ProcessWorkbook(wb)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Normalization failed: %v\n", err)
		os.Exit(1)
	}

	for _, res := range results {
		fmt.Printf("  -> Normalized sheet '%s': %d columns, %d data rows\n", res.Name, len(res.Columns), len(res.Rows))
		if res.Scorecard != nil {
			fmt.Printf("     Quality Score: %.1f/100 (Grade: %s, Anomalies: %d)\n", res.Scorecard.OverallScore, res.Scorecard.Grade, res.Scorecard.AnomaliesCount)
		}
		if len(res.Footnotes) > 0 {
			fmt.Printf("     Separated %d trailing footnote/metadata line(s)\n", len(res.Footnotes))
		}
		if res.FormulasCreated > 0 {
			fmt.Printf("     Synthesized %d native dynamic Excel formula(s)\n", res.FormulasCreated)
		}
		if len(res.Repairs) > 0 {
			fmt.Printf("     Repaired %d broken formula reference(s)\n", len(res.Repairs))
		}
	}

	// Write primary output
	outPath := *output
	if outPath == "" {
		ext := filepath.Ext(*input)
		base := strings.TrimSuffix(*input, ext)
		outPath = base + "_clean.xlsx"
	}

	outWb := &openxml.Workbook{
		Sheets:        make([]*openxml.SheetData, 0),
		SharedStrings: openxml.NewSharedStringsTable(),
	}
	for _, res := range results {
		outWb.Sheets = append(outWb.Sheets, res.CleanSheet)
	}

	if err := pipeline.SaveWorkbookFile(outPath, outWb); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save output to %s: %v\n", outPath, err)
		os.Exit(1)
	}
	fmt.Printf("[gridpipe] Saved clean workbook to: %s\n", outPath)

	// Write JSONL if requested
	if *jsonlOut != "" && len(results) > 0 {
		data, err := pipeline.ExportJSONL(results[0])
		if err == nil {
			_ = os.WriteFile(*jsonlOut, data, 0644)
			fmt.Printf("[gridpipe] Exported JSONL to: %s\n", *jsonlOut)
		}
	}

	// Write SQL if requested
	if *sqlOut != "" && len(results) > 0 {
		sqlText := pipeline.ExportSQL(results[0], "")
		_ = os.WriteFile(*sqlOut, []byte(sqlText), 0644)
		fmt.Printf("[gridpipe] Exported SQL schema and inserts to: %s\n", *sqlOut)
	}
}

func handleReconcileCmd(args []string) {
	fs := flag.NewFlagSet("reconcile", flag.ExitOnError)
	input := fs.String("i", "", "Input workbook path (.xlsx)")
	sheetA := fs.String("sheet-a", "", "Name of baseline sheet (e.g. Budget)")
	sheetB := fs.String("sheet-b", "", "Name of comparison sheet (e.g. Actuals)")
	keysFlag := fs.String("k", "0", "0-based primary key column indices (e.g. '0' or '0,1' for composite)")
	valCol := fs.Int("v", 1, "0-based value column index")
	output := fs.String("o", "", "Output path for reconciled workbook (.xlsx)")
	epsilon := fs.Float64("epsilon", 0.01, "Variance tolerance threshold")
	enableFuzzy := fs.Bool("fuzzy", true, "Enable fuzzy matching for SKU/code variations")
	fuzzyThreshold := fs.Float64("fuzzy-threshold", 0.85, "Fuzzy match similarity threshold (0.0 to 1.0)")
	fs.Parse(args)

	if *input == "" {
		fmt.Fprintln(os.Stderr, "Error: -i (input file) is required")
		os.Exit(1)
	}

	wb, err := pipeline.LoadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load workbook: %v\n", err)
		os.Exit(1)
	}

	if len(wb.Sheets) < 2 {
		fmt.Fprintln(os.Stderr, "Error: Reconcile requires a workbook with at least 2 sheets")
		os.Exit(1)
	}

	var sA, sB *openxml.SheetData
	for _, s := range wb.Sheets {
		if *sheetA != "" && s.Name == *sheetA {
			sA = s
		}
		if *sheetB != "" && s.Name == *sheetB {
			sB = s
		}
	}

	if sA == nil {
		sA = wb.Sheets[0]
	}
	if sB == nil {
		sB = wb.Sheets[1]
	}

	// Parse composite keys
	keyCols := parseIndexList(*keysFlag)

	fmt.Printf("[gridpipe] Reconciling Sheet '%s' against Sheet '%s' (keys: %v, fuzzy: %v)...\n", sA.Name, sB.Name, keyCols, *enableFuzzy)

	hdrA := layout.DetectHeaders(sA, 5)
	hdrB := layout.DetectHeaders(sB, 5)

	opts := reconcile.ReconcileOptions{
		KeyColsA:       keyCols,
		ValColA:        *valCol,
		StartRowA:      hdrA.DataStartRow,
		KeyColsB:       keyCols,
		ValColB:        *valCol,
		StartRowB:      hdrB.DataStartRow,
		Epsilon:        *epsilon,
		EnableFuzzy:    *enableFuzzy,
		FuzzyThreshold: *fuzzyThreshold,
	}

	summary := reconcile.ReconcileSheetsWithOptions(sA, sB, opts)

	fmt.Printf("  -> Matched Items: %d (including %d fuzzy matches)\n", summary.MatchedRows, summary.FuzzyMatches)
	fmt.Printf("  -> Discrepancies: %d\n", summary.Discrepancies)
	fmt.Printf("  -> Missing in Comparison: %d\n", summary.MissingInB)
	fmt.Printf("  -> Net Delta: %.2f\n", summary.NetDelta)

	reconciledSheet := reconcile.GenerateReconciledSheet(summary, "Reconciliation_Audit")
	outWB := &openxml.Workbook{
		Sheets:        []*openxml.SheetData{reconciledSheet, sA, sB},
		SharedStrings: openxml.NewSharedStringsTable(),
	}

	outPath := *output
	if outPath == "" {
		ext := filepath.Ext(*input)
		base := strings.TrimSuffix(*input, ext)
		outPath = base + "_reconciled.xlsx"
	}

	if err := openxml.WriteWorkbookFile(outPath, outWB); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save reconciled workbook: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[gridpipe] Saved auditable reconciliation report to: %s\n", outPath)
}

func handleAuditCmd(args []string) {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	input := fs.String("i", "", "Input spreadsheet path (.xlsx or .csv)")
	jsonOut := fs.String("json", "", "Optional output path for JSON scorecard")
	sheetIdx := fs.Int("sheet", 0, "0-based worksheet index to audit")
	fs.Parse(args)

	if *input == "" {
		fmt.Fprintln(os.Stderr, "Error: -i is required")
		fs.Usage()
		os.Exit(1)
	}

	wb, err := pipeline.LoadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load input: %v\n", err)
		os.Exit(1)
	}

	if *sheetIdx >= len(wb.Sheets) {
		fmt.Fprintf(os.Stderr, "Invalid sheet index %d (workbook has %d sheets)\n", *sheetIdx, len(wb.Sheets))
		os.Exit(1)
	}

	sheet := wb.Sheets[*sheetIdx]
	hdr := layout.DetectHeaders(sheet, 10)
	bounds := layout.DetectTableBounds(sheet, hdr.DataStartRow)

	inferredTypes := make([]types.InferredColumnType, len(hdr.ColumnNames))
	for c := 0; c < len(hdr.ColumnNames); c++ {
		origCol := bounds.MinCol + c
		samples := make([]string, 0)
		for r := bounds.MinRow; r <= bounds.MaxRow; r++ {
			if origCol < len(sheet.Grid[r]) {
				samples = append(samples, sheet.Grid[r][origCol].Resolved)
			}
		}
		inferredTypes[c] = types.InferColumnType(samples, "auto")
	}

	dag := formulas.BuildDAG(wb)
	scorecard := audit.ComputeScorecard(sheet, hdr, bounds, inferredTypes, dag)

	fmt.Printf("\n=== Data Quality Scorecard for '%s' ===\n", sheet.Name)
	fmt.Printf("  Overall Quality Score: %.1f / 100.0  (Grade: %s)\n", scorecard.OverallScore, scorecard.Grade)
	fmt.Printf("  - Completeness:       %.1f%%\n", scorecard.CompletenessScore)
	fmt.Printf("  - Type Conformance:   %.1f%%\n", scorecard.ConformanceScore)
	fmt.Printf("  - Formula Integrity:  %.1f%%\n", scorecard.FormulaIntegrityScore)
	fmt.Printf("  - Uniqueness:         %.1f%%\n", scorecard.UniquenessScore)
	fmt.Printf("  Total Rows: %d | Columns: %d | Cells: %d\n", scorecard.TotalRows, scorecard.TotalColumns, scorecard.TotalCells)
	fmt.Printf("  Anomalies Detected: %d (Duplicates: %d)\n\n", scorecard.AnomaliesCount, scorecard.DuplicateRowsCount)

	if len(scorecard.Issues) > 0 {
		fmt.Println("Detected Data Issues:")
		for i, issue := range scorecard.Issues {
			if i >= 10 {
				fmt.Printf("  ... and %d more issue(s)\n", len(scorecard.Issues)-10)
				break
			}
			fmt.Printf("  [%s] Cell %s (%s): %s\n", issue.Severity, issue.CellRef, issue.Category, issue.Description)
		}
	}

	if *jsonOut != "" {
		data, _ := json.MarshalIndent(scorecard, "", "  ")
		_ = os.WriteFile(*jsonOut, data, 0644)
		fmt.Printf("\n[gridpipe] Saved audit scorecard JSON to: %s\n", *jsonOut)
	}
}

func handleWatchCmd(args []string) {
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	inputDir := fs.String("i", "", "Incoming directory to monitor")
	outputDir := fs.String("o", "", "Directory to output normalized spreadsheets")
	archiveDir := fs.String("archive", "", "Optional archive directory for raw ingested files")
	pollInterval := fs.Duration("poll", 2*time.Second, "Directory polling frequency")
	exportJSONL := fs.Bool("jsonl", true, "Automatically export JSONL alongside XLSX")
	exportSQL := fs.Bool("sql", false, "Automatically export SQL schema alongside XLSX")
	fs.Parse(args)

	if *inputDir == "" || *outputDir == "" {
		fmt.Fprintln(os.Stderr, "Error: Both -i (input dir) and -o (output dir) are required")
		fs.Usage()
		os.Exit(1)
	}

	opts := watcher.WatcherOptions{
		InputDir:     *inputDir,
		OutputDir:    *outputDir,
		ArchiveDir:   *archiveDir,
		PollInterval: *pollInterval,
		ExportJSONL:  *exportJSONL,
		ExportSQL:    *exportSQL,
		Config:       config.DefaultPipelineConfig(),
	}

	w := watcher.NewWatcher(opts)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	if err := w.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		os.Exit(1)
	}
}

func handleFormulaCmd(args []string) {
	fs := flag.NewFlagSet("formula", flag.ExitOnError)
	input := fs.String("i", "", "Input workbook path (.xlsx)")
	output := fs.String("o", "", "Output workbook path (.xlsx)")
	repair := fs.Bool("repair", true, "Heal broken #REF! and #VALUE! references")
	checkDag := fs.Bool("dag", true, "Check calculation dependency DAG for cycles")
	fs.Parse(args)

	if *input == "" {
		fmt.Fprintln(os.Stderr, "Error: -i (input file) is required")
		os.Exit(1)
	}

	wb, err := pipeline.LoadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load workbook: %v\n", err)
		os.Exit(1)
	}

	if *checkDag {
		dag := formulas.BuildDAG(wb)
		fmt.Printf("[gridpipe] Calculation DAG: %d formula nodes, %d dependency edges\n", len(dag.Nodes), dag.TotalEdges)
		if dag.HasCycles {
			fmt.Printf("  [WARNING] Circular reference cycle detected! Nodes involved: %v\n", dag.CycleNodes)
		} else {
			fmt.Printf("  [SUCCESS] Workbook is acyclic. Topological calculation depth: %d\n", len(dag.EvalOrder))
		}
	}

	if *repair {
		report := formulas.RepairWorkbookFormulas(wb)
		fmt.Printf("[gridpipe] Formula repair complete: %d formulas repaired out of %d scanned\n", report.TotalRepaired, report.TotalScanned)
		for _, item := range report.Repairs {
			fmt.Printf("  -> Cell %s!%s: repaired '%s' -> '%s' (%s)\n", item.SheetName, item.CellRef, item.Original, item.Repaired, item.ErrorReason)
		}

		if *output != "" {
			if err := openxml.WriteWorkbookFile(*output, wb); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save repaired workbook: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("[gridpipe] Saved repaired workbook to: %s\n", *output)
		}
	}
}

func handleSchemaCmd(args []string) {
	fs := flag.NewFlagSet("schema", flag.ExitOnError)
	input := fs.String("i", "", "Input spreadsheet path (.xlsx or .csv)")
	output := fs.String("o", "", "Output schema JSON path")
	fs.Parse(args)

	if *input == "" {
		fmt.Fprintln(os.Stderr, "Error: -i is required")
		os.Exit(1)
	}

	wb, err := pipeline.LoadFile(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load file: %v\n", err)
		os.Exit(1)
	}

	if len(wb.Sheets) == 0 {
		fmt.Fprintln(os.Stderr, "Workbook contains no sheets")
		os.Exit(1)
	}

	sheet := wb.Sheets[0]
	hdr := layout.DetectHeaders(sheet, 10)
	bounds := layout.DetectTableBounds(sheet, hdr.DataStartRow)
	fnInfo := layout.AnalyzeFootnotes(sheet, bounds.MinRow, bounds.MaxRow)

	cols := make([]config.TargetColumn, len(hdr.ColumnNames))
	for c, colName := range hdr.ColumnNames {
		samples := make([]string, 0)
		for r := bounds.MinRow; r <= fnInfo.LastDataRow; r++ {
			if c < len(sheet.Grid[r]) {
				samples = append(samples, sheet.Grid[r][c].Resolved)
			}
		}
		inferred := types.InferColumnType(samples, "auto")
		cols[c] = config.TargetColumn{
			Name:     colName,
			Type:     inferred.Type,
			Required: inferred.NonNullCount > 0,
		}
	}

	tblSchema := config.TableSchema{
		SchemaName: sheet.Name,
		Columns:    cols,
	}

	data, _ := json.MarshalIndent(tblSchema, "", "  ")
	if *output != "" {
		_ = os.WriteFile(*output, data, 0644)
		fmt.Printf("[gridpipe] Schema saved to %s\n", *output)
	} else {
		fmt.Println(string(data))
	}
}

func handleStudioCmd(args []string) {
	fs := flag.NewFlagSet("studio", flag.ExitOnError)
	port := fs.Int("p", 7575, "HTTP server listening port")
	fs.Parse(args)

	srv := server.NewServer(*port)
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Studio server error: %v\n", err)
		os.Exit(1)
	}
}

func parseIndexList(s string) []int {
	parts := strings.Split(s, ",")
	res := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if idx, err := strconv.Atoi(p); err == nil {
			res = append(res, idx)
		}
	}
	if len(res) == 0 {
		return []int{0}
	}
	return res
}
