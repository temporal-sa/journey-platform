import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ReactFlowProvider, Position } from '@xyflow/react';

vi.mock('@xyflow/react', async () => {
  const actual = await vi.importActual<typeof import('@xyflow/react')>('@xyflow/react');
  return {
    ...actual,
    EdgeLabelRenderer: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  };
});

import { useEditorStore } from '../../stores/editorStore';
import { Palette, PALETTE_ITEMS } from './Palette';
import { CanvasToolbar } from './CanvasToolbar';
import { validateConnection, wouldCreateCycle } from './CanvasValidation';
import {
  EventStartNode,
  ConditionNode,
  DelayNode,
  WaitForEventNode,
  ExperimentNode,
  EmailNode,
  SMSNode,
  PushNode,
  InAppNode,
  WebhookNode,
  ExitNode,
} from './nodes';
import { LabeledEdge } from './edges/LabeledEdge';
import { Canvas } from './Canvas';

describe('Canvas Editor Components', () => {
  beforeEach(() => {
    useEditorStore.getState().resetStore();
  });

  describe('Palette Drawer Component', () => {
    it('renders all palette categories and items', () => {
      render(<Palette />);
      expect(screen.getByTestId('palette-drawer')).toBeInTheDocument();
      expect(screen.getByText(/Triggers \(1\)/i)).toBeInTheDocument();
      expect(screen.getByText(/Decisions \(4\)/i)).toBeInTheDocument();
      expect(screen.getByText(/Actions \(5\)/i)).toBeInTheDocument();
      expect(screen.getByText(/Utilities \(1\)/i)).toBeInTheDocument();

      PALETTE_ITEMS.forEach((item) => {
        expect(screen.getByTestId(`palette-item-${item.type}`)).toBeInTheDocument();
      });
    });

    it('triggers onAddNode callback when a palette item is clicked', () => {
      const handleAddNode = vi.fn();
      render(<Palette onAddNode={handleAddNode} />);
      const eventStartItem = screen.getByTestId('palette-item-EventStart');
      fireEvent.click(eventStartItem);
      expect(handleAddNode).toHaveBeenCalledWith('EventStart', 'Event Start');
    });
  });

  describe('Custom React Flow Node Components', () => {
    /* eslint-disable @typescript-eslint/no-explicit-any */
    const mockNodeProps: any = {
      id: 'test-node-1',
      type: 'test',
      selected: false,
      zIndex: 1,
      isConnectable: true,
      xPos: 0,
      yPos: 0,
      dragging: false,
      selectable: true,
      deletable: true,
      draggable: true,
    };

    it('renders EventStartNode (Trigger)', () => {
      render(
        <ReactFlowProvider>
          <EventStartNode {...mockNodeProps} data={{ name: 'Order Placed' }} />
        </ReactFlowProvider>
      );
      expect(screen.getByText('Order Placed')).toBeInTheDocument();
      expect(screen.getByText(/Trigger/i)).toBeInTheDocument();
    });

    it('renders ConditionNode (Decision)', () => {
      render(
        <ReactFlowProvider>
          <ConditionNode {...mockNodeProps} data={{ name: 'Is VIP User' }} />
        </ReactFlowProvider>
      );
      expect(screen.getByText('Is VIP User')).toBeInTheDocument();
      expect(screen.getByText('True')).toBeInTheDocument();
      expect(screen.getByText('False')).toBeInTheDocument();
    });

    it('renders DelayNode (Decision)', () => {
      render(
        <ReactFlowProvider>
          <DelayNode {...mockNodeProps} data={{ name: 'Wait 24h', config: { duration: '24h' } }} />
        </ReactFlowProvider>
      );
      expect(screen.getByText('Wait 24h')).toBeInTheDocument();
      expect(screen.getByText('Wait: 24h')).toBeInTheDocument();
    });

    it('renders WaitForEventNode (Decision)', () => {
      render(
        <ReactFlowProvider>
          <WaitForEventNode {...mockNodeProps} data={{ name: 'Wait For Click' }} />
        </ReactFlowProvider>
      );
      expect(screen.getByText('Wait For Click')).toBeInTheDocument();
      expect(screen.getByText('Event')).toBeInTheDocument();
      expect(screen.getByText('Timeout')).toBeInTheDocument();
    });

    it('renders ExperimentNode (Decision)', () => {
      render(
        <ReactFlowProvider>
          <ExperimentNode {...mockNodeProps} data={{ name: 'A/B Test Email' }} />
        </ReactFlowProvider>
      );
      expect(screen.getByText('A/B Test Email')).toBeInTheDocument();
      expect(screen.getByText('Variant A')).toBeInTheDocument();
      expect(screen.getByText('Variant B')).toBeInTheDocument();
    });

    it('renders Action nodes (Email, SMS, Push, InApp, Webhook)', () => {
      const { rerender } = render(
        <ReactFlowProvider>
          <EmailNode {...mockNodeProps} data={{ name: 'Send Welcome Email' }} />
        </ReactFlowProvider>
      );
      expect(screen.getByText('Send Welcome Email')).toBeInTheDocument();

      rerender(
        <ReactFlowProvider>
          <SMSNode {...mockNodeProps} data={{ name: 'Send Verification SMS' }} />
        </ReactFlowProvider>
      );
      expect(screen.getByText('Send Verification SMS')).toBeInTheDocument();

      rerender(
        <ReactFlowProvider>
          <PushNode {...mockNodeProps} data={{ name: 'Send Push Alert' }} />
        </ReactFlowProvider>
      );
      expect(screen.getByText('Send Push Alert')).toBeInTheDocument();

      rerender(
        <ReactFlowProvider>
          <InAppNode {...mockNodeProps} data={{ name: 'Show Banner' }} />
        </ReactFlowProvider>
      );
      expect(screen.getByText('Show Banner')).toBeInTheDocument();

      rerender(
        <ReactFlowProvider>
          <WebhookNode {...mockNodeProps} data={{ name: 'Sync CRM Webhook' }} />
        </ReactFlowProvider>
      );
      expect(screen.getByText('Sync CRM Webhook')).toBeInTheDocument();
    });

    it('renders ExitNode (Utility)', () => {
      render(
        <ReactFlowProvider>
          <ExitNode {...mockNodeProps} data={{ name: 'Complete Journey' }} />
        </ReactFlowProvider>
      );
      expect(screen.getByText('Complete Journey')).toBeInTheDocument();
      expect(screen.getByText(/Utility/i)).toBeInTheDocument();
    });
  });

  describe('Custom LabeledEdge Component', () => {
    it('renders labeled edge with display label', () => {
      /* eslint-disable @typescript-eslint/no-explicit-any */
      const mockEdgeProps: any = {
        id: 'edge-1-2',
        source: 'node-1',
        target: 'node-2',
        sourceX: 100,
        sourceY: 100,
        targetX: 200,
        targetY: 200,
        sourcePosition: Position.Bottom,
        targetPosition: Position.Top,
        label: 'True Branch',
      };

      render(
        <ReactFlowProvider>
          <svg>
            <LabeledEdge {...mockEdgeProps} />
          </svg>
        </ReactFlowProvider>
      );

      expect(screen.getByTestId('edge-label-edge-1-2')).toBeInTheDocument();
      expect(screen.getByText('True Branch')).toBeInTheDocument();
    });
  });

  describe('CanvasToolbar Component', () => {
    it('renders toolbar buttons and handles clicks', () => {
      const handleUndo = vi.fn();
      const handleRedo = vi.fn();
      const handleDelete = vi.fn();
      const handleTogglePan = vi.fn();

      render(
        <CanvasToolbar
          onUndo={handleUndo}
          onRedo={handleRedo}
          canUndo={true}
          canRedo={false}
          onZoomIn={() => {}}
          onZoomOut={() => {}}
          onFitView={() => {}}
          onDeleteSelected={handleDelete}
          hasSelection={true}
          isPanMode={false}
          onTogglePanMode={handleTogglePan}
        />
      );

      expect(screen.getByTestId('canvas-toolbar')).toBeInTheDocument();

      const undoBtn = screen.getByTestId('toolbar-undo');
      expect(undoBtn).not.toBeDisabled();
      fireEvent.click(undoBtn);
      expect(handleUndo).toHaveBeenCalledTimes(1);

      const redoBtn = screen.getByTestId('toolbar-redo');
      expect(redoBtn).toBeDisabled();

      const deleteBtn = screen.getByTestId('toolbar-delete');
      expect(deleteBtn).not.toBeDisabled();
      fireEvent.click(deleteBtn);
      expect(handleDelete).toHaveBeenCalledTimes(1);

      const panBtn = screen.getByTestId('toolbar-pan-mode');
      fireEvent.click(panBtn);
      expect(handleTogglePan).toHaveBeenCalledTimes(1);
    });
  });

  describe('Canvas Validation Rules & Cycle Prevention', () => {
    const nodes = [
      { id: 'start-1', type: 'EventStart', name: 'Start' },
      { id: 'condition-1', type: 'Condition', name: 'Condition 1' },
      { id: 'email-1', type: 'Email', name: 'Email Action' },
      { id: 'exit-1', type: 'Exit', name: 'Exit' },
    ];

    it('detects self-loop cycles', () => {
      expect(wouldCreateCycle(nodes, [], { source: 'condition-1', target: 'condition-1' })).toBe(true);
    });

    it('detects multi-step directed cycles', () => {
      const edges = [
        { id: 'e1', source: 'condition-1', target: 'email-1' },
      ];

      // Connecting email-1 back to condition-1 creates cycle
      expect(wouldCreateCycle(nodes, edges, { source: 'email-1', target: 'condition-1' })).toBe(true);
    });

    it('prevents connections to Trigger nodes', () => {
      const result = validateConnection(
        { source: 'condition-1', target: 'start-1' },
        nodes,
        []
      );
      expect(result.isValid).toBe(false);
      expect(result.reason).toContain('Trigger nodes cannot have incoming connections');
    });

    it('prevents outgoing connections from Exit nodes', () => {
      const result = validateConnection(
        { source: 'exit-1', target: 'email-1' },
        nodes,
        []
      );
      expect(result.isValid).toBe(false);
      expect(result.reason).toContain('Exit nodes cannot have outgoing connections');
    });

    it('prevents duplicate outgoing connections on single-output nodes', () => {
      const edges = [{ id: 'e1', source: 'email-1', target: 'exit-1' }];
      const result = validateConnection(
        { source: 'email-1', target: 'condition-1' },
        nodes,
        edges
      );
      expect(result.isValid).toBe(false);
      expect(result.reason).toContain('Node already has an outgoing connection');
    });

    it('allows separate true and false outgoing branches on Condition nodes', () => {
      const edges = [{ id: 'e1', source: 'condition-1', target: 'email-1', sourceHandle: 'true' }];

      // Connecting false handle should be valid
      const resultFalse = validateConnection(
        { source: 'condition-1', target: 'exit-1', sourceHandle: 'false' },
        nodes,
        edges
      );
      expect(resultFalse.isValid).toBe(true);

      // Connecting true handle again should be invalid
      const resultTrueDuplicate = validateConnection(
        { source: 'condition-1', target: 'exit-1', sourceHandle: 'true' },
        nodes,
        edges
      );
      expect(resultTrueDuplicate.isValid).toBe(false);
      expect(resultTrueDuplicate.reason).toContain("Branch 'true' is already connected");
    });
  });

  describe('Canvas Main Component Integration', () => {
    it('renders Canvas with Palette and ReactFlow workspace', () => {
      useEditorStore.getState().setDraft({
        schema_version: '1.0',
        draft_id: 'draft-test-1',
        tenant_id: 'tenant-1',
        name: 'Test Journey',
        version: 1,
        nodes: [{ id: 'node-start', type: 'EventStart', name: 'Start Event' }],
        edges: [],
      });

      render(<Canvas />);
      expect(screen.getByTestId('canvas-editor-container')).toBeInTheDocument();
      expect(screen.getByTestId('palette-drawer')).toBeInTheDocument();
      expect(screen.getByTestId('canvas-toolbar')).toBeInTheDocument();
    });

    it('adds node to store when clicking a palette item', () => {
      useEditorStore.getState().setDraft({
        schema_version: '1.0',
        draft_id: 'draft-test-2',
        tenant_id: 'tenant-1',
        name: 'Test Journey',
        version: 1,
        nodes: [],
        edges: [],
      });

      render(<Canvas />);
      const conditionItem = screen.getByTestId('palette-item-Condition');
      fireEvent.click(conditionItem);

      const draft = useEditorStore.getState().currentDraft;
      expect(draft?.nodes).toHaveLength(1);
      expect(draft?.nodes[0].type).toBe('Condition');
    });
  });
});
