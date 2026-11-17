package pipeline

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/alexandrmotologa/gridpipe/internal/config"
	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

func TestPerformanceAndMemoryBenchmark(t *testing.T) {
	runtime.GC()
	var memStart runtime.MemStats
	runtime.ReadMemStats(&memStart)

	const rowCount = 50000
	const colCount = 4

	// Construct large 50,000 row sheet in memory
	sheet := &openxml.SheetData{
		Name:   "LargeBenchmark",
		MaxRow: rowCount + 1,
		MaxCol: colCount,
		Grid:   make([][]openxml.Cell, rowCount+1),
	}

	// Header row
	sheet.Grid[0] = []openxml.Cell{
		{Ref: "A1", Resolved: "Transaction_ID", Type: "s"},
		{Ref: "B1", Resolved: "Effective_Date", Type: "s"},
		{Ref: "C1", Resolved: "Item_Count", Type: "s"},
		{Ref: "D1", Resolved: "Amount_USD", Type: "s"},
	}

	// 50,000 rows
	for r := 1; r <= rowCount; r++ {
		sheet.Grid[r] = []openxml.Cell{
			{Ref: fmt.Sprintf("A%d", r+1), Resolved: fmt.Sprintf("TXN-%06d", r), Type: "s"},
			{Ref: fmt.Sprintf("B%d", r+1), Resolved: "2025-03-15", Type: "s"},
			{Ref: fmt.Sprintf("C%d", r+1), Resolved: "12", Type: "n"},
			{Ref: fmt.Sprintf("D%d", r+1), Resolved: "$1,450.50", Type: "s"},
		}
	}

	wb := &openxml.Workbook{
		Sheets:        []*openxml.SheetData{sheet},
		SharedStrings: openxml.NewSharedStringsTable(),
	}

	cfg := config.DefaultPipelineConfig()
	pipe := NewPipeline(cfg)

	start := time.Now()
	results, err := pipe.ProcessWorkbook(wb)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("benchmark failed: %v", err)
	}

	var memEnd runtime.MemStats
	runtime.ReadMemStats(&memEnd)

	memAllocMB := float64(memEnd.Alloc-memStart.Alloc) / (1024 * 1024)

	t.Logf("Processed %d rows in %v (Memory delta: %.2f MB)", len(results[0].Rows), elapsed, memAllocMB)

	if elapsed > 2*time.Second {
		t.Errorf("processing took too long: %v", elapsed)
	}
}
