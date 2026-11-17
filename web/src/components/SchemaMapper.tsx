import React from 'react';
import { Table, Key, CheckCircle, Sliders, ArrowRight } from 'lucide-react';
import { NormalizedSheetResult } from '../types';

interface SchemaMapperProps {
  normalized?: NormalizedSheetResult;
  onUpdateType?: (colIdx: number, newType: string) => void;
}

export const SchemaMapper: React.FC<SchemaMapperProps> = ({ normalized, onUpdateType }) => {
  if (!normalized) {
    return (
      <div className="p-12 text-center text-slate-500 bg-slate-900 border border-slate-800 rounded-lg">
        Run pipeline normalization on an uploaded workbook to inspect inferred schema mappings.
      </div>
    );
  }

  const supportedTypes = ['text', 'integer', 'float', 'currency', 'date', 'boolean'];

  return (
    <div className="flex flex-col h-full bg-slate-900 border border-slate-800 rounded-lg overflow-hidden shadow-xl p-4">
      <div className="flex items-center justify-between pb-3 border-b border-slate-800 mb-4">
        <div>
          <h2 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
            <Sliders className="w-4 h-4 text-sky-400" />
            Enterprise Schema Alignment & Canonical Keys
          </h2>
          <p className="text-xs text-slate-400 mt-0.5">
            Columns synthesized from merged headers, type statistics, and sample distribution
          </p>
        </div>
        <div className="text-xs bg-slate-800 text-slate-300 px-3 py-1 rounded-full border border-slate-700">
          {normalized.columns.length} Canonical Columns
        </div>
      </div>

      <div className="overflow-auto flex-1">
        <table className="w-full text-left text-xs border-collapse">
          <thead>
            <tr className="border-b border-slate-800 text-slate-400 uppercase font-mono text-[11px]">
              <th className="pb-2.5 font-medium">#</th>
              <th className="pb-2.5 font-medium">Canonical Field Key</th>
              <th className="pb-2.5 font-medium">Detected Type</th>
              <th className="pb-2.5 font-medium">Confidence</th>
              <th className="pb-2.5 font-medium">Sample Values</th>
              <th className="pb-2.5 font-medium text-right">Coverage</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800/60 font-mono">
            {normalized.columns.map((col, idx) => {
              const info = normalized.inferred_types[idx];
              const samples = normalized.rows.slice(0, 3).map((r) => r[idx]).filter(Boolean);

              return (
                <tr key={idx} className="hover:bg-slate-800/30 transition-colors">
                  <td className="py-3 text-slate-500">{idx + 1}</td>
                  <td className="py-3 font-semibold text-sky-300 flex items-center gap-1.5">
                    <Key className="w-3 h-3 text-slate-500" />
                    {col}
                  </td>
                  <td className="py-3">
                    <select
                      value={info?.type || 'text'}
                      onChange={(e) => onUpdateType?.(idx, e.target.value)}
                      className="bg-slate-950 border border-slate-700 rounded px-2 py-1 text-slate-200 focus:border-sky-500 outline-none"
                    >
                      {supportedTypes.map((t) => (
                        <option key={t} value={t}>
                          {t.toUpperCase()}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td className="py-3">
                    <div className="flex items-center gap-2">
                      <div className="w-16 bg-slate-800 h-1.5 rounded-full overflow-hidden">
                        <div
                          className="bg-emerald-500 h-full rounded-full"
                          style={{ width: `${Math.round((info?.confidence || 1) * 100)}%` }}
                        />
                      </div>
                      <span className="text-slate-400 text-[11px]">
                        {Math.round((info?.confidence || 1) * 100)}%
                      </span>
                    </div>
                  </td>
                  <td className="py-3 text-slate-400 max-w-xs truncate">
                    {samples.join(' · ') || <span className="italic text-slate-600">empty</span>}
                  </td>
                  <td className="py-3 text-right text-slate-300">
                    {info?.non_null_count ?? normalized.rows.length} / {info?.total_count ?? normalized.rows.length}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
};
