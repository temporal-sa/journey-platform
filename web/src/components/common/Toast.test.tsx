import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { Toast, ToastMessageType } from './Toast';

describe('Toast Component', () => {
  it('renders success toast with required messageType enum, matching icon, and 60% opacity green border', () => {
    const handleClose = vi.fn();
    render(
      <Toast
        isOpen={true}
        messageType={ToastMessageType.SUCCESS}
        title="Journey test execution successfully launched"
        message="Workflow draft-101 is actively running."
        onClose={handleClose}
      />
    );

    const toast = screen.getByTestId('toast-notification');
    expect(toast).toBeInTheDocument();
    expect(toast.className).toContain('border-emerald-500/60');
    expect(screen.getByText('check_circle')).toBeInTheDocument();
    expect(screen.getByText('Journey test execution successfully launched')).toBeInTheDocument();
    expect(screen.getByText('Workflow draft-101 is actively running.')).toBeInTheDocument();
  });

  it('renders error toast with required messageType and 60% opacity red border', () => {
    const handleClose = vi.fn();
    render(
      <Toast
        isOpen={true}
        messageType={ToastMessageType.ERROR}
        title="Failed to launch test execution"
        message="Could not connect to journey control API service."
        onClose={handleClose}
      />
    );

    const toast = screen.getByTestId('toast-notification');
    expect(toast).toBeInTheDocument();
    expect(toast.className).toContain('border-rose-500/60');
    expect(screen.getByText('error')).toBeInTheDocument();
    expect(screen.getByText('Failed to launch test execution')).toBeInTheDocument();
  });

  it('renders warning and info toasts with consistent icons and matching 60% opacity borders', () => {
    const { rerender } = render(
      <Toast
        isOpen={true}
        messageType={ToastMessageType.WARNING}
        title="Warning Toast"
        onClose={() => {}}
      />
    );

    let toast = screen.getByTestId('toast-notification');
    expect(toast.className).toContain('border-amber-500/60');
    expect(screen.getByText('warning')).toBeInTheDocument();

    rerender(
      <Toast
        isOpen={true}
        messageType={ToastMessageType.INFO}
        title="Info Toast"
        onClose={() => {}}
      />
    );

    toast = screen.getByTestId('toast-notification');
    expect(toast.className).toContain('border-cyan-500/60');
    expect(screen.getByText('info')).toBeInTheDocument();
  });

  it('calls onClose when close button is clicked', () => {
    const handleClose = vi.fn();
    render(
      <Toast
        isOpen={true}
        messageType={ToastMessageType.SUCCESS}
        title="Test Notification"
        onClose={handleClose}
      />
    );

    const closeBtn = screen.getByRole('button', { name: 'Close notification' });
    fireEvent.click(closeBtn);
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('does not render when isOpen is false', () => {
    render(
      <Toast
        isOpen={false}
        messageType={ToastMessageType.SUCCESS}
        title="Hidden Toast"
        onClose={() => {}}
      />
    );

    expect(screen.queryByTestId('toast-notification')).not.toBeInTheDocument();
  });

  it('keeps toast open persistently until closed when duration is 0', () => {
    vi.useFakeTimers();
    const handleClose = vi.fn();

    render(
      <Toast
        isOpen={true}
        messageType={ToastMessageType.SUCCESS}
        title="Persistent Toast"
        onClose={handleClose}
      />
    );

    expect(handleClose).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(30000);
    });

    expect(handleClose).not.toHaveBeenCalled();
    vi.useRealTimers();
  });

  it('triggers onClose when custom duration is explicitly supplied', () => {
    vi.useFakeTimers();
    const handleClose = vi.fn();

    render(
      <Toast
        isOpen={true}
        messageType={ToastMessageType.SUCCESS}
        title="Timed Toast"
        duration={5000}
        onClose={handleClose}
      />
    );

    expect(handleClose).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(5000);
    });

    expect(handleClose).toHaveBeenCalledTimes(1);
    vi.useRealTimers();
  });
});
