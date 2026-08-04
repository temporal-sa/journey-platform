import { Handle, Position, NodeProps, Node } from '@xyflow/react';
import type { CustomNodeData } from './EventStartNode';

export type CustomNode = Node<CustomNodeData>;

export function ExperimentNode({ data, selected }: NodeProps<CustomNode>) {
  const displayName = data?.name || data?.label || 'Welcome Email Split';
  const hasBadge = Boolean((data?.errorCount ?? 0) > 0 || (data?.warningCount ?? 0) > 0);

  const rawA = data?.config?.variant_a_weight ?? data?.config?.variant_a_percent;
  const variantAPct = typeof rawA === 'number' ? rawA : (typeof rawA === 'string' && !isNaN(parseInt(rawA, 10)) ? parseInt(rawA, 10) : 50);
  const rawB = data?.config?.variant_b_weight ?? data?.config?.variant_b_percent;
  const parsedB = typeof rawB === 'number' ? rawB : (typeof rawB === 'string' && !isNaN(parseInt(rawB, 10)) ? parseInt(rawB, 10) : NaN);
  const variantBPct = !isNaN(parsedB) ? parsedB : (100 - variantAPct);

  return (
    <div
      role="group"
      aria-label={`Node: ${displayName}, Category: Decision, Type: Experiment`}
      data-node-type="Experiment"
      className={`relative w-60 glass-panel rounded-xl border transition-all ${
        selected
          ? 'border-[#ddb7ff] shadow-[0_0_25px_-5px_rgba(221,183,255,0.6)] scale-105'
          : 'border-[#ddb7ff]/60 shadow-[0_0_25px_-5px_rgba(221,183,255,0.2)]'
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
        className="!w-8 !h-8 !bg-[#ddb7ff] !border-3 !border-[#0B0F19] !-top-4 hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />

      {/* Header Bar */}
      <div className="px-4 py-3.5 bg-[#ddb7ff]/10 border-b border-[#ddb7ff]/20 flex items-center justify-between rounded-t-xl">
        <div className="flex items-center gap-2">
          <span className="material-symbols-outlined text-[#ddb7ff] text-lg">call_split</span>
          <span className="text-[10px] font-bold text-[#ddb7ff] uppercase font-['Outfit'] tracking-wider">A/B TEST</span>
        </div>
        <div className="px-2 py-0.5 bg-[#ddb7ff] text-[#400071] rounded text-[9px] font-bold">LIVE</div>
      </div>

      {/* Card Content */}
      <div className="p-4 space-y-3.5">
        <div className="text-xs font-semibold text-[#dfe2f1] font-['Outfit'] truncate">{displayName}</div>
        <div className="space-y-2">
          <div className="flex items-center justify-between text-[10px] font-mono">
            <span className="text-[#ddb7ff] font-semibold">Variant A</span>
            <span className="text-[#ddb7ff]" data-testid="experiment-node-pct-a">{variantAPct}%</span>
          </div>
          <div className="bg-slate-800 h-1.5 rounded-full overflow-hidden">
            <div className="bg-[#ddb7ff] h-full transition-all duration-300" style={{ width: `${variantAPct}%` }} />
          </div>

          <div className="flex items-center justify-between text-[10px] font-mono mt-1">
            <span className="text-slate-400 font-semibold">Variant B</span>
            <span className="text-slate-400" data-testid="experiment-node-pct-b">{variantBPct}%</span>
          </div>
          <div className="bg-slate-800 h-1.5 rounded-full overflow-hidden">
            <div className="bg-slate-500 h-full transition-all duration-300" style={{ width: `${variantBPct}%` }} />
          </div>
        </div>
      </div>

      {/* Footer Controls */}
      <div className="flex border-t border-white/10 text-slate-400 rounded-b-xl overflow-hidden">
        <div className="flex-1 p-2.5 flex justify-center hover:text-[#ddb7ff] cursor-pointer transition-colors">
          <span className="material-symbols-outlined text-sm">settings</span>
        </div>
        <div className="flex-1 p-2.5 flex justify-center hover:text-[#ddb7ff] border-l border-white/10 cursor-pointer transition-colors">
          <span className="material-symbols-outlined text-sm">analytics</span>
        </div>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        id="variant_a"
        style={{ left: '35%' }}
        className="!w-8 !h-8 !bg-[#ddb7ff] !border-3 !border-[#0B0F19] !-bottom-4 hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        id="variant_b"
        style={{ left: '65%' }}
        className="!w-8 !h-8 !bg-slate-400 !border-3 !border-[#0B0F19] !-bottom-4 hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />
    </div>
  );
}
