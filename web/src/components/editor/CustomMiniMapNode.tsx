import React from 'react';
import { useReactFlow } from '@xyflow/react';

export const NODE_TYPE_COLOR_MAP: Record<string, { bg: string; stroke: string; text: string; icon: string; label: string }> = {
  EventStart: { bg: '#032b36', stroke: '#4cd7f6', text: '#4cd7f6', icon: 'electric_bolt', label: 'Trigger' },
  EventStartNode: { bg: '#032b36', stroke: '#4cd7f6', text: '#4cd7f6', icon: 'electric_bolt', label: 'Trigger' },
  trigger: { bg: '#032b36', stroke: '#4cd7f6', text: '#4cd7f6', icon: 'electric_bolt', label: 'Trigger' },

  Condition: { bg: '#2a1c02', stroke: '#f59e0b', text: '#f59e0b', icon: 'call_split', label: 'Condition' },
  ConditionNode: { bg: '#2a1c02', stroke: '#f59e0b', text: '#f59e0b', icon: 'call_split', label: 'Condition' },
  condition: { bg: '#2a1c02', stroke: '#f59e0b', text: '#f59e0b', icon: 'call_split', label: 'Condition' },

  Delay: { bg: '#2a1c02', stroke: '#f59e0b', text: '#f59e0b', icon: 'schedule', label: 'Delay' },
  DelayNode: { bg: '#2a1c02', stroke: '#f59e0b', text: '#f59e0b', icon: 'schedule', label: 'Delay' },
  delay: { bg: '#2a1c02', stroke: '#f59e0b', text: '#f59e0b', icon: 'schedule', label: 'Delay' },

  WaitForEvent: { bg: '#071a38', stroke: '#3b82f6', text: '#60a5fa', icon: 'hourglass_top', label: 'Wait Event' },
  WaitForEventNode: { bg: '#071a38', stroke: '#3b82f6', text: '#60a5fa', icon: 'hourglass_top', label: 'Wait Event' },
  wait_for_event: { bg: '#071a38', stroke: '#3b82f6', text: '#60a5fa', icon: 'hourglass_top', label: 'Wait Event' },

  Experiment: { bg: '#200738', stroke: '#ddb7ff', text: '#ddb7ff', icon: 'science', label: 'Experiment' },
  ExperimentNode: { bg: '#200738', stroke: '#ddb7ff', text: '#ddb7ff', icon: 'science', label: 'Experiment' },
  experiment: { bg: '#200738', stroke: '#ddb7ff', text: '#ddb7ff', icon: 'science', label: 'Experiment' },

  Email: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'mail', label: 'Email Action' },
  EmailNode: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'mail', label: 'Email Action' },
  action: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'mail', label: 'Action' },

  SMS: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'sms', label: 'SMS Action' },
  SMSNode: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'sms', label: 'SMS Action' },

  Push: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'notifications_active', label: 'Push Action' },
  PushNode: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'notifications_active', label: 'Push Action' },

  InApp: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'smartphone', label: 'In-App Action' },
  InAppNode: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'smartphone', label: 'In-App Action' },

  Webhook: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'webhook', label: 'Webhook' },
  WebhookNode: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'webhook', label: 'Webhook' },

  Exit: { bg: '#2d0909', stroke: '#f87171', text: '#f87171', icon: 'flag', label: 'Exit' },
  ExitNode: { bg: '#2d0909', stroke: '#f87171', text: '#f87171', icon: 'flag', label: 'Exit' },
  exit: { bg: '#2d0909', stroke: '#f87171', text: '#f87171', icon: 'flag', label: 'Exit' },
};

export interface CustomMiniMapNodeProps {
  x: number;
  y: number;
  width?: number;
  height?: number;
  color?: string;
  strokeColor?: string;
  strokeWidth?: number;
  borderRadius?: number;
  id?: string;
}

export function CustomMiniMapNode({
  x,
  y,
  width,
  height,
  color,
  borderRadius,
  id,
}: CustomMiniMapNodeProps) {
  const { getNode, getEdges } = useReactFlow();
  const node = id ? getNode(id) : null;
  if (!node) return null;

  const type = node.type || 'default';
  const colorConfig = NODE_TYPE_COLOR_MAP[type] || {
    bg: '#171b26',
    stroke: color || '#c0c1ff',
    text: color || '#c0c1ff',
    icon: 'extension',
    label: (node.data?.label as string) || (node.data?.name as string) || type,
  };

  const isVisited = Boolean(node.data?.isVisited);
  const isActive = Boolean(node.data?.isActive);
  const strokeColor = isActive ? '#4cd7f6' : isVisited ? '#34d399' : colorConfig.stroke;

  const w = (node.measured?.width as number) || (width && width > 0 ? width : 260);
  const h = (node.measured?.height as number) || (height && height > 0 ? height : 90);
  const displayName = (node.data?.name as string) || (node.data?.label as string) || type;

  const rx = 0;

  const allEdges = getEdges();
  const outgoingEdges = allEdges.filter((e) => e.source === id);

  return (
    <g className="react-flow__minimap-node cursor-pointer">
      {/* Edge Lines connecting to target nodes on the MiniMap */}
      {outgoingEdges.map((edge) => {
        const targetNode = getNode(edge.target);
        if (!targetNode) return null;

        const targetW = (targetNode.measured?.width as number) || 260;

        const x1 = x + w / 2;
        const y1 = y + h;
        const x2 = x + (targetNode.position.x - node.position.x) + targetW / 2;
        const y2 = y + (targetNode.position.y - node.position.y);

        const dy = Math.max(30, Math.abs(y2 - y1) * 0.5);
        const pathD = `M ${x1} ${y1} C ${x1} ${y1 + dy}, ${x2} ${y2 - dy}, ${x2} ${y2}`;
        const isTraversed = Boolean(edge.data?.isTraversed);

        return (
          <g key={`minimap-edge-${edge.id}`}>
            <path
              d={pathD}
              fill="none"
              stroke={isTraversed ? '#4cd7f6' : '#6366f1'}
              strokeWidth={4}
              strokeOpacity={isTraversed ? 0.8 : 0.4}
              style={{ pointerEvents: 'none' }}
            />
            <path
              d={pathD}
              fill="none"
              stroke={isTraversed ? '#4cd7f6' : '#c0c1ff'}
              strokeWidth={2}
              strokeDasharray="4 2"
              style={{ pointerEvents: 'none' }}
            />
          </g>
        );
      })}

      {/* Mini Node Card Background with Category Color Fill Tint & Border */}
      <rect
        x={x}
        y={y}
        width={w}
        height={h}
        rx={0}
        ry={0}
        fill={colorConfig.bg}
        stroke={strokeColor}
        strokeWidth={isActive || isVisited ? 4 : 3}
        fillOpacity={0.95}
      />

      {/* Mini Icon Circle */}
      <circle
        cx={x + 36}
        cy={y + h / 2}
        r={Math.min(h / 3, 20)}
        fill={strokeColor}
        fillOpacity={0.25}
        stroke={strokeColor}
        strokeWidth={1.5}
      />
      <text
        x={x + 36}
        y={y + h / 2 + 1}
        fill={strokeColor}
        fontSize={22}
        fontFamily="Material Symbols Outlined"
        textAnchor="middle"
        dominantBaseline="central"
        fontWeight="bold"
        style={{ pointerEvents: 'none', userSelect: 'none' }}
      >
        {colorConfig.icon}
      </text>

      {/* Mini Node Title Label */}
      <text
        x={x + 68}
        y={y + h / 2 - 6}
        fill="#dfe2f1"
        fontSize={16}
        fontFamily="Outfit, sans-serif"
        fontWeight="600"
        dominantBaseline="central"
        style={{ pointerEvents: 'none', userSelect: 'none' }}
      >
        {displayName.length > 18 ? displayName.slice(0, 16) + '…' : displayName}
      </text>

      {/* Mini Category Subtitle */}
      <text
        x={x + 68}
        y={y + h / 2 + 14}
        fill={strokeColor}
        fontSize={12}
        fontFamily="JetBrains Mono, monospace"
        fontWeight="700"
        dominantBaseline="central"
        style={{ pointerEvents: 'none', userSelect: 'none' }}
      >
        {colorConfig.label.toUpperCase()}
      </text>
    </g>
  );
}
