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
      style={{
        width: '346px',
        minWidth: '346px',
        maxWidth: '346px',
        height: '112px',
        minHeight: '112px',
        maxHeight: '112px',
        boxSizing: 'border-box',
        ...(selected ? { borderColor: '#fb7185', backgroundColor: 'rgba(251, 113, 133, 0.15)' } : {})
      }}
      className={`relative w-[346px] glass-panel rounded-none border-2 px-4 py-3 flex items-center gap-5 shadow-xl transition-all text-left ${
        selected
          ? 'border-[#fb7185] ring-2 ring-[#fb7185]/50 bg-[#fb7185]/10 scale-[1.02]'
          : 'border-[#fb7185]/60 hover:border-[#fb7185]'
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
        className="!w-8 !h-8 !bg-rose-400 !border-3 !border-[#0B0F19] hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />

      {/* Primary Exit Icon Badge */}
      <div className="w-10 h-10 rounded-none bg-rose-500/20 border border-rose-500/40 flex items-center justify-center text-rose-400 shadow-lg shrink-0 overflow-hidden">
        <span className="material-symbols-outlined text-lg">flag</span>
      </div>

      {/* Main Text Content */}
      <div className="flex-grow min-w-0 text-left">
        <div className="text-[10px] font-bold text-rose-400 uppercase tracking-widest font-['Outfit'] text-left">EXIT</div>
        <div className="text-xs font-bold text-[#dfe2f1] leading-tight truncate font-['Outfit'] text-left">{displayName}</div>
        <div className="text-[10px] text-rose-400 font-bold uppercase tracking-tighter font-mono mt-1.5 text-left">Utility / Terminal</div>
      </div>

      <div className="pr-1 text-rose-400 opacity-80 shrink-0">
        <span className="material-symbols-outlined text-lg">drag_indicator</span>
      </div>
    </div>
  );
}
