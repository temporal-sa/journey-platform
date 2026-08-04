import { Handle, Position, NodeProps, Node } from '@xyflow/react';
import type { CustomNodeData } from './EventStartNode';

export type CustomNode = Node<CustomNodeData>;

export function DelayNode({ data, selected }: NodeProps<CustomNode>) {
  const displayName = data?.name || data?.label || 'Delay';
  const duration = (data?.config?.duration as string) || (data?.config?.duration ? `${data.config.duration}` : '1h');
  const hasBadge = Boolean((data?.errorCount ?? 0) > 0 || (data?.warningCount ?? 0) > 0);

  return (
    <div
      role="group"
      aria-label={`Node: ${displayName}, Category: Decision, Type: Delay`}
      data-node-type="Delay"
      className={`relative px-5 py-4 glass-panel rounded-[2px_16px_2px_16px] border-2 transition-all min-w-[190px] ${
        selected
          ? 'border-amber-400 shadow-[0_0_25px_rgba(245,158,11,0.6)] scale-105'
          : 'border-amber-500/40 hover:border-amber-400/80 shadow-lg shadow-black/40'
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
        className="!w-8 !h-8 !bg-amber-400 !border-3 !border-[#0B0F19] !-top-4 hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />

      {/* Header with Icon and Title */}
      <div className="flex items-center gap-2.5 mb-2">
        <div className="w-8 h-8 rounded-lg bg-amber-500/10 border border-amber-500/30 flex items-center justify-center text-amber-400 shrink-0 shadow-inner overflow-hidden">
          <span className="material-symbols-outlined text-lg">schedule</span>
        </div>
        <div className="min-w-0 ml-3">
          <div className="flex items-center gap-1.5">
            <span
              data-testid="node-type-badge"
              className="text-[9px] font-mono font-bold px-1 py-0.5 rounded bg-amber-500/20 text-amber-300 border border-amber-500/30 uppercase"
            >
              Decision
            </span>
          </div>
          <h3 className="font-['Outfit'] font-bold text-xs text-[#dfe2f1] truncate leading-tight mt-0.5">
            {displayName}
          </h3>
        </div>
      </div>

      {/* Duration Badge */}
      <div className="flex items-center justify-between text-[10px] font-mono bg-amber-500/5 px-3 py-1.5 rounded border border-amber-500/15 text-amber-200 mt-1">
        <span className="text-amber-400/70 uppercase text-[9px] font-bold tracking-wider">Duration</span>
        <span className="font-bold text-amber-300">Wait: {duration}</span>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        id="source"
        className="!w-8 !h-8 !bg-amber-400 !border-3 !border-[#0B0F19] !-bottom-4 hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />
    </div>
  );
}
