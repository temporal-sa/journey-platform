import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { TestRunModal } from './TestRunModal';

describe('TestRunModal Component', () => {
  it('does not render modal when isOpen is false', () => {
    const { container } = render(<TestRunModal isOpen={false} onClose={() => {}} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders modal with execution mode dropdown, audience source selector, and fake providers when isOpen is true', () => {
    render(<TestRunModal isOpen={true} onClose={() => {}} draftId="draft-xyz" />);

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Configure Test Run Execution' })).toBeInTheDocument();
    expect(screen.getByTestId('execution-mode-select')).toBeInTheDocument();
    expect(screen.getByTestId('execution-mode-tooltip-icon')).toBeInTheDocument();
    expect(screen.getByText('Realistic Mode')).toBeInTheDocument();
    expect(screen.getByLabelText('Audience Source (Static List)')).toBeInTheDocument();
    expect(screen.getByText('Mock SendGrid (Email)')).toBeInTheDocument();
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

    const modeSelect = screen.getByTestId('execution-mode-select') as HTMLSelectElement;
    fireEvent.change(modeSelect, { target: { value: 'forced_variant_coverage' } });
    expect(modeSelect.value).toBe('forced_variant_coverage');

    // Toggle a provider checkbox
    const sendgridCheckbox = screen.getByLabelText('Mock SendGrid (Email)') as HTMLInputElement;
    expect(sendgridCheckbox.checked).toBe(true);
    fireEvent.click(sendgridCheckbox);
    expect(sendgridCheckbox.checked).toBe(false);

    // Click execute test run without selecting static list -> shows validation error
    const executeBtn = screen.getByRole('button', { name: 'Execute Test Run' });
    fireEvent.click(executeBtn);

    expect(screen.getByRole('alert')).toHaveTextContent('Please select a static audience list');
  });
});
