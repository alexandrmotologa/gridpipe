package formulas

import (
	"strings"
	"testing"

	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

func TestExtractReferences(t *testing.T) {
	formula := "SUM(A1:B10) + C4 + 'Data Sheet'!D5:D20"
	refs := ExtractReferences(formula)

	if len(refs) != 3 {
		t.Fatalf("expected 3 reference matches, got %d", len(refs))
	}

	// Ref 1: A1:B10
	if !refs[0].IsRange || refs[0].StartCol != 0 || refs[0].StartRow != 0 || refs[0].EndCol != 1 || refs[0].EndRow != 9 {
		t.Errorf("ref 0 mismatch: %+v", refs[0])
	}

	// Ref 2: C4
	if refs[1].IsRange || refs[1].StartCol != 2 || refs[1].StartRow != 3 {
		t.Errorf("ref 1 mismatch: %+v", refs[1])
	}

	// Ref 3: 'Data Sheet'!D5:D20
	if refs[2].SheetName != "Data Sheet" || refs[2].StartCol != 3 || refs[2].StartRow != 4 || refs[2].EndRow != 19 {
		t.Errorf("ref 2 mismatch: %+v", refs[2])
	}
}

func TestBuildDAGAndCycleDetection(t *testing.T) {
	// Construct workbook with:
	// A1 = 10
	// B1 = A1 * 2
	// C1 = B1 + 5
	// Valid acyclic DAG
	sheet := &openxml.SheetData{
		Name:   "Sheet1",
		MaxRow: 1,
		MaxCol: 3,
		Grid: [][]openxml.Cell{
			{
				{Ref: "A1", Col: 0, Row: 0, Resolved: "10"},
				{Ref: "B1", Col: 1, Row: 0, Formula: "A1*2"},
				{Ref: "C1", Col: 2, Row: 0, Formula: "B1+5"},
			},
		},
	}

	wb := &openxml.Workbook{
		Sheets: []*openxml.SheetData{sheet},
	}

	dag := BuildDAG(wb)
	if dag.HasCycles {
		t.Errorf("expected no cycles, got cycles: %v", dag.CycleNodes)
	}

	if len(dag.EvalOrder) != 2 {
		t.Errorf("expected 2 nodes in eval order, got %d", len(dag.EvalOrder))
	}

	// B1 should precede C1
	if dag.EvalOrder[0] != "Sheet1!B1" || dag.EvalOrder[1] != "Sheet1!C1" {
		t.Errorf("expected eval order [Sheet1!B1, Sheet1!C1], got %v", dag.EvalOrder)
	}

	// Now introduce a circular reference: B1 = C1, C1 = B1
	sheet.Grid[0][1].Formula = "C1*2"
	sheet.Grid[0][2].Formula = "B1+5"

	cycleDAG := BuildDAG(wb)
	if !cycleDAG.HasCycles {
		t.Errorf("expected cycle detection, but none reported")
	}
	if len(cycleDAG.CycleNodes) == 0 {
		t.Errorf("expected cycle nodes list to be populated")
	}
}

func TestHealFormula(t *testing.T) {
	broken := "SUM(#REF!)"
	healed := HealFormula(broken, 1, 5) // Col 1 (B), Row 5 (B6)
	if strings.Contains(healed, "#REF!") {
		t.Errorf("healed formula still has #REF!: %s", healed)
	}
	if !strings.Contains(healed, "B2:B5") {
		t.Errorf("expected reconstructed range B2:B5, got %s", healed)
	}
}

func TestSynthesizeFormulas(t *testing.T) {
	sheet := &openxml.SheetData{
		Name:   "P&L",
		MaxRow: 4,
		MaxCol: 2,
		Grid: [][]openxml.Cell{
			{{Ref: "A1", Resolved: "Item"}, {Ref: "B1", Resolved: "Amount"}},
			{{Ref: "A2", Resolved: "Sales", Type: "s"}, {Ref: "B2", Resolved: "100", Type: "n"}},
			{{Ref: "A3", Resolved: "Services", Type: "s"}, {Ref: "B3", Resolved: "200", Type: "n"}},
			{{Ref: "A4", Resolved: "Total", Type: "s"}, {Ref: "B4", Resolved: "300", Type: "n"}},
		},
	}

	opts := DefaultSynthesisOptions()
	created := SynthesizeFormulas(sheet, 1, 2, []int{3}, opts)
	if created != 1 {
		t.Fatalf("expected 1 formula created, got %d", created)
	}

	totCell := sheet.Grid[3][1]
	if !strings.Contains(totCell.Formula, "SUM(B2:B3)") {
		t.Errorf("expected SUM(B2:B3) in formula, got %s", totCell.Formula)
	}
}
