import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { JourneysPage } from './JourneysPage';

function renderWithClient(ui: React.ReactElement) {
  const testQueryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, refetchOnWindowFocus: false },
    },
  });
  return render(<QueryClientProvider client={testQueryClient}>{ui}</QueryClientProvider>);
}

describe('JourneysPage Component', () => {
  beforeEach(() => {
    window.history.pushState({}, '', '/');
  });

  it('renders journeys directory with title, search, status filter and journey items', async () => {
    renderWithClient(<JourneysPage />);

    expect(screen.getByText('Journeys Directory')).toBeInTheDocument();
    expect(screen.getByTestId('journeys-search-input')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('Welcome Journey Draft')).toBeInTheDocument();
    });
  });

  it('filters journeys by status filter selection', async () => {
    renderWithClient(<JourneysPage />);

    await waitFor(() => {
      expect(screen.getByText('Welcome Journey Draft')).toBeInTheDocument();
    });

    const triggerBtn = screen.getByTestId('journeys-status-filter-dropdown');
    fireEvent.click(triggerBtn);

    const activeOption = screen.getByRole('option', { name: /Active/i });
    fireEvent.click(activeOption);

    await waitFor(() => {
      expect(screen.getByText('KYC Verification Nudge')).toBeInTheDocument();
    });
  });

  it('filters journeys by search text query', async () => {
    renderWithClient(<JourneysPage />);

    const searchInput = screen.getByTestId('journeys-search-input');
    fireEvent.change(searchInput, { target: { value: 'KYC' } });

    await waitFor(() => {
      expect(screen.getByText('KYC Verification Nudge')).toBeInTheDocument();
      expect(screen.queryByText('Cart Abandonment Recovery')).not.toBeInTheDocument();
    });
  });

  it('renders empty state when search returns zero results', async () => {
    renderWithClient(<JourneysPage />);

    const searchInput = screen.getByTestId('journeys-search-input');
    fireEvent.change(searchInput, { target: { value: 'nonexistent_search_query_xyz' } });

    await waitFor(() => {
      expect(screen.getByText('No Journeys Found')).toBeInTheDocument();
      expect(screen.getByTestId('journeys-clear-filters-btn')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('journeys-clear-filters-btn'));

    await waitFor(() => {
      expect(screen.getByText('Welcome Journey Draft')).toBeInTheDocument();
    });
  });

  it('triggers onSelectJourney callback when Open Canvas is clicked', async () => {
    const handleSelect = vi.fn();
    renderWithClient(<JourneysPage onSelectJourney={handleSelect} />);

    await waitFor(() => {
      expect(screen.getByText('Welcome Journey Draft')).toBeInTheDocument();
    });

    const openCanvasButtons = screen.getAllByRole('button', { name: /Open Canvas/i });
    fireEvent.click(openCanvasButtons[0]);

    expect(handleSelect).toHaveBeenCalledWith(expect.any(String));
  });

  it('disables Previous and Next pagination buttons when no additional pages exist', async () => {
    renderWithClient(<JourneysPage />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Previous/i })).toBeDisabled();
      expect(screen.getByRole('button', { name: /Next/i })).toBeDisabled();
    });
  });
});
