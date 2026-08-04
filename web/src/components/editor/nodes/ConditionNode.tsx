import { Handle, Position, NodeProps, Node } from '@xyflow/react';
import type { CustomNodeData } from './EventStartNode';

export type CustomNode = Node<CustomNodeData>;

export function ConditionNode({ data, selected }: NodeProps<CustomNode>) {
  const displayName = data?.name || data?.label || "plan == 'Pro'";
  const hasBadge = Boolean((data?.errorCount ?? 0) > 0 || (data?.warningCount ?? 0) > 0);

  return (
    <div
      role="group"
      aria-label={`Node: ${displayName}, Category: Decision, Type: Condition`}
      data-node-type="Condition"
      className="relative flex flex-col items-center"
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

      {/* Diamond Shape Filter Container */}
      <div
        className={`w-28 h-28 glass-panel shape-diamond flex items-center justify-center border border-amber-400/50 rotate-45 transition-all ${
          selected ? 'shadow-[0_0_25px_rgba(245,158,11,0.6)] scale-105 border-amber-400' : 'shadow-[0_0_20px_-5px_rgba(245,158,11,0.3)]'
        }`}
      >
        <div className="-rotate-45 flex flex-col items-center">
          <span className="material-symbols-outlined text-amber-400 text-2xl mb-1">filter_alt</span>
          <span className="text-[9px] font-bold text-amber-400 uppercase tracking-wider font-['Outfit']">Filter</span>
        </div>
      </div>

      {/* Condition Expression Card */}
      <div className="mt-3.5 glass-panel rounded-lg px-4 py-2.5 border border-amber-400/20 text-center min-w-[150px]">
        <div className="text-xs font-mono text-amber-100 font-medium truncate">{displayName}</div>
        <div className="text-[10px] text-slate-400 mt-0.5 italic">Exclusive users only</div>
      </div>

      {/* Handles for True / False branches */}
      <div className="flex justify-between items-center w-full px-3 mt-2.5 font-mono text-[9px] font-bold">
        <span className="text-emerald-400">True</span>
        <span className="text-rose-400">False</span>
      </div>
      <Handle
        type="source"
        position={Position.Bottom}
        id="true"
        style={{ left: '35%' }}
        className="!w-8 !h-8 !bg-emerald-400 !border-3 !border-[#0B0F19] !-bottom-4 hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        id="false"
        style={{ left: '65%' }}
        className="!w-8 !h-8 !bg-rose-400 !border-3 !border-[#0B0F19] !-bottom-4 hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />
    </div>
  );
}
