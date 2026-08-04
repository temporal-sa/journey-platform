import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DegradedStateView, DegradedStateType } from './DegradedStateView';

describe('DegradedStateView Component', () => {
  const types: DegradedStateType[] = [
    'api-disconnected',
    'stale-draft-revision',
    'invalid-workflow',
    'failed-simulation',
    'failed-test-run',
    'stale-report',
    'unknown-provider-state',
  ];

  types.forEach((type) => {
    it(`renders degraded state type "${type}" with accessible markup`, () => {
      render(<DegradedStateView type={type} />);

      const alertContainer = screen.getByRole('alert');
      expect(alertContainer).toBeInTheDocument();
      expect(alertContainer).toHaveAttribute('data-testid', `degraded-state-${type}`);

      // Verify labelledby and describedby linking
      const titleId = alertContainer.getAttribute('aria-labelledby');
      const descId = alertContainer.getAttribute('aria-describedby');

      expect(titleId).toBeTruthy();
      expect(descId).toBeTruthy();

      const titleEl = document.getElementById(titleId!);
      const descEl = document.getElementById(descId!);

      expect(titleEl).toBeInTheDocument();
      expect(descEl).toBeInTheDocument();
    });
  });

  it('renders custom title, description, and action button callback', () => {
    const handleRetry = vi.fn();

    render(
      <DegradedStateView
        type="api-disconnected"
        title="Custom Disconnection Title"
        description="Custom backend retry description"
        actionLabel="Try Reconnecting Now"
        onRetry={handleRetry}
      />
    );

    expect(screen.getByText('Custom Disconnection Title')).toBeInTheDocument();
    expect(screen.getByText('Custom backend retry description')).toBeInTheDocument();

    const actionBtn = screen.getByRole('button', { name: /Try Reconnecting Now/i });
    expect(actionBtn).toBeInTheDocument();

    fireEvent.click(actionBtn);
    expect(handleRetry).toHaveBeenCalledTimes(1);
  });

  it('renders details section when string or object provided', () => {
    render(
      <DegradedStateView
        type="failed-simulation"
        details={{ errorCode: 'ERR_SIM_TIMEOUT', stepId: 'step-42' }}
      />
    );

    expect(screen.getByText(/"errorCode": "ERR_SIM_TIMEOUT"/)).toBeInTheDocument();
  });
});
