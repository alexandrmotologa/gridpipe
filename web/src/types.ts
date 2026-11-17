export interface CellData {
  ref: string;
  col: number;
  row: number;
  type: string;
  raw_value: string;
  formula: string;
  style_id?: number;
  resolved: string;
}

export interface SheetOverview {
  index: number;
  name: string;
  sheet_id: string;
  max_row: number;
  max_col: number;
  merge_count: number;
  preview: string[][];
}

export interface NormalizedSheetResult {
  name: string;
  columns: string[];
  inferred_types: {
    type: string;
    currency_code?: string;
    confidence: number;
    non_null_count: number;
    total_count: number;
  }[];
  rows: string[][];
  footnotes: {
    row_index: number;
    content: string;
    kind: string;
  }[];
  repairs: {
    sheet_name: string;
    cell_ref: string;
    original: string;
    repaired: string;
    error_reason: string;
  }[];
  formulas_created: number;
}

export interface RowVarianceItem {
  key: string;
  value_a: number;
  value_b: number;
  delta: number;
  percent_delta: number;
  status: 'MATCH' | 'VARIANCE' | 'MISSING_IN_TARGET' | 'MISSING_IN_SOURCE';
  audit_note: string;
}

export interface ReconciliationReport {
  total_rows_a: number;
  total_rows_b: number;
  matched_rows: number;
  discrepancies: number;
  missing_in_a: number;
  missing_in_b: number;
  sum_value_a: number;
  sum_value_b: number;
  net_delta: number;
  items: RowVarianceItem[];
}

export interface DAGNode {
  id: string;
  cell_ref: string;
  sheet_name: string;
  formula: string;
  depends_on: string[];
  used_by: string[];
  in_degree: number;
}

export interface DAGReport {
  nodes: Record<string, DAGNode>;
  has_cycles: boolean;
  cycle_nodes?: string[];
  eval_order: string[];
  total_edges: number;
}

export interface PipelineStreamEvent {
  stage: string;
  progress_pct: number;
  message: string;
  details?: any;
}
