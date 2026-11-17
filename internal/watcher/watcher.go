package watcher

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandrmotologa/gridpipe/internal/config"
	"github.com/alexandrmotologa/gridpipe/internal/openxml"
	"github.com/alexandrmotologa/gridpipe/internal/pipeline"
)

// WatcherOptions configures automated directory polling and processing.
type WatcherOptions struct {
	InputDir     string        `json:"input_dir"`
	OutputDir    string        `json:"output_dir"`
	ArchiveDir   string        `json:"archive_dir,omitempty"`
	PollInterval time.Duration `json:"poll_interval"`
	ExportJSONL  bool          `json:"export_jsonl"`
	ExportSQL    bool          `json:"export_sql"`
	Config       config.PipelineConfig
}

// Watcher monitors an incoming directory and processes spreadsheets headlessly.
type Watcher struct {
	Opts      WatcherOptions
	processed map[string]bool
}

// NewWatcher creates a directory watcher instance.
func NewWatcher(opts WatcherOptions) *Watcher {
	if opts.PollInterval <= 0 {
		opts.PollInterval = 2 * time.Second
	}
	return &Watcher{
		Opts:      opts,
		processed: make(map[string]bool),
	}
}

// Start begins the monitoring loop until context is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	inputClean := filepath.Clean(w.Opts.InputDir)
	outputClean := filepath.Clean(w.Opts.OutputDir)

	if err := os.MkdirAll(inputClean, 0755); err != nil {
		return fmt.Errorf("failed to create input dir %s: %w", inputClean, err)
	}
	if err := os.MkdirAll(outputClean, 0755); err != nil {
		return fmt.Errorf("failed to create output dir %s: %w", outputClean, err)
	}
	if w.Opts.ArchiveDir != "" {
		if err := os.MkdirAll(filepath.Clean(w.Opts.ArchiveDir), 0755); err != nil {
			return fmt.Errorf("failed to create archive dir: %w", err)
		}
	}

	log.Printf("[gridpipe-watcher] Monitoring '%s' -> '%s' (poll interval: %v)", inputClean, outputClean, w.Opts.PollInterval)

	ticker := time.NewTicker(w.Opts.PollInterval)
	defer ticker.Stop()

	// Initial scan
	w.scanAndProcess()

	for {
		select {
		case <-ctx.Done():
			log.Println("[gridpipe-watcher] Stopping directory watcher...")
			return nil
		case <-ticker.C:
			w.scanAndProcess()
		}
	}
}

func (w *Watcher) scanAndProcess() {
	entries, err := os.ReadDir(w.Opts.InputDir)
	if err != nil {
		log.Printf("[gridpipe-watcher] Error reading input directory: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".xlsx" && ext != ".csv" {
			continue
		}

		// Skip temporary lock files (e.g. ~$file.xlsx)
		if strings.HasPrefix(name, "~$") || strings.HasPrefix(name, ".") {
			continue
		}

		filePath := filepath.Join(w.Opts.InputDir, name)
		if w.processed[filePath] {
			continue
		}

		// Ensure file has finished writing by checking size stability
		fi1, err := os.Stat(filePath)
		if err != nil {
			continue
		}
		time.Sleep(100 * time.Millisecond)
		fi2, err := os.Stat(filePath)
		if err != nil || fi1.Size() != fi2.Size() {
			// Still being written
			continue
		}

		log.Printf("[gridpipe-watcher] Ingesting new file: %s (%d bytes)", name, fi2.Size())
		w.processFile(filePath, name)
		w.processed[filePath] = true
	}
}

func (w *Watcher) processFile(filePath, fileName string) {
	wb, err := pipeline.LoadFile(filePath)
	if err != nil {
		log.Printf("[gridpipe-watcher] Failed to load %s: %v", fileName, err)
		return
	}

	pipe := pipeline.NewPipeline(w.Opts.Config)
	results, err := pipe.ProcessWorkbook(wb)
	if err != nil {
		log.Printf("[gridpipe-watcher] Failed to process %s: %v", fileName, err)
		return
	}

	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)
	outBase := filepath.Join(w.Opts.OutputDir, base+"_audited")

	// 1. Save clean .xlsx
	outWB := &openxml.Workbook{
		Sheets:        make([]*openxml.SheetData, 0, len(results)),
		SharedStrings: openxml.NewSharedStringsTable(),
	}
	for _, res := range results {
		outWB.Sheets = append(outWB.Sheets, res.CleanSheet)
	}

	outPath := outBase + ".xlsx"
	if err := openxml.WriteWorkbookFile(outPath, outWB); err != nil {
		log.Printf("[gridpipe-watcher] Failed to write clean xlsx: %v", err)
	} else {
		log.Printf("[gridpipe-watcher] Created clean workbook: %s", outPath)
	}

	// 2. Export JSONL if requested
	if w.Opts.ExportJSONL && len(results) > 0 {
		jsonlData, err := pipeline.ExportJSONL(results[0])
		if err == nil {
			jsonlPath := outBase + ".jsonl"
			_ = os.WriteFile(jsonlPath, jsonlData, 0644)
			log.Printf("[gridpipe-watcher] Exported JSONL: %s", jsonlPath)
		}
	}

	// 3. Export SQL if requested
	if w.Opts.ExportSQL && len(results) > 0 {
		sqlText := pipeline.ExportSQL(results[0], "")
		sqlPath := outBase + ".sql"
		_ = os.WriteFile(sqlPath, []byte(sqlText), 0644)
		log.Printf("[gridpipe-watcher] Exported SQL: %s", sqlPath)
	}

	// 4. Move to archive directory if configured
	if w.Opts.ArchiveDir != "" {
		destPath := filepath.Join(w.Opts.ArchiveDir, fileName)
		_ = os.Rename(filePath, destPath)
		log.Printf("[gridpipe-watcher] Archived original file to: %s", destPath)
	}
}
