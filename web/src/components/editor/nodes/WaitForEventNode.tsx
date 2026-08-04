import { Handle, Position, NodeProps, Node } from '@xyflow/react';
import type { CustomNodeData } from './EventStartNode';

export type CustomNode = Node<CustomNodeData>;

export function WaitForEventNode({ data, selected }: NodeProps<CustomNode>) {
  const displayName = data?.name || data?.label || 'Wait For Event';
  const hasBadge = Boolean((data?.errorCount ?? 0) > 0 || (data?.warningCount ?? 0) > 0);

  return (
    <div
      role="group"
      aria-label={`Node: ${displayName}, Category: Decision, Type: WaitForEvent`}
      data-node-type="WaitForEvent"
      className={`relative px-5 py-4 rounded-2xl bg-[#0F131D]/90 backdrop-blur-xl border-2 transition-all min-w-[200px] ${
        selected
          ? 'border-blue-400 node-glow-cyan scale-105'
          : 'border-blue-500/40 hover:border-blue-400/80 shadow-lg shadow-black/40'
      }`}
    >
      {hasBadge && (
        <div
          data-testid="node-error-badge"
          role="status"
          aria-label={`${data?.errorCount || 0} error(s), ${data?.warningCount || 0} warning(s)`}
          title={`${data?.errorCount || 0} error(s), ${data?.warningCount || 0} warning(s)`}
          className={`absolute -top-2 -right-2 w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold text-white border-2 border-[#0B0F19] shadow-lg ${
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
        className="!w-8 !h-8 !bg-blue-400 !border-3 !border-[#0B0F19] !-top-4 hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />

      {/* Node Header */}
      <div className="flex items-center gap-2.5 mb-2">
        <div className="w-8 h-8 rounded-lg bg-blue-500/10 border border-blue-500/30 flex items-center justify-center text-blue-400 shrink-0 shadow-inner overflow-hidden">
          <span className="material-symbols-outlined text-lg">hourglass_top</span>
        </div>
        <div className="min-w-0 ml-3">
          <span className="text-[10px] font-mono font-semibold uppercase tracking-wider text-blue-400 block">WAIT / SIGNAL</span>
          <h3 className="font-['Outfit'] font-semibold text-xs text-white truncate leading-tight">{displayName}</h3>
        </div>
      </div>

      {/* Timeout Badge */}
      <div className="mt-1 flex items-center justify-between text-[10px] font-mono bg-blue-500/5 px-3 py-1.5 rounded border border-blue-500/10 text-blue-300">
        <span>event: order_delivered</span>
        <span className="text-blue-400 font-sans">10s max</span>
      </div>

      {/* Branches */}
      <div className="flex justify-between items-center mt-3.5 pt-2.5 border-t border-blue-500/20 text-[10px] font-mono font-semibold px-1">
        <span className="text-cyan-400">Event</span>
        <span className="text-amber-400">Timeout</span>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        id="event"
        style={{ left: '30%' }}
        className="!w-8 !h-8 !bg-cyan-400 !border-3 !border-[#0B0F19] !-bottom-4 hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        id="timeout"
        style={{ left: '70%' }}
        className="!w-8 !h-8 !bg-amber-400 !border-3 !border-[#0B0F19] !-bottom-4 hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />
    </div>
  );
}
