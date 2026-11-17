import React, { useState, useEffect } from 'react';
import { GitCommit, AlertTriangle, CheckCircle2, RefreshCw, ArrowRight } from 'lucide-react';
import { DAGReport } from '../types';

interface FormulaDagViewProps {
  onRefresh?: () => void;
}

export const FormulaDagView: React.FC<FormulaDagViewProps> = ({ onRefresh }) => {
  const [dag, setDag] = useState<DAGReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [selectedNode, setSelectedNode] = useState<string | null>(null);

  const fetchDAG = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/dag');
      if (res.ok) {
        const data = await res.json();
        setDag(data);
      }
    } catch (err) {
      console.error('Failed to fetch DAG:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDAG();
  }, []);

  if (!dag || Object.keys(dag.nodes).length === 0) {
    return (
      <div className="p-12 text-center text-slate-500 bg-slate-900 border border-slate-800 rounded-lg">
        <GitCommit className="w-8 h-8 text-slate-600 mx-auto mb-2" />
        <p className="font-medium text-slate-400">No Formula Nodes Detected</p>
        <p className="text-xs text-slate-500 mt-1">
          Upload a spreadsheet with calculated formulas or run Formula Synthesis to view calculation DAG.
        </p>
        <button
          onClick={fetchDAG}
          className="mt-4 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs rounded transition-colors"
        >
          Refresh Graph
        </button>
      </div>
    );
  }

  const activeNodeData = selectedNode ? dag.nodes[selectedNode] : null;

  return (
    <div className="flex flex-col h-full bg-slate-900 border border-slate-800 rounded-lg overflow-hidden shadow-xl p-4">
      {/* Header and KPI summary */}
      <div className="flex items-center justify-between pb-3 border-b border-slate-800 mb-4">
        <div>
          <h2 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
            <GitCommit className="w-4 h-4 text-sky-400" />
            Formula Dependency DAG & Evaluation Pipeline
          </h2>
          <p className="text-xs text-slate-400 mt-0.5">
            Directed acyclic graph checking circular references and computing execution order
          </p>
        </div>

        <div className="flex items-center gap-3">
          {dag.has_cycles ? (
            <div className="flex items-center gap-1.5 bg-rose-500/10 text-rose-400 border border-rose-500/20 px-3 py-1 rounded-full text-xs font-semibold">
              <AlertTriangle className="w-3.5 h-3.5" />
              Circular Reference Detected!
            </div>
          ) : (
            <div className="flex items-center gap-1.5 bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 px-3 py-1 rounded-full text-xs font-semibold">
              <CheckCircle2 className="w-3.5 h-3.5" />
              Acyclic (Valid)
            </div>
          )}

          <button
            onClick={fetchDAG}
            disabled={loading}
            className="p-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded border border-slate-700"
            title="Refresh DAG"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* DAG Stats */}
      <div className="grid grid-cols-3 gap-3 mb-4">
        <div className="bg-slate-950 p-3 rounded border border-slate-800 text-xs">
          <div className="text-slate-500 mb-1">Formula Cells</div>
          <div className="text-lg font-bold font-mono text-slate-200">{Object.keys(dag.nodes).length}</div>
        </div>
        <div className="bg-slate-950 p-3 rounded border border-slate-800 text-xs">
          <div className="text-slate-500 mb-1">Dependency Edges</div>
          <div className="text-lg font-bold font-mono text-slate-200">{dag.total_edges}</div>
        </div>
        <div className="bg-slate-950 p-3 rounded border border-slate-800 text-xs">
          <div className="text-slate-500 mb-1">Topological Depth</div>
          <div className="text-lg font-bold font-mono text-slate-200">{dag.eval_order.length}</div>
        </div>
      </div>

      {/* Main View: Node List & Details */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 flex-1 overflow-hidden">
        {/* Node Sequence List */}
        <div className="col-span-1 lg:col-span-7 bg-slate-950 rounded border border-slate-800 p-3 overflow-auto flex flex-col gap-2">
          <div className="text-xs font-semibold text-slate-400 mb-1 flex items-center justify-between">
            <span>Evaluation Sequence (Topological Order)</span>
            <span className="text-[11px] font-normal text-slate-500">Earliest to Latest</span>
          </div>
          <div className="space-y-1.5 font-mono text-xs">
            {dag.eval_order.map((nodeId, idx) => {
              const node = dag.nodes[nodeId];
              const isSelected = selectedNode === nodeId;
              return (
                <div
                  key={nodeId}
                  onClick={() => setSelectedNode(nodeId)}
                  className={`p-2 rounded border cursor-pointer transition-colors flex items-center justify-between ${
                    isSelected
                      ? 'bg-sky-500/20 border-sky-500 text-white'
                      : 'bg-slate-900/60 border-slate-800 text-slate-300 hover:bg-slate-800/50'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <span className="text-[10px] text-slate-500 w-4">{idx + 1}</span>
                    <span className="font-semibold text-sky-400">{node?.cell_ref || nodeId}</span>
                    <span className="text-slate-500 text-[11px]">={node?.formula}</span>
                  </div>
                  <div className="text-[11px] text-slate-500">
                    {node?.depends_on.length} dep(s)
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Node Inspector */}
        <div className="col-span-1 lg:col-span-5 bg-slate-950 rounded border border-slate-800 p-3 overflow-auto flex flex-col">
          <div className="text-xs font-semibold text-slate-400 mb-3 pb-2 border-b border-slate-800">
            Node Inspector
          </div>

          {activeNodeData ? (
            <div className="space-y-4 text-xs font-mono">
              <div>
                <div className="text-slate-500 text-[11px]">Cell Identifier</div>
                <div className="text-sky-300 font-bold text-sm mt-0.5">{activeNodeData.id}</div>
              </div>

              <div>
                <div className="text-slate-500 text-[11px]">Formula Expression</div>
                <div className="bg-slate-900 p-2 rounded border border-slate-800 text-emerald-400 mt-1">
                  ={activeNodeData.formula}
                </div>
              </div>

              <div>
                <div className="text-slate-500 text-[11px] mb-1">
                  Depends On ({activeNodeData.depends_on.length} cells)
                </div>
                <div className="space-y-1">
                  {activeNodeData.depends_on.map((dep, i) => (
                    <div key={i} className="bg-slate-900/80 px-2 py-1 rounded text-slate-300 border border-slate-800/80">
                      {dep}
                    </div>
                  ))}
                  {activeNodeData.depends_on.length === 0 && (
                    <div className="text-slate-600 italic">None (root parameter)</div>
                  )}
                </div>
              </div>

              <div>
                <div className="text-slate-500 text-[11px] mb-1">
                  Consumers / Used By ({activeNodeData.used_by.length} cells)
                </div>
                <div className="space-y-1">
                  {activeNodeData.used_by.map((usr, i) => (
                    <div key={i} className="bg-slate-900/80 px-2 py-1 rounded text-slate-300 border border-slate-800/80">
                      {usr}
                    </div>
                  ))}
                  {activeNodeData.used_by.length === 0 && (
                    <div className="text-slate-600 italic">None (leaf result)</div>
                  )}
                </div>
              </div>
            </div>
          ) : (
            <div className="text-center text-slate-600 my-auto text-xs">
              Select a node in the topological sequence to inspect formula inputs and outputs.
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
