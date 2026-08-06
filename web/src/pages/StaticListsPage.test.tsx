import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { StaticListsPage } from './StaticListsPage';
import type { StaticList } from '../types/api';

const { mockLists } = vi.hoisted(() => {
  const lists: StaticList[] = [
    {
      schema_version: '1.0',
      list_id: 'list-vip-users',
      name: 'VIP Customers List',
      description: 'High-value customer dataset',
      item_count: 250,
      data_classification: 'PII',
      items: ['user-1@example.com', 'user-2@example.com'],
      created_at: '2026-08-01T12:00:00Z',
    },
    {
      schema_version: '1.0',
      list_id: 'list-beta-testers',
      name: 'Beta Testers Segment',
      description: 'Opted-in product feature testers',
      item_count: 50,
      data_classification: 'NonPII',
      items: ['tester-a', 'tester-b'],
      created_at: '2026-08-02T14:30:00Z',
    },
  ];
  return { mockLists: lists };
});

vi.mock('../api/client', () => {
  return {
    JourneyApiClient: vi.fn().mockImplementation(() => ({
      listStaticLists: vi.fn().mockResolvedValue(mockLists),
    })),
  };
});

function renderWithQueryClient(ui: React.ReactElement) {
  const testQueryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
  return render(<QueryClientProvider client={testQueryClient}>{ui}</QueryClientProvider>);
}

describe('StaticListsPage Component', () => {
  it('renders header, search box, and static list table', async () => {
    renderWithQueryClient(<StaticListsPage />);

    expect(screen.getByText('Static Lists Directory')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('VIP Customers List')).toBeInTheDocument();
      expect(screen.getByText('Beta Testers Segment')).toBeInTheDocument();
    });
  });

  it('filters static lists by search term', async () => {
    renderWithQueryClient(<StaticListsPage />);

    await waitFor(() => {
      expect(screen.getByText('VIP Customers List')).toBeInTheDocument();
    });

    const searchInput = screen.getByTestId('static-list-search-input');
    fireEvent.change(searchInput, { target: { value: 'Beta' } });

    expect(screen.queryByText('VIP Customers List')).not.toBeInTheDocument();
    expect(screen.getByText('Beta Testers Segment')).toBeInTheDocument();
  });

  it('opens item inspect modal when inspect button is clicked', async () => {
    renderWithQueryClient(<StaticListsPage />);

    await waitFor(() => {
      expect(screen.getByTestId('inspect-static-list-list-vip-users')).toBeInTheDocument();
    });

    const inspectBtn = screen.getByTestId('inspect-static-list-list-vip-users');
    fireEvent.click(inspectBtn);

    expect(screen.getByTestId('static-list-inspect-modal')).toBeInTheDocument();
    expect(screen.getByText(/user-1@example.com/)).toBeInTheDocument();
  });
});
