import { describe, it, expect } from 'vitest';
import { wouldCreateCycle, validateConnection } from './CanvasValidation';
import type { GraphNode, GraphEdge } from '../../stores/editorStore';

describe('CanvasValidation - Pure Graph Commands & Rules', () => {
  const triggerNode: GraphNode = { id: 'node-trigger', type: 'EventStart', name: 'Start Event' };
  const actionNode1: GraphNode = { id: 'node-action1', type: 'Email', name: 'Send Email' };
  const actionNode2: GraphNode = { id: 'node-action2', type: 'SMS', name: 'Send SMS' };
  const conditionNode: GraphNode = { id: 'node-cond', type: 'Condition', name: 'Check VIP' };
  const exitNode: GraphNode = { id: 'node-exit', type: 'Exit', name: 'End Journey' };
  const experimentNode: GraphNode = { id: 'node-exp', type: 'Experiment', name: 'A/B Test' };
  const waitForEventNode: GraphNode = { id: 'node-wait', type: 'WaitForEvent', name: 'Wait for Click' };

  const nodes: GraphNode[] = [
    triggerNode,
    actionNode1,
    actionNode2,
    conditionNode,
    exitNode,
    experimentNode,
    waitForEventNode,
  ];

  describe('wouldCreateCycle', () => {
    it('returns true for self-loop connection', () => {
      const result = wouldCreateCycle(nodes, [], { source: 'node-action1', target: 'node-action1' });
      expect(result).toBe(true);
    });

    it('returns false for valid linear acyclic connections', () => {
      const edges: GraphEdge[] = [
        { id: 'e1', source: 'node-trigger', target: 'node-action1' },
        { id: 'e2', source: 'node-action1', target: 'node-action2' },
      ];
      const result = wouldCreateCycle(nodes, edges, { source: 'node-action2', target: 'node-exit' });
      expect(result).toBe(false);
    });

    it('detects simple 2-node cycle', () => {
      const edges: GraphEdge[] = [
        { id: 'e1', source: 'node-action1', target: 'node-action2' },
      ];
      const result = wouldCreateCycle(nodes, edges, { source: 'node-action2', target: 'node-action1' });
      expect(result).toBe(true);
    });

    it('detects deep multi-hop cycle in graph', () => {
      const edges: GraphEdge[] = [
        { id: 'e1', source: 'node-trigger', target: 'node-action1' },
        { id: 'e2', source: 'node-action1', target: 'node-cond' },
        { id: 'e3', source: 'node-cond', target: 'node-action2' },
      ];
      // Connecting node-action2 back to node-action1 creates a cycle
      const result = wouldCreateCycle(nodes, edges, { source: 'node-action2', target: 'node-action1' });
      expect(result).toBe(true);
    });

    it('respects ignoreEdgeId parameter during edge reconnection', () => {
      const edges: GraphEdge[] = [
        { id: 'e-reconnect', source: 'node-action1', target: 'node-action2' },
      ];
      // Reconnecting e-reconnect should ignore itself
      const result = wouldCreateCycle(
        nodes,
        edges,
        { source: 'node-action1', target: 'node-action2' },
        'e-reconnect'
      );
      expect(result).toBe(false);
    });
  });

  describe('validateConnection', () => {
    it('rejects connection if source or target is missing', () => {
      const res = validateConnection({ source: '', target: 'node-action1' }, nodes, []);
      expect(res.isValid).toBe(false);
      expect(res.reason).toMatch(/Invalid connection parameters/i);
    });

    it('rejects self-connection', () => {
      const res = validateConnection({ source: 'node-action1', target: 'node-action1' }, nodes, []);
      expect(res.isValid).toBe(false);
      expect(res.reason).toMatch(/Cannot connect a node to itself/i);
    });

    it('rejects connection to EventStart trigger node', () => {
      const res = validateConnection(
        { source: 'node-action1', target: 'node-trigger' },
        nodes,
        []
      );
      expect(res.isValid).toBe(false);
      expect(res.reason).toMatch(/Trigger nodes cannot have incoming connections/i);
    });

    it('rejects connection from Exit node', () => {
      const res = validateConnection(
        { source: 'node-exit', target: 'node-action1' },
        nodes,
        []
      );
      expect(res.isValid).toBe(false);
      expect(res.reason).toMatch(/Exit nodes cannot have outgoing connections/i);
    });

    it('rejects connection that creates a cycle', () => {
      const edges: GraphEdge[] = [
        { id: 'e1', source: 'node-action1', target: 'node-action2' },
      ];
      const res = validateConnection(
        { source: 'node-action2', target: 'node-action1' },
        nodes,
        edges
      );
      expect(res.isValid).toBe(false);
      expect(res.reason).toMatch(/create a cycle/i);
    });

    it('enforces single outgoing edge limit for standard action nodes', () => {
      const edges: GraphEdge[] = [
        { id: 'e1', source: 'node-action1', target: 'node-action2' },
      ];
      const res = validateConnection(
        { source: 'node-action1', target: 'node-exit' },
        nodes,
        edges
      );
      expect(res.isValid).toBe(false);
      expect(res.reason).toMatch(/already has an outgoing connection/i);
    });

    it('allows distinct branch connections on Condition nodes (true/false handles)', () => {
      const edges: GraphEdge[] = [
        { id: 'e-true', source: 'node-cond', sourceHandle: 'true', target: 'node-action1' },
      ];
      // Connecting false handle should succeed
      const resFalse = validateConnection(
        { source: 'node-cond', sourceHandle: 'false', target: 'node-action2' },
        nodes,
        edges
      );
      expect(resFalse.isValid).toBe(true);

      // Connecting true handle again should fail
      const resTrueDup = validateConnection(
        { source: 'node-cond', sourceHandle: 'true', target: 'node-exit' },
        nodes,
        edges
      );
      expect(resTrueDup.isValid).toBe(false);
      expect(resTrueDup.reason).toMatch(/Branch 'true' is already connected/i);
    });

    it('allows distinct branch connections on WaitForEvent nodes (event/timeout handles)', () => {
      const edges: GraphEdge[] = [
        { id: 'e-evt', source: 'node-wait', sourceHandle: 'event', target: 'node-action1' },
      ];
      // Connecting timeout handle should succeed
      const resTimeout = validateConnection(
        { source: 'node-wait', sourceHandle: 'timeout', target: 'node-action2' },
        nodes,
        edges
      );
      expect(resTimeout.isValid).toBe(true);

      // Duplicate event handle connection should fail
      const resEvtDup = validateConnection(
        { source: 'node-wait', sourceHandle: 'event', target: 'node-exit' },
        nodes,
        edges
      );
      expect(resEvtDup.isValid).toBe(false);
      expect(resEvtDup.reason).toMatch(/Branch 'event' is already connected/i);
    });

    it('allows distinct variant branch connections on Experiment nodes', () => {
      const edges: GraphEdge[] = [
        { id: 'e-var-a', source: 'node-exp', sourceHandle: 'variant_a', target: 'node-action1' },
      ];
      // Connecting variant_b handle should succeed
      const resB = validateConnection(
        { source: 'node-exp', sourceHandle: 'variant_b', target: 'node-action2' },
        nodes,
        edges
      );
      expect(resB.isValid).toBe(true);

      // Connecting variant_a handle again should fail
      const resADup = validateConnection(
        { source: 'node-exp', sourceHandle: 'variant_a', target: 'node-exit' },
        nodes,
        edges
      );
      expect(resADup.isValid).toBe(false);
      expect(resADup.reason).toMatch(/Variant branch 'variant_a' is already connected/i);
    });
  });
});
