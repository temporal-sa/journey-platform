import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { LogStreamModal } from './LogStreamModal';

describe('LogStreamModal Component', () => {
  beforeEach(() => {
    class MockEventSource {
      onmessage: ((event: MessageEvent) => void) | null = null;
      onerror: ((event: Event) => void) | null = null;
      close() {}
    }
    vi.stubGlobal('EventSource', MockEventSource);
  });

  it('does not render modal when isOpen is false', () => {
    render(<LogStreamModal isOpen={false} onClose={() => {}} />);
    expect(screen.queryByTestId('log-stream-modal')).not.toBeInTheDocument();
  });

  it('renders modal with status badge, action buttons and terminal output when isOpen is true', () => {
    render(<LogStreamModal isOpen={true} onClose={() => {}} />);

    expect(screen.getByTestId('log-stream-modal')).toBeInTheDocument();
    expect(screen.getByText('Live Server Log Stream')).toBeInTheDocument();
    expect(screen.getByTestId('log-stream-status-live')).toHaveTextContent('STREAMING');
    expect(screen.getByTestId('toggle-log-stream-btn')).toHaveTextContent('Pause Stream');
    expect(screen.getByTestId('clear-logs-btn')).toBeInTheDocument();
    expect(screen.getByTestId('log-terminal-output')).toBeInTheDocument();
  });

  it('toggles stream status between streaming and paused when toggle button is clicked', () => {
    render(<LogStreamModal isOpen={true} onClose={() => {}} />);

    const toggleBtn = screen.getByTestId('toggle-log-stream-btn');
    fireEvent.click(toggleBtn);

    expect(screen.getByTestId('log-stream-status-paused')).toHaveTextContent('PAUSED');
    expect(toggleBtn).toHaveTextContent('Resume Stream');

    fireEvent.click(toggleBtn);

    expect(screen.getByTestId('log-stream-status-live')).toHaveTextContent('STREAMING');
    expect(toggleBtn).toHaveTextContent('Pause Stream');
  });

  it('calls onClose callback when modal close button is clicked', () => {
    const handleClose = vi.fn();
    render(<LogStreamModal isOpen={true} onClose={handleClose} />);

    const closeBtn = screen.getByRole('button', { name: 'Close modal' });
    fireEvent.click(closeBtn);

    expect(handleClose).toHaveBeenCalledOnce();
  });
  it('renders log entries with alternating background colors and pretty printed JSON payloads', async () => {
    class MockEventSource {
      onmessage: ((event: MessageEvent) => void) | null = null;
      onerror: ((event: Event) => void) | null = null;
      constructor() {
        setTimeout(() => {
          if (this.onmessage) {
            this.onmessage({
              data: JSON.stringify({
                timestamp: '2026-08-14T10:00:00Z',
                level: 'info',
                message: 'Processing event',
                fields: { trace_id: 'tr-100', user_id: 'usr_42' },
              }),
            } as MessageEvent);
            this.onmessage({
              data: JSON.stringify({
                timestamp: '2026-08-14T10:00:01Z',
                level: 'error',
                message: '{"status":"failed","code":500}',
              }),
            } as MessageEvent);
          }
        }, 10);
      }
      close() {}
    }
    vi.stubGlobal('EventSource', MockEventSource);

    render(<LogStreamModal isOpen={true} onClose={() => {}} />);

    await waitFor(() => {
      const logEntries = screen.getAllByTestId('log-entry');
      expect(logEntries.length).toBe(2);
      expect(logEntries[0]).toHaveClass('bg-[#0a0e19]');
      expect(logEntries[1]).toHaveClass('bg-[#121828]');
      expect(screen.getByText(/"trace_id": "tr-100"/)).toBeInTheDocument();
      expect(screen.getByText(/"status": "failed"/)).toBeInTheDocument();
    });
  });
});
