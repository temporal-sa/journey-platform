import { Connection } from '@xyflow/react';
import type { GraphNode, GraphEdge } from '../../stores/editorStore';

export interface ValidationResult {
  isValid: boolean;
  reason?: string;
}

/**
 * Checks whether adding/reconnecting an edge from source to target creates a directed cycle.
 */
export function wouldCreateCycle(
  nodes: GraphNode[],
  edges: GraphEdge[],
  connection: { source: string; target: string },
  ignoreEdgeId?: string
): boolean {
  const { source, target } = connection;
  if (!source || !target) return false;
  if (source === target) return true; // Self loop is a cycle

  // Build adjacency graph from existing edges excluding ignored edge
  const adj = new Map<string, string[]>();
  nodes.forEach((n) => adj.set(n.id, []));

  edges.forEach((e) => {
    if (ignoreEdgeId && e.id === ignoreEdgeId) return;
    const list = adj.get(e.source) || [];
    list.push(e.target);
    adj.set(e.source, list);
  });

  // Temporarily add candidate connection source -> target
  const candidateList = adj.get(source) || [];
  candidateList.push(target);
  adj.set(source, candidateList);

  // Perform BFS/DFS from `target` to see if `source` can be reached
  const visited = new Set<string>();
  const queue: string[] = [target];
  visited.add(target);

  while (queue.length > 0) {
    const current = queue.shift()!;
    if (current === source) {
      return true; // Cycle detected: path from target back to source
    }

    const neighbors = adj.get(current) || [];
    for (const neighbor of neighbors) {
      if (!visited.has(neighbor)) {
        visited.add(neighbor);
        queue.push(neighbor);
      }
    }
  }

  return false;
}

/**
 * Validates connection cardinality and compiler rules for node connections.
 */
export function validateConnection(
  connection: Connection | { source: string; target: string; sourceHandle?: string | null; targetHandle?: string | null },
  nodes: GraphNode[],
  edges: GraphEdge[],
  ignoreEdgeId?: string
): ValidationResult {
  let { source, target, sourceHandle, targetHandle } = connection;

  if (!source || !target) {
    return { isValid: false, reason: 'Invalid connection parameters' };
  }

  const isSourceATargetHandle = sourceHandle === 'target';
  const isTargetASourceHandle =
    targetHandle === 'source' ||
    targetHandle === 'true' ||
    targetHandle === 'false' ||
    targetHandle === 'event' ||
    targetHandle === 'timeout' ||
    (targetHandle && targetHandle.startsWith('variant_'));

  if (isSourceATargetHandle || isTargetASourceHandle) {
    const tempSource = source;
    const tempSourceHandle = sourceHandle;
    source = target;
    sourceHandle = targetHandle;
    target = tempSource;
    targetHandle = tempSourceHandle;
  }
  if (source === target) {
    return { isValid: false, reason: 'Cannot connect a node to itself' };
  }

  const sourceNode = nodes.find((n) => n.id === source);
  const targetNode = nodes.find((n) => n.id === target);

  if (!sourceNode || !targetNode) {
    return { isValid: false, reason: 'Source or target node not found' };
  }

  const sourceType = sourceNode.type;
  const targetType = targetNode.type;

  // Rule 1: Triggers (EventStart) cannot have target handles (incoming connections)
  if (targetType === 'EventStart' || targetType === 'EventStartNode' || targetType === 'trigger') {
    return { isValid: false, reason: 'Trigger nodes cannot have incoming connections' };
  }

  // Rule 2: Exits (Exit) cannot have source handles (outgoing connections)
  if (sourceType === 'Exit' || sourceType === 'ExitNode') {
    return { isValid: false, reason: 'Exit nodes cannot have outgoing connections' };
  }

  // Rule 3: Cycle prevention
  if (wouldCreateCycle(nodes, edges, { source, target }, ignoreEdgeId)) {
    return { isValid: false, reason: 'Connection would create a cycle in workflow graph' };
  }

  // Rule 4: Outgoing Cardinality checks
  const existingSourceEdges = edges.filter(
    (e) => e.source === source && (!ignoreEdgeId || e.id !== ignoreEdgeId)
  );

  // Condition node: max 1 edge per handle ('true', 'false')
  if (sourceType === 'Condition' || sourceType === 'ConditionNode') {
    const handle = sourceHandle || 'true';
    const edgeOnHandle = existingSourceEdges.find((e) => (e.sourceHandle || 'true') === handle);
    if (edgeOnHandle) {
      return { isValid: false, reason: `Branch '${handle}' is already connected` };
    }
  }
  // WaitForEvent node: max 1 edge per handle ('event', 'timeout')
  else if (sourceType === 'WaitForEvent' || sourceType === 'WaitForEventNode') {
    const handle = sourceHandle || 'event';
    const edgeOnHandle = existingSourceEdges.find((e) => (e.sourceHandle || 'event') === handle);
    if (edgeOnHandle) {
      return { isValid: false, reason: `Branch '${handle}' is already connected` };
    }
  }
  // Experiment node: max 1 edge per handle ('variant_a', 'variant_b', etc.)
  else if (sourceType === 'Experiment' || sourceType === 'ExperimentNode') {
    const handle = sourceHandle || 'variant_a';
    const edgeOnHandle = existingSourceEdges.find(
      (e) => (e.sourceHandle || 'variant_a') === handle
    );
    if (edgeOnHandle) {
      return { isValid: false, reason: `Variant branch '${handle}' is already connected` };
    }
  }
  // Single-output nodes (Triggers, Actions, Delay, Webhook, etc.): max 1 outgoing edge total
  else {
    if (existingSourceEdges.length >= 1) {
      return { isValid: false, reason: 'Node already has an outgoing connection' };
    }
  }

  return { isValid: true };
}
