import React, { useState } from 'react';
import { Calculator, Hash, Calendar, DollarSign, AlertCircle, FileSpreadsheet, Check } from 'lucide-react';
import { NormalizedSheetResult } from '../types';

interface GridCanvasProps {
  normalized?: NormalizedSheetResult;
  rawPreview?: string[][];
  sheetName: string;
  sheetIdx?: number;
  onCellEdited?: (row: number, col: number, newVal: string) => void;
}

export const GridCanvas: React.FC<GridCanvasProps> = ({
  normalized,
  rawPreview,
  sheetName,
  sheetIdx = 0,
  onCellEdited,
}) => {
  const [selectedCell, setSelectedCell] = useState<{ r: number; c: number } | null>({ r: 0, c: 0 });
  const [viewMode, setViewMode] = useState<'clean' | 'raw'>('clean');
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState('');
  const [saving, setSaving] = useState(false);

  const isClean = viewMode === 'clean' && normalized;
  const columns = isClean ? normalized.columns : rawPreview?.[0]?.map((_, i) => String.fromCharCode(65 + i)) || [];
  const rows = isClean ? normalized.rows : rawPreview?.slice(1) || [];

  const activeVal = selectedCell
    ? (isClean ? normalized?.rows[selectedCell.r]?.[selectedCell.c] : rawPreview?.[selectedCell.r + 1]?.[selectedCell.c]) || ''
    : '';

  const activeColName = selectedCell && columns[selectedCell.c] ? columns[selectedCell.c] : '';
  const activeColType = isClean && selectedCell ? normalized?.inferred_types[selectedCell.c] : undefined;

  const colLetter = selectedCell ? String.fromCharCode(65 + selectedCell.c) : 'A';
  const rowNumber = selectedCell ? selectedCell.r + 1 : 1;
  const cellCoord = `${colLetter}${rowNumber}`;

  const handleStartEdit = (r: number, c: number, currentVal: string) => {
    setSelectedCell({ r, c });
    setEditValue(currentVal);
    setIsEditing(true);
  };

  const handleCommitEdit = async () => {
    if (!selectedCell) return;
    setIsEditing(false);
    setSaving(true);

    try {
      const res = await fetch('/api/cell/edit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          sheet_idx: sheetIdx,
          row: isClean ? selectedCell.r + 1 : selectedCell.r + 1, // data row
          col: selectedCell.c,
          value: editValue,
        }),
      });

      if (res.ok) {
        if (isClean && normalized && normalized.rows[selectedCell.r]) {
          normalized.rows[selectedCell.r][selectedCell.c] = editValue;
        }
        onCellEdited?.(selectedCell.r, selectedCell.c, editValue);
      }
    } catch (err) {
      console.error('Failed to update cell:', err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex flex-col h-full bg-slate-900 border border-slate-800 rounded-lg overflow-hidden shadow-xl">
      {/* Top Toolbar & Formula Bar */}
      <div className="bg-slate-950 border-b border-slate-800 p-2.5 flex flex-col gap-2">
        <div className="flex items-center justify-between gap-2 overflow-x-auto scrollbar-none">
          <div className="flex items-center gap-2 shrink-0 whitespace-nowrap">
            <span className="text-xs font-semibold px-2 py-0.5 rounded bg-sky-500/10 text-sky-400 border border-sky-500/20 flex items-center gap-1 whitespace-nowrap shrink-0">
              <FileSpreadsheet className="w-3.5 h-3.5 shrink-0" />
              <span>{sheetName}</span>
            </span>
            <div className="flex bg-slate-900 rounded-md p-0.5 border border-slate-800 text-xs shrink-0 whitespace-nowrap">
              <button
                onClick={() => setViewMode('clean')}
                className={`px-2.5 py-1 rounded transition-colors whitespace-nowrap shrink-0 ${
                  viewMode === 'clean' ? 'bg-sky-600 text-white font-medium' : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                Canonical Grid {normalized && `(${normalized.rows.length} rows)`}
              </button>
              <button
                onClick={() => setViewMode('raw')}
                className={`px-2.5 py-1 rounded transition-colors whitespace-nowrap shrink-0 ${
                  viewMode === 'raw' ? 'bg-sky-600 text-white font-medium' : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                Raw Input View
              </button>
            </div>
            {saving && (
              <span className="text-xs text-sky-400 font-mono animate-pulse whitespace-nowrap shrink-0">
                Saving cell...
              </span>
            )}
          </div>

          {activeColType && (
            <div className="flex items-center gap-2 text-xs shrink-0 whitespace-nowrap">
              <span className="text-slate-400 whitespace-nowrap">Column Type:</span>
              <span className="px-2 py-0.5 rounded-full bg-slate-800 text-slate-300 border border-slate-700 font-mono whitespace-nowrap">
                {activeColType.type.toUpperCase()}
                {activeColType.currency_code && ` (${activeColType.currency_code})`}
              </span>
              <span className="text-slate-500 text-[11px] whitespace-nowrap">
                {Math.round(activeColType.confidence * 100)}% confidence
              </span>
            </div>
          )}
        </div>

        {/* Formula Bar */}
        <div className="flex items-center gap-2 bg-slate-900 border border-slate-800 rounded px-2.5 py-1.5 text-xs font-mono">
          <span className="text-sky-400 font-bold w-12 border-r border-slate-800 pr-2">
            {cellCoord}
          </span>
          <span className="text-slate-500">fx</span>
          <input
            type="text"
            value={isEditing ? editValue : activeVal}
            onChange={(e) => {
              if (isEditing) setEditValue(e.target.value);
            }}
            onFocus={() => {
              if (selectedCell) handleStartEdit(selectedCell.r, selectedCell.c, activeVal);
            }}
            onBlur={handleCommitEdit}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleCommitEdit();
              if (e.key === 'Escape') setIsEditing(false);
            }}
            className="flex-1 bg-transparent text-slate-200 outline-none"
            placeholder="Selected cell value (double-click to edit)"
          />
        </div>
      </div>

      {/* Grid Canvas */}
      <div className="flex-1 overflow-auto bg-slate-950/60 relative">
        <table className="w-full border-collapse text-xs font-mono select-none">
          <thead>
            <tr className="sticky top-0 bg-slate-900 z-10 border-b border-slate-800">
              <th className="w-12 bg-slate-950 border-r border-slate-800 text-slate-500 font-semibold p-1.5 text-center">
                #
              </th>
              {columns.map((col, idx) => (
                <th
                  key={idx}
                  className="border-r border-slate-800 text-slate-300 font-medium p-2 text-left min-w-[140px] bg-slate-900/90 backdrop-blur"
                >
                  <div className="flex items-center justify-between gap-1">
                    <span className="truncate" title={col}>
                      {col}
                    </span>
                    {isClean && normalized?.inferred_types[idx] && (
                      <span className="text-[10px] text-slate-500 font-normal">
                        {normalized.inferred_types[idx].type[0].toUpperCase()}
                      </span>
                    )}
                  </div>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, rIdx) => (
              <tr key={rIdx} className="hover:bg-slate-900/40 border-b border-slate-800/60 transition-colors">
                <td className="sticky left-0 bg-slate-950 border-r border-slate-800 text-slate-500 text-center font-mono py-1.5 px-2">
                  {rIdx + 1}
                </td>
                {columns.map((_, cIdx) => {
                  const val = row[cIdx] || '';
                  const isSelected = selectedCell?.r === rIdx && selectedCell?.c === cIdx;
                  const isThisEditing = isEditing && isSelected;

                  return (
                    <td
                      key={cIdx}
                      onClick={() => setSelectedCell({ r: rIdx, c: cIdx })}
                      onDoubleClick={() => handleStartEdit(rIdx, cIdx, val)}
                      className={`border-r border-slate-800/50 p-2 cursor-pointer truncate max-w-[200px] transition-all relative ${
                        isSelected
                          ? 'bg-sky-500/20 ring-1 ring-inset ring-sky-400 text-white'
                          : 'text-slate-300 hover:bg-slate-800/40'
                      }`}
                      title={val}
                    >
                      {isThisEditing ? (
                        <input
                          autoFocus
                          type="text"
                          value={editValue}
                          onChange={(e) => setEditValue(e.target.value)}
                          onBlur={handleCommitEdit}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') handleCommitEdit();
                            if (e.key === 'Escape') setIsEditing(false);
                          }}
                          className="w-full bg-slate-900 text-white border border-sky-400 rounded px-1 py-0.5 outline-none font-mono text-xs"
                        />
                      ) : (
                        val || <span className="text-slate-700 italic">null</span>
                      )}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>

        {rows.length === 0 && (
          <div className="p-12 text-center text-slate-500">
            No rows available. Upload a workbook or run Clean to view table grid.
          </div>
        )}
      </div>

      {/* Trailing Footnotes Bar */}
      {isClean && normalized && normalized.footnotes.length > 0 && (
        <div className="bg-slate-950 border-t border-slate-800 px-3 py-2 text-xs">
          <div className="flex items-center gap-1.5 text-amber-400 font-semibold mb-1">
            <AlertCircle className="w-3.5 h-3.5" />
            <span>Separated Metadata & Footnotes ({normalized.footnotes.length})</span>
          </div>
          <div className="space-y-1">
            {normalized.footnotes.map((fn, i) => (
              <div key={i} className="text-slate-400 font-mono text-[11px] truncate" title={fn.content}>
                Row {fn.row_index + 1}: {fn.content}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
