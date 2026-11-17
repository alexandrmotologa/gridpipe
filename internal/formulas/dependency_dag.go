package formulas

import (
	"fmt"

	"github.com/alexandrmotologa/gridpipe/internal/openxml"
)

// DAGNode represents a cell in the formula dependency graph.
type DAGNode struct {
	ID        string   `json:"id"`        // e.g. "Sheet1!B10" or "B10"
	CellRef   string   `json:"cell_ref"`  // "B10"
	SheetName string   `json:"sheet_name"`
	Formula   string   `json:"formula"`
	DependsOn []string `json:"depends_on"` // Nodes that this node reads from
	UsedBy    []string `json:"used_by"`    // Nodes that read from this node
	InDegree  int      `json:"in_degree"`
}

// DependencyDAG stores the entire calculation graph across sheets.
type DependencyDAG struct {
	Nodes        map[string]*DAGNode `json:"nodes"`
	HasCycles    bool                `json:"has_cycles"`
	CycleNodes   []string            `json:"cycle_nodes,omitempty"`
	EvalOrder    []string            `json:"eval_order"`
	TotalEdges   int                 `json:"total_edges"`
}

// NewDependencyDAG initializes an empty DAG.
func NewDependencyDAG() *DependencyDAG {
	return &DependencyDAG{
		Nodes:     make(map[string]*DAGNode),
		EvalOrder: make([]string, 0),
	}
}

// BuildDAG creates the dependency graph for all formula cells in a workbook.
func BuildDAG(wb *openxml.Workbook) *DependencyDAG {
	dag := NewDependencyDAG()

	// 1. Register all formula cells as nodes
	for _, sheet := range wb.Sheets {
		sheetName := sheet.Name
		for r := 0; r < sheet.MaxRow; r++ {
			for c := 0; c < sheet.MaxCol; c++ {
				cell := sheet.Grid[r][c]
				if cell.Formula != "" {
					nodeID := fmt.Sprintf("%s!%s", sheetName, cell.Ref)
					dag.Nodes[nodeID] = &DAGNode{
						ID:        nodeID,
						CellRef:   cell.Ref,
						SheetName: sheetName,
						Formula:   cell.Formula,
						DependsOn: make([]string, 0),
						UsedBy:    make([]string, 0),
					}
				}
			}
		}
	}

	// 2. Link dependencies
	for nodeID, node := range dag.Nodes {
		refs := ExtractReferences(node.Formula)
		for _, ref := range refs {
			targetSheet := ref.SheetName
			if targetSheet == "" {
				targetSheet = node.SheetName
			}

			// If it's a range, expand all cells in range
			for r := ref.StartRow; r <= ref.EndRow; r++ {
				for c := ref.StartCol; c <= ref.EndCol; c++ {
					targetCellRef := openxml.CoordsToCellRef(c, r)
					targetNodeID := fmt.Sprintf("%s!%s", targetSheet, targetCellRef)

					// Add dependency
					node.DependsOn = append(node.DependsOn, targetNodeID)
					dag.TotalEdges++

					// If target is also a formula node, record UsedBy
					if targetNode, exists := dag.Nodes[targetNodeID]; exists {
						targetNode.UsedBy = append(targetNode.UsedBy, nodeID)
					}
				}
			}
		}
	}

	// 3. Cycle detection and topological sort using Kahn's algorithm
	inDegree := make(map[string]int)
	for id, n := range dag.Nodes {
		inDegree[id] = 0
		for _, depID := range n.DependsOn {
			// Only count dependencies that are themselves formula nodes
			if _, isFormula := dag.Nodes[depID]; isFormula {
				inDegree[id]++
			}
		}
		n.InDegree = inDegree[id]
	}

	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	order := make([]string, 0, len(dag.Nodes))
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		order = append(order, curr)

		if node, exists := dag.Nodes[curr]; exists {
			for _, user := range node.UsedBy {
				inDegree[user]--
				if inDegree[user] == 0 {
					queue = append(queue, user)
				}
			}
		}
	}

	dag.EvalOrder = order
	if len(order) < len(dag.Nodes) {
		dag.HasCycles = true
		// Find remaining nodes with inDegree > 0
		for id, deg := range inDegree {
			if deg > 0 {
				dag.CycleNodes = append(dag.CycleNodes, id)
			}
		}
	}

	return dag
}
