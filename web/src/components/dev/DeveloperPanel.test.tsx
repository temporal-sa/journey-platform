import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DeveloperPanel } from './DeveloperPanel';
import { ToastMessageType } from '../common/Toast';
import {
  isSimulatedApiFailureEnabled,
  setSimulatedApiFailureEnabled,
} from '../../api/simulatedFailure';

describe('DeveloperPanel Component', () => {
  beforeEach(() => {
    setSimulatedApiFailureEnabled(false);
  });

  afterEach(() => {
    setSimulatedApiFailureEnabled(false);
  });

  it('renders modal when open with toast trigger buttons and API failure toggle', () => {
    const handleClose = vi.fn();
    const handleFireToast = vi.fn();

    render(
      <DeveloperPanel
        isOpen={true}
        onClose={handleClose}
        onFireToast={handleFireToast}
      />
    );

    expect(screen.getByTestId('developer-panel')).toBeInTheDocument();
    expect(screen.getByText('Developer Control Panel')).toBeInTheDocument();
    expect(screen.getByTestId('toggle-api-failure-btn')).toBeInTheDocument();
    expect(screen.getByTestId('dev-toast-success')).toBeInTheDocument();
    expect(screen.getByTestId('dev-toast-error')).toBeInTheDocument();
    expect(screen.getByTestId('dev-toast-warning')).toBeInTheDocument();
    expect(screen.getByTestId('dev-toast-info')).toBeInTheDocument();
  });

  it('toggles API failure mode state when failure toggle button is clicked', () => {
    const handleClose = vi.fn();
    const handleFireToast = vi.fn();

    render(
      <DeveloperPanel
        isOpen={true}
        onClose={handleClose}
        onFireToast={handleFireToast}
      />
    );

    const toggleBtn = screen.getByTestId('toggle-api-failure-btn');
    expect(isSimulatedApiFailureEnabled()).toBe(false);
    expect(screen.getByText('DISABLED')).toBeInTheDocument();
    expect(screen.getByText('Enable Failure Mode')).toBeInTheDocument();

    fireEvent.click(toggleBtn);
    expect(isSimulatedApiFailureEnabled()).toBe(true);
    expect(screen.getByText('ACTIVE')).toBeInTheDocument();
    expect(screen.getByText('Disable Failure Mode')).toBeInTheDocument();
    expect(handleFireToast).toHaveBeenCalledWith(
      ToastMessageType.WARNING,
      'API Failure Mode Enabled',
      expect.any(String)
    );

    fireEvent.click(toggleBtn);
    expect(isSimulatedApiFailureEnabled()).toBe(false);
    expect(screen.getByText('DISABLED')).toBeInTheDocument();
    expect(screen.getByText('Enable Failure Mode')).toBeInTheDocument();
    expect(handleFireToast).toHaveBeenCalledWith(
      ToastMessageType.INFO,
      'API Failure Mode Disabled',
      expect.any(String)
    );
  });

  it('triggers onFireToast callback with correct parameters for each button', () => {
    const handleClose = vi.fn();
    const handleFireToast = vi.fn();

    render(
      <DeveloperPanel
        isOpen={true}
        onClose={handleClose}
        onFireToast={handleFireToast}
      />
    );

    fireEvent.click(screen.getByTestId('dev-toast-success'));
    expect(handleFireToast).toHaveBeenLastCalledWith(
      ToastMessageType.SUCCESS,
      'Journey Execution Succeeded',
      'Test run scenario executed cleanly across 4 mock nodes.'
    );

    fireEvent.click(screen.getByTestId('dev-toast-error'));
    expect(handleFireToast).toHaveBeenLastCalledWith(
      ToastMessageType.ERROR,
      'Workflow Execution Failed',
      'State transition error on node-301: Connection timed out.'
    );

    fireEvent.click(screen.getByTestId('dev-toast-warning'));
    expect(handleFireToast).toHaveBeenLastCalledWith(
      ToastMessageType.WARNING,
      'Rate Limit Approaching',
      'Provider mock throughput is operating at 92% capacity.'
    );

    fireEvent.click(screen.getByTestId('dev-toast-info'));
    expect(handleFireToast).toHaveBeenLastCalledWith(
      ToastMessageType.INFO,
      'System State Synced',
      'ETag draft hash updated to draft-101-v5 revision.'
    );
  });
});
