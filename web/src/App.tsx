import React, { useState, useEffect, useRef } from 'react';
import {
  Upload,
  Play,
  Wrench,
  Download,
  Activity,
  Layers,
  Sliders,
  GitCommit,
  GitCompare,
  FileSpreadsheet,
  CheckCircle2,
  AlertCircle,
  ShieldCheck,
} from 'lucide-react';
import { GridCanvas } from './components/GridCanvas';
import { SchemaMapper } from './components/SchemaMapper';
import { FormulaDagView } from './components/FormulaDagView';
import { VarianceHeatmap } from './components/VarianceHeatmap';
import { AuditScorecard } from './components/AuditScorecard';
import { PipelineStream } from './components/PipelineStream';
import { SheetOverview, NormalizedSheetResult, ReconciliationReport, PipelineStreamEvent } from './types';

export const App: React.FC = () => {
  const [sheets, setSheets] = useState<SheetOverview[]>([]);
  const [activeSheetIdx, setActiveSheetIdx] = useState(0);
  const [activeTab, setActiveTab] = useState<'grid' | 'schema' | 'dag' | 'variance' | 'audit'>('grid');
  const [fileName, setFileName] = useState<string>('');
  const [normalized, setNormalized] = useState<NormalizedSheetResult[]>([]);
  const [reconciliationReport, setReconciliationReport] = useState<ReconciliationReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [streamEvents, setStreamEvents] = useState<PipelineStreamEvent[]>([]);
  const [streamOpen, setStreamOpen] = useState(false);
  const [sseConnected, setSseConnected] = useState(false);
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);

  // Connect to SSE stream
  useEffect(() => {
    const es = new EventSource('/api/stream');
    es.onopen = () => setSseConnected(true);
    es.onerror = () => setSseConnected(false);

    const stages = [
      'HEADER_SCAN',
      'BOUNDARY_DETECTION',
      'FORMULA_SYNTHESIS',
      'FORMULA_REPAIR',
      'RECONCILE_START',
      'RECONCILE_COMPLETE',
      'COMPLETE',
    ];

    stages.forEach((st) => {
      es.addEventListener(st, (ev: MessageEvent) => {
        try {
          const parsed = JSON.parse(ev.data);
          setStreamEvents((prev) => [parsed, ...prev.slice(0, 50)]);
        } catch (e) {
          // ignore
        }
      });
    });

    return () => es.close();
  }, []);

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type });
    setTimeout(() => setToast(null), 4000);
  };

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setLoading(true);
    const formData = new FormData();
    formData.append('file', file);

    try {
      const res = await fetch('/api/upload', {
        method: 'POST',
        body: formData,
      });

      if (!res.ok) {
        const err = await res.text();
        throw new Error(err);
      }

      const data = await res.json();
      setFileName(data.filename);
      setSheets(data.sheets || []);
      setActiveSheetIdx(0);
      setNormalized([]);
      setReconciliationReport(null);
      showToast(`Loaded '${data.filename}' successfully`);
    } catch (err: any) {
      showToast(err.message || 'File upload failed', 'error');
    } finally {
      setLoading(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  };

  const handleRunClean = async () => {
    setLoading(true);
    setStreamOpen(true);
    try {
      const res = await fetch('/api/clean', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          header_scan_rows: 10,
          numeric_locale: 'auto',
          trim_footnotes: true,
          synthesize_math: true,
          repair_broken_refs: true,
        }),
      });

      if (!res.ok) {
        const err = await res.text();
        throw new Error(err);
      }

      const data = await res.json();
      setNormalized(data.results || []);
      showToast('Spreadsheet normalization completed with native OpenXML formulas!');
    } catch (err: any) {
      showToast(err.message || 'Clean failed', 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleRunRepair = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/formulas/repair', { method: 'POST' });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      showToast(`Repaired ${data.total_repaired} formula references`);
    } catch (err: any) {
      showToast(err.message || 'Formula repair failed', 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleRunReconcile = async (
    sA: string,
    sB: string,
    keyCols: number[],
    vCol: number,
    eps: number,
    enableFuzzy: boolean,
    fuzzyThreshold: number
  ) => {
    setLoading(true);
    setStreamOpen(true);
    try {
      const res = await fetch('/api/reconcile', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          sheet_a: sA,
          sheet_b: sB,
          key_cols_a: keyCols,
          val_col_a: vCol,
          key_cols_b: keyCols,
          val_col_b: vCol,
          start_row: 1,
          epsilon: eps,
          enable_fuzzy: enableFuzzy,
          fuzzy_threshold: fuzzyThreshold,
        }),
      });

      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setReconciliationReport(data);
      showToast(`Reconciliation complete! Matched ${data.matched_rows} items.`);
    } catch (err: any) {
      showToast(err.message || 'Reconciliation failed', 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleExport = (format: string) => {
    window.open(`/api/export?format=${format}`, '_blank');
  };

  const currentSheet = sheets[activeSheetIdx];
  const currentNormalized = normalized[activeSheetIdx];

  return (
    <div className="flex flex-col h-screen bg-slate-950 text-slate-100 overflow-hidden font-sans">
      {/* Toast Notification */}
      {toast && (
        <div
          className={`fixed top-4 right-4 z-50 px-4 py-2.5 rounded-lg border text-xs font-medium shadow-2xl flex items-center gap-2 transition-all ${
            toast.type === 'success'
              ? 'bg-emerald-950/90 text-emerald-300 border-emerald-500/30'
              : 'bg-rose-950/90 text-rose-300 border-rose-500/30'
          }`}
        >
          {toast.type === 'success' ? (
            <CheckCircle2 className="w-4 h-4 text-emerald-400" />
          ) : (
            <AlertCircle className="w-4 h-4 text-rose-400" />
          )}
          {toast.msg}
        </div>
      )}

      {/* Top Main Navigation Bar */}
      <header className="h-14 border-b border-slate-800 bg-slate-950/80 backdrop-blur px-3 sm:px-4 flex items-center justify-between shrink-0 gap-2">
        <div className="flex items-center gap-2 sm:gap-3 shrink-0">
          <div className="flex items-center gap-2 font-bold tracking-tight text-white text-base">
            <div className="w-7 h-7 rounded-lg bg-sky-500 flex items-center justify-center text-slate-950 font-black shadow-lg shadow-sky-500/20">
              G
            </div>
            <span className="hidden xs:inline">gridpipe</span>
            <span className="hidden sm:inline text-[10px] text-sky-400 bg-sky-500/10 border border-sky-500/20 px-1.5 py-0.5 rounded font-mono font-normal">
              v1.1
            </span>
          </div>

          {fileName && (
            <div className="hidden md:flex items-center gap-2 text-xs bg-slate-900 border border-slate-800 px-3 py-1 rounded-full text-slate-300">
              <FileSpreadsheet className="w-3.5 h-3.5 text-slate-400" />
              <span className="truncate max-w-[160px] lg:max-w-[240px]">{fileName}</span>
            </div>
          )}
        </div>

        {/* Center Navigation Tabs */}
        <div className="flex items-center gap-0.5 sm:gap-1 bg-slate-900/80 border border-slate-800 p-1 rounded-lg text-xs overflow-x-auto scrollbar-none max-w-full">
          <button
            onClick={() => setActiveTab('grid')}
            title="Grid Canvas"
            className={`flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 rounded-md transition-colors shrink-0 ${
              activeTab === 'grid' ? 'bg-sky-600 text-white font-medium shadow-sm' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Layers className="w-3.5 h-3.5" />
            <span className="hidden md:inline">Grid Canvas</span>
          </button>
          <button
            onClick={() => setActiveTab('schema')}
            title="Schema Alignment"
            className={`flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 rounded-md transition-colors shrink-0 ${
              activeTab === 'schema' ? 'bg-sky-600 text-white font-medium shadow-sm' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Sliders className="w-3.5 h-3.5" />
            <span className="hidden md:inline">Schema</span>
          </button>
          <button
            onClick={() => setActiveTab('dag')}
            title="Formula DAG"
            className={`flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 rounded-md transition-colors shrink-0 ${
              activeTab === 'dag' ? 'bg-sky-600 text-white font-medium shadow-sm' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <GitCommit className="w-3.5 h-3.5" />
            <span className="hidden md:inline">Formula DAG</span>
          </button>
          <button
            onClick={() => setActiveTab('variance')}
            title="Reconciliation"
            className={`flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 rounded-md transition-colors shrink-0 ${
              activeTab === 'variance' ? 'bg-sky-600 text-white font-medium shadow-sm' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <GitCompare className="w-3.5 h-3.5" />
            <span className="hidden md:inline">Reconciliation</span>
          </button>
          <button
            onClick={() => setActiveTab('audit')}
            title="Quality & Audit"
            className={`flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 rounded-md transition-colors shrink-0 ${
              activeTab === 'audit' ? 'bg-sky-600 text-white font-medium shadow-sm' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <ShieldCheck className="w-3.5 h-3.5" />
            <span className="hidden md:inline">Quality & Audit</span>
          </button>
        </div>

        {/* Action Controls */}
        <div className="flex items-center gap-1.5 sm:gap-2 shrink-0">
          <input
            type="file"
            ref={fileInputRef}
            onChange={handleFileUpload}
            accept=".xlsx,.csv"
            className="hidden"
          />
          <button
            onClick={() => fileInputRef.current?.click()}
            disabled={loading}
            className="px-2.5 sm:px-3 py-1.5 bg-slate-900 hover:bg-slate-800 text-slate-200 border border-slate-700 rounded-md text-xs font-medium flex items-center gap-1.5 transition-colors"
            title="Upload File"
          >
            <Upload className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">Upload</span>
          </button>

          <button
            onClick={handleRunClean}
            disabled={loading || sheets.length === 0}
            className="px-2.5 sm:px-3 py-1.5 bg-sky-600 hover:bg-sky-500 disabled:bg-slate-800 disabled:text-slate-600 text-white rounded-md text-xs font-semibold flex items-center gap-1.5 transition-all shadow-md shadow-sky-600/20"
            title="Normalize & Synthesize Formulas"
          >
            <Play className="w-3.5 h-3.5 fill-white" />
            <span className="hidden sm:inline">Normalize</span>
          </button>

          <button
            onClick={handleRunRepair}
            disabled={loading || sheets.length === 0}
            className="px-2 sm:px-3 py-1.5 bg-slate-900 hover:bg-slate-800 text-slate-300 border border-slate-800 rounded-md text-xs flex items-center gap-1.5 transition-colors"
            title="Scan and heal broken references (#REF!, #VALUE!)"
          >
            <Wrench className="w-3.5 h-3.5 text-amber-400" />
            <span className="hidden lg:inline">Heal</span>
          </button>

          {/* Export Dropdown */}
          <div className="relative group">
            <button
              disabled={sheets.length === 0}
              className="px-2.5 sm:px-3 py-1.5 bg-slate-900 hover:bg-slate-800 disabled:opacity-50 text-slate-200 border border-slate-700 rounded-md text-xs font-medium flex items-center gap-1.5"
              title="Export Data"
            >
              <Download className="w-3.5 h-3.5 text-sky-400" />
              <span className="hidden sm:inline">Export</span>
            </button>
            <div className="absolute right-0 top-full mt-1 hidden group-hover:block bg-slate-900 border border-slate-800 rounded-lg shadow-xl py-1 w-44 z-40 text-xs">
              <button
                onClick={() => handleExport('xlsx')}
                className="w-full text-left px-3 py-1.5 text-slate-300 hover:bg-slate-800 hover:text-white flex items-center gap-2"
              >
                <span>Audited Excel (.xlsx)</span>
              </button>
              <button
                onClick={() => handleExport('csv')}
                className="w-full text-left px-3 py-1.5 text-slate-300 hover:bg-slate-800 hover:text-white flex items-center gap-2"
              >
                <span>Relational CSV (.csv)</span>
              </button>
              <button
                onClick={() => handleExport('jsonl')}
                className="w-full text-left px-3 py-1.5 text-slate-300 hover:bg-slate-800 hover:text-white flex items-center gap-2"
              >
                <span>JSON Lines (.jsonl)</span>
              </button>
              <button
                onClick={() => handleExport('sql')}
                className="w-full text-left px-3 py-1.5 text-slate-300 hover:bg-slate-800 hover:text-white flex items-center gap-2"
              >
                <span>SQL Schema & Inserts</span>
              </button>
            </div>
          </div>

          <button
            onClick={() => setStreamOpen(!streamOpen)}
            className={`p-1.5 rounded-md border text-xs transition-colors ${
              streamOpen
                ? 'bg-sky-500/20 text-sky-400 border-sky-500/40'
                : 'bg-slate-900 text-slate-400 border-slate-800 hover:text-slate-200'
            }`}
            title="Toggle Live Execution Stream"
          >
            <Activity className="w-4 h-4" />
          </button>
        </div>
      </header>

      {/* Sheets Tab Bar (if multi-sheet workbook) */}
      {sheets.length > 0 && (
        <div className="h-9 bg-slate-950 border-b border-slate-800/80 px-4 flex items-center gap-2 shrink-0 overflow-x-auto text-xs">
          <span className="text-slate-500 text-[11px] font-semibold uppercase tracking-wider mr-1">
            Worksheets:
          </span>
          {sheets.map((s, idx) => (
            <button
              key={s.name}
              onClick={() => setActiveSheetIdx(idx)}
              className={`px-3 py-1 rounded transition-colors flex items-center gap-1.5 font-mono ${
                activeSheetIdx === idx
                  ? 'bg-slate-800 text-sky-400 font-semibold border-b-2 border-sky-500'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              <FileSpreadsheet className="w-3 h-3" />
              {s.name}
            </button>
          ))}
        </div>
      )}

      {/* Main Workspace Body */}
      <main className="flex-1 p-4 overflow-hidden relative">
        {sheets.length === 0 ? (
          <div className="h-full border-2 border-dashed border-slate-800 rounded-xl flex flex-col items-center justify-center p-8 bg-slate-900/30">
            <div className="w-16 h-16 rounded-2xl bg-sky-500/10 border border-sky-500/20 flex items-center justify-center text-sky-400 mb-4 shadow-xl">
              <Upload className="w-8 h-8" />
            </div>
            <h3 className="text-base font-bold text-slate-200">Ingest Unstructured Spreadsheet</h3>
            <p className="text-xs text-slate-400 mt-1 max-w-md text-center leading-relaxed">
              Drag and drop an Excel (.xlsx) or CSV file with multi-tier merged headers, mixed date conventions, or overwritten formula calculations.
            </p>
            <button
              onClick={() => fileInputRef.current?.click()}
              className="mt-5 px-5 py-2.5 bg-sky-600 hover:bg-sky-500 text-white font-semibold text-xs rounded-lg shadow-lg shadow-sky-600/20 transition-all flex items-center gap-2"
            >
              <Upload className="w-4 h-4" />
              Select Spreadsheet
            </button>
          </div>
        ) : (
          <>
            {activeTab === 'grid' && (
              <GridCanvas
                sheetName={currentSheet?.name || 'Sheet'}
                sheetIdx={activeSheetIdx}
                normalized={currentNormalized}
                rawPreview={currentSheet?.preview}
              />
            )}
            {activeTab === 'schema' && (
              <SchemaMapper normalized={currentNormalized} />
            )}
            {activeTab === 'dag' && <FormulaDagView />}
            {activeTab === 'variance' && (
              <VarianceHeatmap
                sheets={sheets}
                onReconcile={handleRunReconcile}
                report={reconciliationReport}
                loading={loading}
              />
            )}
            {activeTab === 'audit' && (
              <AuditScorecard sheetIdx={activeSheetIdx} />
            )}
          </>
        )}
      </main>

      {/* Real-time SSE Execution Drawer */}
      <PipelineStream
        events={streamEvents}
        isOpen={streamOpen}
        onClose={() => setStreamOpen(false)}
        connected={sseConnected}
      />
    </div>
  );
};
