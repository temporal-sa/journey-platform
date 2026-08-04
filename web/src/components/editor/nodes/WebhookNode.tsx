import { Handle, Position, NodeProps, Node } from '@xyflow/react';
import type { CustomNodeData } from './EventStartNode';

export type CustomNode = Node<CustomNodeData>;

export function WebhookNode({ data, selected }: NodeProps<CustomNode>) {
  const displayName = data?.name || data?.label || 'Webhook';
  const hasBadge = Boolean((data?.errorCount ?? 0) > 0 || (data?.warningCount ?? 0) > 0);

  return (
    <div
      role="group"
      aria-label={`Node: ${displayName}, Category: Action, Type: Webhook`}
      data-node-type="Webhook"
      style={{
        width: '346px',
        minWidth: '346px',
        maxWidth: '346px',
        height: '112px',
        minHeight: '112px',
        maxHeight: '112px',
        boxSizing: 'border-box',
        ...(selected ? { borderColor: '#c0c1ff', backgroundColor: 'rgba(192, 193, 255, 0.15)' } : {})
      }}
      className={`relative w-[346px] glass-panel rounded-none border-2 px-4 py-3 flex items-center gap-5 shadow-xl transition-all text-left ${
        selected
          ? 'border-[#c0c1ff] ring-2 ring-[#c0c1ff]/50 bg-[#c0c1ff]/10 scale-[1.02]'
          : 'border-[#c0c1ff]/60 hover:border-[#c0c1ff]'
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
        className="!w-8 !h-8 !bg-[#c0c1ff] !border-3 !border-[#0B0F19] hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />

      <div className="w-10 h-10 rounded-none bg-[#c0c1ff]/20 border border-[#c0c1ff]/40 flex items-center justify-center text-[#c0c1ff] shadow-lg shrink-0 overflow-hidden">
        <span className="material-symbols-outlined text-lg">webhook</span>
      </div>

      <div className="flex-grow min-w-0 text-left">
        <div className="text-[10px] font-bold text-[#c0c1ff] uppercase tracking-widest font-['Outfit'] text-left">ACTION</div>
        <div className="text-xs font-bold text-[#dfe2f1] leading-tight truncate font-['Outfit'] text-left">{displayName}</div>
        <div className="flex items-center gap-2 mt-1.5 text-left">
          <span className="text-[10px] font-mono bg-slate-800 px-1.5 py-0.5 rounded text-slate-300">Webhook</span>
          <span className="w-1 h-1 rounded-full bg-slate-500" />
          <span className="text-[10px] text-slate-400 uppercase tracking-tighter font-mono">Draft</span>
        </div>
      </div>

      <div className="pr-1 text-[#c0c1ff] opacity-80 shrink-0">
        <span className="material-symbols-outlined text-lg">drag_indicator</span>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        id="source"
        className="!w-8 !h-8 !bg-[#c0c1ff] !border-3 !border-[#0B0F19] hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />
    </div>
  );
}
