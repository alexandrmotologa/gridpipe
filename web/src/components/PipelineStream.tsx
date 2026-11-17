import React from 'react';
import { Activity, CheckCircle, Clock, X } from 'lucide-react';
import { PipelineStreamEvent } from '../types';

interface PipelineStreamProps {
  events: PipelineStreamEvent[];
  isOpen: boolean;
  onClose: () => void;
  connected: boolean;
}

export const PipelineStream: React.FC<PipelineStreamProps> = ({
  events,
  isOpen,
  onClose,
  connected,
}) => {
  if (!isOpen) return null;

  return (
    <div className="fixed bottom-4 right-4 w-96 max-h-[450px] bg-slate-900 border border-slate-700/80 rounded-lg shadow-2xl z-50 flex flex-col overflow-hidden backdrop-blur-md">
      <div className="bg-slate-950 px-3 py-2 border-b border-slate-800 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Activity className="w-4 h-4 text-sky-400 animate-pulse" />
          <span className="text-xs font-semibold text-slate-200">Execution Pipeline Stream</span>
          <span
            className={`w-2 h-2 rounded-full ${
              connected ? 'bg-emerald-400' : 'bg-rose-400'
            }`}
            title={connected ? 'SSE Connected' : 'SSE Disconnected'}
          />
        </div>
        <button
          onClick={onClose}
          className="text-slate-500 hover:text-slate-300 p-0.5 rounded transition-colors"
        >
          <X className="w-3.5 h-3.5" />
        </button>
      </div>

      <div className="flex-1 overflow-auto p-3 space-y-2.5 font-mono text-xs">
        {events.length === 0 ? (
          <div className="text-slate-500 text-center py-8">
            <Clock className="w-6 h-6 mx-auto mb-2 text-slate-600" />
            Waiting for pipeline events...
          </div>
        ) : (
          events.map((ev, i) => (
            <div key={i} className="bg-slate-950 p-2 rounded border border-slate-800/80">
              <div className="flex items-center justify-between mb-1">
                <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-sky-500/10 text-sky-400 border border-sky-500/20">
                  {ev.stage}
                </span>
                <span className="text-[10px] text-slate-500">{ev.progress_pct}%</span>
              </div>
              <p className="text-slate-300 text-[11px] mt-0.5">{ev.message}</p>
            </div>
          ))
        )}
      </div>
    </div>
  );
};
