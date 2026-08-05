import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClientProvider } from '@tanstack/react-query';
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
});
