import { Handle, Position, NodeProps, Node } from '@xyflow/react';
import type { CustomNodeData } from './EventStartNode';

export type CustomNode = Node<CustomNodeData>;

function getExperimentPercentages(config: Record<string, unknown> | undefined): { aPct: number; bPct: number } {
  if (!config) return { aPct: 50, bPct: 50 };

  // 1. Check direct top-level fields: variant_a_weight, variant_a_percent, variantAPct
  const rawA = config.variant_a_weight ?? config.variant_a_percent ?? config.variantAPct;
  if (typeof rawA === 'number' && !isNaN(rawA)) {
    const a = Math.max(0, Math.min(100, Math.round(rawA)));
    return { aPct: a, bPct: 100 - a };
  }
  if (typeof rawA === 'string' && !isNaN(parseInt(rawA, 10))) {
    const a = Math.max(0, Math.min(100, parseInt(rawA, 10)));
    return { aPct: a, bPct: 100 - a };
  }

  // 2. Check variants array
  const variants = config.variants as Array<Record<string, unknown>> | undefined;
  if (Array.isArray(variants) && variants.length >= 2) {
    const rawWA = variants[0]?.weight ?? variants[0]?.percent ?? variants[0]?.allocation;
    const rawWB = variants[1]?.weight ?? variants[1]?.percent ?? variants[1]?.allocation;

    const wA = typeof rawWA === 'number' ? rawWA : (typeof rawWA === 'string' ? parseInt(rawWA, 10) : NaN);
    const wB = typeof rawWB === 'number' ? rawWB : (typeof rawWB === 'string' ? parseInt(rawWB, 10) : NaN);

    if (!isNaN(wA) && !isNaN(wB)) {
      const total = wA + wB || 10000;
      const a = Math.max(0, Math.min(100, Math.round((wA / total) * 100)));
      return { aPct: a, bPct: 100 - a };
    }
  }

  return { aPct: 50, bPct: 50 };
}

export function ExperimentNode({ data, selected }: NodeProps<CustomNode>) {
  const displayName = data?.name || data?.label || 'Welcome Email Split';
  const hasBadge = Boolean((data?.errorCount ?? 0) > 0 || (data?.warningCount ?? 0) > 0);

  const { aPct: variantAPct, bPct: variantBPct } = getExperimentPercentages(data?.config);

  return (
    <div
      role="group"
      aria-label={`Node: ${displayName}, Category: Decision, Type: Experiment`}
      data-node-type="Experiment"
      style={{
        width: '346px',
        minWidth: '346px',
        maxWidth: '346px',
        height: '138px',
        minHeight: '138px',
        maxHeight: '138px',
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

      <div className="flex items-center gap-4 w-full text-left">
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

      {/* Prominent Amber Fill Split Bar (No internal text, guaranteed rendering via explicit inline styles) */}
      <div
        style={{
          width: '100%',
          height: '14px',
          minHeight: '14px',
          maxHeight: '14px',
          backgroundColor: '#0f172a',
          border: '1.5px solid #f59e0b',
          borderRadius: '0px',
          overflow: 'hidden',
          display: 'flex',
          marginTop: '6px',
          marginBottom: '4px',
          boxSizing: 'border-box',
        }}
        className="shrink-0 select-none shadow-md"
      >
        <div
          data-testid="experiment-node-bar-a"
          style={{
            width: `${variantAPct}%`,
            height: '100%',
            backgroundColor: '#f59e0b',
            transition: 'all 300ms ease-in-out',
            flexShrink: 0,
          }}
          title={`Variant A: ${variantAPct}%`}
        />
        <div
          data-testid="experiment-node-bar-b"
          style={{
            width: `${variantBPct}%`,
            height: '100%',
            backgroundColor: '#1e293b',
            borderLeft: '1.5px solid #f59e0b',
            transition: 'all 300ms ease-in-out',
            flex: 1,
            minWidth: 0,
          }}
          title={`Variant B: ${variantBPct}%`}
        />
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
