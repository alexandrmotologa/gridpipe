package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexandrmotologa/gridpipe/internal/config"
)

func TestWatcherScanAndProcess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gridpipe_watcher_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	inDir := filepath.Join(tempDir, "incoming")
	outDir := filepath.Join(tempDir, "processed")
	arcDir := filepath.Join(tempDir, "archive")

	_ = os.MkdirAll(inDir, 0755)

	// Write sample CSV into incoming
	csvContent := "SKU,Date,Price\nITEM-1,15/01/2025,$250.00\nITEM-2,2025-02-10,$150.00\n"
	samplePath := filepath.Join(inDir, "sample_invoice.csv")
	if err := os.WriteFile(samplePath, []byte(csvContent), 0644); err != nil {
		t.Fatalf("failed to write sample file: %v", err)
	}

	opts := WatcherOptions{
		InputDir:     inDir,
		OutputDir:    outDir,
		ArchiveDir:   arcDir,
		PollInterval: 100 * time.Millisecond,
		ExportJSONL:  true,
		Config:       config.DefaultPipelineConfig(),
	}

	w := NewWatcher(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Run watcher in goroutine
	go func() {
		_ = w.Start(ctx)
	}()

	// Wait for watcher to pick up and process file
	time.Sleep(500 * time.Millisecond)

	// Verify output xlsx and jsonl exist in outDir
	expectedXlsx := filepath.Join(outDir, "sample_invoice_audited.xlsx")
	if _, err := os.Stat(expectedXlsx); os.IsNotExist(err) {
		t.Errorf("expected clean xlsx file at %s, but not found", expectedXlsx)
	}

	expectedJsonl := filepath.Join(outDir, "sample_invoice_audited.jsonl")
	if _, err := os.Stat(expectedJsonl); os.IsNotExist(err) {
		t.Errorf("expected jsonl file at %s, but not found", expectedJsonl)
	}

	// Verify original file was archived
	expectedArchived := filepath.Join(arcDir, "sample_invoice.csv")
	if _, err := os.Stat(expectedArchived); os.IsNotExist(err) {
		t.Errorf("expected archived file at %s, but not found", expectedArchived)
	}
}
