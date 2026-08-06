import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ParameterModal, ParameterItem } from './ParameterModal';

describe('ParameterModal Component', () => {
  const mockParameters: ParameterItem[] = [
    { key: 'user_id', token: '{{event.user_id}}', name: 'User ID', category: 'event', categoryLabel: 'Event Field', description: 'User ID', dataClassification: 'PII', exampleValue: 'usr_1' },
    { key: 'first_name', token: '{{subject.first_name}}', name: 'First Name', category: 'subject', categoryLabel: 'Subject Attribute', description: 'First Name', dataClassification: 'PII', exampleValue: 'Jane' },
    { key: 'email', token: '{{subject.email}}', name: 'Email Address', category: 'subject', categoryLabel: 'Subject Attribute', description: 'Email Address', dataClassification: 'PII', exampleValue: 'jane@example.com' },
    { key: 'variant_id', token: '{{experiment.variant_id}}', name: 'Variant ID', category: 'experiment', categoryLabel: 'Experiment Context', description: 'Variant ID', dataClassification: 'NonPII', exampleValue: 'var_a' },
    { key: 'global_support_email', token: '{{params.global_support_email}}', name: 'Support Email', category: 'parameter', categoryLabel: 'Shared Parameter', description: 'Support Email', dataClassification: 'NonPII', exampleValue: 'support@example.com' },
  ];

  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    onSelectToken: vi.fn(),
    targetFieldLabel: 'Recipient Email',
    customParameters: mockParameters,
  };

  it('does not render modal when isOpen is false', () => {
    render(<ParameterModal {...defaultProps} isOpen={false} />);
    expect(screen.queryByTestId('parameter-library-modal')).not.toBeInTheDocument();
  });

  it('renders Parameter Library Modal with title, search input, and default items', () => {
    render(<ParameterModal {...defaultProps} />);

    expect(screen.getByTestId('parameter-modal')).toBeInTheDocument();
    expect(screen.getByText('Parameter Library')).toBeInTheDocument();
    expect(screen.getByText(/Insert dynamic parameter token into field "Recipient Email"/i)).toBeInTheDocument();
    expect(screen.getByTestId('parameter-search-input')).toBeInTheDocument();

    // Contains default tokens
    expect(screen.getByTestId('param-item-user_id')).toBeInTheDocument();
    expect(screen.getByTestId('param-item-email')).toBeInTheDocument();
    expect(screen.getByTestId('param-item-global_support_email')).toBeInTheDocument();
  });

  it('filters parameters list in real-time when search query is entered', () => {
    render(<ParameterModal {...defaultProps} />);

    const searchInput = screen.getByTestId('parameter-search-input');
    fireEvent.change(searchInput, { target: { value: 'first_name' } });

    expect(screen.getByTestId('param-item-first_name')).toBeInTheDocument();
    expect(screen.queryByTestId('param-item-user_id')).not.toBeInTheDocument();
  });

  it('displays empty state when search query matches no items', () => {
    render(<ParameterModal {...defaultProps} />);

    const searchInput = screen.getByTestId('parameter-search-input');
    fireEvent.change(searchInput, { target: { value: 'nonexistent_token_xyz_99' } });

    expect(screen.getByTestId('no-parameters-found')).toBeInTheDocument();
    expect(screen.getByText('No parameters match your search')).toBeInTheDocument();
  });

  it('filters parameters by category tabs', () => {
    render(<ParameterModal {...defaultProps} />);

    const subjectTab = screen.getByTestId('param-category-tab-subject');
    fireEvent.click(subjectTab);

    expect(screen.getByTestId('param-item-email')).toBeInTheDocument();
    expect(screen.queryByTestId('param-item-user_id')).not.toBeInTheDocument();

    const experimentTab = screen.getByTestId('param-category-tab-experiment');
    fireEvent.click(experimentTab);

    expect(screen.getByTestId('param-item-variant_id')).toBeInTheDocument();
    expect(screen.queryByTestId('param-item-email')).not.toBeInTheDocument();
  });

  it('calls onSelectToken and onClose when Insert Token button is clicked', () => {
    const onSelectTokenMock = vi.fn();
    const onCloseMock = vi.fn();

    render(
      <ParameterModal
        {...defaultProps}
        onSelectToken={onSelectTokenMock}
        onClose={onCloseMock}
      />
    );

    const insertBtn = screen.getByTestId('insert-token-btn-first_name');
    fireEvent.click(insertBtn);

    expect(onSelectTokenMock).toHaveBeenCalledTimes(1);
    expect(onSelectTokenMock).toHaveBeenCalledWith('{{subject.first_name}}');
    expect(onCloseMock).toHaveBeenCalledTimes(1);
  });

  it('calls onClose when close button is clicked', () => {
    const onCloseMock = vi.fn();

    render(<ParameterModal {...defaultProps} onClose={onCloseMock} />);

    const closeBtn = screen.getByRole('button', { name: 'Close modal' });
    fireEvent.click(closeBtn);

    expect(onCloseMock).toHaveBeenCalledTimes(1);
  });
});
