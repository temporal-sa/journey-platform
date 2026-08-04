import { render } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { axe } from 'vitest-axe';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import App from '../App';
import JourneysPage from '../pages/JourneysPage';
import RunDetailPage from '../pages/RunDetailPage';
import { ParameterModal } from '../components/inspector/ParameterModal';
import { ConflictDialog } from '../components/inspector/ConflictDialog';
import { KeyboardShortcutsModal } from '../components/KeyboardShortcutsModal';
import { TestRunModal } from '../components/testlane/TestRunModal';
import { StaticListUploadModal } from '../components/testlane/StaticListUploadModal';
import { ExperimentInspector } from '../components/experiments/ExperimentInspector';
import { ExperimentReportView } from '../components/experiments/ExperimentReportView';
import { DegradedStateView } from '../components/DegradedStateView';

function renderWithQueryClient(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe('Accessibility Test Suite (axe-core)', () => {
  describe('Primary Routes', () => {
    it('App root route passes accessibility checks', async () => {
      const { container } = renderWithQueryClient(<App />);
      const results = await axe(container);
      expect(results).toHaveNoViolations();
    });

    it('JourneysPage passes accessibility checks', async () => {
      const { container } = renderWithQueryClient(<JourneysPage />);
      const results = await axe(container);
      expect(results).toHaveNoViolations();
    });

    it('RunDetailPage passes accessibility checks', async () => {
      const { container } = renderWithQueryClient(<RunDetailPage runId="run-601" />);
      const results = await axe(container);
      expect(results).toHaveNoViolations();
    });
  });

  describe('Modal States', () => {
    it('ParameterModal passes accessibility checks when open', async () => {
      const { container } = render(
        <ParameterModal
          isOpen={true}
          onClose={vi.fn()}
          onSelectToken={vi.fn()}
          targetFieldLabel="Recipient Email"
        />
      );
      const results = await axe(container);
      expect(results).toHaveNoViolations();
    });

    it('ConflictDialog passes accessibility checks when open', async () => {
      const { container } = render(
        <ConflictDialog
          isOpen={true}
          localDraft={null}
          serverDraft={null}
          onKeepLocal={vi.fn()}
          onAcceptServer={vi.fn()}
          onCancel={vi.fn()}
        />
      );
      const results = await axe(container);
      expect(results).toHaveNoViolations();
    });

    it('KeyboardShortcutsModal passes accessibility checks when open', async () => {
      const { container } = render(
        <KeyboardShortcutsModal isOpen={true} onClose={vi.fn()} />
      );
      const results = await axe(container);
      expect(results).toHaveNoViolations();
    });

    it('TestRunModal passes accessibility checks when open', async () => {
      const { container } = render(
        <TestRunModal isOpen={true} onClose={vi.fn()} onStartTestRun={vi.fn()} />
      );
      const results = await axe(container);
      expect(results).toHaveNoViolations();
    });

    it('StaticListUploadModal passes accessibility checks when open', async () => {
      const { container } = render(
        <StaticListUploadModal isOpen={true} onClose={vi.fn()} />
      );
      const results = await axe(container);
      expect(results).toHaveNoViolations();
    });
  });

  describe('Complex Views & State Banner Components', () => {
    it('ExperimentInspector passes accessibility checks', async () => {
      const { container } = render(<ExperimentInspector />);
      const results = await axe(container);
      expect(results).toHaveNoViolations();
    });

    it('ExperimentReportView passes accessibility checks', async () => {
      const { container } = render(<ExperimentReportView />);
      const results = await axe(container);
      expect(results).toHaveNoViolations();
    });

    it('DegradedStateView passes accessibility checks for error state', async () => {
      const { container } = render(
        <DegradedStateView type="api-disconnected" onRetry={vi.fn()} />
      );
      const results = await axe(container);
      expect(results).toHaveNoViolations();
    });

    it('DegradedStateView passes accessibility checks for warning state', async () => {
      const { container } = render(
        <DegradedStateView type="stale-draft-revision" onRefresh={vi.fn()} />
      );
      const results = await axe(container);
      expect(results).toHaveNoViolations();
    });
  });
});
