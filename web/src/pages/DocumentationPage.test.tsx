import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { SafeQueryClientProvider } from '../components/SafeQueryClientProvider';
import { DocumentationPage } from './DocumentationPage';

describe('DocumentationPage Component', () => {
  it('renders documentation header, tabs, and html rendered markdown content', async () => {
    render(
      <SafeQueryClientProvider>
        <DocumentationPage />
      </SafeQueryClientProvider>
    );

    expect(screen.getByText('User Documentation & Architecture Guide')).toBeInTheDocument();
    expect(screen.getByTestId('docs-tab-overview')).toBeInTheDocument();
    expect(screen.getByTestId('docs-tab-pages')).toBeInTheDocument();
    expect(screen.getByTestId('docs-tab-components')).toBeInTheDocument();

    const contentArea = screen.getByTestId('rendered-docs-content');
    expect(contentArea).toBeInTheDocument();
    expect(contentArea).toHaveClass('docs-markdown-theme');

    await waitFor(() => {
      expect(screen.getByText('User Guide: Event-Driven Customer Journey Engine')).toBeInTheDocument();
    });
  });

  it('switches documentation section tabs on tab click', async () => {
    render(
      <SafeQueryClientProvider>
        <DocumentationPage />
      </SafeQueryClientProvider>
    );

    const pagesTab = screen.getByTestId('docs-tab-pages');
    fireEvent.click(pagesTab);

    await waitFor(() => {
      expect(screen.getByText('User Guide: Application Pages & Views')).toBeInTheDocument();
    });
  });

  it('filters sections and highlights search terms when text is typed in search input', async () => {
    render(
      <SafeQueryClientProvider>
        <DocumentationPage />
      </SafeQueryClientProvider>
    );

    const searchInput = screen.getByTestId('docs-search-input');
    fireEvent.change(searchInput, { target: { value: 'Canvas' } });

    await waitFor(() => {
      const contentArea = screen.getByTestId('rendered-docs-content');
      expect(contentArea.innerHTML).toContain('<mark');
    });
  });
});
