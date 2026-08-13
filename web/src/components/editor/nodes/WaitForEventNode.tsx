import { Handle, Position, NodeProps, Node } from '@xyflow/react';
import type { CustomNodeData } from './EventStartNode';

export type CustomNode = Node<CustomNodeData>;

export function WaitForEventNode({ data, selected }: NodeProps<CustomNode>) {
  const displayName = data?.name || data?.label || 'Wait For Event';
  const eventType = (data?.config?.event_type as string) || 'order.completed';
  const rawDuration = (data?.config?.duration ?? data?.config?.timeout ?? data?.config?.timeout_seconds) as number | string | undefined;
  const rawUnit = (data?.config?.unit as string) || (data?.config?.timeout_seconds ? 'seconds' : 'hours');

  let timeoutText = '24h max';
  if (rawDuration !== undefined && rawDuration !== null && rawDuration !== '') {
    const durStr = String(rawDuration).trim();
    if (/[a-zA-Z]/.test(durStr)) {
      timeoutText = `${durStr} max`;
    } else if (rawUnit) {
      const u = rawUnit.charAt(0).toLowerCase();
      timeoutText = `${durStr}${u} max`;
    } else {
      timeoutText = `${durStr}s max`;
    }
  }
  const hasBadge = Boolean((data?.errorCount ?? 0) > 0 || (data?.warningCount ?? 0) > 0);
  return (
    <div
      role="group"
      aria-label={`Node: ${displayName}, Category: Decision, Type: WaitForEvent`}
      data-node-type="WaitForEvent"
      style={{
        width: '346px',
        minWidth: '346px',
        maxWidth: '346px',
        height: '112px',
        minHeight: '112px',
        maxHeight: '112px',
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

      <div className="flex items-center gap-5 w-full text-left">
        {/* Primary Wait Icon Badge */}
        <div className="w-10 h-10 rounded-none bg-amber-500/20 border border-amber-500/40 flex items-center justify-center text-amber-400 shadow-lg shrink-0 overflow-hidden">
          <span className="material-symbols-outlined text-lg">hourglass_empty</span>
        </div>

        {/* Main Text Content */}
        <div className="flex-grow min-w-0 text-left">
          <div className="text-[10px] font-bold text-amber-400 uppercase tracking-widest font-['Outfit'] text-left">DECISION</div>
          <div className="text-xs font-bold text-[#dfe2f1] leading-tight truncate font-['Outfit'] text-left">{displayName}</div>
          <div className="flex items-center gap-2 mt-1.5 text-[10px] font-mono text-amber-300 text-left">
            <span className="truncate max-w-[120px]">{eventType}</span>
            <span className="text-slate-400 shrink-0">({timeoutText})</span>
          </div>
        </div>

        <div className="pr-1 text-amber-400 opacity-80 shrink-0">
          <span className="material-symbols-outlined text-lg">drag_indicator</span>
        </div>
      </div>

      {/* Branches */}
      <div className="flex justify-between items-center w-full px-2 mt-1 font-mono text-[9px] font-bold">
        <span className="text-cyan-400">Event</span>
        <span className="text-amber-400">Timeout</span>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        id="event"
        style={{ left: '30%' }}
        className="!w-8 !h-8 !bg-cyan-400 !border-3 !border-[#0B0F19] hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        id="timeout"
        style={{ left: '70%' }}
        className="!w-8 !h-8 !bg-amber-400 !border-3 !border-[#0B0F19] hover:!scale-125 transition-all cursor-crosshair z-20 shadow-xl"
      />
    </div>
  );
}
