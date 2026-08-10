import React, { useMemo } from 'react';
import {
  ReactFlow,
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  Node,
  Edge,
  NodeProps,
  EdgeProps,
  getBezierPath,
  EdgeLabelRenderer,
  BaseEdge,
  ReactFlowProvider,
  useReactFlow,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import { GraphNode, GraphEdge } from '../../types/api';
import { nodeTypes as baseNodeTypes } from '../editor/nodes';

export interface NodeVisitStep {
  stepIndex: number;
  nodeId: string;
  nodeName?: string;
  nodeType?: string;
  status: 'passed' | 'failed' | 'running' | 'completed' | 'suppressed';
  timestamp?: string;
  output?: Record<string, unknown>;
}

export interface ExecutionGraphViewProps {
  nodes: GraphNode[];
  edges: GraphEdge[];
  visitSteps: NodeVisitStep[];
  currentNodeId?: string;
  status?: 'passed' | 'failed' | 'running';
  height?: string;
}
const NODE_TYPE_COLOR_MAP: Record<string, { bg: string; stroke: string; text: string; icon: string; label: string }> = {
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

  Email: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'mail', label: 'Email Action' },
  EmailNode: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'mail', label: 'Email Action' },
  action: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'mail', label: 'Action' },

  SMS: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'sms', label: 'SMS Action' },
  SMSNode: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'sms', label: 'SMS Action' },

  Push: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'notifications_active', label: 'Push Action' },
  PushNode: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'notifications_active', label: 'Push Action' },

  InApp: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'terminal', label: 'In-App' },
  InAppNode: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'terminal', label: 'In-App' },

  Webhook: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'webhook', label: 'Webhook' },
  WebhookNode: { bg: '#121330', stroke: '#c0c1ff', text: '#c0c1ff', icon: 'webhook', label: 'Webhook' },

  Exit: { bg: '#2d0909', stroke: '#f87171', text: '#f87171', icon: 'flag', label: 'Exit' },
  ExitNode: { bg: '#2d0909', stroke: '#f87171', text: '#f87171', icon: 'flag', label: 'Exit' },
};

function CustomMiniMapNode({
  x,
  y,
  width,
  height,
  color,
  id,
}: {
  x: number;
  y: number;
  width?: number;
  height?: number;
  color?: string;
  id?: string;
}) {
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

  const isAction = ['Email', 'SMS', 'Push', 'InApp', 'Webhook', 'action'].includes(type);
  const rx = isAction ? h / 2 : 12;

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

      {/* Mini Node Card Background */}
      <rect
        x={x}
        y={y}
        width={w}
        height={h}
        rx={rx}
        ry={rx}
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

function ExecutionNodeWrapper(props: NodeProps) {
  const { id, type, data } = props;
  const isVisited = Boolean(data?.isVisited);
  const isActive = Boolean(data?.isActive);
  const isUnvisited = !isVisited && !isActive;
  const stepIndex = data?.stepIndex;

  const BaseComponent = baseNodeTypes[type] || baseNodeTypes.EventStartNode || baseNodeTypes.EventStart;

  return (
    <div
      data-testid={`execution-node-${id}`}
      data-node-id={id}
      data-node-type={type}
      data-visited={isVisited}
      data-active={isActive}
      className={`relative transition-all duration-300 ${
        isUnvisited ? 'opacity-45' : 'opacity-100'
      }`}
      style={{
        opacity: isUnvisited ? 0.45 : 1,
      }}
    >
      {/* Active Badge */}
      {isActive && (
        <div
          data-testid={`node-badge-active-${id}`}
          className="absolute -top-3 -left-2 z-30 px-2 py-0.5 rounded text-[10px] font-bold bg-[#4cd7f6] text-[#0B0F19] shadow-[0_0_10px_rgba(76,215,246,0.6)] font-mono uppercase tracking-wider animate-pulse flex items-center gap-1"
        >
          <span className="w-1.5 h-1.5 rounded-full bg-[#0B0F19] animate-ping" />
          Active
        </div>
      )}

      {/* Visited Step Badge */}
      {!isActive && isVisited && stepIndex !== undefined && (
        <div
          data-testid={`node-badge-step-${id}`}
          className="absolute -top-3 -left-2 z-30 px-2 py-0.5 rounded text-[10px] font-bold bg-[#34d399] text-[#0B0F19] shadow-[0_0_10px_rgba(52,211,153,0.5)] font-mono uppercase tracking-wider flex items-center gap-1"
        >
          Step {stepIndex}
        </div>
      )}

      {/* Node border & glow overlay container */}
      <div
        className={`rounded-none transition-all duration-300 ${
          isActive
            ? 'border-2 border-[#4cd7f6] ring-2 ring-[#4cd7f6]/60 animate-pulse shadow-[0_0_20px_rgba(76,215,246,0.6)]'
            : isVisited
            ? 'border-2 border-[#34d399] shadow-[0_0_15px_rgba(52,211,153,0.4)]'
            : ''
        }`}
        style={{
          ...(isActive
            ? { borderColor: '#4cd7f6', boxShadow: '0 0 20px rgba(76, 215, 246, 0.6)' }
            : isVisited
            ? { borderColor: '#34d399', boxShadow: '0 0 15px rgba(52, 211, 153, 0.4)' }
            : {}),
        }}
      >
        <BaseComponent {...props} />
      </div>
    </div>
  );
}

const executionNodeTypes: Record<string, React.ComponentType<NodeProps>> = Object.keys(
  baseNodeTypes
).reduce((acc, key) => {
  acc[key] = (props: NodeProps) => <ExecutionNodeWrapper {...props} />;
  return acc;
}, {} as Record<string, React.ComponentType<NodeProps>>);

// Fallback for custom or missing node types
executionNodeTypes.default = (props: NodeProps) => <ExecutionNodeWrapper {...props} />;

export function ExecutionEdge(props: EdgeProps) {
  const { id, sourceX, sourceY, targetX, targetY, sourcePosition, targetPosition, style, label, data } = props;
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const displayLabel = (label as string) || (data?.label as string) || (data?.condition as string) || '';
  const isTraversed = Boolean(data?.isTraversed);

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        style={{
          stroke: isTraversed ? '#4cd7f6' : '#464554',
          strokeWidth: isTraversed ? 2.5 : 1.5,
          strokeDasharray: isTraversed ? undefined : '6 4',
          filter: isTraversed ? 'drop-shadow(0 0 8px rgba(76, 215, 246, 0.6))' : undefined,
          ...style,
        }}
      />
      {displayLabel && (
        <EdgeLabelRenderer>
          <div
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
              pointerEvents: 'all',
            }}
            className={`nodrag nopan px-2 py-0.5 rounded-none text-[10px] font-mono font-semibold transition-all backdrop-blur-md border ${
              isTraversed
                ? 'bg-[#032b36]/90 text-[#4cd7f6] border-[#4cd7f6] shadow-[0_0_10px_rgba(76,215,246,0.4)]'
                : 'bg-[#171b26]/90 text-[#8e8c9e] border-[#464554]'
            }`}
            data-testid={`edge-label-${id}`}
          >
            {displayLabel}
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  );
}

const edgeTypes = {
  labeled: ExecutionEdge,
  default: ExecutionEdge,
};

function ExecutionGraphViewInner({
  nodes,
  edges,
  visitSteps,
  currentNodeId,
  status,
  height = '600px',
}: ExecutionGraphViewProps) {
  const isCompleted = useMemo(() => {
    const s = (status || '').toLowerCase();
    return s === 'completed' || s === 'passed' || s === 'success' || s === 'succeeded';
  }, [status]);

  // Determine active node ID (only active while run is running)
  const activeNodeId = useMemo(() => {
    if (isCompleted) return undefined;
    if (currentNodeId) return currentNodeId;
    if (status === 'running' && visitSteps.length > 0) {
      return visitSteps[visitSteps.length - 1].nodeId;
    }
    return undefined;
  }, [currentNodeId, status, visitSteps, isCompleted]);

  // Helper to find visit step matching a graph node
  const findMatchingStep = useMemo(() => {
    return (node: GraphNode, index: number): NodeVisitStep | undefined => {
      if (!visitSteps || visitSteps.length === 0) return undefined;

      const sanitize = (str?: string) => (str || '').toLowerCase().replace(/[-_\s]/g, '');
      const normId = sanitize(node.id);
      const normName = sanitize(node.name);
      const normType = sanitize(node.type);

      // 1. Direct ID match
      const directMatch = visitSteps.find((s) => sanitize(s.nodeId) === normId && normId.length > 0);
      if (directMatch) return directMatch;

      // 2. Name or Type match
      const nameOrTypeMatch = visitSteps.find((s) => {
        const stepNormId = sanitize(s.nodeId);
        const stepNormName = sanitize(s.nodeName);
        const stepNormType = sanitize(s.nodeType);

        return (
          (normName.length > 0 && (stepNormId === normName || stepNormName === normName)) ||
          (normType.length > 0 && (stepNormId === normType || stepNormType === normType))
        );
      });
      if (nameOrTypeMatch) return nameOrTypeMatch;

      // 3. Fallback match by step index
      const indexMatch = visitSteps.find((s) => {
        if (s.stepIndex === index + 1) return true;
        if (sanitize(s.nodeId) === `node${index + 1}` || sanitize(s.nodeId) === `step${index + 1}`) return true;
        return false;
      });
      if (indexMatch && (visitSteps.length === nodes.length || nodes.length <= 3)) return indexMatch;

      return undefined;
    };
  }, [visitSteps, nodes]);

  // Set of traversed edge keys: "sourceId->targetId"
  const traversedEdgePairs = useMemo(() => {
    const pairs = new Set<string>();

    const visitedNodeIds = new Set<string>();
    nodes.forEach((n, idx) => {
      const step = findMatchingStep(n, idx);
      if (step || isCompleted) {
        visitedNodeIds.add(n.id);
      }
    });

    for (let i = 0; i < visitSteps.length - 1; i++) {
      const from = visitSteps[i].nodeId;
      const to = visitSteps[i + 1].nodeId;
      pairs.add(`${from}->${to}`);
    }

    edges.forEach((edge) => {
      const sourceVisited = visitedNodeIds.has(edge.source);
      const targetVisited = visitedNodeIds.has(edge.target);
      if (sourceVisited && targetVisited) {
        pairs.add(`${edge.source}->${edge.target}`);
      }
    });

    return pairs;
  }, [visitSteps, nodes, edges, findMatchingStep, isCompleted]);

  // Map input nodes to ReactFlow nodes
  const flowNodes: Node[] = useMemo(() => {
    return nodes.map((node, index) => {
      const stepMatch = findMatchingStep(node, index);
      const isVisited = Boolean(stepMatch) || isCompleted;
      const isActive = activeNodeId !== undefined && (node.id === activeNodeId || Boolean(stepMatch && stepMatch.nodeId === activeNodeId));
      const stepIndex = stepMatch ? stepMatch.stepIndex : (isCompleted ? index + 1 : undefined);

      return {
        id: node.id,
        type: node.type || 'EventStart',
        position: {
          x: node.position?.x ?? (index % 3) * 380 + 50,
          y: node.position?.y ?? Math.floor(index / 3) * 180 + 50,
        },
        measured: { width: 346, height: 112 },
        data: {
          label: node.name,
          name: node.name,
          config: node.config || {},
          isVisited,
          isActive,
          stepIndex,
        },
      };
    });
  }, [nodes, findMatchingStep, isCompleted, activeNodeId]);
  // Map input edges to ReactFlow edges
  const flowEdges: Edge[] = useMemo(() => {
    return edges.map((edge) => {
      const isTraversed = traversedEdgePairs.has(`${edge.source}->${edge.target}`);
      const displayLabel = edge.label || edge.condition;

      return {
        id: edge.id,
        source: edge.source,
        target: edge.target,
        sourceHandle: edge.sourceHandle,
        targetHandle: edge.targetHandle,
        type: 'labeled',
        animated: false,
        style: {
          stroke: isTraversed ? '#4cd7f6' : '#464554',
          strokeWidth: isTraversed ? 2.5 : 1.5,
        },
        data: {
          label: displayLabel,
          condition: edge.condition,
          isTraversed,
        },
      };
    });
  }, [edges, traversedEdgePairs]);

  return (
    <div
      data-testid="execution-graph-view"
      className="relative w-full h-full min-h-[480px] overflow-hidden bg-[#0B0F19] border border-[#464554]/40"
      style={{ height: height || '540px' }}
    >
      <ReactFlow
        nodes={flowNodes}
        edges={flowEdges}
        nodeTypes={executionNodeTypes}
        edgeTypes={edgeTypes}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={true}
        panOnDrag={true}
        zoomOnScroll={true}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        proOptions={{ hideAttribution: true }}
      >
        <Background color="#464554" variant={BackgroundVariant.Dots} gap={20} size={1} />
        <Controls className="!bg-[#171b26] !border-[#464554] !fill-[#dfe2f1] !text-[#dfe2f1]" />
        <MiniMap
          position="bottom-right"
          style={{ width: 300, height: 180 }}
          pannable={true}
          zoomable={true}
          maskColor="rgba(11, 15, 25, 0.7)"
          maskStrokeColor="#4cd7f6"
          maskStrokeWidth={2}
          nodeComponent={CustomMiniMapNode}
          className="!bg-[#171b26] !border-[#464554] cursor-pointer shadow-xl"
        />
      </ReactFlow>
    </div>
  );
}

export function ExecutionGraphView(props: ExecutionGraphViewProps) {
  return (
    <ReactFlowProvider>
      <ExecutionGraphViewInner {...props} />
    </ReactFlowProvider>
  );
}
