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
	StatusMatch      VarianceStatus = "MATCH"
	StatusVariance   VarianceStatus = "VARIANCE"
	StatusMissingInB VarianceStatus = "MISSING_IN_TARGET"
	StatusMissingInA VarianceStatus = "MISSING_IN_SOURCE"
)

// RowVariance represents a reconciled line item comparison.
type RowVariance struct {
	Key          string         `json:"key"`
	MatchedKeyB  string         `json:"matched_key_b,omitempty"`
	MatchType    string         `json:"match_type"` // EXACT, FUZZY, NONE
	Confidence   float64        `json:"confidence"` // 0.0 to 1.0
	ValueA       float64        `json:"value_a"`
	ValueB       float64        `json:"value_b"`
	Delta        float64        `json:"delta"`         // B - A
	PercentDelta float64        `json:"percent_delta"` // (B - A) / A * 100
	Status       VarianceStatus `json:"status"`
	AuditNote    string         `json:"audit_note"`
}

// ReconciliationSummary aggregates totals across matched rows.
type ReconciliationSummary struct {
	TotalRowsA    int           `json:"total_rows_a"`
	TotalRowsB    int           `json:"total_rows_b"`
	MatchedRows   int           `json:"matched_rows"`
	Discrepancies int           `json:"discrepancies"`
	FuzzyMatches  int           `json:"fuzzy_matches"`
	MissingInA    int           `json:"missing_in_a"`
	MissingInB    int           `json:"missing_in_b"`
	SumValueA     float64       `json:"sum_value_a"`
	SumValueB     float64       `json:"sum_value_b"`
	NetDelta      float64       `json:"net_delta"`
	Items         []RowVariance `json:"items"`
}

// ReconcileOptions holds full configuration for cross-tab reconciliation.
type ReconcileOptions struct {
	KeyColsA       []int   `json:"key_cols_a"`
	ValColA        int     `json:"val_col_a"`
	StartRowA      int     `json:"start_row_a"`
	KeyColsB       []int   `json:"key_cols_b"`
	ValColB        int     `json:"val_col_b"`
	StartRowB      int     `json:"start_row_b"`
	Epsilon        float64 `json:"epsilon"`
	EnableFuzzy    bool    `json:"enable_fuzzy"`
	FuzzyThreshold float64 `json:"fuzzy_threshold"`
}

// DefaultReconcileOptions returns standard options with exact matching on column 0 and value on column 1.
func DefaultReconcileOptions() ReconcileOptions {
	return ReconcileOptions{
		KeyColsA:       []int{0},
		ValColA:        1,
		StartRowA:      1,
		KeyColsB:       []int{0},
		ValColB:        1,
		StartRowB:      1,
		Epsilon:        0.0001,
		EnableFuzzy:    false,
		FuzzyThreshold: 0.85,
	}
}

// ExtractCompositeKey builds a composite key string from multiple column indices.
func ExtractCompositeKey(row []openxml.Cell, cols []int) string {
	parts := make([]string, 0, len(cols))
	for _, c := range cols {
		if c < len(row) {
			val := strings.TrimSpace(row[c].Resolved)
			if val != "" {
				parts = append(parts, val)
			}
		}
	}
	return strings.Join(parts, " | ")
}

// ReconcileSheets aligns SheetA (baseline) and SheetB (comparison) using key and value columns.
func ReconcileSheets(
	sheetA *openxml.SheetData, keyColA, valColA, startRowA int,
	sheetB *openxml.SheetData, keyColB, valColB, startRowB int,
	epsilon float64,
) ReconciliationSummary {
	opts := ReconcileOptions{
		KeyColsA:       []int{keyColA},
		ValColA:        valColA,
		StartRowA:      startRowA,
		KeyColsB:       []int{keyColB},
		ValColB:        valColB,
		StartRowB:      startRowB,
		Epsilon:        epsilon,
		EnableFuzzy:    false,
		FuzzyThreshold: 0.85,
	}
	return ReconcileSheetsWithOptions(sheetA, sheetB, opts)
}

// ReconcileSheetsWithOptions aligns sheets supporting composite keys and fuzzy similarity.
func ReconcileSheetsWithOptions(
	sheetA, sheetB *openxml.SheetData,
	opts ReconcileOptions,
) ReconciliationSummary {
	if opts.Epsilon <= 0 {
		opts.Epsilon = 0.0001
	}
	if opts.FuzzyThreshold <= 0 {
		opts.FuzzyThreshold = 0.85
	}
	if len(opts.KeyColsA) == 0 {
		opts.KeyColsA = []int{0}
	}
	if len(opts.KeyColsB) == 0 {
		opts.KeyColsB = []int{0}
	}

	// 1. Build map of Sheet A items
	type itemData struct {
		rawKey string
		val    float64
	}

	mapA := make(map[string]float64)
	keysA := make([]string, 0)
	for r := opts.StartRowA; r < sheetA.MaxRow; r++ {
		key := ExtractCompositeKey(sheetA.Grid[r], opts.KeyColsA)
		if key == "" {
			continue
		}
		rawVal := ""
		if opts.ValColA < len(sheetA.Grid[r]) {
			rawVal = sheetA.Grid[r][opts.ValColA].Resolved
		}
		numRes := types.ParseNumber(rawVal, "auto")
		val := 0.0
		if numRes.Success {
			val = numRes.Value
		}
		mapA[key] = val
		keysA = append(keysA, key)
	}

	// 2. Build map of Sheet B items
	mapB := make(map[string]float64)
	keysB := make([]string, 0)
	for r := opts.StartRowB; r < sheetB.MaxRow; r++ {
		key := ExtractCompositeKey(sheetB.Grid[r], opts.KeyColsB)
		if key == "" {
			continue
		}
		rawVal := ""
		if opts.ValColB < len(sheetB.Grid[r]) {
			rawVal = sheetB.Grid[r][opts.ValColB].Resolved
		}
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

	matchedKeysB := make(map[string]bool)
	unmatchedKeysA := make([]string, 0)

	// Step 1: Exact matching
	for _, kA := range keysA {
		valA := mapA[kA]
		summary.SumValueA += valA

		valB, existsInB := mapB[kA]
		if existsInB && !matchedKeysB[kA] {
			matchedKeysB[kA] = true
			summary.SumValueB += valB
			delta := valB - valA
			pct := 0.0
			if math.Abs(valA) > 1e-9 {
				pct = (delta / math.Abs(valA)) * 100.0
			}

			status := StatusMatch
			auditNote := "Exact key match. Variance within tolerance."
			if math.Abs(delta) > opts.Epsilon {
				status = StatusVariance
				summary.Discrepancies++
				auditNote = fmt.Sprintf("Exact match. Discrepancy of %.2f (%.1f%%) detected.", delta, pct)
			} else {
				summary.MatchedRows++
			}

			summary.Items = append(summary.Items, RowVariance{
				Key:          kA,
				MatchedKeyB:  kA,
				MatchType:    "EXACT",
				Confidence:   1.0,
				ValueA:       valA,
				ValueB:       valB,
				Delta:        delta,
				PercentDelta: pct,
				Status:       status,
				AuditNote:    auditNote,
			})
		} else {
			unmatchedKeysA = append(unmatchedKeysA, kA)
		}
	}

	// Step 2: Fuzzy matching for unmatched keys in A
	remainingUnmatchedA := make([]string, 0)
	if opts.EnableFuzzy {
		for _, kA := range unmatchedKeysA {
			bestScore := 0.0
			bestKeyB := ""

			for _, kB := range keysB {
				if matchedKeysB[kB] {
					continue
				}
				score := SimilarityScore(kA, kB)
				if score > bestScore {
					bestScore = score
					bestKeyB = kB
				}
			}

			if bestScore >= opts.FuzzyThreshold && bestKeyB != "" {
				matchedKeysB[bestKeyB] = true
				valA := mapA[kA]
				valB := mapB[bestKeyB]
				summary.SumValueB += valB
				delta := valB - valA
				pct := 0.0
				if math.Abs(valA) > 1e-9 {
					pct = (delta / math.Abs(valA)) * 100.0
				}

				summary.FuzzyMatches++
				status := StatusMatch
				auditNote := fmt.Sprintf("Fuzzy match (%.0f%% confidence with '%s').", bestScore*100, bestKeyB)
				if math.Abs(delta) > opts.Epsilon {
					status = StatusVariance
					summary.Discrepancies++
					auditNote += fmt.Sprintf(" Discrepancy of %.2f (%.1f%%).", delta, pct)
				} else {
					summary.MatchedRows++
				}

				summary.Items = append(summary.Items, RowVariance{
					Key:          kA,
					MatchedKeyB:  bestKeyB,
					MatchType:    "FUZZY",
					Confidence:   bestScore,
					ValueA:       valA,
					ValueB:       valB,
					Delta:        delta,
					PercentDelta: pct,
					Status:       status,
					AuditNote:    auditNote,
				})
			} else {
				remainingUnmatchedA = append(remainingUnmatchedA, kA)
			}
		}
	} else {
		remainingUnmatchedA = unmatchedKeysA
	}

	// Step 3: Record items missing in target (B)
	for _, kA := range remainingUnmatchedA {
		valA := mapA[kA]
		summary.MissingInB++
		summary.Items = append(summary.Items, RowVariance{
			Key:        kA,
			MatchType:  "NONE",
			Confidence: 0.0,
			ValueA:     valA,
			ValueB:     0,
			Delta:      -valA,
			Status:     StatusMissingInB,
			AuditNote:  fmt.Sprintf("Key '%s' present in source but missing in comparison sheet", kA),
		})
	}

	// Step 4: Record items present in target (B) but missing in source (A)
	for _, kB := range keysB {
		if matchedKeysB[kB] {
			continue
		}
		valB := mapB[kB]
		summary.SumValueB += valB
		summary.MissingInA++
		summary.Items = append(summary.Items, RowVariance{
			Key:         kB,
			MatchedKeyB: kB,
			MatchType:   "NONE",
			Confidence:  0.0,
			ValueA:      0,
			ValueB:      valB,
			Delta:       valB,
			Status:      StatusMissingInA,
			AuditNote:   fmt.Sprintf("Key '%s' found in comparison sheet but missing in baseline source", kB),
		})
	}

	summary.NetDelta = summary.SumValueB - summary.SumValueA
	return summary
}
