import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { SafeQueryClientProvider } from '../components/SafeQueryClientProvider';
import { RunListPage } from './RunListPage';

describe('RunListPage Component', () => {
  it('renders execution run directory title and filter controls', async () => {
    render(
      <SafeQueryClientProvider>
        <RunListPage />
      </SafeQueryClientProvider>
    );

    expect(screen.getByRole('heading', { name: /Journey Execution Runs/i })).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Run ID, Workflow ID, or Tenant...')).toBeInTheDocument();
  });
});
