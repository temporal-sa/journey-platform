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

      {/* Hexagon Trigger Box */}
      <div
        className={`w-32 h-28 glass-panel shape-hexagon flex flex-col items-center justify-center border-t-2 border-[#4cd7f6] p-4 transition-all ${
          selected ? 'shadow-[0_0_25px_rgba(76,215,246,0.6)] scale-105' : 'shadow-[0_0_20px_-5px_rgba(76,215,246,0.3)]'
        }`}
      >
        <span className="material-symbols-outlined text-[#4cd7f6] text-3xl mb-1" style={{ fontVariationSettings: "'FILL' 1" }}>
          electric_bolt
        </span>
        <span className="text-[10px] font-bold text-[#4cd7f6] tracking-widest uppercase font-['Outfit']">Trigger</span>
      </div>

      {/* Event Details Card */}
      <div className="mt-3 glass-panel rounded-lg px-3 py-2 border border-[#4cd7f6]/30 text-center min-w-[140px]">
        <div className="text-xs font-mono text-[#dfe2f1] font-medium truncate">{displayName}</div>
        <div className="flex items-center justify-center gap-1.5 mt-1">
          <span className="w-1.5 h-1.5 rounded-full bg-[#4cd7f6] animate-pulse" />
          <span className="text-[9px] text-[#4cd7f6] font-bold uppercase tracking-tighter">Schema Verified</span>
        </div>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        id="source"
        className="!w-8 !h-8 !bg-[#4cd7f6] !border-3 !border-[#0B0F19] hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl !-bottom-4"
      />
    </div>
  );
}
