import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { TestRunModal } from './TestRunModal';

describe('TestRunModal Component', () => {
  it('does not render modal when isOpen is false', () => {
    const { container } = render(<TestRunModal isOpen={false} onClose={() => {}} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders modal with explicit TEST MODE badge, execution modes, fixture picker, fake providers, and preview metrics', () => {
    render(<TestRunModal isOpen={true} onClose={() => {}} draftId="draft-xyz" />);

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Configure Test Run Execution' })).toBeInTheDocument();
    expect(screen.getByTestId('test-mode-badge')).toHaveTextContent('TEST MODE');
    expect(screen.getByText('Realistic Mode')).toBeInTheDocument();
    expect(screen.getByText('Forced Variant Coverage')).toBeInTheDocument();
    expect(screen.getByLabelText('Audience Source (Static List)')).toBeInTheDocument();
    expect(screen.getByText('Mock SendGrid (Email)')).toBeInTheDocument();
    expect(screen.getByText('Target Member Count')).toBeInTheDocument();
  });

  it('allows toggling execution mode and provider checkboxes and starting test run', async () => {
    const onStartTestRun = vi.fn();
    render(
      <TestRunModal
        isOpen={true}
        onClose={() => {}}
        onStartTestRun={onStartTestRun}
        draftId="draft-abc"
      />
    );

    // Click Forced Variant Coverage mode
    const forcedModeBtn = screen.getByText('Forced Variant Coverage');
    fireEvent.click(forcedModeBtn);

    // Toggle a provider checkbox
    const sendgridCheckbox = screen.getByLabelText('Mock SendGrid (Email)') as HTMLInputElement;
    expect(sendgridCheckbox.checked).toBe(true);
    fireEvent.click(sendgridCheckbox);
    expect(sendgridCheckbox.checked).toBe(false);

    // Click execute test run
    const executeBtn = screen.getByRole('button', { name: 'Execute Test Run' });
    fireEvent.click(executeBtn);

    await new Promise((resolve) => setTimeout(resolve, 300));

    expect(onStartTestRun).toHaveBeenCalledWith(
      expect.objectContaining({
        draftId: 'draft-abc',
        executionMode: 'forced_variant_coverage',
      })
    );
  });
});
