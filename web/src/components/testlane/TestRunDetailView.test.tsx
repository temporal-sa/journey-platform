import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { TestRunDetailView } from './TestRunDetailView';

describe('TestRunDetailView Component', () => {
  it('renders header with prominent TEST MODE badge and metadata summary', async () => {
    render(<TestRunDetailView testRunId="tr-static-901" />);

    expect(await screen.findByText(/Test-Run Results:/i)).toBeInTheDocument();
    expect(screen.getByText('tr-static-901')).toBeInTheDocument();
    expect(screen.getByTestId('test-mode-badge')).toHaveTextContent('TEST MODE');
    expect(screen.getAllByText('passed')[0]).toBeInTheDocument();
    expect(screen.getByText('145 ms')).toBeInTheDocument();
  });

  it('renders summary KPI metrics', async () => {
    render(<TestRunDetailView />);

    expect(await screen.findByText('Total Test Members')).toBeInTheDocument();
    expect(screen.getByText('50')).toBeInTheDocument(); // total members
    expect(screen.getByText('Completed Journey')).toBeInTheDocument();
    expect(screen.getByText('46')).toBeInTheDocument(); // completed members
    expect(screen.getByText('Suppressions')).toBeInTheDocument();
    expect(screen.getAllByText('4')[0]).toBeInTheDocument(); // suppressions count
  });

  it('switches between detail tabs: per-member visit steps, branch decisions, suppressions, and fake deliveries', async () => {
    render(<TestRunDetailView />);

    // Default tab is Per-Member Node Visit Steps
    expect(await screen.findByText(/Execution Path for/i)).toBeInTheDocument();
    expect(screen.getByText('User Signup Trigger')).toBeInTheDocument();
    expect(screen.getByText('Is Premium User?')).toBeInTheDocument();

    // Switch to Member MBR-002
    const member2Btn = screen.getByRole('button', { name: /MBR-002/i });
    fireEvent.click(member2Btn);
    expect(screen.getByText('Send Standard Onboarding Email')).toBeInTheDocument();

    // Switch to Branch Decisions tab
    const branchTab = screen.getByRole('button', { name: /Branch Decisions/i });
    fireEvent.click(branchTab);
    expect(screen.getByText('Condition & Experiment Branch Decisions')).toBeInTheDocument();
    expect(screen.getAllByText('attributes.user_tier == "premium"')[0]).toBeInTheDocument();
    expect(screen.getByText('True Branch (node-action-1)')).toBeInTheDocument();

    // Switch to Suppressions tab
    const suppTab = screen.getByRole('button', { name: /Suppressions/i });
    fireEvent.click(suppTab);
    expect(screen.getByText('Target Suppressions & Opt-Out Log')).toBeInTheDocument();
    expect(screen.getByText(/frequency cap/i)).toBeInTheDocument();
    expect(screen.getByText(/global opt out/i)).toBeInTheDocument();

    // Switch to Fake Delivery Status tab
    const deliveryTab = screen.getByRole('button', { name: /Fake Delivery Status/i });
    fireEvent.click(deliveryTab);
    expect(screen.getByText('Fake Delivery Dispatch Log')).toBeInTheDocument();
    expect(screen.getAllByText('mock-sendgrid')[0]).toBeInTheDocument();
    expect(screen.getByText('mock-twilio')).toBeInTheDocument();
    expect(screen.getAllByText('delivered')[0]).toBeInTheDocument();
    expect(screen.getByText('bounced')).toBeInTheDocument();
  });

  it('triggers onBackToList callback when back button is clicked', async () => {
    const onBackToList = vi.fn();
    render(<TestRunDetailView onBackToList={onBackToList} />);

    const backBtn = await screen.findByRole('button', { name: '← Back to Runs' });
    fireEvent.click(backBtn);
    expect(onBackToList).toHaveBeenCalledTimes(1);
  });
});
