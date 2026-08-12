import { Handle, Position, NodeProps, Node } from '@xyflow/react';
import type { CustomNodeData } from './EventStartNode';

export type CustomNode = Node<CustomNodeData>;

export function ExperimentNode({ data, selected }: NodeProps<CustomNode>) {
  const displayName = data?.name || data?.label || 'Welcome Email Split';
  const hasBadge = Boolean((data?.errorCount ?? 0) > 0 || (data?.warningCount ?? 0) > 0);

  // Support both explicit variant_a_weight/percent (from node selector) and nested variants array
  const rawA = data?.config?.variant_a_weight ?? data?.config?.variant_a_percent;
  let variantAPct = 50;

  if (typeof rawA === 'number') {
    variantAPct = rawA;
  } else if (typeof rawA === 'string' && !isNaN(parseInt(rawA, 10))) {
    variantAPct = parseInt(rawA, 10);
  } else {
    const variants = data?.config?.variants as Array<{ weight?: number; percent?: number }> | undefined;
    if (Array.isArray(variants) && variants.length >= 2) {
      const wA = variants[0]?.weight ?? variants[0]?.percent ?? 5000;
      const wB = variants[1]?.weight ?? variants[1]?.percent ?? 5000;
      const total = wA + wB || 10000;
      variantAPct = Math.round((wA / total) * 100);
    }
  }

  const rawB = data?.config?.variant_b_weight ?? data?.config?.variant_b_percent;
  const parsedB = typeof rawB === 'number' ? rawB : (typeof rawB === 'string' && !isNaN(parseInt(rawB, 10)) ? parseInt(rawB, 10) : NaN);
  const variantBPct = !isNaN(parsedB) ? parsedB : (100 - variantAPct);

  return (
    <div
      role="group"
      aria-label={`Node: ${displayName}, Category: Decision, Type: Experiment`}
      data-node-type="Experiment"
      style={{
        width: '346px',
        minWidth: '346px',
        maxWidth: '346px',
        height: '112px',
        minHeight: '112px',
        maxHeight: '112px',
        boxSizing: 'border-box',
        ...(selected ? { borderColor: '#f59e0b', backgroundColor: 'rgba(245, 158, 11, 0.15)' } : {})
      }}
      className={`relative w-[346px] glass-panel rounded-none border-2 px-4 py-3 flex flex-col justify-between shadow-xl transition-all text-left ${
        selected
          ? 'border-[#f59e0b] ring-2 ring-[#f59e0b]/50 bg-[#f59e0b]/10 scale-[1.02]'
          : 'border-[#f59e0b]/60 hover:border-[#f59e0b]'
      }`}
    >
      {hasBadge && (
        <div
          data-testid="node-error-badge"
          role="status"
          aria-label={`${data?.errorCount || 0} error(s), ${data?.warningCount || 0} warning(s)`}
          title={`${data?.errorCount || 0} error(s), ${data?.warningCount || 0} warning(s)`}
          className={`absolute -top-2 -right-2 z-10 w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold text-white border-2 border-[#0B0F19] shadow-lg ${
            (data?.errorCount ?? 0) > 0 ? 'bg-rose-500 animate-bounce' : 'bg-amber-500'
          }`}
        >
          {(data?.errorCount ?? 0) + (data?.warningCount ?? 0)}
        </div>
      )}

      <Handle
        type="target"
        position={Position.Top}
        id="target"
        className="!w-8 !h-8 !bg-amber-400 !border-3 !border-[#0B0F19] hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />

      <div className="flex items-center gap-5 w-full text-left">
        {/* Primary Experiment Icon Badge */}
        <div className="w-10 h-10 rounded-none bg-amber-500/20 border border-amber-500/40 flex items-center justify-center text-amber-400 shadow-lg shrink-0 overflow-hidden">
          <span className="material-symbols-outlined text-lg">science</span>
        </div>

        {/* Main Text Content */}
        <div className="flex-grow min-w-0 text-left">
          <div className="text-[10px] font-bold text-amber-400 uppercase tracking-widest font-['Outfit'] text-left">DECISION</div>
          <div className="text-xs font-bold text-[#dfe2f1] leading-tight truncate font-['Outfit'] text-left">{displayName}</div>
          <div className="flex items-center gap-2 mt-1 text-[10px] font-mono text-amber-300 text-left">
            <span><span>Variant A</span>: <strong data-testid="experiment-node-pct-a">{variantAPct}%</strong></span>
            <span className="text-slate-500">|</span>
            <span><span>Variant B</span>: <strong data-testid="experiment-node-pct-b" className="text-slate-400">{variantBPct}%</strong></span>
          </div>
        </div>

        <div className="pr-1 text-amber-400 opacity-80 shrink-0">
          <span className="material-symbols-outlined text-lg">drag_indicator</span>
        </div>
      </div>

      {/* Percentage Bar reflecting user selected split */}
      <div className="w-full bg-[#11141d] border border-amber-500/40 h-3.5 rounded-none overflow-hidden flex mt-1 shadow-inner select-none">
        <div
          data-testid="experiment-node-bar-a"
          className="bg-amber-400 text-[#0B0F19] h-full flex items-center justify-center transition-all duration-300 text-[9px] font-mono font-bold shrink-0"
          style={{ width: `${variantAPct}%` }}
          title={`Variant A: ${variantAPct}%`}
        >
          {variantAPct >= 18 && <span>{variantAPct}%</span>}
        </div>
        <div
          data-testid="experiment-node-bar-b"
          className="bg-slate-700 text-[#dfe2f1] border-l border-amber-500/30 h-full flex items-center justify-center transition-all duration-300 text-[9px] font-mono font-bold flex-1 min-w-0"
          style={{ width: `${variantBPct}%` }}
          title={`Variant B: ${variantBPct}%`}
        >
          {variantBPct >= 18 && <span>{variantBPct}%</span>}
        </div>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        id="variant_a"
        style={{ left: '30%' }}
        className="!w-8 !h-8 !bg-amber-400 !border-3 !border-[#0B0F19] hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        id="variant_b"
        style={{ left: '70%' }}
        className="!w-8 !h-8 !bg-indigo-400 !border-3 !border-[#0B0F19] hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />
    </div>
  );
}
