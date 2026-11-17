import React, { useState, useEffect } from 'react';
import { ShieldCheck, AlertTriangle, AlertOctagon, Info, CheckCircle2, RefreshCw } from 'lucide-react';
import { DataQualityScorecard } from '../types';

interface AuditScorecardProps {
  sheetIdx: number;
}

export const AuditScorecard: React.FC<AuditScorecardProps> = ({ sheetIdx }) => {
  const [scorecard, setScorecard] = useState<DataQualityScorecard | null>(null);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState<'ALL' | 'HIGH' | 'MEDIUM' | 'LOW'>('ALL');

  const fetchScorecard = async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/audit?sheet=${sheetIdx}`);
      if (res.ok) {
        const data = await res.json();
        setScorecard(data);
      }
    } catch (err) {
      console.error('Failed to fetch scorecard:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchScorecard();
  }, [sheetIdx]);

  if (!scorecard) {
    return (
      <div className="p-12 text-center text-slate-500 bg-slate-900 border border-slate-800 rounded-lg">
        <ShieldCheck className="w-8 h-8 text-slate-600 mx-auto mb-2" />
        <p className="font-medium text-slate-400">Loading Quality Scorecard...</p>
      </div>
    );
  }

  const filteredIssues = scorecard.issues.filter((issue) => {
    if (filter === 'HIGH') return issue.severity === 'HIGH';
    if (filter === 'MEDIUM') return issue.severity === 'MEDIUM';
    if (filter === 'LOW') return issue.severity === 'LOW';
    return true;
  });

  const getGradeColor = (grade: string) => {
    switch (grade) {
      case 'A':
        return 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20';
      case 'B':
        return 'text-sky-400 bg-sky-500/10 border-sky-500/20';
      case 'C':
        return 'text-amber-400 bg-amber-500/10 border-amber-500/20';
      default:
        return 'text-rose-400 bg-rose-500/10 border-rose-500/20';
    }
  };

  return (
    <div className="flex flex-col h-full bg-slate-900 border border-slate-800 rounded-lg overflow-hidden shadow-xl p-4">
      {/* Header */}
      <div className="flex items-center justify-between pb-3 border-b border-slate-800 mb-4">
        <div>
          <h2 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
            <ShieldCheck className="w-4 h-4 text-emerald-400" />
            Autonomous Data Quality & Compliance Scorecard
          </h2>
          <p className="text-xs text-slate-400 mt-0.5">
            Statistical outlier detection, duplicate analysis, sign inversion flags, and type conformance
          </p>
        </div>

        <button
          onClick={fetchScorecard}
          disabled={loading}
          className="p-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded border border-slate-700 transition-colors"
          title="Refresh Audit"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
        </button>
      </div>

      {/* Top Quality Score Gauge & Metrics */}
      <div className="grid grid-cols-12 gap-4 mb-4">
        {/* Main Scorecard Card */}
        <div className="col-span-4 bg-slate-950 p-4 rounded-xl border border-slate-800 flex items-center justify-between">
          <div>
            <div className="text-xs font-medium text-slate-400">Overall Quality Index</div>
            <div className="text-3xl font-extrabold font-mono text-white mt-1">
              {scorecard.overall_score.toFixed(1)}
              <span className="text-sm font-normal text-slate-500"> / 100</span>
            </div>
            <div className="text-[11px] text-slate-500 mt-1">
              {scorecard.total_rows} rows · {scorecard.total_columns} columns · {scorecard.total_cells} cells
            </div>
          </div>

          <div
            className={`w-14 h-14 rounded-2xl flex items-center justify-center font-black text-2xl border shadow-inner ${getGradeColor(
              scorecard.grade
            )}`}
          >
            {scorecard.grade}
          </div>
        </div>

        {/* 4 Pillars Breakdown */}
        <div className="col-span-8 bg-slate-950 p-4 rounded-xl border border-slate-800 grid grid-cols-4 gap-4 text-xs font-mono">
          <div>
            <div className="text-slate-500 mb-1 flex items-center justify-between">
              <span>Completeness</span>
              <span className="text-slate-300">{Math.round(scorecard.completeness_score)}%</span>
            </div>
            <div className="w-full bg-slate-900 h-1.5 rounded-full overflow-hidden">
              <div
                className="bg-emerald-500 h-full rounded-full"
                style={{ width: `${scorecard.completeness_score}%` }}
              />
            </div>
            <div className="text-[10px] text-slate-600 mt-1">{scorecard.empty_cells} empty cells</div>
          </div>

          <div>
            <div className="text-slate-500 mb-1 flex items-center justify-between">
              <span>Conformance</span>
              <span className="text-slate-300">{Math.round(scorecard.conformance_score)}%</span>
            </div>
            <div className="w-full bg-slate-900 h-1.5 rounded-full overflow-hidden">
              <div
                className="bg-sky-500 h-full rounded-full"
                style={{ width: `${scorecard.conformance_score}%` }}
              />
            </div>
            <div className="text-[10px] text-slate-600 mt-1">Type strictness</div>
          </div>

          <div>
            <div className="text-slate-500 mb-1 flex items-center justify-between">
              <span>DAG Integrity</span>
              <span className="text-slate-300">{Math.round(scorecard.formula_integrity_score)}%</span>
            </div>
            <div className="w-full bg-slate-900 h-1.5 rounded-full overflow-hidden">
              <div
                className="bg-indigo-500 h-full rounded-full"
                style={{ width: `${scorecard.formula_integrity_score}%` }}
              />
            </div>
            <div className="text-[10px] text-slate-600 mt-1">Zero formula error</div>
          </div>

          <div>
            <div className="text-slate-500 mb-1 flex items-center justify-between">
              <span>Uniqueness</span>
              <span className="text-slate-300">{Math.round(scorecard.uniqueness_score)}%</span>
            </div>
            <div className="w-full bg-slate-900 h-1.5 rounded-full overflow-hidden">
              <div
                className="bg-teal-500 h-full rounded-full"
                style={{ width: `${scorecard.uniqueness_score}%` }}
              />
            </div>
            <div className="text-[10px] text-slate-600 mt-1">{scorecard.duplicate_rows_count} duplicates</div>
          </div>
        </div>
      </div>

      {/* Filter and Anomalies List */}
      <div className="flex-1 flex flex-col bg-slate-950 rounded-xl border border-slate-800 overflow-hidden">
        <div className="p-3 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-xs font-semibold text-slate-300">
              Detected Anomalies & Outliers ({scorecard.anomalies_count})
            </span>
          </div>

          <div className="flex gap-1 bg-slate-900 p-0.5 rounded border border-slate-800 text-xs">
            <button
              onClick={() => setFilter('ALL')}
              className={`px-2.5 py-0.5 rounded transition-colors ${
                filter === 'ALL' ? 'bg-sky-600 text-white font-medium' : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              All ({scorecard.issues.length})
            </button>
            <button
              onClick={() => setFilter('HIGH')}
              className={`px-2.5 py-0.5 rounded transition-colors ${
                filter === 'HIGH' ? 'bg-rose-600 text-white font-medium' : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              High Severity
            </button>
            <button
              onClick={() => setFilter('MEDIUM')}
              className={`px-2.5 py-0.5 rounded transition-colors ${
                filter === 'MEDIUM' ? 'bg-amber-600 text-white font-medium' : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              Medium
            </button>
            <button
              onClick={() => setFilter('LOW')}
              className={`px-2.5 py-0.5 rounded transition-colors ${
                filter === 'LOW' ? 'bg-slate-700 text-white font-medium' : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              Low / Outliers
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-auto">
          {filteredIssues.length === 0 ? (
            <div className="p-12 text-center text-slate-500 font-mono text-xs">
              <CheckCircle2 className="w-8 h-8 text-emerald-500 mx-auto mb-2" />
              <p className="font-semibold text-slate-300">No Data Quality Violations</p>
              <p className="text-slate-500 text-[11px] mt-0.5">
                All records conform to inferred types with zero statistical outliers or duplicate keys.
              </p>
            </div>
          ) : (
            <table className="w-full text-left text-xs border-collapse font-mono">
              <thead className="sticky top-0 bg-slate-900 text-slate-400 border-b border-slate-800">
                <tr>
                  <th className="p-2.5">Severity</th>
                  <th className="p-2.5">Category</th>
                  <th className="p-2.5">Cell / Coordinate</th>
                  <th className="p-2.5">Field Name</th>
                  <th className="p-2.5">Value</th>
                  <th className="p-2.5">Audit Findings</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60">
                {filteredIssues.map((issue, idx) => (
                  <tr key={idx} className="hover:bg-slate-900/50 transition-colors">
                    <td className="p-2.5">
                      <span
                        className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${
                          issue.severity === 'HIGH'
                            ? 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                            : issue.severity === 'MEDIUM'
                            ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                            : 'bg-slate-800 text-slate-400 border border-slate-700'
                        }`}
                      >
                        {issue.severity}
                      </span>
                    </td>
                    <td className="p-2.5 text-slate-300 font-semibold">{issue.category}</td>
                    <td className="p-2.5 text-sky-400">{issue.cell_ref || `Row ${issue.row + 1}`}</td>
                    <td className="p-2.5 text-slate-300">{issue.column_name}</td>
                    <td className="p-2.5 text-slate-200 truncate max-w-[120px]">{issue.value || '-'}</td>
                    <td className="p-2.5 text-slate-400 max-w-md">{issue.description}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </div>
  );
};
