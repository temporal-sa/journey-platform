import React, { useState, useCallback, useMemo, useEffect } from 'react';
import {
  ReactFlow,
  Background,
  BackgroundVariant,
  MiniMap,
  Connection,
  Edge,
  Node,
  ReactFlowProvider,
  useReactFlow,
  OnNodesChange,
  OnEdgesChange,
  OnReconnect,
  SelectionMode,
} from '@xyflow/react';

import { useEditorStore, GraphNode, GraphEdge } from '../../stores/editorStore';
import { nodeTypes } from './nodes';
import { LabeledEdge } from './edges/LabeledEdge';
import { Palette } from './Palette';
import { CanvasToolbar } from './CanvasToolbar';
import { validateConnection } from './CanvasValidation';
import { getNodeValidationCounts, getNodeIssues } from '../inspector/validationMapping';
import { CloseButton } from '../common/CloseButton';

const edgeTypes = {
  labeled: LabeledEdge,
  default: LabeledEdge,
};

const DEFAULT_VIEWPORT = { x: 0, y: 0, zoom: 1 };
const MULTI_SELECTION_KEY_CODES = ['Control', 'Meta', 'Shift'];

function getBranchLabel(sourceHandle?: string | null): string | undefined {
  if (!sourceHandle) return undefined;
  if (sourceHandle === 'true') return 'True';
  if (sourceHandle === 'false') return 'False';
  if (sourceHandle === 'event') return 'Event';
  if (sourceHandle === 'timeout') return 'Timeout';
  if (sourceHandle.startsWith('variant_')) {
    const variantName = sourceHandle.replace('variant_', '');
    return `Variant ${variantName.toUpperCase()}`;
  }
  return sourceHandle;
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
  exit: { bg: '#36080b', stroke: '#ffb4ab', text: '#ffb4ab', icon: 'flag', label: 'Exit' },
};

const nodeDataCache = new Map<string, { key: string; data: Record<string, unknown> }>();
const nodePositionCache = new Map<string, { x: number; y: number }>();
const nodeStyleCache = new Map<number, { width: number }>();
const edgeDataCache = new Map<string, { key: string; data: Record<string, unknown> }>();
const nodeObjectCache = new Map<string, { key: string; node: Node }>();
const edgeObjectCache = new Map<string, { key: string; edge: Edge }>();

function getMemoizedNodeStyle(width: number) {
  let cached = nodeStyleCache.get(width);
  if (!cached) {
    cached = { width };
    nodeStyleCache.set(width, cached);
  }
  return cached;
}

function getMemoizedEdgeData(id: string, condition?: string, label?: string) {
  const cacheKey = `${id}:${condition || ''}:${label || ''}`;
  const cached = edgeDataCache.get(id);
  if (cached && cached.key === cacheKey) {
    return cached.data;
  }
  const newData = { condition, label };
  edgeDataCache.set(id, { key: cacheKey, data: newData });
  return newData;
}

function getMemoizedNodePosition(id: string, x: number, y: number) {
  const cached = nodePositionCache.get(id);
  if (cached && cached.x === x && cached.y === y) {
    return cached;
  }
  const newPos = { x, y };
  nodePositionCache.set(id, newPos);
  return newPos;
}

function getMemoizedNodeData(
  id: string,
  name: string,
  config: Record<string, unknown> | undefined,
  errors: number,
  warnings: number,
  issues: unknown[]
) {
  const cacheKey = `${id}:${name}:${JSON.stringify(config || {})}:${errors}:${warnings}:${issues.length}`;
  const cached = nodeDataCache.get(id);
  if (cached && cached.key === cacheKey) {
    return cached.data;
  }
  const newData = {
    label: name,
    name: name,
    config: config || {},
    errorCount: errors,
    warningCount: warnings,
    validationIssues: issues,
  };
  nodeDataCache.set(id, { key: cacheKey, data: newData });
  return newData;
}

function getMemoizedNodeObject(
  id: string,
  type: string,
  position: { x: number; y: number },
  width: number,
  height: number,
  style: { width: number },
  data: Record<string, unknown>,
  selected: boolean
): Node {
  const dataKey = nodeDataCache.get(id)?.key || '';
  const cacheKey = `${id}:${type}:${position.x}:${position.y}:${width}:${height}:${selected}:${dataKey}`;
  const cached = nodeObjectCache.get(id);
  if (cached && cached.key === cacheKey) {
    return cached.node;
  }
  const newNode: Node = {
    id,
    type,
    position,
    width,
    height,
    measured: { width, height },
    style,
    data,
    selected,
  };
  nodeObjectCache.set(id, { key: cacheKey, node: newNode });
  return newNode;
}

function getMemoizedEdgeObject(
  id: string,
  source: string,
  target: string,
  sourceHandle: string | undefined,
  targetHandle: string | undefined,
  label: string | undefined,
  type: string,
  selected: boolean,
  data: Record<string, unknown>
): Edge {
  const dataKey = edgeDataCache.get(id)?.key || '';
  const cacheKey = `${id}:${source}:${target}:${sourceHandle || ''}:${targetHandle || ''}:${label || ''}:${selected}:${dataKey}`;
  const cached = edgeObjectCache.get(id);
  if (cached && cached.key === cacheKey) {
    return cached.edge;
  }
  const newEdge: Edge = {
    id,
    source,
    target,
    sourceHandle,
    targetHandle,
    label,
    type,
    selected,
    data,
  };
  edgeObjectCache.set(id, { key: cacheKey, edge: newEdge });
  return newEdge;
}


function CustomMiniMapNode({
  x,
  y,
  width,
  height,
  color,
  borderRadius,
  id,
}: {
  x: number;
  y: number;
  width?: number;
  height?: number;
  color?: string;
  strokeColor?: string;
  strokeWidth?: number;
  borderRadius?: number;
  id?: string;
}) {
  const { getNode } = useReactFlow();
  const { currentDraft } = useEditorStore();
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

  const w = (node.measured?.width as number) || (width && width > 0 ? width : 260);
  const h = (node.measured?.height as number) || (height && height > 0 ? height : 90);
  const displayName = (node.data?.name as string) || (node.data?.label as string) || type;

  const isAction = ['Email', 'SMS', 'Push', 'InApp', 'Webhook', 'action'].includes(type);
  const rx = isAction ? h / 2 : (borderRadius || 12);

  const outgoingEdges = currentDraft?.edges.filter((e) => e.source === id) || [];

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

        return (
          <g key={`minimap-edge-${edge.id}`}>
            <path
              d={pathD}
              fill="none"
              stroke="#6366f1"
              strokeWidth={4}
              strokeOpacity={0.4}
              style={{ pointerEvents: 'none' }}
            />
            <path
              d={pathD}
              fill="none"
              stroke="#c0c1ff"
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
        rx={rx}
        ry={rx}
        fill={colorConfig.bg}
        stroke={colorConfig.stroke}
        strokeWidth={3}
        fillOpacity={0.95}
      />

      {/* Mini Icon Circle */}
      <circle
        cx={x + 36}
        cy={y + h / 2}
        r={Math.min(h / 3, 20)}
        fill={colorConfig.stroke}
        fillOpacity={0.25}
        stroke={colorConfig.stroke}
        strokeWidth={1.5}
      />
      <text
        x={x + 36}
        y={y + h / 2 + 1}
        fill={colorConfig.text}
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
        fill={colorConfig.text}
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

interface CanvasProps {
  onSaveDraft?: () => void;
  onSimulateConflict?: () => void;
  onPublish?: () => void;
  onTestMode?: () => void;
  onUploadStaticList?: () => void;
  onLaunchTestRun?: () => void;
  onKeyboardShortcuts?: () => void;
  onAgyContext?: () => void;
}

export function CanvasInner({ onSaveDraft, onSimulateConflict, onPublish, onTestMode, onUploadStaticList, onLaunchTestRun, onKeyboardShortcuts, onAgyContext }: CanvasProps) {
  const { screenToFlowPosition, zoomIn, zoomOut, fitView } = useReactFlow();
  const currentDraft = useEditorStore((s) => s.currentDraft);

  const handleZoomIn = useCallback(() => {
    zoomIn({ duration: 200 });
  }, [zoomIn]);

  const handleZoomOut = useCallback(() => {
    zoomOut({ duration: 200 });
  }, [zoomOut]);

  const handleFitView = useCallback(() => {
    fitView({ padding: 0.2, duration: 200 });
  }, [fitView]);
  const selectedNodeId = useEditorStore((s) => s.selectedNodeId);
  const selectedEdgeId = useEditorStore((s) => s.selectedEdgeId);
  const selectedNodeIds = useEditorStore((s) => s.selectedNodeIds);
  const selectedEdgeIds = useEditorStore((s) => s.selectedEdgeIds);
  const addNode = useEditorStore((s) => s.addNode);
  const setSelectedNodeId = useEditorStore((s) => s.setSelectedNodeId);
  const setSelectedNodeIds = useEditorStore((s) => s.setSelectedNodeIds);
  const setSelectedEdgeId = useEditorStore((s) => s.setSelectedEdgeId);
  const setSelectedEdgeIds = useEditorStore((s) => s.setSelectedEdgeIds);
  const updateNodes = useEditorStore((s) => s.updateNodes);
  const updateEdges = useEditorStore((s) => s.updateEdges);
  const storeAddEdge = useEditorStore((s) => s.addEdge);
  const deleteElements = useEditorStore((s) => s.deleteElements);
  const undo = useEditorStore((s) => s.undo);
  const redo = useEditorStore((s) => s.redo);
  const canUndo = useEditorStore((s) => s.canUndo);
  const canRedo = useEditorStore((s) => s.canRedo);
  const snapToGridEnabled = useEditorStore((s) => s.snapToGridEnabled);
  const setSnapToGridEnabled = useEditorStore((s) => s.setSnapToGridEnabled);
  const snapAllNodesToGrid = useEditorStore((s) => s.snapAllNodesToGrid);
  const autoArrangeHorizontal = useEditorStore((s) => s.autoArrangeHorizontal);
  const autoArrangeVertical = useEditorStore((s) => s.autoArrangeVertical);
  const validationResult = useEditorStore((s) => s.validationResult);

  const isCanvasLocked = useEditorStore((s) => s.isCanvasLocked);
  const toggleCanvasLock = useEditorStore((s) => s.toggleCanvasLock);

  const [validationError, setValidationError] = useState<string | null>(null);
  const [isPanMode, setIsPanMode] = useState(false);

  // Auto-clear validation notification after 4s
  useEffect(() => {
    if (validationError) {
      const timer = setTimeout(() => setValidationError(null), 4000);
      return () => clearTimeout(timer);
    }
  }, [validationError]);
  useEffect(() => {
    (window as any).__EDITOR_STORE__ = useEditorStore.getState();
  }, [currentDraft]);
  useEffect(() => {
    return () => {
      setSelectedNodeId(null);
      setSelectedNodeIds([]);
      setSelectedEdgeId(null);
      setSelectedEdgeIds([]);
    };
  }, [setSelectedNodeId, setSelectedNodeIds, setSelectedEdgeId, setSelectedEdgeIds]);

  // Convert GraphDraft nodes -> ReactFlow Nodes with validation issues & badges
  const nodes: Node[] = useMemo(() => {
    if (!currentDraft?.nodes) return [];
    return currentDraft.nodes.map((n, i) => {
      const { errors, warnings } = getNodeValidationCounts(n.id, validationResult);
      const isAction = ['Email', 'SMS', 'Push', 'InApp', 'Webhook', 'action'].includes(n.type);
      const nodeWidth = isAction ? 288 : (n.type === 'Condition' || n.type === 'Experiment' ? 240 : 220);
      const nodeHeight = n.type === 'EventStart' ? 160 : (n.type === 'Experiment' ? 150 : 90);

      const defaultX = 100 + (i % 3) * 360;
      const defaultY = 150 + Math.floor(i / 3) * 220;
      const posX = n.position?.x ?? defaultX;
      const posY = n.position?.y ?? defaultY;
      const isSelected = selectedNodeIds.includes(n.id) || n.id === selectedNodeId;

      return getMemoizedNodeObject(
        n.id,
        n.type,
        getMemoizedNodePosition(n.id, posX, posY),
        nodeWidth,
        nodeHeight,
        getMemoizedNodeStyle(nodeWidth),
        getMemoizedNodeData(n.id, n.name, n.config, errors, warnings, getNodeIssues(n.id, validationResult)),
        isSelected
      );
    });
  }, [currentDraft?.nodes, selectedNodeId, selectedNodeIds, validationResult]);

  // Convert GraphDraft edges -> ReactFlow Edges
  const edges: Edge[] = useMemo(() => {
    if (!currentDraft?.edges) return [];
    return currentDraft.edges.map((e) => {
      const resolvedSourceHandle =
        e.sourceHandle ||
        (e.condition === 'true' || e.id.includes('-true-')
          ? 'true'
          : e.condition === 'false' || e.id.includes('-false-')
          ? 'false'
          : e.condition === 'event' || e.id.includes('-event-')
          ? 'event'
          : e.condition === 'timeout' || e.id.includes('-timeout-')
          ? 'timeout'
          : e.condition || undefined);

      const isSelected = selectedEdgeIds.includes(e.id) || e.id === selectedEdgeId;
      const edgeLabel = e.label || getBranchLabel(resolvedSourceHandle);
      const edgeData = getMemoizedEdgeData(e.id, e.condition || resolvedSourceHandle, edgeLabel);

      return getMemoizedEdgeObject(
        e.id,
        e.source,
        e.target,
        resolvedSourceHandle,
        e.targetHandle,
        edgeLabel,
        'labeled',
        isSelected,
        edgeData
      );
    });
  }, [currentDraft?.edges, selectedEdgeId, selectedEdgeIds]);

  // Handle Node Position Changes
  // Handle Node Position & Deletion Changes
  const onNodesChange: OnNodesChange = useCallback(
    (changes) => {
      if (!currentDraft) return;

      const removedNodeIds = changes
        .filter((c) => c.type === 'remove')
        .map((c) => (c as { id: string }).id);

      if (removedNodeIds.length > 0) {
        deleteElements(removedNodeIds, []);
        return;
      }

      let positionChanged = false;
      let isDragging = false;

      const updatedNodes = currentDraft.nodes.map((node) => {
        const change = changes.find((c) => 'id' in c && c.id === node.id);
        if (change && change.type === 'position') {
          const pos = change.position || (change as any).positionAbsolute;
          if (pos) {
            const currentX = node.position?.x ?? 0;
            const currentY = node.position?.y ?? 0;
            const dx = Math.abs(pos.x - currentX);
            const dy = Math.abs(pos.y - currentY);

            if (dx > 0.01 || dy > 0.01) {
              positionChanged = true;
              if (change.dragging) {
                isDragging = true;
              }
              return {
                ...node,
                position: { x: pos.x, y: pos.y },
              };
            }
          }
        }
        return node;
      });

      if (positionChanged) {
        updateNodes(updatedNodes, isDragging);
      }
    },
    [currentDraft, updateNodes, deleteElements]
  );

  const onEdgesChange: OnEdgesChange = useCallback(
    (changes) => {
      if (!currentDraft) return;

      const removedEdgeIds = changes
        .filter((c) => c.type === 'remove')
        .map((c) => (c as { id: string }).id);

      if (removedEdgeIds.length > 0) {
        deleteElements([], removedEdgeIds);
      }
    },
    [currentDraft, deleteElements]
  );

  const onEdgesDelete = useCallback(
    (deletedEdges: Edge[]) => {
      deleteElements([], deletedEdges.map((e) => e.id));
    },
    [deleteElements]
  );
  const onNodeDragStop = useCallback(
    (_: unknown, node: Node) => {
      if (!currentDraft) return;
      const GRID_SIZE = 16;
      const updatedNodes = currentDraft.nodes.map((n) => {
        if (n.id === node.id) {
          const finalX = snapToGridEnabled ? Math.round(node.position.x / GRID_SIZE) * GRID_SIZE : node.position.x;
          const finalY = snapToGridEnabled ? Math.round(node.position.y / GRID_SIZE) * GRID_SIZE : node.position.y;
          return {
            ...n,
            position: { x: finalX, y: finalY },
          };
        }
        return n;
      });
      updateNodes(updatedNodes);
    },
    [currentDraft, updateNodes, snapToGridEnabled]
  );

  const onSelectionChange = useCallback(
    ({ nodes: selNodes, edges: selEdges }: { nodes: Node[]; edges: Edge[] }) => {
      const nodeIds = selNodes.map((n) => n.id);
      const edgeIds = selEdges.map((e) => e.id);

      const state = useEditorStore.getState();
      const currentNodes = state.selectedNodeIds;
      const currentEdges = state.selectedEdgeIds;

      const nodesChanged = nodeIds.length !== currentNodes.length || nodeIds.some((id, idx) => id !== currentNodes[idx]);
      const edgesChanged = edgeIds.length !== currentEdges.length || edgeIds.some((id, idx) => id !== currentEdges[idx]);

      if (nodesChanged) {
        setSelectedNodeIds(nodeIds);
      }
      if (edgesChanged) {
        setSelectedEdgeIds(edgeIds);
      }
    },
    [setSelectedNodeIds, setSelectedEdgeIds]
  );

  const onNodesDelete = useCallback(
    (deletedNodes: Node[]) => {
      deleteElements(deletedNodes.map((n) => n.id), []);
    },
    [deleteElements]
  );

  const onNodeClick = useCallback(
    (event: React.MouseEvent, node: Node) => {
      const isMulti = event.shiftKey || event.metaKey || event.ctrlKey;
      if (isMulti) {
        const currentIds = useEditorStore.getState().selectedNodeIds;
        const exists = currentIds.includes(node.id);
        const newIds = exists ? currentIds.filter((id) => id !== node.id) : [...currentIds, node.id];
        setSelectedNodeIds(newIds);
      } else {
        setSelectedNodeId(node.id);
      }
    },
    [setSelectedNodeId, setSelectedNodeIds]
  );

  // Handle New Edge Connection with validation
  const checkIsValidConnection = useCallback(
    (connection: Connection | Edge) => {
      if (!currentDraft) return false;
      const result = validateConnection(
        connection as Connection,
        currentDraft.nodes,
        currentDraft.edges
      );
      return result.isValid;
    },
    [currentDraft]
  );

  const onConnect = useCallback(
    (connection: Connection) => {
      if (isCanvasLocked) return;
      if (!currentDraft) return;
      setValidationError(null);

      const result = validateConnection(
        connection,
        currentDraft.nodes,
        currentDraft.edges
      );

      if (!result.isValid) {
        setValidationError(result.reason || 'Invalid connection');
        return;
      }

      let sourceId = connection.source;
      let targetId = connection.target;
      let sHandle = connection.sourceHandle;
      let tHandle = connection.targetHandle;

      if (
        sHandle === 'target' ||
        tHandle === 'source' ||
        tHandle === 'true' ||
        tHandle === 'false' ||
        tHandle === 'event' ||
        tHandle === 'timeout' ||
        (tHandle && tHandle.startsWith('variant_'))
      ) {
        const temp = sourceId;
        const tempH = sHandle;
        sourceId = targetId;
        sHandle = tHandle;
        targetId = temp;
        tHandle = tempH;
      }

      const edgeLabel = getBranchLabel(sHandle);
      const newEdge: GraphEdge = {
        id: `edge-${sourceId}-${sHandle || 'default'}-${targetId}-${Date.now()}`,
        source: sourceId,
        target: targetId,
        sourceHandle: sHandle || undefined,
        targetHandle: tHandle || undefined,
        label: edgeLabel,
        condition: sHandle || undefined,
      };

      storeAddEdge(newEdge);
    },
    [currentDraft, storeAddEdge, isCanvasLocked]
  );

  // Handle Edge Reconnection
  const onReconnect: OnReconnect = useCallback(
    (oldEdge, newConnection) => {
      if (isCanvasLocked || !currentDraft) return;

      const result = validateConnection(
        newConnection,
        currentDraft.nodes,
        currentDraft.edges,
        oldEdge.id
      );

      if (!result.isValid) {
        setValidationError(result.reason || 'Invalid connection');
        return;
      }

      setValidationError(null);
      const edgeLabel = getBranchLabel(newConnection.sourceHandle);
      const updatedEdges = currentDraft.edges.map((e) => {
        if (e.id === oldEdge.id) {
          return {
            ...e,
            source: newConnection.source,
            target: newConnection.target,
            sourceHandle: newConnection.sourceHandle || undefined,
            targetHandle: newConnection.targetHandle || undefined,
            label: edgeLabel,
          };
        }
        return e;
      });

      updateEdges(updatedEdges);
    },
    [currentDraft, updateEdges, isCanvasLocked]
  );

  // Handle Palette Drag-and-Drop Drop
  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    if (isCanvasLocked) return;
    event.dataTransfer.dropEffect = 'move';
  }, [isCanvasLocked]);

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();
      if (isCanvasLocked) return;

      const type = event.dataTransfer.getData('application/reactflow/type');
      const name = event.dataTransfer.getData('application/reactflow/name');

      if (!type) return;

      const position = screenToFlowPosition({
        x: event.clientX,
        y: event.clientY,
      });

      const GRID_SIZE = 16;
      const finalPosition = snapToGridEnabled
        ? {
            x: Math.round(position.x / GRID_SIZE) * GRID_SIZE,
            y: Math.round(position.y / GRID_SIZE) * GRID_SIZE,
          }
        : position;

      const newNode: GraphNode = {
        id: `node-${Date.now()}`,
        type,
        name: name || type,
        position: finalPosition,
        config: {},
      };

      addNode(newNode);
      setSelectedNodeId(newNode.id);
    },
    [screenToFlowPosition, addNode, setSelectedNodeId, snapToGridEnabled, isCanvasLocked]
  );

  // Handle Direct Node Creation from Palette Click
  const handleAddNodeFromPalette = useCallback(
    (type: string, name: string) => {
      if (isCanvasLocked) return;
      const GRID_SIZE = 16;
      const initialPos = { x: 300 + Math.random() * 50, y: 200 + Math.random() * 50 };
      const finalPosition = snapToGridEnabled
        ? {
            x: Math.round(initialPos.x / GRID_SIZE) * GRID_SIZE,
            y: Math.round(initialPos.y / GRID_SIZE) * GRID_SIZE,
          }
        : initialPos;

      const newNode: GraphNode = {
        id: `node-${Date.now()}`,
        type,
        name,
        position: finalPosition,
        config: {},
      };

      addNode(newNode);
      setSelectedNodeId(newNode.id);
    },
    [addNode, setSelectedNodeId, snapToGridEnabled]
  );

  // Delete Selected Elements
  const handleDeleteSelected = useCallback(() => {
    const nodesToDelete = selectedNodeIds.length > 0 ? selectedNodeIds : (selectedNodeId ? [selectedNodeId] : []);
    const edgesToDelete = selectedEdgeIds.length > 0 ? selectedEdgeIds : (selectedEdgeId ? [selectedEdgeId] : []);
    if (nodesToDelete.length > 0 || edgesToDelete.length > 0) {
      deleteElements(nodesToDelete, edgesToDelete);
    }
  }, [selectedNodeIds, selectedEdgeIds, selectedNodeId, selectedEdgeId, deleteElements]);

  // Keyboard Shortcuts (Undo, Redo, Delete)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (
        e.target instanceof HTMLInputElement ||
        e.target instanceof HTMLTextAreaElement
      ) {
        return;
      }

      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'z') {
        if (e.shiftKey) {
          redo();
        } else {
          undo();
        }
        e.preventDefault();
      } else if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'y') {
        redo();
        e.preventDefault();
      } else if (e.key === 'Delete' || e.key === 'Backspace') {
        handleDeleteSelected();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [undo, redo, handleDeleteSelected]);

  const hasSelection = selectedNodeIds.length > 0 || selectedEdgeIds.length > 0 || Boolean(selectedNodeId || selectedEdgeId);

  return (
    <div
      style={{
        display: 'flex',
        width: '100%',
        height: '100%',
        maxHeight: '100%',
        minHeight: 0,
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: '#0b0f19',
        overflow: 'hidden',
      }}
      data-testid="canvas-editor-container"
    >
      <Palette onAddNode={handleAddNodeFromPalette} />

      <div
        style={{ flex: 1, position: 'relative', height: '100%', width: '100%', minWidth: 0, minHeight: 0, overflow: 'hidden' }}
        onDragOver={onDragOver}
        onDrop={onDrop}
        data-testid="reactflow-drop-zone"
      >
        <CanvasToolbar
          onUndo={undo}
          onRedo={redo}
          canUndo={canUndo}
          canRedo={canRedo}
          onZoomIn={handleZoomIn}
          onZoomOut={handleZoomOut}
          onFitView={handleFitView}
          onDeleteSelected={handleDeleteSelected}
          hasSelection={hasSelection}
          isPanMode={isPanMode}
          onTogglePanMode={() => setIsPanMode((prev) => !prev)}
          isLocked={isCanvasLocked}
          onToggleLock={toggleCanvasLock}
          onSaveDraft={onSaveDraft}
          onSimulateConflict={onSimulateConflict}
          onPublish={onPublish}
          onTestMode={onTestMode}
          onUploadStaticList={onUploadStaticList}
          onLaunchTestRun={onLaunchTestRun}
          onKeyboardShortcuts={onKeyboardShortcuts}
          onAgyContext={onAgyContext}
          snapToGridEnabled={snapToGridEnabled}
          onToggleSnapToGrid={() => setSnapToGridEnabled(!snapToGridEnabled)}
          onSnapAllNodesToGrid={snapAllNodesToGrid}
          onAutoArrangeHorizontal={autoArrangeHorizontal}
          onAutoArrangeVertical={autoArrangeVertical}
        />

        {validationError && (
          <div
            className="absolute top-16 left-4 z-50 glass-panel px-4 py-2 rounded-xl border border-rose-500/30 text-rose-300 text-xs font-semibold shadow-2xl flex items-center gap-2 animate-bounce"
            data-testid="validation-error-banner"
          >
            <span className="material-symbols-outlined text-rose-400 text-sm">warning</span>
            <span>{validationError}</span>
            <CloseButton
              onClick={() => setValidationError(null)}
              ariaLabel="Dismiss validation error"
              size="sm"
              className="ml-2 text-rose-400 hover:text-white"
            />
          </div>
        )}

        <ReactFlow
          className="w-full h-full relative"
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onNodeClick={onNodeClick}
          onConnect={onConnect}
          onReconnect={onReconnect}
          isValidConnection={checkIsValidConnection}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          selectionMode={SelectionMode.Partial}
          selectionKeyCode="Shift"
          onSelectionChange={onSelectionChange}
          onNodesDelete={onNodesDelete}
          onEdgesDelete={onEdgesDelete}
          onNodeDragStop={onNodeDragStop}
          selectionOnDrag={!isPanMode}
          panOnDrag={isPanMode}
          multiSelectionKeyCode={MULTI_SELECTION_KEY_CODES}
          nodesDraggable={!isCanvasLocked}
          nodesConnectable={!isCanvasLocked}
          elementsSelectable={!isCanvasLocked}
          edgesFocusable={!isCanvasLocked}
          nodesFocusable={!isCanvasLocked}
          minZoom={0.1}
          maxZoom={4}
          defaultViewport={DEFAULT_VIEWPORT}
          fitViewOptions={{ padding: 0.2, maxZoom: 1.5 }}
          snapToGrid={snapToGridEnabled}
          snapGrid={[16, 16]}
        >
          <Background variant={BackgroundVariant.Lines} color="#1f2937" gap={16} size={1} />
          <MiniMap
            position="bottom-right"
            style={{ width: 320, height: 200 }}
            maskColor="rgba(183, 109, 255, 0.2)"
            maskStrokeColor="#ddb7ff"
            maskStrokeWidth={2.5}
            nodeStrokeColor="#464554"
            nodeStrokeWidth={2}
            nodeBorderRadius={10}
            nodeComponent={CustomMiniMapNode}
            nodeColor={(node) => {
              const t = (node.type || '').toLowerCase();
              if (t.includes('trigger') || t.includes('eventstart')) return '#4cd7f6';
              if (t.includes('condition') || t.includes('delay')) return '#f59e0b';
              if (t.includes('wait')) return '#3b82f6';
              if (t.includes('experiment')) return '#ddb7ff';
              if (t.includes('exit')) return '#ffb4ab';
              return '#c0c1ff';
            }}
            zoomable
            pannable
            className="!rounded-2xl !overflow-hidden shadow-2xl"
          />
        </ReactFlow>
      </div>
    </div>
  );
}

export function Canvas(props: CanvasProps) {
  return (
    <ReactFlowProvider>
      <CanvasInner {...props} />
    </ReactFlowProvider>
  );
}
