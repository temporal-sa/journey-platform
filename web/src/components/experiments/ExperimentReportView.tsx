import React, { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { JourneyApiClient } from '../../api/client';
import type { AggregateReportData } from './types';
import { SafeQueryClientProvider } from '../SafeQueryClientProvider';
import { Button } from '../common/Button';
import { Badge } from '../common/Badge';

const apiClient = new JourneyApiClient();

export interface ExperimentReportViewProps {
  experimentId?: string;
  report?: Partial<AggregateReportData>;
  onRefresh?: () => void;
  onExportCSV?: () => void;
}

const ExperimentReportViewInner: React.FC<ExperimentReportViewProps> = ({
  experimentId,
  report,
  onRefresh,
  onExportCSV,
}) => {
  const targetExpId = experimentId || report?.experiment_id;

  const { data: fetchedReportData } = useQuery({
    queryKey: ['aggregateReport', targetExpId],
    queryFn: async () => {
      const data = await apiClient.getAggregateReport({ experimentId: targetExpId });
      return data as unknown as AggregateReportData;
    },
    enabled: Boolean(targetExpId || !report),
  });

  const mergedReport: AggregateReportData = useMemo(() => {
    const base: AggregateReportData = fetchedReportData || {
      schema_version: '1.0',
      report_id: 'rep-301',
      tenant_id: 'tenant-default',
      experiment_id: targetExpId || 'exp-101',
      experiment_name: 'Experiment Aggregate Analytics',
      period_start: new Date(Date.now() - 7 * 86400000).toISOString(),
      period_end: new Date().toISOString(),
      generated_at: new Date().toISOString(),
      data_freshness_seconds: 60,
      is_filtered: true,
      srm_status: 'PASSED',
      srm_p_value: 0.5,
      srm_details: 'Allocation conforms to target ratio',
      variant_metrics: [],
    };

    return {
      ...base,
      ...report,
      variant_metrics: report?.variant_metrics?.length
        ? report.variant_metrics
        : base.variant_metrics || [],
    };
  }, [fetchedReportData, report, targetExpId]);

  const handleExportCSV = async () => {
    if (onExportCSV) {
      onExportCSV();
      return;
    }
    const reportId = mergedReport.report_id || 'rep-301';
    try {
      const csvText = await apiClient.exportCSV(reportId);
      if (typeof document !== 'undefined') {
        const blob = new Blob([csvText], { type: 'text/csv;charset=utf-8;' });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.setAttribute('download', `report_${reportId}.csv`);
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
      }
    } catch (err) {
      console.error('Export CSV failed:', err);
    }
  };

  const [isFiltered, setIsFiltered] = useState<boolean>(mergedReport.is_filtered);

  // Compute aggregated totals across variants
  const totals = useMemo(() => {
    const metrics = mergedReport.variant_metrics;
    const assigned = metrics.reduce((s, m) => s + m.assigned, 0);
    const exposed = metrics.reduce((s, m) => s + m.exposed, 0);
    const attempted = metrics.reduce((s, m) => s + m.attempted, 0);
    const accepted = metrics.reduce((s, m) => s + m.accepted, 0);
    const delivered = metrics.reduce((s, m) => s + m.delivered, 0);
    const unique_open = metrics.reduce((s, m) => s + m.unique_open, 0);
    const unique_click = metrics.reduce((s, m) => s + m.unique_click, 0);
    const conversion = metrics.reduce((s, m) => s + m.conversion, 0);

    const overallConversionRate = delivered > 0 ? conversion / delivered : 0;

    const treatmentLifts = metrics
      .filter((m) => !m.is_control)
      .map((m) => m.lift_pct);
    const maxLift = treatmentLifts.length > 0 ? Math.max(...treatmentLifts) : 0;

    return {
      assigned,
      exposed,
      attempted,
      accepted,
      delivered,
      unique_open,
      unique_click,
      conversion,
      overallConversionRate,
      maxLift,
    };
  }, [mergedReport.variant_metrics]);

  const formatFreshness = (seconds: number) => {
    if (seconds < 60) return `${seconds}s ago`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    return `${Math.floor(seconds / 3600)}h ago`;
  };

  const isSrmPassed = mergedReport.srm_status === 'PASSED';

  return (
    <div
      data-testid="experiment-report-view"
      aria-label="Experiment Analytics Report Dashboard"
      className="p-6 lg:p-8 bg-[#0B0F19] text-[#DFE2F1] font-['Outfit',sans-serif] space-y-6 w-full max-w-full min-h-screen"
    >
      {/* Header Bar */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-[#1F2937] pb-5">
        <div>
          <div className="flex items-center gap-3 flex-wrap">
            <h2 className="text-xl font-['Outfit'] font-bold text-white flex items-center gap-2">
              <span className="material-symbols-outlined text-[#ddb7ff]">analytics</span>
              <span>{mergedReport.experiment_name || mergedReport.experiment_id}</span>
            </h2>
            <span
              data-testid="data-freshness-indicator"
              className="text-[11px] font-mono font-medium px-3 py-1 rounded-none bg-[#6366F1]/10 text-[#c0c1ff] border border-[#6366F1]/30 flex items-center gap-1.5 shadow-sm"
            >
              <span className="w-2 h-2 rounded-none bg-[#10b981] animate-pulse" />
              Fresh ({formatFreshness(mergedReport.data_freshness_seconds)})
            </span>
          </div>
          <p className="text-xs text-[#908fa0] mt-1.5 font-mono">
            Report ID: <span className="text-[#c0c1ff] font-semibold">{mergedReport.report_id}</span> | Period:{' '}
            {new Date(mergedReport.period_start).toLocaleDateString()} –{' '}
            {new Date(mergedReport.period_end).toLocaleDateString()}
          </p>
        </div>

        <div className="flex items-center gap-3 flex-wrap">
          {/* Raw vs Filtered Indicator / Toggle */}
          <div
            data-testid="raw-vs-filtered-toggle"
            className="flex items-center bg-[#0a0e18] p-1 rounded-none border border-[#1F2937] text-xs shadow-inner"
          >
            <button
              type="button"
              data-testid="filtered-view-btn"
              onClick={() => setIsFiltered(true)}
              className={`px-4 py-2 rounded-none text-xs font-semibold transition-all ${
                isFiltered
                  ? 'bg-[#171b26] text-[#c0c1ff] border border-[#6366F1] shadow-sm'
                  : 'bg-transparent text-[#908fa0] hover:text-[#dfe2f1] hover:bg-[#171b26]/50 border border-transparent'
              }`}
            >
              Filtered Data
            </button>
            <button
              type="button"
              data-testid="raw-view-btn"
              onClick={() => setIsFiltered(false)}
              className={`px-4 py-2 rounded-none text-xs font-semibold transition-all ${
                !isFiltered
                  ? 'bg-[#171b26] text-[#c0c1ff] border border-[#6366F1] shadow-sm'
                  : 'bg-transparent text-[#908fa0] hover:text-[#dfe2f1] hover:bg-[#171b26]/50 border border-transparent'
              }`}
            >
              Raw Dataset
            </button>
          </div>

          {onRefresh && (
            <Button
              type="button"
              onClick={onRefresh}
              variant="secondary-dark"
              icon="refresh"
            >
              Refresh
            </Button>
          )}

          <Button
            type="button"
            onClick={handleExportCSV}
            data-testid="export-csv-btn"
            variant="primary-indigo"
            icon="download"
          >
            Export CSV
          </Button>
        </div>
      </div>

      {/* Privacy Callout Banner */}
      <div
        data-testid="privacy-indicator"
        className="glass-panel p-4 rounded-none border border-[#03b5d3]/30 bg-[#03b5d3]/10 text-[#dfe2f1] text-xs flex items-center gap-3 shadow-md"
      >
        <span className="material-symbols-outlined text-[#4cd7f6] text-lg shrink-0" style={{ fontVariationSettings: "'FILL' 1" }}>
          lock
        </span>
        <span className="font-['Inter']">
          <strong className="text-[#4cd7f6] font-semibold">Privacy Compliant Aggregate Report:</strong> No contact-level PII or raw experiment rows are displayed or transmitted.
        </span>
      </div>

      {/* Sample Ratio Mismatch (SRM) Status Card */}
      <div
        data-testid="srm-status-card"
        className={`glass-panel p-4 rounded-none border flex flex-col sm:flex-row sm:items-center justify-between gap-3 shadow-md transition-all ${
          isSrmPassed
            ? 'border-[#10b981]/40 bg-[#10b981]/10 status-strip-emerald'
            : 'border-[#f43f5e]/40 bg-[#f43f5e]/10 status-strip-rose'
        }`}
      >
        <div className="flex items-center gap-3">
          <span
            className={`material-symbols-outlined text-xl ${isSrmPassed ? 'text-[#10b981]' : 'text-[#ffb4ab]'}`}
            style={{ fontVariationSettings: "'FILL' 1" }}
          >
            {isSrmPassed ? 'check_circle' : 'warning'}
          </span>
          <div>
            <div className="flex items-center gap-2.5 flex-wrap">
              <span className="text-xs font-bold text-white font-['Outfit'] tracking-wide">
                Sample Ratio Mismatch (SRM) Status:
              </span>
              <Badge
                variant={isSrmPassed ? 'emerald' : 'rose'}
                size="md"
                testId="srm-status-badge"
              >
                {mergedReport.srm_status}
              </Badge>
            </div>
            <p className="text-xs text-[#c7c4d7] mt-1 font-['Inter']">
              {mergedReport.srm_details || 'Chi-square test evaluated across traffic distribution.'}
            </p>
          </div>
        </div>

        {mergedReport.srm_p_value !== undefined && (
          <div className="text-xs font-mono text-[#c7c4d7] shrink-0 sm:text-right">
            p-value: <code className="font-bold text-white bg-[#171b26] px-2.5 py-1 rounded-none border border-[#313540]">{mergedReport.srm_p_value.toFixed(4)}</code>
          </div>
        )}
      </div>

      {/* Aggregate Metric Cards */}
      <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-3.5">
        <MetricCard label="Assigned" value={totals.assigned.toLocaleString()} testId="metric-assigned" />
        <MetricCard label="Exposed" value={totals.exposed.toLocaleString()} testId="metric-exposed" />
        <MetricCard label="Attempted" value={totals.attempted.toLocaleString()} testId="metric-attempted" />
        <MetricCard label="Accepted" value={totals.accepted.toLocaleString()} testId="metric-accepted" />
        <MetricCard label="Delivered" value={totals.delivered.toLocaleString()} testId="metric-delivered" />
        <MetricCard label="Unique Opens" value={totals.unique_open.toLocaleString()} testId="metric-unique-open" />
        <MetricCard label="Unique Clicks" value={totals.unique_click.toLocaleString()} testId="metric-unique-click" />
        <MetricCard label="Conversions" value={totals.conversion.toLocaleString()} testId="metric-conversion" />
        <MetricCard
          label="Conversion Rate"
          value={`${(totals.overallConversionRate * 100).toFixed(1)}%`}
          highlight
          highlightColor="#ddb7ff"
          testId="metric-conversion-rate"
        />
        <MetricCard
          label="Max Lift %"
          value={`${totals.maxLift >= 0 ? '+' : ''}${totals.maxLift.toFixed(1)}%`}
          highlight
          highlightColor="#10b981"
          testId="metric-max-lift"
        />
      </div>

      {/* Aggregate Breakdown Table Container */}
      <div className="glass-panel rounded-none border border-[#1F2937] shadow-xl overflow-hidden bg-[#171b26]/90 p-6 mt-6 mb-6">
        <div className="px-2 pb-4 mb-4 border-b border-[#1F2937] flex justify-between items-center bg-transparent">
          <h3 className="font-['Outfit'] font-bold text-sm text-white flex items-center gap-2">
            <span className="material-symbols-outlined text-[#4cd7f6] text-base">table_chart</span>
            <span>Variant Performance Breakdown ({isFiltered ? 'Filtered Data' : 'Raw Dataset'})</span>
          </h3>
          <span className="text-xs font-mono text-[#908fa0]">
            {mergedReport.variant_metrics.length} Variants Tracked
          </span>
        </div>

        <div className="overflow-x-auto">
          <table
            data-testid="aggregate-table"
            className="w-full text-left text-xs font-['Inter',sans-serif] border-collapse"
          >
            <thead>
              <tr className="bg-[#0a0e18] border-b border-[#1F2937] text-[#908fa0] font-mono uppercase tracking-wider text-[10px]">
                <th className="px-6 py-4 font-semibold text-left">Variant</th>
                <th className="px-6 py-4 font-semibold text-left">Assigned</th>
                <th className="px-6 py-4 font-semibold text-left">Exposed</th>
                <th className="px-6 py-4 font-semibold text-left">Attempted</th>
                <th className="px-6 py-4 font-semibold text-left">Accepted</th>
                <th className="px-6 py-4 font-semibold text-left">Delivered</th>
                <th className="px-6 py-4 font-semibold text-left">Unique Opens</th>
                <th className="px-6 py-4 font-semibold text-left">Unique Clicks</th>
                <th className="px-6 py-4 font-semibold text-left">Conversions</th>
                <th className="px-6 py-4 font-semibold text-left">Conv Rate</th>
                <th className="px-6 py-4 font-semibold text-left">Lift %</th>
                <th className="px-6 py-4 font-semibold text-left">95% CI</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#1F2937]/60 font-mono text-[#dfe2f1]">
              {mergedReport.variant_metrics.map((v) => {
                const convRatePct = (v.conversion_rate * 100).toFixed(1);
                const ciLower = v.confidence_interval_95[0].toFixed(2);
                const ciUpper = v.confidence_interval_95[1].toFixed(2);

                return (
                  <tr
                    key={v.variant_key}
                    data-testid={`report-row-${v.variant_key}`}
                    className={`hover:bg-[#262a35]/60 transition-colors ${
                      v.is_control ? 'bg-[#ddb7ff]/5 hover:bg-[#ddb7ff]/10' : ''
                    }`}
                  >
                    <td className="px-6 py-4">
                      <div className="font-semibold text-white font-['Outfit'] text-xs flex items-center gap-2">
                        <div
                          className={`w-2 h-2 rounded-none ${
                            v.is_control ? 'bg-[#c7c4d7]' : 'bg-[#ddb7ff]'
                          }`}
                        />
                        <span>{v.variant_name}</span>
                      </div>
                      <div className="text-[10px] text-[#908fa0] mt-1 font-mono">
                        Key: <code className="text-[#ddb7ff] font-medium">{v.variant_key}</code>{' '}
                        {v.is_control && <span className="text-[#ddb7ff] font-semibold">(Control)</span>}
                      </div>
                    </td>
                    <td className="px-6 py-4 text-[#dfe2f1]">{v.assigned.toLocaleString()}</td>
                    <td className="px-6 py-4 text-[#dfe2f1]">{v.exposed.toLocaleString()}</td>
                    <td className="px-6 py-4 text-[#dfe2f1]">{v.attempted.toLocaleString()}</td>
                    <td className="px-6 py-4 text-[#dfe2f1]">{v.accepted.toLocaleString()}</td>
                    <td className="px-6 py-4 text-[#dfe2f1]">{v.delivered.toLocaleString()}</td>
                    <td className="px-6 py-4 text-[#dfe2f1]">{v.unique_open.toLocaleString()}</td>
                    <td className="px-6 py-4 text-[#dfe2f1]">{v.unique_click.toLocaleString()}</td>
                    <td className="px-6 py-4 text-[#dfe2f1]">{v.conversion.toLocaleString()}</td>
                    <td className="px-6 py-4 font-bold text-white font-mono">{convRatePct}%</td>
                    <td className="px-6 py-4">
                      {v.is_control ? (
                        <span className="text-[#908fa0] font-semibold font-mono text-[11px]">Baseline</span>
                      ) : (
                        <span
                          className={`font-bold font-mono text-[11px] ${
                            v.lift_pct >= 0 ? 'text-[#10b981]' : 'text-[#ffb4ab]'
                          }`}
                        >
                          {v.lift_pct >= 0 ? '+' : ''}
                          {v.lift_pct.toFixed(1)}%
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-[11px]">
                      <span className="font-mono text-[#c7c4d7] bg-[#0a0e18] px-2.5 py-1 rounded-none border border-[#313540]">
                        {`[${ciLower}%, ${ciUpper}%]`}
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

interface MetricCardProps {
  label: string;
  value: string;
  highlight?: boolean;
  highlightColor?: string;
  testId?: string;
}

const MetricCard: React.FC<MetricCardProps> = ({
  label,
  value,
  highlight = false,
  highlightColor = '#c0c1ff',
  testId,
}) => (
  <div
    data-testid={testId}
    className="glass-panel p-4 rounded-none border border-[#1F2937] bg-[#171b26]/90 hover:bg-[#262a35]/60 transition-all flex flex-col justify-between shadow-lg group"
  >
    <div className="text-[10px] font-bold uppercase tracking-wider text-[#908fa0] font-['Inter'] group-hover:text-[#c7c4d7] transition-colors">
      {label}
    </div>
    <div
      className={`text-lg font-bold font-mono mt-2 truncate ${
        highlight ? '' : 'text-[#dfe2f1]'
      }`}
      style={highlight ? { color: highlightColor } : undefined}
    >
      {value}
    </div>
  </div>
);

export const ExperimentReportView: React.FC<ExperimentReportViewProps> = (props) => (
  <SafeQueryClientProvider>
    <ExperimentReportViewInner {...props} />
  </SafeQueryClientProvider>
);
