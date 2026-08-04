import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, within } from '@testing-library/react';
import { ExperimentReportView } from './ExperimentReportView';
import type { AggregateReportData } from './types';

const testReport: AggregateReportData = {
  schema_version: '1.0',
  report_id: 'rep-test-999',
  tenant_id: 'tenant-test',
  experiment_id: 'exp-hero-banner',
  experiment_name: 'Homepage Hero Banner Test',
  period_start: '2026-07-01T00:00:00Z',
  period_end: '2026-07-27T12:00:00Z',
  generated_at: '2026-07-27T12:00:00Z',
  data_freshness_seconds: 300, // 5 minutes ago
  is_filtered: true,
  srm_status: 'PASSED',
  srm_p_value: 0.521,
  srm_details: 'Sample ratio conforms to expected distribution (p = 0.521).',
  variant_metrics: [
    {
      variant_key: 'control',
      variant_name: 'Banner Original (Control)',
      is_control: true,
      assigned: 10000,
      exposed: 9950,
      attempted: 9950,
      accepted: 9900,
      delivered: 9850,
      unique_open: 4432,
      unique_click: 1772,
      conversion: 985,
      conversion_rate: 0.10, // 10%
      lift_pct: 0.0,
      confidence_interval_95: [9.40, 10.60],
    },
    {
      variant_key: 'variant_hero_b',
      variant_name: 'Banner Modern (Variant B)',
      is_control: false,
      assigned: 10000,
      exposed: 9960,
      attempted: 9960,
      accepted: 9920,
      delivered: 9880,
      unique_open: 5137,
      unique_click: 2272,
      conversion: 1482,
      conversion_rate: 0.15, // 15%
      lift_pct: 50.0, // +50% lift
      confidence_interval_95: [14.30, 15.70],
    },
  ],
};

describe('ExperimentReportView Component', () => {
  it('renders aggregate report dashboard with SRM status, data freshness, and raw vs filtered indicators', () => {
    render(<ExperimentReportView report={testReport} />);

    expect(screen.getByTestId('experiment-report-view')).toBeInTheDocument();
    expect(screen.getByText(/Homepage Hero Banner Test/i)).toBeInTheDocument();
    expect(screen.getByTestId('data-freshness-indicator')).toHaveTextContent('5m ago');
    expect(screen.getByTestId('srm-status-badge')).toHaveTextContent('PASSED');
    expect(screen.getByTestId('raw-vs-filtered-toggle')).toBeInTheDocument();
  });

  it('renders summary metric cards for assigned, exposed, attempted, accepted, delivered, opens, clicks, conversion, lift', () => {
    render(<ExperimentReportView report={testReport} />);

    expect(screen.getByTestId('metric-assigned')).toHaveTextContent('20,000');
    expect(screen.getByTestId('metric-exposed')).toHaveTextContent('19,910');
    expect(screen.getByTestId('metric-attempted')).toHaveTextContent('19,910');
    expect(screen.getByTestId('metric-accepted')).toHaveTextContent('19,820');
    expect(screen.getByTestId('metric-delivered')).toHaveTextContent('19,730');
    expect(screen.getByTestId('metric-unique-open')).toHaveTextContent('9,569');
    expect(screen.getByTestId('metric-unique-click')).toHaveTextContent('4,044');
    expect(screen.getByTestId('metric-conversion')).toHaveTextContent('2,467');
    expect(screen.getByTestId('metric-conversion-rate')).toHaveTextContent('12.5%');
    expect(screen.getByTestId('metric-max-lift')).toHaveTextContent('+50.0%');
  });

  it('renders aggregate breakdown table with 95% confidence intervals and lift %', () => {
    render(<ExperimentReportView report={testReport} />);

    expect(screen.getByTestId('aggregate-table')).toBeInTheDocument();
    const rowVariantB = screen.getByTestId('report-row-variant_hero_b');
    expect(within(rowVariantB).getByText(/Banner Modern \(Variant B\)/i)).toBeInTheDocument();
    expect(within(rowVariantB).getByText(/\+50.0%/i)).toBeInTheDocument();
    expect(within(rowVariantB).getByText(/\[14.30%, 15.70%\]/i)).toBeInTheDocument();
  });

  it('enforces privacy compliance by displaying privacy indicator and omitting contact-level PII or raw rows', () => {
    render(<ExperimentReportView report={testReport} />);

    expect(screen.getByTestId('privacy-indicator')).toHaveTextContent(
      'Privacy Compliant Aggregate Report: No contact-level PII or raw experiment rows are displayed'
    );

    // Verify absence of common PII identifiers or raw row tables
    expect(screen.queryByText(/user@example.com/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/contact_id/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/phone_number/i)).not.toBeInTheDocument();
  });

  it('allows toggling between Filtered Data and Raw Dataset views', () => {
    render(<ExperimentReportView report={testReport} />);

    const rawBtn = screen.getByTestId('raw-view-btn');
    const filteredBtn = screen.getByTestId('filtered-view-btn');

    fireEvent.click(rawBtn);
    expect(screen.getByText(/Variant Performance Breakdown \(Raw Dataset\)/i)).toBeInTheDocument();

    fireEvent.click(filteredBtn);
    expect(screen.getByText(/Variant Performance Breakdown \(Filtered Data\)/i)).toBeInTheDocument();
  });

  it('triggers onRefresh and onExportCSV handlers when buttons are clicked', () => {
    const handleRefresh = vi.fn();
    const handleExport = vi.fn();

    render(<ExperimentReportView report={testReport} onRefresh={handleRefresh} onExportCSV={handleExport} />);

    const refreshBtn = screen.getByRole('button', { name: /Refresh/i });
    fireEvent.click(refreshBtn);
    expect(handleRefresh).toHaveBeenCalledTimes(1);

    const exportBtn = screen.getByRole('button', { name: /Export CSV/i });
    fireEvent.click(exportBtn);
    expect(handleExport).toHaveBeenCalledTimes(1);
  });
});
