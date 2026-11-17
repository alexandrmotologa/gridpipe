package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/alexandrmotologa/gridpipe/internal/config"
	"github.com/alexandrmotologa/gridpipe/internal/formulas"
	"github.com/alexandrmotologa/gridpipe/internal/openxml"
	"github.com/alexandrmotologa/gridpipe/internal/pipeline"
	"github.com/alexandrmotologa/gridpipe/internal/reconcile"
)

// WorkbookStore holds state in memory for active studio sessions.
type WorkbookStore struct {
	mu           sync.RWMutex
	ActiveWB     *openxml.Workbook
	FileName     string
	Normalized   []*pipeline.NormalizedSheet
	LastSummary  *reconcile.ReconciliationSummary
	ReconciledWB *openxml.Workbook
}

var store = &WorkbookStore{}

// RegisterRoutes sets up REST endpoints and handles CORS.
func RegisterRoutes(mux *http.ServeMux, hub *SSEHub) {
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/upload", handleUpload(hub))
	mux.HandleFunc("/api/sheets", handleGetSheets)
	mux.HandleFunc("/api/clean", handleClean(hub))
	mux.HandleFunc("/api/reconcile", handleReconcile(hub))
	mux.HandleFunc("/api/formulas/repair", handleRepair(hub))
	mux.HandleFunc("/api/dag", handleGetDAG)
	mux.HandleFunc("/api/export", handleExport)
	mux.HandleFunc("/api/stream", hub.HandleSSE)
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"service": "gridpipe",
		"version": "1.0.0",
	})
}

func handleUpload(hub *SSEHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := r.ParseMultipartForm(50 << 20) // 50MB max in memory
		if err != nil {
			http.Error(w, "File upload exceeds limits: "+err.Error(), http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Missing 'file' field in multipart form", http.StatusBadRequest)
			return
		}
		defer file.Close()

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Failed to read uploaded file", http.StatusInternalServerError)
			return
		}

		ext := strings.ToLower(filepath.Ext(header.Filename))
		var wb *openxml.Workbook

		if ext == ".csv" {
			sheet, parseErr := openxml.ReadCSV(bytes.NewReader(fileBytes), strings.TrimSuffix(header.Filename, ext))
			if parseErr != nil {
				http.Error(w, "CSV parse failed: "+parseErr.Error(), http.StatusBadRequest)
				return
			}
			wb = &openxml.Workbook{
				Sheets:        []*openxml.SheetData{sheet},
				SharedStrings: openxml.NewSharedStringsTable(),
			}
		} else if ext == ".xlsx" {
			wb, err = openxml.ReadWorkbookFromBytes(fileBytes)
			if err != nil {
				http.Error(w, "OpenXML parse failed: "+err.Error(), http.StatusBadRequest)
				return
			}
		} else {
			http.Error(w, "Unsupported file format. Please upload .xlsx or .csv", http.StatusBadRequest)
			return
		}

		store.mu.Lock()
		store.ActiveWB = wb
		store.FileName = header.Filename
		store.Normalized = nil
		store.LastSummary = nil
		store.ReconciledWB = nil
		store.mu.Unlock()

		hub.Broadcast(pipeline.PipelineEvent{
			Stage:       "UPLOAD_SUCCESS",
			ProgressPct: 100,
			Message:     fmt.Sprintf("Loaded '%s' with %d sheet(s)", header.Filename, len(wb.Sheets)),
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"filename":    header.Filename,
			"sheet_count": len(wb.Sheets),
			"sheets":      getSheetOverview(wb),
		})
	}
}

func handleGetSheets(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}

	store.mu.RLock()
	wb := store.ActiveWB
	store.mu.RUnlock()

	if wb == nil {
		http.Error(w, "No workbook loaded. Please upload a file first.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"filename": store.FileName,
		"sheets":   getSheetOverview(wb),
	})
}

func handleClean(hub *SSEHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		store.mu.Lock()
		wb := store.ActiveWB
		store.mu.Unlock()

		if wb == nil {
			http.Error(w, "No active workbook loaded", http.StatusBadRequest)
			return
		}

		cfg := config.DefaultPipelineConfig()
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&cfg)
		}

		pipe := pipeline.NewPipeline(cfg)
		pipe.OnEvent = func(e pipeline.PipelineEvent) {
			hub.Broadcast(e)
		}

		results, err := pipe.ProcessWorkbook(wb)
		if err != nil {
			http.Error(w, "Processing failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		store.mu.Lock()
		store.Normalized = results
		store.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "success",
			"results": results,
		})
	}
}

func handleReconcile(hub *SSEHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		type ReconcileReq struct {
			SheetA   string  `json:"sheet_a"`
			SheetB   string  `json:"sheet_b"`
			KeyColA  int     `json:"key_col_a"`
			ValColA  int     `json:"val_col_a"`
			KeyColB  int     `json:"key_col_b"`
			ValColB  int     `json:"val_col_b"`
			StartRow int     `json:"start_row"`
			Epsilon  float64 `json:"epsilon"`
		}

		var req ReconcileReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid json body: "+err.Error(), http.StatusBadRequest)
			return
		}

		store.mu.RLock()
		wb := store.ActiveWB
		store.mu.RUnlock()

		if wb == nil || len(wb.Sheets) < 2 {
			http.Error(w, "Workbook must contain at least 2 sheets for reconciliation", http.StatusBadRequest)
			return
		}

		var sA, sB *openxml.SheetData
		for _, s := range wb.Sheets {
			if s.Name == req.SheetA {
				sA = s
			}
			if s.Name == req.SheetB {
				sB = s
			}
		}

		if sA == nil || sB == nil {
			sA = wb.Sheets[0]
			sB = wb.Sheets[1]
		}

		hub.Broadcast(pipeline.PipelineEvent{
			Stage:       "RECONCILE_START",
			ProgressPct: 30,
			Message:     fmt.Sprintf("Reconciling '%s' against '%s'", sA.Name, sB.Name),
		})

		summary := reconcile.ReconcileSheets(
			sA, req.KeyColA, req.ValColA, req.StartRow,
			sB, req.KeyColB, req.ValColB, req.StartRow,
			req.Epsilon,
		)

		reconciledSheet := reconcile.GenerateReconciledSheet(summary, "AuditReport")

		reconciledWB := &openxml.Workbook{
			Sheets:        []*openxml.SheetData{reconciledSheet, sA, sB},
			SharedStrings: openxml.NewSharedStringsTable(),
		}

		store.mu.Lock()
		store.LastSummary = &summary
		store.ReconciledWB = reconciledWB
		store.mu.Unlock()

		hub.Broadcast(pipeline.PipelineEvent{
			Stage:       "RECONCILE_COMPLETE",
			ProgressPct: 100,
			Message:     fmt.Sprintf("Reconciliation complete: %d matched, %d variances", summary.MatchedRows, summary.Discrepancies),
			Details:     summary,
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(summary)
	}
}

func handleRepair(hub *SSEHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == http.MethodOptions {
			return
		}

		store.mu.Lock()
		wb := store.ActiveWB
		store.mu.Unlock()

		if wb == nil {
			http.Error(w, "No active workbook", http.StatusBadRequest)
			return
		}

		hub.Broadcast(pipeline.PipelineEvent{
			Stage:       "REPAIR_START",
			ProgressPct: 50,
			Message:     "Scanning workbook for formula reference errors",
		})

		report := formulas.RepairWorkbookFormulas(wb)

		hub.Broadcast(pipeline.PipelineEvent{
			Stage:       "REPAIR_COMPLETE",
			ProgressPct: 100,
			Message:     fmt.Sprintf("Formula repair complete: fixed %d of %d scanned", report.TotalRepaired, report.TotalScanned),
			Details:     report,
		})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(report)
	}
}

func handleGetDAG(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}

	store.mu.RLock()
	wb := store.ActiveWB
	store.mu.RUnlock()

	if wb == nil {
		http.Error(w, "No active workbook", http.StatusBadRequest)
		return
	}

	dag := formulas.BuildDAG(wb)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dag)
}

func handleExport(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		return
	}

	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "xlsx"
	}

	store.mu.RLock()
	norm := store.Normalized
	recWB := store.ReconciledWB
	activeWB := store.ActiveWB
	store.mu.RUnlock()

	switch format {
	case "xlsx":
		var exportWB *openxml.Workbook
		if recWB != nil {
			exportWB = recWB
		} else if len(norm) > 0 {
			exportWB = &openxml.Workbook{
				Sheets:        make([]*openxml.SheetData, 0),
				SharedStrings: openxml.NewSharedStringsTable(),
			}
			for _, n := range norm {
				exportWB.Sheets = append(exportWB.Sheets, n.CleanSheet)
			}
		} else if activeWB != nil {
			exportWB = activeWB
		} else {
			http.Error(w, "Nothing to export", http.StatusBadRequest)
			return
		}

		var buf bytes.Buffer
		if err := openxml.WriteWorkbook(&buf, exportWB); err != nil {
			http.Error(w, "Failed to generate Excel file: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename=\"gridpipe_audited.xlsx\"")
		w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
		_, _ = w.Write(buf.Bytes())

	case "jsonl":
		if len(norm) == 0 {
			http.Error(w, "Run /api/clean before exporting JSONL", http.StatusBadRequest)
			return
		}
		data, err := pipeline.ExportJSONL(norm[0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Content-Disposition", "attachment; filename=\"gridpipe_clean.jsonl\"")
		_, _ = w.Write(data)

	case "sql":
		if len(norm) == 0 {
			http.Error(w, "Run /api/clean before exporting SQL", http.StatusBadRequest)
			return
		}
		sqlDump := pipeline.ExportSQL(norm[0], "gridpipe_clean_table")
		w.Header().Set("Content-Type", "application/sql")
		w.Header().Set("Content-Disposition", "attachment; filename=\"gridpipe_clean.sql\"")
		_, _ = w.Write([]byte(sqlDump))

	case "csv":
		if len(norm) == 0 {
			http.Error(w, "Run /api/clean before exporting CSV", http.StatusBadRequest)
			return
		}
		var buf bytes.Buffer
		if err := openxml.WriteCSV(&buf, norm[0].CleanSheet, ','); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=\"gridpipe_clean.csv\"")
		_, _ = w.Write(buf.Bytes())

	default:
		http.Error(w, "Unsupported export format: "+format, http.StatusBadRequest)
	}
}

func getSheetOverview(wb *openxml.Workbook) []map[string]any {
	out := make([]map[string]any, len(wb.Sheets))
	for i, s := range wb.Sheets {
		previewRows := make([][]string, 0)
		for r := 0; r < 5 && r < s.MaxRow; r++ {
			rowVals := make([]string, 0)
			for c := 0; c < 10 && c < s.MaxCol; c++ {
				rowVals = append(rowVals, s.Grid[r][c].Resolved)
			}
			previewRows = append(previewRows, rowVals)
		}

		out[i] = map[string]any{
			"index":       i,
			"name":        s.Name,
			"sheet_id":    s.SheetID,
			"max_row":     s.MaxRow,
			"max_col":     s.MaxCol,
			"merge_count": len(s.MergeCells),
			"preview":     previewRows,
		}
	}
	return out
}
