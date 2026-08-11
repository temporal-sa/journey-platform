import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { RunDetailPage } from './RunDetailPage';

function renderWithClient(ui: React.ReactElement) {
  const testQueryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, refetchOnWindowFocus: false },
    },
  });
  return render(<QueryClientProvider client={testQueryClient}>{ui}</QueryClientProvider>);
}

describe('RunDetailPage Component', () => {
  beforeEach(() => {
    window.history.pushState({}, '', '/?runId=run-601');
  });

  it('renders run detail header, parameters, node visits timeline, and action executions', async () => {
    renderWithClient(<RunDetailPage runId="run-601" />);

    expect(screen.getByText(/Run Execution Detail:/i)).toBeInTheDocument();
    expect(screen.getByText('run-601')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('tenant-default')).toBeInTheDocument();
      expect(screen.getByText((content) => content.includes('wf-welcome-series') || content.includes('wf-run-601'))).toBeInTheDocument();
      expect(screen.getByText(/Action Executions/i)).toBeInTheDocument();
      expect(screen.getByText(/Suppressions & Policy Checks/i)).toBeInTheDocument();
    });
  });

  it('executes replay / retry trigger mutation on button click', async () => {
    renderWithClient(<RunDetailPage runId="run-601" />);

    const replayBtn = screen.getByRole('button', { name: /Trigger Replay \/ Retry/i });
    fireEvent.click(replayBtn);

    await waitFor(() => {
      expect(screen.getByText(/Successfully triggered event emission replay for run/i)).toBeInTheDocument();
    });
  });

  it('calls onBackToList when Back to Run List button is clicked', async () => {
    const handleBack = vi.fn();
    renderWithClient(<RunDetailPage runId="run-601" onBackToList={handleBack} />);

    const backBtn = screen.getByRole('button', { name: /Back to Run List/i });
    fireEvent.click(backBtn);

    expect(handleBack).toHaveBeenCalled();
  });
  it('renders view segment toggle and toggles between Visual Graph Trace and Timeline List', async () => {
    renderWithClient(<RunDetailPage runId="run-601" />);

    const graphToggle = await screen.findByTestId('view-toggle-graph');
    const timelineToggle = await screen.findByTestId('view-toggle-timeline');

    expect(graphToggle).toBeInTheDocument();
    expect(timelineToggle).toBeInTheDocument();

    expect(screen.getAllByText('Trace Graph').length).toBeGreaterThan(0);
    expect(screen.getByTestId('execution-graph-container')).toBeInTheDocument();

    fireEvent.click(timelineToggle);

    await waitFor(() => {
      expect(screen.getByText(/Node Visits & Outcome Timeline/i)).toBeInTheDocument();
      expect(screen.queryByTestId('execution-graph-container')).not.toBeInTheDocument();
    });

    fireEvent.click(graphToggle);

    await waitFor(() => {
      expect(screen.getAllByText('Trace Graph').length).toBeGreaterThan(0);
      expect(screen.getByTestId('execution-graph-container')).toBeInTheDocument();
    });
  });
});
