import React, { useState } from 'react';
import { GitCompare, CheckCircle, AlertCircle, HelpCircle, Sparkles, Filter } from 'lucide-react';
import { ReconciliationReport, SheetOverview } from '../types';

interface VarianceHeatmapProps {
  sheets: SheetOverview[];
  onReconcile: (
    sheetA: string,
    sheetB: string,
    keyCols: number[],
    valCol: number,
    epsilon: number,
    enableFuzzy: boolean,
    fuzzyThreshold: number
  ) => void;
  report: ReconciliationReport | null;
  loading: boolean;
}

export const VarianceHeatmap: React.FC<VarianceHeatmapProps> = ({
  sheets,
  onReconcile,
  report,
  loading,
}) => {
  const [sheetA, setSheetA] = useState(sheets[0]?.name || '');
  const [sheetB, setSheetB] = useState(sheets[1]?.name || sheets[0]?.name || '');
  const [keyColsStr, setKeyColsStr] = useState('0');
  const [valCol, setValCol] = useState(1);
  const [epsilon, setEpsilon] = useState(0.01);
  const [enableFuzzy, setEnableFuzzy] = useState(true);
  const [fuzzyThreshold, setFuzzyThreshold] = useState(0.85);
  const [filter, setFilter] = useState<'ALL' | 'VARIANCE' | 'FUZZY' | 'MISSING'>('ALL');

  const parseKeyCols = (s: string): number[] => {
    return s
      .split(',')
      .map((p) => parseInt(p.trim(), 10))
      .filter((n) => !isNaN(n));
  };

  const handleRun = () => {
    const kCols = parseKeyCols(keyColsStr);
    onReconcile(
      sheetA,
      sheetB,
      kCols.length > 0 ? kCols : [0],
      valCol,
      epsilon,
      enableFuzzy,
      fuzzyThreshold
    );
  };

  const filteredItems = report?.items.filter((item) => {
    if (filter === 'VARIANCE') return item.status === 'VARIANCE';
    if (filter === 'FUZZY') return item.match_type === 'FUZZY';
    if (filter === 'MISSING')
      return item.status === 'MISSING_IN_TARGET' || item.status === 'MISSING_IN_SOURCE';
    return true;
  });

  return (
    <div className="flex flex-col h-full bg-slate-900 border border-slate-800 rounded-lg overflow-hidden shadow-xl p-4">
      {/* Configuration Header */}
      <div className="bg-slate-950 p-3 rounded-lg border border-slate-800 mb-4 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex flex-col gap-1 text-xs">
            <span className="text-slate-400 font-medium">Baseline (Sheet A)</span>
            <select
              value={sheetA}
              onChange={(e) => setSheetA(e.target.value)}
              className="bg-slate-900 border border-slate-700 text-slate-200 text-xs rounded px-2.5 py-1.5 outline-none focus:border-sky-500"
            >
              {sheets.map((s) => (
                <option key={s.name} value={s.name}>
                  {s.name}
                </option>
              ))}
            </select>
          </div>

          <div className="flex flex-col gap-1 text-xs">
            <span className="text-slate-400 font-medium">Comparison (Sheet B)</span>
            <select
              value={sheetB}
              onChange={(e) => setSheetB(e.target.value)}
              className="bg-slate-900 border border-slate-700 text-slate-200 text-xs rounded px-2.5 py-1.5 outline-none focus:border-sky-500"
            >
              {sheets.map((s) => (
                <option key={s.name} value={s.name}>
                  {s.name}
                </option>
              ))}
            </select>
          </div>

          <div className="flex flex-col gap-1 text-xs">
            <span className="text-slate-400 font-medium">Key Col(s) (e.g. 0,1)</span>
            <input
              type="text"
              value={keyColsStr}
              onChange={(e) => setKeyColsStr(e.target.value)}
              placeholder="0 or 0,1"
              className="w-20 bg-slate-900 border border-slate-700 text-slate-200 text-xs rounded px-2 py-1.5 text-center outline-none focus:border-sky-500 font-mono"
            />
          </div>

          <div className="flex flex-col gap-1 text-xs">
            <span className="text-slate-400 font-medium">Value Col</span>
            <input
              type="number"
              min="0"
              value={valCol}
              onChange={(e) => setValCol(Number(e.target.value))}
              className="w-16 bg-slate-900 border border-slate-700 text-slate-200 text-xs rounded px-2 py-1.5 text-center outline-none focus:border-sky-500"
            />
          </div>

          {/* Fuzzy Matching Controls */}
          <div className="flex items-center gap-2 pl-2 border-l border-slate-800 text-xs">
            <label className="flex items-center gap-1.5 cursor-pointer text-slate-300">
              <input
                type="checkbox"
                checked={enableFuzzy}
                onChange={(e) => setEnableFuzzy(e.target.checked)}
                className="rounded bg-slate-900 border-slate-700 text-sky-500 focus:ring-0"
              />
              <span className="flex items-center gap-1">
                <Sparkles className="w-3.5 h-3.5 text-sky-400" />
                Fuzzy Match
              </span>
            </label>

            {enableFuzzy && (
              <div className="flex items-center gap-1.5 text-[11px] text-slate-400 font-mono">
                <span>{Math.round(fuzzyThreshold * 100)}%</span>
                <input
                  type="range"
                  min="0.70"
                  max="0.95"
                  step="0.05"
                  value={fuzzyThreshold}
                  onChange={(e) => setFuzzyThreshold(parseFloat(e.target.value))}
                  className="w-16 accent-sky-500 cursor-pointer"
                  title="Fuzzy similarity threshold"
                />
              </div>
            )}
          </div>
        </div>

        <button
          onClick={handleRun}
          disabled={loading || sheets.length < 2}
          className="px-4 py-2 bg-sky-600 hover:bg-sky-500 disabled:bg-slate-800 disabled:text-slate-600 text-white font-medium text-xs rounded-md shadow flex items-center gap-1.5 transition-all"
        >
          <GitCompare className="w-3.5 h-3.5" />
          {loading ? 'Reconciling...' : 'Run Reconciliation'}
        </button>
      </div>

      {/* KPI Cards */}
      {report && (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3 mb-4">
          <div className="bg-slate-950 p-3 rounded border border-slate-800 text-xs">
            <div className="text-slate-500 mb-1">Baseline Total (A)</div>
            <div className="text-sm font-bold font-mono text-slate-200">
              {report.sum_value_a.toLocaleString(undefined, { minimumFractionDigits: 2 })}
            </div>
          </div>
          <div className="bg-slate-950 p-3 rounded border border-slate-800 text-xs">
            <div className="text-slate-500 mb-1">Comparison Total (B)</div>
            <div className="text-sm font-bold font-mono text-slate-200">
              {report.sum_value_b.toLocaleString(undefined, { minimumFractionDigits: 2 })}
            </div>
          </div>
          <div className="bg-slate-950 p-3 rounded border border-slate-800 text-xs">
            <div className="text-slate-500 mb-1">Net Variance Delta</div>
            <div
              className={`text-sm font-bold font-mono flex items-center gap-1 ${
                report.net_delta === 0 ? 'text-emerald-400' : 'text-amber-400'
              }`}
            >
              {report.net_delta > 0 && '+'}
              {report.net_delta.toLocaleString(undefined, { minimumFractionDigits: 2 })}
            </div>
          </div>
          <div className="bg-slate-950 p-3 rounded border border-slate-800 text-xs">
            <div className="text-slate-500 mb-1">Matched Items</div>
            <div className="text-sm font-bold font-mono text-emerald-400">
              {report.matched_rows}
              {report.fuzzy_matches > 0 && (
                <span className="text-[11px] text-sky-400 font-normal ml-1">
                  ({report.fuzzy_matches} fuzzy)
                </span>
              )}
            </div>
          </div>
          <div className="bg-slate-950 p-3 rounded border border-slate-800 text-xs">
            <div className="text-slate-500 mb-1">Discrepancies / Missing</div>
            <div className="text-sm font-bold font-mono text-rose-400">
              {report.discrepancies + report.missing_in_a + report.missing_in_b}
            </div>
          </div>
        </div>
      )}

      {/* Filter and Table View */}
      {report ? (
        <div className="flex-1 flex flex-col bg-slate-950 rounded border border-slate-800 overflow-hidden">
          <div className="p-2.5 border-b border-slate-800 flex items-center justify-between">
            <div className="flex gap-1 bg-slate-900 p-0.5 rounded border border-slate-800 text-xs">
              <button
                onClick={() => setFilter('ALL')}
                className={`px-3 py-1 rounded transition-colors ${
                  filter === 'ALL' ? 'bg-sky-600 text-white font-medium' : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                All Items ({report.items.length})
              </button>
              <button
                onClick={() => setFilter('VARIANCE')}
                className={`px-3 py-1 rounded transition-colors ${
                  filter === 'VARIANCE' ? 'bg-sky-600 text-white font-medium' : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                Variances ({report.discrepancies})
              </button>
              <button
                onClick={() => setFilter('FUZZY')}
                className={`px-3 py-1 rounded transition-colors ${
                  filter === 'FUZZY' ? 'bg-sky-600 text-white font-medium' : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                Fuzzy Matches ({report.fuzzy_matches})
              </button>
              <button
                onClick={() => setFilter('MISSING')}
                className={`px-3 py-1 rounded transition-colors ${
                  filter === 'MISSING' ? 'bg-sky-600 text-white font-medium' : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                Missing ({report.missing_in_a + report.missing_in_b})
              </button>
            </div>
          </div>

          <div className="flex-1 overflow-auto">
            <table className="w-full text-left text-xs border-collapse font-mono">
              <thead className="sticky top-0 bg-slate-900 border-b border-slate-800 text-slate-400">
                <tr>
                  <th className="p-2.5">Key (A)</th>
                  <th className="p-2.5">Matched Key (B)</th>
                  <th className="p-2.5">Match Type</th>
                  <th className="p-2.5 text-right">Baseline (A)</th>
                  <th className="p-2.5 text-right">Comparison (B)</th>
                  <th className="p-2.5 text-right">Delta (B - A)</th>
                  <th className="p-2.5 text-right">% Variance</th>
                  <th className="p-2.5 text-center">Status</th>
                  <th className="p-2.5">Audit Note</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60">
                {filteredItems?.map((row, idx) => {
                  const isMatch = row.status === 'MATCH';
                  const isFuzzy = row.match_type === 'FUZZY';

                  return (
                    <tr
                      key={idx}
                      className={`hover:bg-slate-800/30 transition-colors ${
                        isMatch ? 'bg-emerald-950/20' : 'bg-rose-950/20'
                      }`}
                    >
                      <td className="p-2.5 font-bold text-slate-200">{row.key}</td>
                      <td className="p-2.5 text-slate-400 font-mono text-[11px]">
                        {row.matched_key_b || '-'}
                      </td>
                      <td className="p-2.5">
                        <span
                          className={`px-2 py-0.5 rounded text-[10px] font-semibold ${
                            row.match_type === 'EXACT'
                              ? 'bg-slate-800 text-slate-300'
                              : isFuzzy
                              ? 'bg-sky-500/10 text-sky-400 border border-sky-500/20'
                              : 'text-slate-600'
                          }`}
                        >
                          {row.match_type}
                          {isFuzzy && ` (${Math.round(row.confidence * 100)}%)`}
                        </span>
                      </td>
                      <td className="p-2.5 text-right text-slate-300">
                        {row.value_a.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                      </td>
                      <td className="p-2.5 text-right text-slate-300">
                        {row.value_b.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                      </td>
                      <td
                        className={`p-2.5 text-right font-bold ${
                          isMatch ? 'text-emerald-400' : 'text-rose-400'
                        }`}
                      >
                        {row.delta > 0 && '+'}
                        {row.delta.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                      </td>
                      <td className="p-2.5 text-right text-slate-400">
                        {row.percent_delta !== 0 ? `${row.percent_delta.toFixed(1)}%` : '0.0%'}
                      </td>
                      <td className="p-2.5 text-center">
                        <span
                          className={`px-2 py-0.5 rounded-full text-[10px] font-semibold tracking-wider ${
                            isMatch
                              ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                              : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                          }`}
                        >
                          {row.status}
                        </span>
                      </td>
                      <td className="p-2.5 text-slate-400 truncate max-w-sm" title={row.audit_note}>
                        {row.audit_note}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <div className="flex-1 flex flex-col items-center justify-center p-12 text-slate-500 bg-slate-950 rounded border border-slate-800">
          <GitCompare className="w-10 h-10 text-slate-700 mb-2" />
          <p className="font-medium text-slate-400">Ready for Cross-Tab Reconciliation</p>
          <p className="text-xs text-slate-600 mt-1 max-w-sm text-center">
            Select source and comparison sheets above and click "Run Reconciliation" to audit variances and inject formulas.
          </p>
        </div>
      )}
    </div>
  );
};
