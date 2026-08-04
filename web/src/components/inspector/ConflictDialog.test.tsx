import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ConflictDialog } from './ConflictDialog';
import type { GraphDraft } from '../../types/api';

describe('ConflictDialog Component', () => {
  const localDraft: GraphDraft = {
    schema_version: '1.0',
    draft_id: 'draft-1',
    tenant_id: 'tenant-1',
    name: 'Local Modified Journey',
    version: 1,
    nodes: [{ id: 'node-1', type: 'trigger', name: 'Local Start' }],
    edges: [],
    content_hash: 'etag-local-123',
  };

  const serverDraft: GraphDraft = {
    schema_version: '1.0',
    draft_id: 'draft-1',
    tenant_id: 'tenant-1',
    name: 'Server Modified Journey',
    version: 2,
    nodes: [
      { id: 'node-1', type: 'trigger', name: 'Server Start' },
      { id: 'node-2', type: 'action', name: 'Server Email' },
    ],
    edges: [],
    content_hash: 'etag-server-456',
  };

  const defaultProps = {
    isOpen: true,
    localDraft,
    serverDraft,
    localETag: 'etag-local-123',
    serverETag: 'etag-server-456',
    onKeepLocal: vi.fn(),
    onAcceptServer: vi.fn(),
    onMerge: vi.fn(),
    onCancel: vi.fn(),
  };

  it('does not render when isOpen is false', () => {
    render(<ConflictDialog {...defaultProps} isOpen={false} />);
    expect(screen.queryByTestId('etag-conflict-dialog')).not.toBeInTheDocument();
  });

  it('renders ETag conflict dialog with side-by-side comparison of local vs server draft', () => {
    render(<ConflictDialog {...defaultProps} />);

    expect(screen.getByTestId('etag-conflict-dialog')).toBeInTheDocument();
    expect(screen.getByText(/ETag Version Conflict Detected/i)).toBeInTheDocument();

    // Cards
    expect(screen.getByTestId('local-edits-card')).toBeInTheDocument();
    expect(screen.getByTestId('server-revision-card')).toBeInTheDocument();

    expect(screen.getByText('Local Modified Journey')).toBeInTheDocument();
    expect(screen.getByText('Server Modified Journey')).toBeInTheDocument();

    // Node counts inside cards
    expect(screen.getByTestId('local-edits-card')).toHaveTextContent('Nodes Count: 1');
    expect(screen.getByTestId('server-revision-card')).toHaveTextContent('Nodes Count: 2');
  });

  it('triggers onKeepLocal when Overwrite button is clicked', () => {
    const onKeepLocalMock = vi.fn();
    render(<ConflictDialog {...defaultProps} onKeepLocal={onKeepLocalMock} />);

    const keepBtn = screen.getByTestId('keep-local-btn');
    fireEvent.click(keepBtn);

    expect(onKeepLocalMock).toHaveBeenCalledTimes(1);
  });

  it('triggers onAcceptServer when Accept Server Revision button is clicked', () => {
    const onAcceptServerMock = vi.fn();
    render(<ConflictDialog {...defaultProps} onAcceptServer={onAcceptServerMock} />);

    const acceptBtn = screen.getByTestId('accept-server-btn');
    fireEvent.click(acceptBtn);

    expect(onAcceptServerMock).toHaveBeenCalledTimes(1);
  });

  it('triggers onMerge when Merge Local & Server button is clicked', () => {
    const onMergeMock = vi.fn();
    render(<ConflictDialog {...defaultProps} onMerge={onMergeMock} />);

    const mergeBtn = screen.getByTestId('merge-conflict-btn');
    fireEvent.click(mergeBtn);

    expect(onMergeMock).toHaveBeenCalledTimes(1);
  });

  it('triggers onCancel when Cancel button is clicked', () => {
    const onCancelMock = vi.fn();
    render(<ConflictDialog {...defaultProps} onCancel={onCancelMock} />);

    const cancelBtn = screen.getByTestId('cancel-conflict-btn');
    fireEvent.click(cancelBtn);

    expect(onCancelMock).toHaveBeenCalledTimes(1);
  });
});
