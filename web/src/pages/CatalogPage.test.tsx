import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { SafeQueryClientProvider } from '../components/SafeQueryClientProvider';
import { CatalogPage } from './CatalogPage';

describe('CatalogPage Component', () => {
  it('renders catalog header and navigation tabs', async () => {
    render(
      <SafeQueryClientProvider>
        <CatalogPage />
      </SafeQueryClientProvider>
    );

    expect(screen.getByRole('heading', { name: /Component Catalog/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Events Catalog/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Actions Catalog/i })).toBeInTheDocument();
  });

  it('renders paginated table headers when items load', async () => {
    render(
      <SafeQueryClientProvider>
        <CatalogPage />
      </SafeQueryClientProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Component Name & ID')).toBeInTheDocument();
      expect(screen.getByText('Type')).toBeInTheDocument();
      expect(screen.getByText('Version')).toBeInTheDocument();
      expect(screen.getByText('Status')).toBeInTheDocument();
    });
  });

  it('opens inspect modal showing component details when Inspect button is clicked', async () => {
    render(
      <SafeQueryClientProvider>
        <CatalogPage />
      </SafeQueryClientProvider>
    );

    const inspectButtons = await screen.findAllByRole('button', { name: /Inspect/i });
    expect(inspectButtons.length).toBeGreaterThan(0);

    fireEvent.click(inspectButtons[0]);

    await waitFor(() => {
      expect(screen.getByTestId('catalog-inspect-modal')).toBeInTheDocument();
      expect(screen.getByText('Component Type:')).toBeInTheDocument();
      expect(screen.getByText('JSON Schema Definition')).toBeInTheDocument();
    });
  });
  it('opens catalog creation modal when Create Component button is clicked', async () => {
    render(
      <SafeQueryClientProvider>
        <CatalogPage />
      </SafeQueryClientProvider>
    );

    const createBtn = screen.getByTestId('create-component-btn');
    expect(createBtn).toBeInTheDocument();

    fireEvent.click(createBtn);

    await waitFor(() => {
      expect(screen.getByTestId('catalog-creation-modal')).toBeInTheDocument();
      expect(screen.getByText('Create Catalog Component')).toBeInTheDocument();
      expect(screen.getByTestId('catalog-name-input')).toBeInTheDocument();
    });
  });
});
