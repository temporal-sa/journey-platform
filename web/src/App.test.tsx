import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import App from './App';
import { ErrorBoundary } from './components/ErrorBoundary';
import { LoadingState } from './components/LoadingState';
import { EmptyState } from './components/EmptyState';
import { useEditorStore } from './stores/editorStore';

describe('App & Foundation Components', () => {
  beforeEach(() => {
    useEditorStore.getState().resetStore();
  });

  it('renders LoadingState component correctly', () => {
    render(<LoadingState message="Fetching data..." size="large" />);
    expect(screen.getByRole('status')).toBeInTheDocument();
    expect(screen.getByText('Fetching data...')).toBeInTheDocument();
  });

  it('renders EmptyState component with title and action', () => {
    render(
      <EmptyState
        title="No Items Found"
        description="Try adjusting your filter settings."
        action={<button>Create Item</button>}
      />
    );
    expect(screen.getByRole('region')).toBeInTheDocument();
    expect(screen.getByText('No Items Found')).toBeInTheDocument();
    expect(screen.getByText('Try adjusting your filter settings.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create Item' })).toBeInTheDocument();
  });

  it('catches runtime errors in ErrorBoundary and renders fallback UI', () => {
    const ProblematicComponent = () => {
      throw new Error('Test boundary trigger');
    };

    // Suppress console.error during expected error boundary test
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    render(
      <ErrorBoundary>
        <ProblematicComponent />
      </ErrorBoundary>
    );

    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.getByText('Test boundary trigger')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Try Again' })).toBeInTheDocument();

    consoleSpy.mockRestore();
  });

  it('renders App component with QueryClientProvider and top header', async () => {
    render(<App />);

    expect(screen.getByText('Journey Control Engine')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Journeys Directory' })).toBeInTheDocument();
    });
  });

  it('interacts with navigation routes, modals, and inspector state in App', async () => {
    render(<App />);

    // Switch navigation route to Catalog
    const catalogNavBtn = screen.getByRole('button', { name: 'Component Catalog' });
    fireEvent.click(catalogNavBtn);
    expect(screen.getByText('Component Catalog & Schemas')).toBeInTheDocument();

    // Switch navigation route to Canvas & test Save Draft modal
    const canvasNavBtn = screen.getByRole('button', { name: 'Journey Canvas' });
    fireEvent.click(canvasNavBtn);

    const saveBtn = screen.getByRole('button', { name: 'Save Draft' });
    await act(async () => {
      fireEvent.click(saveBtn);
    });

    expect(screen.getByTestId('app-toast-notification')).toBeInTheDocument();
    expect(screen.getByText('Draft Saved Successfully')).toBeInTheDocument();
  });

  it('navigates to Experiment Analytics route and syncs location hash', async () => {
    window.location.hash = '';
    render(<App />);

    const expNavBtn = screen.getByRole('button', { name: 'Experiment Analytics' });
    fireEvent.click(expNavBtn);

    expect(screen.getByTestId('experiment-report-view')).toBeInTheDocument();
    expect(window.location.hash).toBe('#/experiments');
  });

  it('initializes activeRoute from window.location.hash and handles hashchange event', async () => {
    window.location.hash = '#/experiments';
    render(<App />);

    expect(screen.getByTestId('experiment-report-view')).toBeInTheDocument();

    // Trigger hashchange event
    window.location.hash = '#/catalog';
    fireEvent(window, new Event('hashchange'));

    await waitFor(() => {
      expect(screen.getByText('Component Catalog & Schemas')).toBeInTheDocument();
    });
  });

  it('renders dynamic build tag in sidebar header', () => {
    render(<App />);
    const versionTag = screen.getByTestId('sidebar-version-tag');

    expect(versionTag).toBeInTheDocument();
    expect(versionTag.textContent).toContain('ui-improvements-obsidian-feat/ui-improvements-obsidian-');
  });

  it('triggers error toast when publishing workflow during simulated API failure mode', async () => {
    const { setSimulatedApiFailureEnabled } = await import('./api/simulatedFailure');
    setSimulatedApiFailureEnabled(true);

    try {
      render(<App />);
      const canvasNavBtn = screen.getByRole('button', { name: 'Journey Canvas' });
      fireEvent.click(canvasNavBtn);

      const publishBtn = screen.getByTestId('toolbar-publish');
      await act(async () => {
        fireEvent.click(publishBtn);
      });

      expect(screen.getByTestId('app-toast-notification')).toBeInTheDocument();
      expect(screen.getByText('Failed to Publish Workflow')).toBeInTheDocument();
    } finally {
      setSimulatedApiFailureEnabled(false);
    }
  });

  it('triggers error toast when launching test run during simulated API failure mode', async () => {
    const { setSimulatedApiFailureEnabled } = await import('./api/simulatedFailure');
    setSimulatedApiFailureEnabled(true);

    try {
      render(<App />);
      const canvasNavBtn = screen.getByRole('button', { name: 'Journey Canvas' });
      fireEvent.click(canvasNavBtn);

      const testRunBtn = screen.getByRole('button', { name: 'Launch Test Execution' });
      fireEvent.click(testRunBtn);

      const launchBtn = screen.getByRole('button', { name: /Execute Test Run/i });
      await act(async () => {
        fireEvent.click(launchBtn);
      });

      expect(screen.getByTestId('app-toast-notification')).toBeInTheDocument();
      expect(screen.getByText('Failed to Launch Test Execution')).toBeInTheDocument();
    } finally {
      setSimulatedApiFailureEnabled(false);
    }
  });

  it('triggers error toast when renaming journey draft during simulated API failure mode', async () => {
    const { setSimulatedApiFailureEnabled } = await import('./api/simulatedFailure');
    setSimulatedApiFailureEnabled(true);

    try {
      render(<App />);
      const canvasNavBtn = screen.getByRole('button', { name: 'Journey Canvas' });
      fireEvent.click(canvasNavBtn);

      const nameBtn = screen.getByRole('button', { name: /Journey name:/i });
      fireEvent.click(nameBtn);

      const input = screen.getByTestId('inline-journey-name-input');
      fireEvent.change(input, { target: { value: 'Renamed Journey' } });
      await act(async () => {
        fireEvent.blur(input);
      });

      expect(screen.getByTestId('app-toast-notification')).toBeInTheDocument();
      expect(screen.getByText('Failed to Rename Journey')).toBeInTheDocument();
    } finally {
      setSimulatedApiFailureEnabled(false);
    }
  });

  it('triggers success toast when renaming journey draft successfully', async () => {
    render(<App />);
    const canvasNavBtn = screen.getByRole('button', { name: 'Journey Canvas' });
    fireEvent.click(canvasNavBtn);

    const nameBtn = screen.getByRole('button', { name: /Journey name:/i });
    fireEvent.click(nameBtn);

    const input = screen.getByTestId('inline-journey-name-input');
    fireEvent.change(input, { target: { value: 'New Onboarding Flow' } });
    await act(async () => {
      fireEvent.blur(input);
    });

    expect(screen.getByTestId('app-toast-notification')).toBeInTheDocument();
    expect(screen.getByText('Journey Renamed Successfully')).toBeInTheDocument();
  });
});
