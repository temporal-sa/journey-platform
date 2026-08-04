import { Handle, Position, NodeProps, Node } from '@xyflow/react';
import type { CustomNodeData } from './EventStartNode';

export type CustomNode = Node<CustomNodeData>;

export function ExitNode({ data, selected }: NodeProps<CustomNode>) {
  const displayName = data?.name || data?.label || 'Journey Complete';
  const hasBadge = Boolean((data?.errorCount ?? 0) > 0 || (data?.warningCount ?? 0) > 0);

  return (
    <div
      role="group"
      aria-label={`Node: ${displayName}, Category: Utility, Type: Exit`}
      data-node-type="Exit"
      className="relative flex items-center gap-3.5 group"
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
        className="!w-8 !h-8 !bg-rose-400 !border-3 !border-[#0B0F19] !-top-4 hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />

      {/* Circle Icon Button */}
      <div
        className={`w-12 h-12 rounded-full glass-panel border flex items-center justify-center transition-colors shadow-lg p-2.5 shrink-0 overflow-hidden ${
          selected
            ? 'border-rose-400 bg-rose-500/20 scale-105 shadow-[0_0_20px_rgba(244,63,94,0.5)]'
            : 'border-rose-500/50 hover:bg-rose-500/10'
        }`}
      >
        <span className="material-symbols-outlined text-rose-400 text-xl">flag</span>
      </div>

      {/* Text Card */}
      <div className="glass-panel px-4 py-2 rounded-lg border border-rose-500/20 text-left min-w-[130px] ml-3.5">
        <div className="text-[10px] font-bold text-rose-400 uppercase font-['Outfit'] tracking-wider">Utility / Exit</div>
        <div className="text-xs font-semibold text-[#dfe2f1] font-['Outfit'] truncate">{displayName}</div>
      </div>
    </div>
  );
}
