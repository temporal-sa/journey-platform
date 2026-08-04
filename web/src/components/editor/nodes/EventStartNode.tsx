import { Handle, Position, NodeProps, Node } from '@xyflow/react';

export type CustomNodeData = {
  label?: string;
  name?: string;
  config?: Record<string, unknown>;
  errorCount?: number;
  warningCount?: number;
  validationIssues?: unknown[];
};

export type CustomNode = Node<CustomNodeData>;

export function EventStartNode({ data, selected }: NodeProps<CustomNode>) {
  const displayName = data?.name || data?.label || 'user_signed_up';
  const hasBadge = Boolean((data?.errorCount ?? 0) > 0 || (data?.warningCount ?? 0) > 0);

  return (
    <div
      role="group"
      aria-label={`Node: ${displayName}, Category: Trigger, Type: EventStart`}
      data-node-type="EventStart"
      style={{
        width: '346px',
        minWidth: '346px',
        maxWidth: '346px',
        height: '112px',
        minHeight: '112px',
        maxHeight: '112px',
        boxSizing: 'border-box',
        ...(selected ? { borderColor: '#4cd7f6', backgroundColor: 'rgba(76, 215, 246, 0.15)' } : {})
      }}
      className={`relative w-[346px] glass-panel rounded-none border-2 px-4 py-3 flex items-center gap-5 shadow-xl transition-all text-left ${
        selected
          ? 'border-[#4cd7f6] ring-2 ring-[#4cd7f6]/50 bg-[#4cd7f6]/10 scale-[1.02]'
          : 'border-[#4cd7f6]/60 hover:border-[#4cd7f6]'
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

      {/* Primary Trigger Icon Badge */}
      <div className="w-10 h-10 rounded-none bg-[#4cd7f6]/20 border border-[#4cd7f6]/40 flex items-center justify-center text-[#4cd7f6] shadow-lg shrink-0 overflow-hidden">
        <span className="material-symbols-outlined text-lg">
          bolt
        </span>
      </div>

      {/* Main Text Content */}
      <div className="flex-grow min-w-0 text-left">
        <div className="text-[10px] font-bold text-[#4cd7f6] uppercase tracking-widest font-['Outfit'] text-left">TRIGGER</div>
        <div className="text-xs font-bold text-[#dfe2f1] leading-tight truncate font-['Outfit'] text-left">{displayName}</div>
        <div className="text-[10px] text-[#4cd7f6] font-bold uppercase tracking-tighter font-mono mt-1.5 text-left">Schema Verified</div>
      </div>

      <div className="pr-1 text-[#4cd7f6] opacity-80 shrink-0">
        <span className="material-symbols-outlined text-lg">drag_indicator</span>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        id="source"
        className="!w-8 !h-8 !bg-[#4cd7f6] !border-3 !border-[#0B0F19] hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />
    </div>
  );
}
