import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { NodeInspector } from './NodeInspector';
import { ParameterModal } from './ParameterModal';
import { useEditorStore } from '../../stores/editorStore';
import type { GraphDraft, ValidationResult } from '../../types/api';

describe('NodeInspector Component', () => {
  const sampleDraft: GraphDraft = {
    schema_version: '1.0',
    draft_id: 'draft-test-101',
    tenant_id: 'tenant-test',
    name: 'Inspector Test Journey',
    version: 1,
    nodes: [
      {
        id: 'trigger-1',
        type: 'EventStart',
        name: 'Order Placed Trigger',
        config: {
          event_name: 'order_completed',
          event_filter: 'event.data.amount > 50',
        },
      },
      {
        id: 'email-1',
        type: 'Email',
        name: 'Welcome Email Action',
        config: {
          recipient: 'user@example.com',
          subject: 'Welcome to Acme!',
          template_id: 'tmpl_welcome',
        },
      },
      {
        id: 'condition-1',
        type: 'Condition',
        name: 'VIP Check Decision',
        config: {
          condition_expression: "subject.tier == 'VIP'",
        },
      },
      {
        id: 'delay-1',
        type: 'Delay',
        name: 'Wait 24 Hours',
        config: {
          duration: 24,
          unit: 'hours',
        },
      },
    ],
    edges: [],
  };

  beforeEach(() => {
    useEditorStore.getState().resetStore();
    useEditorStore.getState().setDraft(sampleDraft);
  });

  it('renders empty selection prompt when no nodeId is selected', () => {
    render(<NodeInspector nodeId={null} />);
    expect(screen.getByTestId('node-inspector-empty')).toBeInTheDocument();
    expect(screen.getByText('No Node Selected')).toBeInTheDocument();
  });

  it('renders schema-driven form controls for selected Trigger node', () => {
    render(<NodeInspector nodeId="trigger-1" />);

    expect(screen.getByTestId('node-inspector-panel')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Order Placed Trigger')).toBeInTheDocument();
    expect(screen.getByDisplayValue('order_completed')).toBeInTheDocument();
    expect(screen.getByDisplayValue('event.data.amount > 50')).toBeInTheDocument();
  });

  it('renders schema-driven form controls for selected Email node', () => {
    render(<NodeInspector nodeId="email-1" />);

    expect(screen.getByDisplayValue('Welcome Email Action')).toBeInTheDocument();
    expect(screen.getByDisplayValue('user@example.com')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Welcome to Acme!')).toBeInTheDocument();
    expect(screen.getByDisplayValue('tmpl_welcome')).toBeInTheDocument();
  });

  it('isolated draft buffer: Cancel restores original graph node state', () => {
    render(<NodeInspector nodeId="email-1" />);

    const recipientInput = screen.getByTestId('inspector-input-recipient') as HTMLInputElement;
    expect(recipientInput).toHaveValue('user@example.com');

    // Modify value in isolated draft buffer
    fireEvent.change(recipientInput, { target: { value: 'changed@example.com' } });
    expect(recipientInput).toHaveValue('changed@example.com');
    expect(screen.getByTestId('unsaved-inspector-badge')).toBeInTheDocument();

    // Click Cancel to discard isolated edits
    const cancelBtn = screen.getByTestId('inspector-cancel-btn');
    fireEvent.click(cancelBtn);

    expect(recipientInput).toHaveValue('user@example.com');
    expect(screen.queryByTestId('unsaved-inspector-badge')).not.toBeInTheDocument();

    // Store state remains untouched
    const currentNodes = useEditorStore.getState().currentDraft?.nodes;
    const emailNode = currentNodes?.find((n) => n.id === 'email-1');
    expect(emailNode?.config?.recipient).toBe('user@example.com');
  });

  it('isolated draft buffer: Save applies edits to editorStore graph state', () => {
    render(<NodeInspector nodeId="email-1" />);

    const recipientInput = screen.getByTestId('inspector-input-recipient');
    const subjectInput = screen.getByTestId('inspector-input-subject');

    fireEvent.change(recipientInput, { target: { value: 'new.user@domain.com' } });
    fireEvent.change(subjectInput, { target: { value: 'Updated Email Subject' } });

    expect(screen.getByTestId('unsaved-inspector-badge')).toBeInTheDocument();

    const saveBtn = screen.getByTestId('inspector-save-btn');
    fireEvent.click(saveBtn);

    expect(screen.queryByTestId('unsaved-inspector-badge')).not.toBeInTheDocument();

    // Store state updated
    const currentNodes = useEditorStore.getState().currentDraft?.nodes;
    const emailNode = currentNodes?.find((n) => n.id === 'email-1');
    expect(emailNode?.config?.recipient).toBe('new.user@domain.com');
    expect(emailNode?.config?.subject).toBe('Updated Email Subject');
  });

  it('maps server validation issues directly to node alert banner and field errors', () => {
    const sampleValidationResult: ValidationResult = {
      draft_id: 'draft-test-101',
      is_valid: false,
      issues: [
        {
          schema_version: '1.0',
          issue_id: 'issue-1',
          node_id: 'email-1',
          severity: 'error',
          code: 'ERR_REQUIRED_RECIPIENT',
          message: 'Recipient email address is invalid or missing',
          field_path: 'config.recipient',
        },
        {
          schema_version: '1.0',
          issue_id: 'issue-2',
          node_id: 'email-1',
          severity: 'warning',
          code: 'WARN_SUBJECT_SHORT',
          message: 'Subject line is under recommended length',
          field_path: 'config.subject',
        },
      ],
    };

    useEditorStore.getState().setValidationResult(sampleValidationResult);

    render(<NodeInspector nodeId="email-1" />);

    // Node-level alert banner
    expect(screen.getByTestId('node-validation-alert')).toBeInTheDocument();
    expect(screen.getAllByText(/Recipient email address is invalid or missing/i)[0]).toBeInTheDocument();

    // Field-level error messages
    expect(screen.getByTestId('field-error-recipient')).toBeInTheDocument();
    expect(screen.getByTestId('field-error-subject')).toBeInTheDocument();
  });

  it('opens Parameter Library Modal and inserts token into active field', () => {
    let isOpen = false;
    let targetField: string | undefined = undefined;

    const mockParamsList = [
      { key: 'email', token: '{{subject.email}}', name: 'Email Address', category: 'subject' as const, categoryLabel: 'Subject Attribute', description: 'Email', dataClassification: 'PII' as const },
    ];

    const { rerender } = render(
      <>
        <NodeInspector nodeId="email-1" />
        <ParameterModal
          isOpen={isOpen}
          onClose={() => { isOpen = false; }}
          onSelectToken={(token) => {
            window.dispatchEvent(new CustomEvent('select-param-token', { detail: { token, fieldKey: targetField } }));
          }}
          targetFieldLabel={targetField}
          customParameters={mockParamsList}
        />
      </>
    );

    const tokenBtn = screen.getByTestId('param-btn-recipient');
    fireEvent.click(tokenBtn);

    isOpen = true;
    targetField = 'recipient';

    rerender(
      <>
        <NodeInspector nodeId="email-1" />
        <ParameterModal
          isOpen={isOpen}
          onClose={() => { isOpen = false; }}
          onSelectToken={(token) => {
            window.dispatchEvent(new CustomEvent('select-param-token', { detail: { token, fieldKey: targetField } }));
          }}
          targetFieldLabel={targetField}
          customParameters={mockParamsList}
        />
      </>
    );

    // Modal opens
    expect(screen.getByTestId('parameter-modal')).toBeInTheDocument();

    // Select token
    const insertBtn = screen.getByTestId('insert-token-btn-email');
    fireEvent.click(insertBtn);

    // Token inserted into input
    expect(screen.getByTestId('inspector-input-recipient')).toHaveValue('user@example.com{{subject.email}}');
  });

  it('invokes onClose callback when close button is clicked', () => {
    const onCloseMock = vi.fn();
    render(<NodeInspector nodeId="email-1" onClose={onCloseMock} />);

    const closeBtn = screen.getByTestId('close-inspector-btn');
    fireEvent.click(closeBtn);

    expect(onCloseMock).toHaveBeenCalledTimes(1);
  });
});
