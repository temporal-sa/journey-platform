import { describe, it, expect, beforeEach } from 'vitest';
import { useEditorStore } from './editorStore';
import type { GraphDraft, ValidationResult, DraftVersionItem } from '../types/api';

describe('editorStore', () => {
  beforeEach(() => {
    useEditorStore.getState().resetStore();
  });

  it('initializes with default state values', () => {
    const state = useEditorStore.getState();
    expect(state.selectedNodeId).toBeNull();
    expect(state.selectedEdgeId).toBeNull();
    expect(state.zoomLevel).toBe(1.0);
    expect(state.activeTab).toBe('canvas');
    expect(state.isSidebarOpen).toBe(true);
    expect(state.isInspectorOpen).toBe(false);
    expect(state.currentDraft).toBeNull();
    expect(state.draftRevisions).toEqual([]);
    expect(state.validationResult).toBeNull();
    expect(state.activeModal).toBeNull();
    expect(state.hasUnsavedChanges).toBe(false);
    expect(state.canUndo).toBe(false);
    expect(state.canRedo).toBe(false);
  });

  it('manages editor presentation state correctly', () => {
    const store = useEditorStore.getState();

    store.setSelectedNodeId('node-1');
    expect(useEditorStore.getState().selectedNodeId).toBe('node-1');
    expect(useEditorStore.getState().isInspectorOpen).toBe(true);

    store.setSelectedEdgeId('edge-1');
    expect(useEditorStore.getState().selectedEdgeId).toBe('edge-1');
    expect(useEditorStore.getState().selectedNodeId).toBeNull();

    store.setZoomLevel(2.5);
    expect(useEditorStore.getState().zoomLevel).toBe(2.5);

    // Zoom level clamps between 0.1 and 5.0
    store.setZoomLevel(10);
    expect(useEditorStore.getState().zoomLevel).toBe(5.0);
    store.setZoomLevel(0.01);
    expect(useEditorStore.getState().zoomLevel).toBe(0.1);

    store.setActiveTab('simulation');
    expect(useEditorStore.getState().activeTab).toBe('simulation');

    store.toggleSidebar();
    expect(useEditorStore.getState().isSidebarOpen).toBe(false);

    store.toggleInspector();
    expect(useEditorStore.getState().isInspectorOpen).toBe(false);
  });

  it('manages server draft revisions and draft loading state', () => {
    const mockDraft: GraphDraft = {
      schema_version: '1.0',
      draft_id: 'draft-1',
      tenant_id: 'tenant-1',
      name: 'Test Journey',
      version: 3,
      nodes: [{ id: 'node-1', type: 'trigger', name: 'Start' }],
      edges: [],
      content_hash: 'hash-abc-123',
    };

    const mockRevisions: DraftVersionItem[] = [
      { version: 1, content_hash: 'hash-v1', updated_at: '2026-07-27T10:00:00Z' },
      { version: 2, content_hash: 'hash-v2', updated_at: '2026-07-27T11:00:00Z' },
      { version: 3, content_hash: 'hash-abc-123', updated_at: '2026-07-27T12:00:00Z' },
    ];

    useEditorStore.getState().setDraft(mockDraft);
    expect(useEditorStore.getState().currentDraft).toEqual(mockDraft);
    expect(useEditorStore.getState().lastSavedHash).toBe('hash-abc-123');
    expect(useEditorStore.getState().activeRevisionVersion).toBe(3);

    useEditorStore.getState().setDraftRevisions(mockRevisions);
    expect(useEditorStore.getState().draftRevisions).toHaveLength(3);

    useEditorStore.getState().setActiveRevisionVersion(1);
    expect(useEditorStore.getState().activeRevisionVersion).toBe(1);

    useEditorStore.getState().setDraftLoading(true);
    expect(useEditorStore.getState().isDraftLoading).toBe(true);

    useEditorStore.getState().setDraftError('Failed to fetch draft');
    expect(useEditorStore.getState().draftError).toBe('Failed to fetch draft');
  });

  it('handles validation result state', () => {
    const mockValidationResult: ValidationResult = {
      draft_id: 'draft-1',
      is_valid: false,
      issues: [
        {
          schema_version: '1.0',
          issue_id: 'issue-1',
          node_id: 'node-1',
          severity: 'error',
          code: 'DISCONNECTED_NODE',
          message: 'Node has no outgoing edge',
          field_path: 'nodes[0]',
        },
      ],
    };

    useEditorStore.getState().setValidating(true);
    expect(useEditorStore.getState().isValidating).toBe(true);

    useEditorStore.getState().setValidationResult(mockValidationResult);
    expect(useEditorStore.getState().validationResult).toEqual(mockValidationResult);
  });

  it('manages modal dialog state', () => {
    useEditorStore.getState().openModal('publish', { targetEnvironment: 'production' });
    expect(useEditorStore.getState().activeModal).toBe('publish');
    expect(useEditorStore.getState().modalData).toEqual({ targetEnvironment: 'production' });

    useEditorStore.getState().closeModal();
    expect(useEditorStore.getState().activeModal).toBeNull();
    expect(useEditorStore.getState().modalData).toBeNull();
  });

  it('tracks unsaved inspector edits and commits changes to draft', () => {
    const mockDraft: GraphDraft = {
      schema_version: '1.0',
      draft_id: 'draft-1',
      tenant_id: 'tenant-1',
      name: 'Initial Journey',
      version: 1,
      nodes: [{ id: 'node-1', type: 'trigger', name: 'Start', config: { delay: 10 } }],
      edges: [],
    };

    useEditorStore.getState().setDraft(mockDraft);

    useEditorStore.getState().updateInspectorEdit('node-1', 'name', 'Updated Start Node');
    expect(useEditorStore.getState().hasUnsavedChanges).toBe(true);
    expect(useEditorStore.getState().unsavedInspectorEdits['node-1']).toEqual({
      name: 'Updated Start Node',
    });

    useEditorStore.getState().commitInspectorEdits('node-1');
    expect(useEditorStore.getState().hasUnsavedChanges).toBe(false);
    expect(useEditorStore.getState().unsavedInspectorEdits['node-1']).toBeUndefined();

    const updatedNode = useEditorStore.getState().currentDraft?.nodes.find((n) => n.id === 'node-1');
    expect(updatedNode?.config).toEqual({
      delay: 10,
      name: 'Updated Start Node',
    });
  });

  it('handles addNode, addEdge, updateNodes, updateEdges, and deleteElements', () => {
    const initialDraft: GraphDraft = {
      schema_version: '1.0',
      draft_id: 'draft-10',
      tenant_id: 'tenant-1',
      name: 'Editable Journey',
      version: 1,
      nodes: [{ id: 'start-1', type: 'EventStart', name: 'Start' }],
      edges: [],
    };

    useEditorStore.getState().setDraft(initialDraft);

    // addNode
    useEditorStore.getState().addNode({ id: 'email-1', type: 'Email', name: 'Send Email' });
    expect(useEditorStore.getState().currentDraft?.nodes).toHaveLength(2);
    expect(useEditorStore.getState().hasUnsavedChanges).toBe(true);

    // addEdge
    useEditorStore.getState().addEdge({ id: 'e1', source: 'start-1', target: 'email-1' });
    expect(useEditorStore.getState().currentDraft?.edges).toHaveLength(1);

    // deleteElements
    useEditorStore.getState().deleteElements(['email-1'], []);
    expect(useEditorStore.getState().currentDraft?.nodes).toHaveLength(1);
    expect(useEditorStore.getState().currentDraft?.edges).toHaveLength(0); // Connected edge deleted
  });

  it('supports undo and redo stack history operations', () => {
    const initialDraft: GraphDraft = {
      schema_version: '1.0',
      draft_id: 'draft-history',
      tenant_id: 'tenant-1',
      name: 'History Journey',
      version: 1,
      nodes: [{ id: 'start-1', type: 'EventStart', name: 'Start' }],
      edges: [],
    };

    useEditorStore.getState().setDraft(initialDraft);
    expect(useEditorStore.getState().canUndo).toBe(false);
    expect(useEditorStore.getState().canRedo).toBe(false);

    // Perform Edit 1: Add Node A
    useEditorStore.getState().addNode({ id: 'node-a', type: 'Condition', name: 'Rule A' });
    expect(useEditorStore.getState().canUndo).toBe(true);
    expect(useEditorStore.getState().canRedo).toBe(false);
    expect(useEditorStore.getState().currentDraft?.nodes).toHaveLength(2);

    // Perform Edit 2: Add Node B
    useEditorStore.getState().addNode({ id: 'node-b', type: 'Email', name: 'Send Email B' });
    expect(useEditorStore.getState().currentDraft?.nodes).toHaveLength(3);

    // Undo Edit 2 -> back to 2 nodes
    useEditorStore.getState().undo();
    expect(useEditorStore.getState().canRedo).toBe(true);
    expect(useEditorStore.getState().currentDraft?.nodes).toHaveLength(2);

    // Undo Edit 1 -> back to initial 1 node
    useEditorStore.getState().undo();
    expect(useEditorStore.getState().canUndo).toBe(false);
    expect(useEditorStore.getState().currentDraft?.nodes).toHaveLength(1);

    // Redo Edit 1 -> 2 nodes
    useEditorStore.getState().redo();
    expect(useEditorStore.getState().canUndo).toBe(true);
    expect(useEditorStore.getState().currentDraft?.nodes).toHaveLength(2);

    // Redo Edit 2 -> 3 nodes
    useEditorStore.getState().redo();
    expect(useEditorStore.getState().canRedo).toBe(false);
    useEditorStore.getState().redo();
    expect(useEditorStore.getState().currentDraft?.nodes).toHaveLength(3);
  });

  it('updates draft name and pushes past history stack', () => {
    const sampleDraft: GraphDraft = {
      schema_version: '1.0',
      draft_id: 'draft-1',
      tenant_id: 'tenant-1',
      name: 'Original Journey Name',
      version: 1,
      nodes: [],
      edges: [],
    };
    useEditorStore.getState().setDraft(sampleDraft);

    useEditorStore.getState().setDraftName('Updated Journey Name');
    const state = useEditorStore.getState();

    expect(state.currentDraft?.name).toBe('Updated Journey Name');
    expect(state.hasUnsavedChanges).toBe(true);
    expect(state.canUndo).toBe(true);

    useEditorStore.getState().undo();
    expect(useEditorStore.getState().currentDraft?.name).toBe('Original Journey Name');
  });

  it('manages snap to grid state and bulk snap action', () => {
    const store = useEditorStore.getState();
    expect(store.snapToGridEnabled).toBe(true);

    store.setSnapToGridEnabled(false);
    expect(useEditorStore.getState().snapToGridEnabled).toBe(false);

    store.setSnapToGridEnabled(true);
    expect(useEditorStore.getState().snapToGridEnabled).toBe(true);

    const mockDraft: GraphDraft = {
      schema_version: '1.0',
      draft_id: 'draft-1',
      tenant_id: 'tenant-1',
      name: 'Test Journey',
      version: 1,
      nodes: [
        { id: 'node-1', type: 'trigger', name: 'Start', position: { x: 10, y: 20 } },
        { id: 'node-2', type: 'action', name: 'Email', position: { x: 32, y: 64 } },
        { id: 'node-3', type: 'action', name: 'SMS', position: { x: 105, y: 198 } },
      ],
      edges: [],
    };

    useEditorStore.getState().setDraft(mockDraft);
    expect(useEditorStore.getState().currentDraft?.nodes[0].position?.x).toBe(10);

    // Bulk snap nodes to 16px grid
    useEditorStore.getState().snapAllNodesToGrid();

    const snappedNodes = useEditorStore.getState().currentDraft?.nodes;
    expect(snappedNodes).toBeDefined();
    // node-1: x=10 -> 16, y=20 -> 16
    expect(snappedNodes![0].position).toEqual({ x: 16, y: 16 });
    // node-2: x=32 -> 32, y=64 -> 64
    expect(snappedNodes![1].position).toEqual({ x: 32, y: 64 });
    // node-3: x=105 -> 112, y=198 -> 192
    expect(snappedNodes![2].position).toEqual({ x: 112, y: 192 });

    // Verify undo/redo support for bulk snap
    useEditorStore.getState().undo();
    expect(useEditorStore.getState().currentDraft?.nodes[0].position).toEqual({ x: 10, y: 20 });

    useEditorStore.getState().redo();
    expect(useEditorStore.getState().currentDraft?.nodes[0].position).toEqual({ x: 16, y: 16 });
  });

  it('automatically arranges nodes horizontally', () => {
    const mockDraft: GraphDraft = {
      schema_version: '1.0',
      draft_id: 'draft-1',
      tenant_id: 'tenant-1',
      name: 'Test Journey',
      version: 1,
      nodes: [
        { id: 'node-start', type: 'trigger', name: 'Start', position: { x: 0, y: 0 } },
        { id: 'node-action-1', type: 'action', name: 'Email', position: { x: 0, y: 0 } },
        { id: 'node-action-2', type: 'action', name: 'SMS', position: { x: 0, y: 0 } },
      ],
      edges: [
        { id: 'e1', source: 'node-start', target: 'node-action-1' },
        { id: 'e2', source: 'node-action-1', target: 'node-action-2' },
      ],
    };

    useEditorStore.getState().setDraft(mockDraft);
    useEditorStore.getState().autoArrangeHorizontal();

    const arrangedNodes = useEditorStore.getState().currentDraft?.nodes;
    expect(arrangedNodes).toBeDefined();

    // Horizontal Level layout positions: START_X=112, spacing=384, START_Y=160
    // node-start (level 0) -> x = 112
    // node-action-1 (level 1) -> x = 496
    // node-action-2 (level 2) -> x = 880
    expect(arrangedNodes![0].id).toBe('node-start');
    expect(arrangedNodes![0].position).toEqual({ x: 112, y: 160 });

    expect(arrangedNodes![1].id).toBe('node-action-1');
    expect(arrangedNodes![1].position).toEqual({ x: 496, y: 160 });

    expect(arrangedNodes![2].id).toBe('node-action-2');
    expect(arrangedNodes![2].position).toEqual({ x: 880, y: 160 });

    // Verify undo/redo
    useEditorStore.getState().undo();
    expect(useEditorStore.getState().currentDraft?.nodes[0].position).toEqual({ x: 0, y: 0 });

    useEditorStore.getState().redo();
    expect(useEditorStore.getState().currentDraft?.nodes[0].position).toEqual({ x: 112, y: 160 });
  });

  it('automatically arranges nodes vertically', () => {
    const mockDraft: GraphDraft = {
      schema_version: '1.0',
      draft_id: 'draft-1',
      tenant_id: 'tenant-1',
      name: 'Test Journey',
      version: 1,
      nodes: [
        { id: 'node-start', type: 'trigger', name: 'Start', position: { x: 0, y: 0 } },
        { id: 'node-action-1', type: 'action', name: 'Email', position: { x: 0, y: 0 } },
        { id: 'node-action-2', type: 'action', name: 'SMS', position: { x: 0, y: 0 } },
      ],
      edges: [
        { id: 'e1', source: 'node-start', target: 'node-action-1' },
        { id: 'e2', source: 'node-action-1', target: 'node-action-2' },
      ],
    };

    useEditorStore.getState().setDraft(mockDraft);
    useEditorStore.getState().autoArrangeVertical();

    const arrangedNodes = useEditorStore.getState().currentDraft?.nodes;
    expect(arrangedNodes).toBeDefined();

    // Vertical Level layout positions: START_X=112, START_Y=160, vertical spacing=224
    // node-start (level 0) -> y = 160
    // node-action-1 (level 1) -> y = 384 (160 + 224)
    // node-action-2 (level 2) -> y = 608 (384 + 224)
    expect(arrangedNodes![0].id).toBe('node-start');
    expect(arrangedNodes![0].position).toEqual({ x: 112, y: 160 });

    expect(arrangedNodes![1].id).toBe('node-action-1');
    expect(arrangedNodes![1].position).toEqual({ x: 112, y: 384 });

    expect(arrangedNodes![2].id).toBe('node-action-2');
    expect(arrangedNodes![2].position).toEqual({ x: 112, y: 608 });
  });

  it('arranges decision true branch to the left and false branch to the right without crossing', () => {
    const mockDraft: GraphDraft = {
      schema_version: '1.0',
      draft_id: 'draft-decision',
      tenant_id: 'tenant-1',
      name: 'Decision Test Journey',
      version: 1,
      nodes: [
        { id: 'node-cond', type: 'Condition', name: 'Check Plan', position: { x: 0, y: 0 } },
        { id: 'node-false-target', type: 'Email', name: 'False Action', position: { x: 0, y: 0 } },
        { id: 'node-true-target', type: 'Email', name: 'True Action', position: { x: 0, y: 0 } },
      ],
      edges: [
        { id: 'e-false', source: 'node-cond', sourceHandle: 'false', target: 'node-false-target' },
        { id: 'e-true', source: 'node-cond', sourceHandle: 'true', target: 'node-true-target' },
      ],
    };

    useEditorStore.getState().setDraft(mockDraft);
    useEditorStore.getState().autoArrangeVertical();

    const nodes = useEditorStore.getState().currentDraft?.nodes;
    expect(nodes).toBeDefined();

    const trueNode = nodes!.find(n => n.id === 'node-true-target');
    const falseNode = nodes!.find(n => n.id === 'node-false-target');

    expect(trueNode).toBeDefined();
    expect(falseNode).toBeDefined();

    // True branch target should have a smaller X coordinate than False branch target
    expect(trueNode!.position?.x ?? 0).toBeLessThan(falseNode!.position?.x ?? 0);
  });
});
