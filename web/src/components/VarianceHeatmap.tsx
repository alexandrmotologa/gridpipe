import React, { useState } from 'react';
import { GitCompare, CheckCircle, AlertCircle, HelpCircle, ArrowUpRight, ArrowDownRight } from 'lucide-react';
import { ReconciliationReport, SheetOverview } from '../types';

interface VarianceHeatmapProps {
  sheets: SheetOverview[];
  onReconcile: (sheetA: string, sheetB: string, keyCol: number, valCol: number, epsilon: number) => void;
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
  const [keyCol, setKeyCol] = useState(0);
  const [valCol, setValCol] = useState(1);
  const [epsilon, setEpsilon] = useState(0.01);
  const [filter, setFilter] = useState<'ALL' | 'VARIANCE' | 'MISSING'>('ALL');

  const filteredItems = report?.items.filter((item) => {
    if (filter === 'VARIANCE') return item.status === 'VARIANCE';
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
            <span className="text-slate-400 font-medium">Key Col Idx</span>
            <input
              type="number"
              min="0"
              value={keyCol}
              onChange={(e) => setKeyCol(Number(e.target.value))}
              className="w-16 bg-slate-900 border border-slate-700 text-slate-200 text-xs rounded px-2 py-1.5 text-center outline-none focus:border-sky-500"
            />
          </div>

          <div className="flex flex-col gap-1 text-xs">
            <span className="text-slate-400 font-medium">Value Col Idx</span>
            <input
              type="number"
              min="0"
              value={valCol}
              onChange={(e) => setValCol(Number(e.target.value))}
              className="w-16 bg-slate-900 border border-slate-700 text-slate-200 text-xs rounded px-2 py-1.5 text-center outline-none focus:border-sky-500"
            />
          </div>
        </div>

        <button
          onClick={() => onReconcile(sheetA, sheetB, keyCol, valCol, epsilon)}
          disabled={loading || sheets.length < 2}
          className="px-4 py-2 bg-sky-600 hover:bg-sky-500 disabled:bg-slate-800 disabled:text-slate-600 text-white font-medium text-xs rounded-md shadow flex items-center gap-1.5 transition-all"
        >
          <GitCompare className="w-3.5 h-3.5" />
          {loading ? 'Reconciling...' : 'Run Cross-Tab Reconciliation'}
        </button>
      </div>

      {/* KPI Cards */}
      {report && (
        <div className="grid grid-cols-5 gap-3 mb-4">
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
            <div className="text-sm font-bold font-mono text-emerald-400">{report.matched_rows}</div>
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
                  <th className="p-2.5">Line Item Key</th>
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
                  const isVariance = row.status === 'VARIANCE';

                  return (
                    <tr
                      key={idx}
                      className={`hover:bg-slate-800/30 transition-colors ${
                        isMatch ? 'bg-emerald-950/20' : 'bg-rose-950/20'
                      }`}
                    >
                      <td className="p-2.5 font-bold text-slate-200">{row.key}</td>
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
            Select source and comparison sheets above and click "Run Cross-Tab Reconciliation" to audit variances and inject formulas.
          </p>
        </div>
      )}
    </div>
  );
};
