package reconcile

import (
	"fmt"
	"math"
	"strings"

	"github.com/alexandrmotologa/gridpipe/internal/openxml"
	"github.com/alexandrmotologa/gridpipe/internal/types"
)

// VarianceStatus indicates audit status for an item.
type VarianceStatus string

const (
	StatusMatch        VarianceStatus = "MATCH"
	StatusVariance     VarianceStatus = "VARIANCE"
	StatusMissingInB   VarianceStatus = "MISSING_IN_TARGET"
	StatusMissingInA   VarianceStatus = "MISSING_IN_SOURCE"
)

// RowVariance represents a reconciled line item comparison.
type RowVariance struct {
	Key          string         `json:"key"`
	ValueA       float64        `json:"value_a"`
	ValueB       float64        `json:"value_b"`
	Delta        float64        `json:"delta"`         // B - A
	PercentDelta float64        `json:"percent_delta"` // (B - A) / A * 100
	Status       VarianceStatus `json:"status"`
	AuditNote    string         `json:"audit_note"`
}

// ReconciliationSummary aggregates totals across matched rows.
type ReconciliationSummary struct {
	TotalRowsA     int           `json:"total_rows_a"`
	TotalRowsB     int           `json:"total_rows_b"`
	MatchedRows    int           `json:"matched_rows"`
	Discrepancies  int           `json:"discrepancies"`
	MissingInA     int           `json:"missing_in_a"`
	MissingInB     int           `json:"missing_in_b"`
	SumValueA      float64       `json:"sum_value_a"`
	SumValueB      float64       `json:"sum_value_b"`
	NetDelta       float64       `json:"net_delta"`
	Items          []RowVariance `json:"items"`
}

// ReconcileSheets aligns SheetA (baseline) and SheetB (comparison) using key and value columns.
func ReconcileSheets(
	sheetA *openxml.SheetData, keyColA, valColA, startRowA int,
	sheetB *openxml.SheetData, keyColB, valColB, startRowB int,
	epsilon float64,
) ReconciliationSummary {
	if epsilon <= 0 {
		epsilon = 0.0001
	}

	mapA := make(map[string]float64)
	keysA := make([]string, 0)
	for r := startRowA; r < sheetA.MaxRow; r++ {
		key := strings.TrimSpace(sheetA.Grid[r][keyColA].Resolved)
		if key == "" {
			continue
		}
		rawVal := sheetA.Grid[r][valColA].Resolved
		numRes := types.ParseNumber(rawVal, "auto")
		val := 0.0
		if numRes.Success {
			val = numRes.Value
		}
		mapA[key] = val
		keysA = append(keysA, key)
	}

	mapB := make(map[string]float64)
	keysB := make([]string, 0)
	for r := startRowB; r < sheetB.MaxRow; r++ {
		key := strings.TrimSpace(sheetB.Grid[r][keyColB].Resolved)
		if key == "" {
			continue
		}
		rawVal := sheetB.Grid[r][valColB].Resolved
		numRes := types.ParseNumber(rawVal, "auto")
		val := 0.0
		if numRes.Success {
			val = numRes.Value
		}
		mapB[key] = val
		keysB = append(keysB, key)
	}

	summary := ReconciliationSummary{
		TotalRowsA: len(mapA),
		TotalRowsB: len(mapB),
		Items:      make([]RowVariance, 0),
	}

	seenKeys := make(map[string]bool)

	// Process all keys from Sheet A
	for _, k := range keysA {
		if seenKeys[k] {
			continue
		}
		seenKeys[k] = true

		valA := mapA[k]
		summary.SumValueA += valA

		valB, existsInB := mapB[k]
		if !existsInB {
			summary.MissingInB++
			summary.Items = append(summary.Items, RowVariance{
				Key:       k,
				ValueA:    valA,
				ValueB:    0,
				Delta:     -valA,
				Status:    StatusMissingInB,
				AuditNote: fmt.Sprintf("Key '%s' present in source but missing in comparison sheet", k),
			})
			continue
		}

		summary.SumValueB += valB
		delta := valB - valA
		pct := 0.0
		if math.Abs(valA) > 1e-9 {
			pct = (delta / math.Abs(valA)) * 100.0
		}

		status := StatusMatch
		auditNote := "Variance within tolerance"
		if math.Abs(delta) > epsilon {
			status = StatusVariance
			summary.Discrepancies++
			auditNote = fmt.Sprintf("Discrepancy of %.2f (%.1f%%) detected between sheets", delta, pct)
		} else {
			summary.MatchedRows++
		}

		summary.Items = append(summary.Items, RowVariance{
			Key:          k,
			ValueA:       valA,
			ValueB:       valB,
			Delta:        delta,
			PercentDelta: pct,
			Status:       status,
			AuditNote:    auditNote,
		})
	}

	// Process keys in B that were missing in A
	for _, k := range keysB {
		if seenKeys[k] {
			continue
		}
		seenKeys[k] = true

		valB := mapB[k]
		summary.SumValueB += valB
		summary.MissingInA++

		summary.Items = append(summary.Items, RowVariance{
			Key:       k,
			ValueA:    0,
			ValueB:    valB,
			Delta:     valB,
			Status:    StatusMissingInA,
			AuditNote: fmt.Sprintf("Key '%s' found in comparison sheet but was not in baseline source", k),
		})
	}

	summary.NetDelta = summary.SumValueB - summary.SumValueA
	return summary
}
