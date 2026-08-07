import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ExecutionGraphView, ExecutionEdge, NodeVisitStep } from './ExecutionGraphView';
import { Position, ReactFlowProvider } from '@xyflow/react';
import { GraphNode, GraphEdge } from '../../types/api';

// Mock ResizeObserver for happy-dom/jsdom environment used by ReactFlow
class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}
global.ResizeObserver = ResizeObserverMock as unknown as typeof ResizeObserver;

describe('ExecutionGraphView Component', () => {
  const sampleNodes: GraphNode[] = [
    {
      id: 'node-start',
      type: 'EventStart',
      name: 'User Signup Trigger',
      position: { x: 0, y: 0 },
    },
    {
      id: 'node-cond',
      type: 'Condition',
      name: 'Is Premium User?',
      position: { x: 0, y: 150 },
    },
    {
      id: 'node-[#34d399]-action',
      type: 'Email',
      name: 'Send Welcome Email',
      position: { x: -150, y: 300 },
    },
    {
      id: 'node-unvisited-action',
      type: 'SMS',
      name: 'Send SMS Alert',
      position: { x: 150, y: 300 },
    },
  ];

  const sampleEdges: GraphEdge[] = [
    {
      id: 'e1',
      source: 'node-start',
      target: 'node-cond',
    },
    {
      id: 'e2',
      source: 'node-cond',
      target: 'node-[#34d399]-action',
      condition: 'True',
    },
    {
      id: 'e3',
      source: 'node-cond',
      target: 'node-unvisited-action',
      condition: 'False',
    },
  ];

  const sampleVisitSteps: NodeVisitStep[] = [
    {
      stepIndex: 1,
      nodeId: 'node-start',
      nodeName: 'User Signup Trigger',
      nodeType: 'EventStart',
      status: 'passed',
    },
    {
      stepIndex: 2,
      nodeId: 'node-cond',
      nodeName: 'Is Premium User?',
      nodeType: 'Condition',
      status: 'passed',
    },
    {
      stepIndex: 3,
      nodeId: 'node-[#34d399]-action',
      nodeName: 'Send Welcome Email',
      nodeType: 'Email',
      status: 'passed',
    },
  ];

  it('renders the graph container with Obsidian theme background', () => {
    render(
      <ExecutionGraphView
        nodes={sampleNodes}
        edges={sampleEdges}
        visitSteps={sampleVisitSteps}
        height="500px"
      />
    );

    const container = screen.getByTestId('execution-graph-view');
    expect(container).toBeInTheDocument();
    expect(container).toHaveStyle({ height: '500px' });
  });

  it('calculates node visited states correctly with step badges', () => {
    render(
      <ExecutionGraphView
        nodes={sampleNodes}
        edges={sampleEdges}
        visitSteps={sampleVisitSteps}
      />
    );

    // Node 1: Visited, Step 1
    const nodeStart = screen.getByTestId('execution-node-node-start');
    expect(nodeStart).toHaveAttribute('data-visited', 'true');
    expect(screen.getByTestId('node-badge-step-node-start')).toHaveTextContent('Step 1');

    // Node 2: Visited, Step 2
    const nodeCond = screen.getByTestId('execution-node-node-cond');
    expect(nodeCond).toHaveAttribute('data-visited', 'true');
    expect(screen.getByTestId('node-badge-step-node-cond')).toHaveTextContent('Step 2');

    // Unvisited node
    const nodeUnvisited = screen.getByTestId('execution-node-node-unvisited-action');
    expect(nodeUnvisited).toHaveAttribute('data-visited', 'false');
    expect(nodeUnvisited).toHaveAttribute('data-active', 'false');
    expect(nodeUnvisited).toHaveClass('opacity-45');
  });

  it('calculates active currentNodeId state with active badge and pulsing indicator', () => {
    render(
      <ExecutionGraphView
        nodes={sampleNodes}
        edges={sampleEdges}
        visitSteps={sampleVisitSteps}
        currentNodeId="node-cond"
      />
    );

    const activeNode = screen.getByTestId('execution-node-node-cond');
    expect(activeNode).toHaveAttribute('data-active', 'true');
    expect(screen.getByTestId('node-badge-active-node-cond')).toHaveTextContent('Active');
  });

  it('infers active node when status is running and currentNodeId is not specified', () => {
    render(
      <ExecutionGraphView
        nodes={sampleNodes}
        edges={sampleEdges}
        visitSteps={sampleVisitSteps}
        status="running"
      />
    );

    // The last step in visitSteps is node-[#34d399]-action
    const lastVisitedNode = screen.getByTestId('execution-node-node-[#34d399]-action');
    expect(lastVisitedNode).toHaveAttribute('data-active', 'true');
  });

  it('renders traversed edge with cyan stroke and unvisited edge with muted stroke when rendered with ExecutionEdge', () => {
    const traversedProps = {
      id: 'e-traversed',
      sourceX: 0,
      sourceY: 100,
      targetX: 100,
      targetY: 200,
      sourcePosition: Position.Bottom,
      targetPosition: Position.Top,
      label: 'True',
      data: { isTraversed: true },
    } as unknown as Parameters<typeof ExecutionEdge>[0];

    const unvisitedProps = {
      id: 'e-unvisited',
      sourceX: 0,
      sourceY: 100,
      targetX: 100,
      targetY: 200,
      sourcePosition: Position.Bottom,
      targetPosition: Position.Top,
      label: 'False',
      data: { isTraversed: false },
    } as unknown as Parameters<typeof ExecutionEdge>[0];

    const { container } = render(
      <ReactFlowProvider>
        <svg>
          <ExecutionEdge {...traversedProps} />
          <ExecutionEdge {...unvisitedProps} />
        </svg>
      </ReactFlowProvider>
    );

    const pathTraversed = container.querySelector('#e-traversed');
    expect(pathTraversed).toBeInTheDocument();
    expect(pathTraversed).toHaveStyle({ stroke: '#4cd7f6' });

    const pathUnvisited = container.querySelector('#e-unvisited');
    expect(pathUnvisited).toBeInTheDocument();
    expect(pathUnvisited).toHaveStyle({ stroke: '#464554' });
  });
});
