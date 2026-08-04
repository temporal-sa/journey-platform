import { create } from 'zustand';
import type { GraphDraft, DraftVersionItem, ValidationResult, GraphNode, GraphEdge } from '../types/api';

export type { GraphNode, GraphEdge, GraphDraft };

export type EditorTab = 'canvas' | 'code' | 'simulation' | 'history' | 'settings';

export type ModalType =
  | 'publish'
  | 'save'
  | 'versionHistory'
  | 'simulationConfig'
  | 'confirmDiscard'
  | 'keyboardShortcuts'
  | 'staticListUpload'
  | 'testRunConfig'
  | null;

export interface EditorPresentationState {
  selectedNodeId: string | null;
  selectedEdgeId: string | null;
  selectedNodeIds: string[];
  selectedEdgeIds: string[];
  zoomLevel: number;
  panPosition: { x: number; y: number };
  activeTab: EditorTab;
  isSidebarOpen: boolean;
  isInspectorOpen: boolean;
  snapToGridEnabled: boolean;
}

export interface HistoryState {
  past: GraphDraft[];
  future: GraphDraft[];
  canUndo: boolean;
  canRedo: boolean;
}

export interface ServerDraftState {
  currentDraft: GraphDraft | null;
  draftRevisions: DraftVersionItem[];
  activeRevisionVersion: number | null;
  isDraftLoading: boolean;
  draftError: string | null;
  lastSavedHash: string | null;
}

export interface ValidationState {
  validationResult: ValidationResult | null;
  isValidating: boolean;
}

export interface ModalState {
  activeModal: ModalType;
  modalData: Record<string, unknown> | null;
}

export interface InspectorEditsState {
  unsavedInspectorEdits: Record<string, Record<string, unknown>>;
  hasUnsavedChanges: boolean;
}

export interface EditorActions {
  // Presentation Actions
  setSelectedNodeId: (id: string | null) => void;
  setSelectedNodeIds: (ids: string[]) => void;
  setSelectedEdgeId: (id: string | null) => void;
  setSelectedEdgeIds: (ids: string[]) => void;
  deleteElements: (nodeIds?: string[], edgeIds?: string[]) => void;
  setZoomLevel: (zoom: number) => void;
  setPanPosition: (pos: { x: number; y: number }) => void;
  setActiveTab: (tab: EditorTab) => void;
  toggleSidebar: () => void;
  toggleInspector: () => void;
  setSnapToGridEnabled: (enabled: boolean) => void;
  snapAllNodesToGrid: () => void;
  autoArrangeHorizontal: () => void;
  autoArrangeVertical: () => void;

  // Server Draft Actions
  setDraft: (draft: GraphDraft | null) => void;
  setDraftName: (name: string) => void;
  setDraftRevisions: (revisions: DraftVersionItem[]) => void;
  setActiveRevisionVersion: (version: number | null) => void;
  setDraftLoading: (loading: boolean) => void;
  setDraftError: (error: string | null) => void;
  setLastSavedHash: (hash: string | null) => void;
  markSaved: () => void;
  // Graph Editing Actions
  addNode: (node: GraphNode) => void;
  updateNodes: (nodes: GraphNode[], skipHistory?: boolean) => void;
  updateEdges: (edges: GraphEdge[]) => void;
  addEdge: (edge: GraphEdge) => void;
  // Undo / Redo Actions
  undo: () => void;
  redo: () => void;

  // Validation Actions
  setValidationResult: (result: ValidationResult | null) => void;
  setValidating: (validating: boolean) => void;

  // Modal Actions
  openModal: (modal: ModalType, data?: Record<string, unknown>) => void;
  closeModal: () => void;

  // Unsaved Inspector Edits Actions
  updateInspectorEdit: (nodeId: string, field: string, value: unknown) => void;
  clearInspectorEdits: (nodeId?: string) => void;
  commitInspectorEdits: (nodeId: string) => void;

  // Reset State
  resetStore: () => void;
}

export type EditorStore = EditorPresentationState &
  HistoryState &
  ServerDraftState &
  ValidationState &
  ModalState &
  InspectorEditsState &
  EditorActions;

const initialPresentationState: EditorPresentationState = {
  selectedNodeId: null,
  selectedEdgeId: null,
  selectedNodeIds: [],
  selectedEdgeIds: [],
  zoomLevel: 1.0,
  panPosition: { x: 0, y: 0 },
  activeTab: 'canvas',
  isSidebarOpen: true,
  isInspectorOpen: false,
  snapToGridEnabled: true,
};

const initialHistoryState: HistoryState = {
  past: [],
  future: [],
  canUndo: false,
  canRedo: false,
};



const initialDraftState: ServerDraftState = {
  currentDraft: null,
  draftRevisions: [],
  activeRevisionVersion: null,
  isDraftLoading: false,
  draftError: null,
  lastSavedHash: null,
};

const initialValidationState: ValidationState = {
  validationResult: null,
  isValidating: false,
};

const initialModalState: ModalState = {
  activeModal: null,
  modalData: null,
};

const initialInspectorEditsState: InspectorEditsState = {
  unsavedInspectorEdits: {},
  hasUnsavedChanges: false,
};

const MAX_HISTORY_LENGTH = 50;

function computeNodeLevels(nodes: GraphNode[], edges: GraphEdge[]) {
  const inDegree: Record<string, number> = {};
  const adjList: Record<string, string[]> = {};
  nodes.forEach(n => {
    inDegree[n.id] = 0;
    adjList[n.id] = [];
  });
  edges.forEach(e => {
    if (adjList[e.source]) {
      adjList[e.source].push(e.target);
    }
    if (inDegree[e.target] !== undefined) {
      inDegree[e.target]++;
    }
  });

  const levels: Record<string, number> = {};
  nodes.forEach(n => {
    levels[n.id] = -1;
  });

  let queue: string[] = [];

  nodes.forEach(n => {
    if (inDegree[n.id] === 0) {
      levels[n.id] = 0;
      queue.push(n.id);
    }
  });

  while (queue.length > 0) {
    const currId = queue.shift()!;
    const currLevel = levels[currId];
    const neighbors = adjList[currId] || [];
    for (const neighbor of neighbors) {
      if (levels[neighbor] < currLevel + 1) {
        levels[neighbor] = currLevel + 1;
        queue.push(neighbor);
      }
    }
  }

  let hasUnvisited = nodes.some(n => levels[n.id] === -1);
  while (hasUnvisited) {
    const firstUnvisited = nodes.find(n => levels[n.id] === -1);
    if (!firstUnvisited) break;

    levels[firstUnvisited.id] = 0;
    queue.push(firstUnvisited.id);

    while (queue.length > 0) {
      const currId = queue.shift()!;
      const currLevel = levels[currId];
      const neighbors = adjList[currId] || [];
      for (const neighbor of neighbors) {
        if (levels[neighbor] < currLevel + 1) {
          levels[neighbor] = currLevel + 1;
          queue.push(neighbor);
        }
      }
    }
    hasUnvisited = nodes.some(n => levels[n.id] === -1);
  }

  return levels;
}

export const useEditorStore = create<EditorStore>()((set, get) => ({
  ...initialPresentationState,
  ...initialHistoryState,
  ...initialDraftState,
  ...initialValidationState,
  ...initialModalState,
  ...initialInspectorEditsState,

  // Presentation Actions
  setSelectedNodeId: (id) => {
    const state = get();
    if (state.selectedNodeId === id && state.selectedNodeIds.length === (id ? 1 : 0) && (id ? state.selectedNodeIds[0] === id : true)) return;
    set({
      selectedNodeId: id,
      selectedNodeIds: id ? [id] : [],
      selectedEdgeId: id ? null : state.selectedEdgeId,
      selectedEdgeIds: id ? [] : state.selectedEdgeIds,
      isInspectorOpen: id !== null,
    });
  },

  setSelectedNodeIds: (ids) => {
    const state = get();
    const prev = state.selectedNodeIds;
    if (prev.length === ids.length && prev.every((id, idx) => id === ids[idx])) return;
    set({
      selectedNodeIds: ids,
      selectedNodeId: ids.length > 0 ? ids[ids.length - 1] : null,
      selectedEdgeId: ids.length > 0 ? null : state.selectedEdgeId,
      selectedEdgeIds: ids.length > 0 ? [] : state.selectedEdgeIds,
      isInspectorOpen: ids.length === 1,
    });
  },

  setSelectedEdgeId: (id) => {
    const state = get();
    if (state.selectedEdgeId === id && state.selectedEdgeIds.length === (id ? 1 : 0) && (id ? state.selectedEdgeIds[0] === id : true)) return;
    set({
      selectedEdgeId: id,
      selectedEdgeIds: id ? [id] : [],
      selectedNodeId: id ? null : state.selectedNodeId,
      selectedNodeIds: id ? [] : state.selectedNodeIds,
    });
  },

  setSelectedEdgeIds: (ids) => {
    const state = get();
    const prev = state.selectedEdgeIds;
    if (prev.length === ids.length && prev.every((id, idx) => id === ids[idx])) return;
    set({
      selectedEdgeIds: ids,
      selectedEdgeId: ids.length > 0 ? ids[ids.length - 1] : null,
      selectedNodeId: ids.length > 0 ? null : state.selectedNodeId,
      selectedNodeIds: ids.length > 0 ? [] : state.selectedNodeIds,
    });
  },

  setZoomLevel: (zoomLevel) => set({ zoomLevel: Math.max(0.1, Math.min(zoomLevel, 5.0)) }),

  setPanPosition: (panPosition) => set({ panPosition }),

  setActiveTab: (activeTab) => set({ activeTab }),

  toggleSidebar: () => set((state) => ({ isSidebarOpen: !state.isSidebarOpen })),

  toggleInspector: () => set((state) => ({ isInspectorOpen: !state.isInspectorOpen })),

  setSnapToGridEnabled: (snapToGridEnabled) => set({ snapToGridEnabled }),

  snapAllNodesToGrid: () => {
    const { currentDraft, past } = get();
    if (!currentDraft || !currentDraft.nodes) return;

    const GRID_SIZE = 16;
    const snappedNodes = currentDraft.nodes.map((node) => {
      const x = node.position?.x ?? 0;
      const y = node.position?.y ?? 0;
      return {
        ...node,
        position: {
          x: Math.round(x / GRID_SIZE) * GRID_SIZE,
          y: Math.round(y / GRID_SIZE) * GRID_SIZE,
        },
      };
    });

    const anyChanged = snappedNodes.some((node, i) => {
      const original = currentDraft.nodes[i];
      const origX = original.position?.x ?? 0;
      const origY = original.position?.y ?? 0;
      return node.position.x !== origX || node.position.y !== origY;
    });

    if (!anyChanged) return;

    const newPast = [...past, currentDraft].slice(-MAX_HISTORY_LENGTH);
    set({
      currentDraft: {
        ...currentDraft,
        nodes: snappedNodes,
      },
      past: newPast,
      future: [],
      canUndo: true,
      canRedo: false,
      hasUnsavedChanges: true,
    });
  },

  autoArrangeHorizontal: () => {
    const { currentDraft, past } = get();
    if (!currentDraft || !currentDraft.nodes) return;

    const nodes = currentDraft.nodes;
    const edges = currentDraft.edges || [];
    const levels = computeNodeLevels(nodes, edges);

    const nodesByLevel: Record<number, string[]> = {};
    nodes.forEach(n => {
      const lvl = levels[n.id];
      if (!nodesByLevel[lvl]) {
        nodesByLevel[lvl] = [];
      }
      nodesByLevel[lvl].push(n.id);
    });

    const HORIZONTAL_SPACING = 384;
    const VERTICAL_SPACING = 224;
    const START_X = 112;
    const START_Y = 160;

    const arrangedNodes = nodes.map(node => {
      const lvl = levels[node.id];
      const levelNodes = nodesByLevel[lvl] || [];
      const indexInLevel = levelNodes.indexOf(node.id);

      const x = START_X + lvl * HORIZONTAL_SPACING;
      const y = START_Y + indexInLevel * VERTICAL_SPACING;

      return {
        ...node,
        position: { x, y }
      };
    });

    const anyChanged = arrangedNodes.some((node, i) => {
      const original = currentDraft.nodes[i];
      const origX = original.position?.x ?? 0;
      const origY = original.position?.y ?? 0;
      return node.position.x !== origX || node.position.y !== origY;
    });

    if (!anyChanged) return;

    const newPast = [...past, currentDraft].slice(-MAX_HISTORY_LENGTH);
    set({
      currentDraft: {
        ...currentDraft,
        nodes: arrangedNodes,
      },
      past: newPast,
      future: [],
      canUndo: true,
      canRedo: false,
      hasUnsavedChanges: true,
    });
  },

  autoArrangeVertical: () => {
    const { currentDraft, past } = get();
    if (!currentDraft || !currentDraft.nodes) return;

    const nodes = currentDraft.nodes;
    const edges = currentDraft.edges || [];
    const levels = computeNodeLevels(nodes, edges);

    const nodesByLevel: Record<number, string[]> = {};
    nodes.forEach(n => {
      const lvl = levels[n.id];
      if (!nodesByLevel[lvl]) {
        nodesByLevel[lvl] = [];
      }
      nodesByLevel[lvl].push(n.id);
    });

    const HORIZONTAL_SPACING = 384;
    const VERTICAL_SPACING = 224;
    const START_X = 112;
    const START_Y = 160;

    const arrangedNodes = nodes.map(node => {
      const lvl = levels[node.id];
      const levelNodes = nodesByLevel[lvl] || [];
      const indexInLevel = levelNodes.indexOf(node.id);

      const x = START_X + indexInLevel * HORIZONTAL_SPACING;
      const y = START_Y + lvl * VERTICAL_SPACING;

      return {
        ...node,
        position: { x, y }
      };
    });

    const anyChanged = arrangedNodes.some((node, i) => {
      const original = currentDraft.nodes[i];
      const origX = original.position?.x ?? 0;
      const origY = original.position?.y ?? 0;
      return node.position.x !== origX || node.position.y !== origY;
    });

    if (!anyChanged) return;

    const newPast = [...past, currentDraft].slice(-MAX_HISTORY_LENGTH);
    set({
      currentDraft: {
        ...currentDraft,
        nodes: arrangedNodes,
      },
      past: newPast,
      future: [],
      canUndo: true,
      canRedo: false,
      hasUnsavedChanges: true,
    });
  },

  // Server Draft Actions
  setDraft: (currentDraft) =>
    set({
      currentDraft,
      lastSavedHash: currentDraft?.content_hash || null,
      activeRevisionVersion: currentDraft?.version || null,
      hasUnsavedChanges: false,
      past: [],
      future: [],
      canUndo: false,
      canRedo: false,
    }),

  setDraftRevisions: (draftRevisions) => set({ draftRevisions }),

  setActiveRevisionVersion: (activeRevisionVersion) => set({ activeRevisionVersion }),

  setDraftLoading: (isDraftLoading) => set({ isDraftLoading }),

  setDraftError: (draftError) => set({ draftError }),

  setLastSavedHash: (lastSavedHash) => set({ lastSavedHash }),

  markSaved: () => set({ hasUnsavedChanges: false }),

  setDraftName: (name) => {
    const { currentDraft, past } = get();
    if (!currentDraft || currentDraft.name === name) return;

    const newPast = [...past, currentDraft].slice(-MAX_HISTORY_LENGTH);
    const updatedDraft: GraphDraft = {
      ...currentDraft,
      name,
    };

    set({
      currentDraft: updatedDraft,
      past: newPast,
      future: [],
      canUndo: true,
      canRedo: false,
      hasUnsavedChanges: true,
    });
  },
  // Graph Editing Actions
  addNode: (node) => {
    const { currentDraft, past } = get();
    if (!currentDraft) {
      const newDraft: GraphDraft = {
        schema_version: '1.0',
        draft_id: 'draft-new',
        tenant_id: 'tenant-default',
        name: 'New Journey',
        version: 1,
        nodes: [node],
        edges: [],
      };
      set({
        currentDraft: newDraft,
        past: [],
        future: [],
        canUndo: false,
        canRedo: false,
        hasUnsavedChanges: true,
      });
      return;
    }

    const newPast = [...past, currentDraft].slice(-MAX_HISTORY_LENGTH);
    const updatedDraft: GraphDraft = {
      ...currentDraft,
      nodes: [...currentDraft.nodes, node],
    };

    set({
      currentDraft: updatedDraft,
      past: newPast,
      future: [],
      canUndo: true,
      canRedo: false,
      hasUnsavedChanges: true,
    });
  },

  updateNodes: (nodes, skipHistory = false) => {
    const { currentDraft, past } = get();
    if (!currentDraft) return;

    if (skipHistory) {
      set({
        currentDraft: {
          ...currentDraft,
          nodes,
        },
        hasUnsavedChanges: true,
      });
      return;
    }

    const newPast = [...past, currentDraft].slice(-MAX_HISTORY_LENGTH);
    const updatedDraft: GraphDraft = {
      ...currentDraft,
      nodes,
    };

    set({
      currentDraft: updatedDraft,
      past: newPast,
      future: [],
      canUndo: true,
      canRedo: false,
      hasUnsavedChanges: true,
    });
  },

  updateEdges: (edges) => {
    const { currentDraft, past } = get();
    if (!currentDraft) return;

    const newPast = [...past, currentDraft].slice(-MAX_HISTORY_LENGTH);
    const updatedDraft: GraphDraft = {
      ...currentDraft,
      edges,
    };

    set({
      currentDraft: updatedDraft,
      past: newPast,
      future: [],
      canUndo: true,
      canRedo: false,
      hasUnsavedChanges: true,
    });
  },

  addEdge: (edge) => {
    const { currentDraft, past } = get();
    if (!currentDraft) return;

    const filteredEdges = currentDraft.edges.filter(
      (e) => e.id !== edge.id && !(e.source === edge.source && e.target === edge.target)
    );

    const newPast = [...past, currentDraft].slice(-MAX_HISTORY_LENGTH);
    const updatedDraft: GraphDraft = {
      ...currentDraft,
      edges: [...filteredEdges, edge],
    };

    set({
      currentDraft: updatedDraft,
      past: newPast,
      future: [],
      canUndo: true,
      canRedo: false,
      hasUnsavedChanges: true,
    });
  },

  deleteElements: (nodeIds, edgeIds) => {
    const { currentDraft, past, selectedNodeId, selectedEdgeId, selectedNodeIds, selectedEdgeIds } = get();
    if (!currentDraft) return;

    const targetNodeIds =
      nodeIds && nodeIds.length > 0
        ? nodeIds
        : selectedNodeIds.length > 0
        ? selectedNodeIds
        : selectedNodeId
        ? [selectedNodeId]
        : [];
    const targetEdgeIds =
      edgeIds && edgeIds.length > 0
        ? edgeIds
        : selectedEdgeIds.length > 0
        ? selectedEdgeIds
        : selectedEdgeId
        ? [selectedEdgeId]
        : [];

    if (targetNodeIds.length === 0 && targetEdgeIds.length === 0) return;

    const nodeSet = new Set(targetNodeIds);
    const edgeSet = new Set(targetEdgeIds);

    const updatedNodes = currentDraft.nodes.filter((n) => !nodeSet.has(n.id));
    const updatedEdges = currentDraft.edges.filter(
      (e) => !edgeSet.has(e.id) && !nodeSet.has(e.source) && !nodeSet.has(e.target)
    );

    const newPast = [...past, currentDraft].slice(-MAX_HISTORY_LENGTH);
    const updatedDraft: GraphDraft = {
      ...currentDraft,
      nodes: updatedNodes,
      edges: updatedEdges,
    };

    const remainingNodeIds = selectedNodeIds.filter((id) => !nodeSet.has(id));
    const remainingEdgeIds = selectedEdgeIds.filter((id) => !edgeSet.has(id));

    set({
      currentDraft: updatedDraft,
      past: newPast,
      future: [],
      canUndo: true,
      canRedo: false,
      hasUnsavedChanges: true,
      selectedNodeIds: remainingNodeIds,
      selectedEdgeIds: remainingEdgeIds,
      selectedNodeId: remainingNodeIds.length > 0 ? remainingNodeIds[remainingNodeIds.length - 1] : null,
      selectedEdgeId: remainingEdgeIds.length > 0 ? remainingEdgeIds[remainingEdgeIds.length - 1] : null,
      isInspectorOpen: remainingNodeIds.length === 1,
    });
  },

  // Undo / Redo Actions
  undo: () => {
    const { currentDraft, past, future } = get();
    if (!currentDraft || past.length === 0) return;

    const previousDraft = past[past.length - 1];
    const newPast = past.slice(0, past.length - 1);
    const newFuture = [currentDraft, ...future];

    set({
      currentDraft: previousDraft,
      past: newPast,
      future: newFuture,
      canUndo: newPast.length > 0,
      canRedo: true,
      hasUnsavedChanges: true,
    });
  },

  redo: () => {
    const { currentDraft, past, future } = get();
    if (!currentDraft || future.length === 0) return;

    const nextDraft = future[0];
    const newFuture = future.slice(1);
    const newPast = [...past, currentDraft];

    set({
      currentDraft: nextDraft,
      past: newPast,
      future: newFuture,
      canUndo: true,
      canRedo: newFuture.length > 0,
      hasUnsavedChanges: true,
    });
  },

  // Validation Actions
  setValidationResult: (validationResult) => set({ validationResult }),

  setValidating: (isValidating) => set({ isValidating }),

  // Modal Actions
  openModal: (activeModal, modalData) => set({ activeModal, modalData: modalData ?? null }),

  closeModal: () => set({ activeModal: null, modalData: null }),

  // Unsaved Inspector Edits Actions
  updateInspectorEdit: (nodeId, field, value) => {
    const state = get();
    const nodeEdits = { ...(state.unsavedInspectorEdits[nodeId] || {}), [field]: value };
    const updatedEdits = { ...state.unsavedInspectorEdits, [nodeId]: nodeEdits };
    const hasUnsavedChanges = Object.keys(updatedEdits).some(
      (key) => Object.keys(updatedEdits[key]).length > 0
    );
    set({
      unsavedInspectorEdits: updatedEdits,
      hasUnsavedChanges,
    });
  },

  clearInspectorEdits: (nodeId) => {
    const state = get();
    if (!nodeId) {
      set({ unsavedInspectorEdits: {}, hasUnsavedChanges: false });
      return;
    }
    const updatedEdits = { ...state.unsavedInspectorEdits };
    delete updatedEdits[nodeId];
    const hasUnsavedChanges = Object.keys(updatedEdits).some(
      (key) => Object.keys(updatedEdits[key]).length > 0
    );
    set({ unsavedInspectorEdits: updatedEdits, hasUnsavedChanges });
  },

  commitInspectorEdits: (nodeId) => {
    const state = get();
    const edits = state.unsavedInspectorEdits[nodeId];
    if (!edits || !state.currentDraft) return;

    const updatedNodes = state.currentDraft.nodes.map((node) => {
      if (node.id === nodeId) {
        return {
          ...node,
          config: { ...(node.config || {}), ...edits },
        };
      }
      return node;
    });

    const updatedDraft: GraphDraft = {
      ...state.currentDraft,
      nodes: updatedNodes,
    };

    get().setDraft(updatedDraft);
    get().clearInspectorEdits(nodeId);
  },

  // Reset Store
  resetStore: () =>
    set({
      ...initialPresentationState,
      ...initialHistoryState,
      ...initialDraftState,
      ...initialValidationState,
      ...initialModalState,
      ...initialInspectorEditsState,
    }),
}));

if (typeof window !== 'undefined') {
  (window as any).useEditorStore = useEditorStore;
}
