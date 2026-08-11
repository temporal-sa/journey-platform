import React, { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { JourneyApiClient } from '../api/client';
import { Skeleton } from '../components/Skeleton';
import { DegradedStateView } from '../components/DegradedStateView';
import { Button } from '../components/common/Button';
import { PaginatedTable, ColumnDef } from '../components/common/PaginatedTable';
import { DirectoryLayout } from '../components/common/DirectoryLayout';
import { StatusFilterDropdown } from '../components/common/StatusFilterDropdown';
import { useRouteParams } from '../hooks/useRouteParams';
import type { RunProjection } from '../types/api';

const apiClient = new JourneyApiClient();

export interface ExtendedRunItem extends RunProjection {
  execution_mode: 'production' | 'test';
}


interface RunListPageProps {
  onSelectRun?: (runId: string) => void;
}

const DEFAULT_RUN_LIST_PARAMS = {
  mode: 'all',
  status: 'all',
  search: '',
  startDate: '',
  endDate: '',
  page: '1',
};

export const RunListPage: React.FC<RunListPageProps> = ({ onSelectRun }) => {
  const [params, setParams] = useRouteParams(DEFAULT_RUN_LIST_PARAMS);

  const {
    data: fetchedRuns,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['runs', 'list', params.status, params.mode],
    queryFn: async () => {
      const res = await apiClient.listJourneyRuns({
        status: params.status !== 'all' ? params.status : undefined,
      });
      if (!res) return [];
      const mapped: ExtendedRunItem[] = res.map((r) => {
        const vars = r.variables as Record<string, unknown> | undefined;
        const rawMode = r.execution_mode || (vars?.execution_mode as string) || (vars?.data as Record<string, unknown> | undefined)?.execution_mode;
        const isTest = rawMode === 'test' || rawMode === 'realistic' || rawMode === 'forced_variant_coverage' || r.run_id.includes('testrun') || r.run_id.startsWith('tr-');
        const execution_mode: 'production' | 'test' = isTest ? 'test' : 'production';
        return {
          ...r,
          execution_mode,
        };
      });
      return mapped;
    },
  });

  const runs: ExtendedRunItem[] = fetchedRuns || [];

  // Filter Logic (Execution mode, status, search, date range)
  const filteredRuns = runs.filter((run) => {
    const matchesMode = params.mode === 'all' || run.execution_mode === params.mode;
    const matchesStatus = params.status === 'all' || run.status === params.status;
    const matchesSearch =
      !params.search ||
      run.run_id.toLowerCase().includes(params.search.toLowerCase()) ||
      run.workflow_id.toLowerCase().includes(params.search.toLowerCase()) ||
      run.tenant_id.toLowerCase().includes(params.search.toLowerCase());

    const runStart = new Date(run.started_at).getTime();
    const afterStart = !params.startDate || runStart >= new Date(params.startDate).getTime();
    const beforeEnd = !params.endDate || runStart <= new Date(`${params.endDate}T23:59:59`).getTime();

    return matchesMode && matchesStatus && matchesSearch && afterStart && beforeEnd;
  });

  const getModeStyles = (mode: string) => {
    const m = mode.toLowerCase();
    if (m === 'production') return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20';
    if (m === 'test') return 'bg-amber-500/10 text-amber-400 border-amber-500/20';
    return 'bg-slate-800 text-slate-300 border-slate-700';
  };

  const getStatusStyles = (status: string) => {
    const s = status.toLowerCase();
    if (s === 'completed') return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20';
    if (s === 'running') return 'bg-blue-500/10 text-blue-400 border-blue-500/20';
    if (s === 'failed') return 'bg-rose-500/10 text-rose-400 border-rose-500/20';
    if (s === 'terminated') return 'bg-purple-500/10 text-purple-300 border-purple-500/20';
    return 'bg-slate-800 text-slate-300 border-slate-700';
  };

  const runColumns: ColumnDef<ExtendedRunItem>[] = useMemo(
    () => [
      {
        key: 'run_id',
        header: 'Run ID',
        cell: (run) => (
          <span className="font-mono text-[#dfe2f1] text-xs">
            {run.run_id}
          </span>
        ),
      },
      {
        key: 'workflow_id',
        header: 'Workflow ID',
        cell: (run) => <span className="text-white font-medium">{run.workflow_id}</span>,
      },
      {
        key: 'execution_mode',
        header: 'Mode',
        cell: (run) => (
          <span
            className={`px-2 py-0.5 rounded-none text-[10px] font-mono font-semibold uppercase border tracking-wider ${
              run.execution_mode === 'production'
                ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
            }`}
          >
            {run.execution_mode}
          </span>
        ),
      },
      {
        key: 'status',
        header: 'Status',
        cell: (run) => (
          <span className={`px-2 py-0.5 rounded-none text-[10px] font-mono font-semibold uppercase border tracking-wider ${getStatusStyles(run.status)}`}>
            {run.status}
          </span>
        ),
      },
      {
        key: 'current_nodes',
        header: 'Current Active Node',
        cell: (run) => (
          <span className="font-mono text-[#c0c1ff]">
            {run.current_nodes?.join(', ') || 'N/A'}
          </span>
        ),
      },
      {
        key: 'started_at',
        header: 'Started At',
        cell: (run) => (
          <span className="text-[#908fa0] text-xs font-mono">
            {new Date(run.started_at).toLocaleString()}
          </span>
        ),
      },
      {
        key: 'actions',
        header: 'Actions',
        cell: (run) => (
          <Button
            variant="secondary-dark"
            size="sm"
            icon="visibility"
            onClick={() => onSelectRun?.(run.run_id)}
            className="bg-[#b76dff]/20 hover:bg-[#b76dff]/40 text-[#ddb7ff] border-[#ddb7ff]/30 font-semibold"
          >
            Inspect
          </Button>
        ),
      },
    ],
    [onSelectRun]
  );

  return (
    <DirectoryLayout
      title="Journey Execution Runs"
      subtitle="Inspect active execution state machines, production traces, and test scenario runs."
      icon="route"
      iconAccentColor="#34d399"
      controls={
        <>
          {/* Execution Mode Selector */}
          <div>
            <label className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
              Execution Mode Filter
            </label>
            <div className="flex flex-wrap items-center gap-1.5 h-[34px] bg-[#171b26] px-1.5 rounded-none border border-[#464554]" role="group" aria-label="Execution Mode Filter">
              {[
                { mode: 'all', label: 'All Modes' },
                { mode: 'production', label: 'Production' },
                { mode: 'test', label: 'Test' },
              ].map((item) => {
                const isActive = params.mode === item.mode;
                return (
                  <button
                    key={item.mode}
                    type="button"
                    onClick={() => setParams({ mode: item.mode, page: '1' })}
                    className={`px-2.5 py-1 rounded-none text-[10px] font-mono font-semibold uppercase border tracking-wider transition-all cursor-pointer ${getModeStyles(item.mode)} ${
                      isActive
                        ? 'ring-2 ring-current font-bold shadow-md scale-105 opacity-100'
                        : 'opacity-70 hover:opacity-100 hover:brightness-125'
                    }`}
                  >
                    {item.label}
                  </button>
                );
              })}
            </div>
          </div>

          {/* Status Filter Selector */}
          <StatusFilterDropdown
            label="Run Status Filter"
            value={params.status}
            onChange={(newStatus) => setParams({ status: newStatus, page: '1' })}
            options={[
              { status: 'all', label: 'All Statuses' },
              { status: 'running', label: 'Running' },
              { status: 'completed', label: 'Completed' },
              { status: 'failed', label: 'Failed' },
              { status: 'terminated', label: 'Terminated' },
            ]}
            getStatusStyles={getStatusStyles}
            dataTestId="runs-status-filter-dropdown"
          />

          {/* Date Range Inputs */}
          <div className="flex gap-2 items-center">
            <div>
              <label className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
                Start Date
              </label>
              <input
                type="date"
                aria-label="Start Date"
                value={params.startDate}
                onChange={(e) => setParams({ startDate: e.target.value, page: '1' })}
                className="h-[34px] px-3 rounded-none bg-[#171b26] border border-[#464554] text-white text-xs focus:outline-none focus:border-[#c0c1ff] font-['Outfit',sans-serif] cursor-pointer"
              />
            </div>
            <div>
              <label className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
                End Date
              </label>
              <input
                type="date"
                aria-label="End Date"
                value={params.endDate}
                onChange={(e) => setParams({ endDate: e.target.value, page: '1' })}
                className="h-[34px] px-3 rounded-none bg-[#171b26] border border-[#464554] text-white text-xs focus:outline-none focus:border-[#c0c1ff] font-['Outfit',sans-serif] cursor-pointer"
              />
            </div>
          </div>

          {/* Search Bar Input */}
          <div className="flex-1 min-w-[200px]">
            <label className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
              Search Runs
            </label>
            <input
              type="text"
              placeholder="Run ID, Workflow ID, or Tenant..."
              value={params.search}
              onChange={(e) => setParams({ search: e.target.value, page: '1' })}
              className="w-full h-[34px] px-4 rounded-none bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
            />
          </div>

          {/* Always Present Clear Filters Button */}
          <div className="self-end pb-0.5">
            <Button
              type="button"
              onClick={() =>
                setParams({
                  mode: 'all',
                  status: 'all',
                  search: '',
                  startDate: '',
                  endDate: '',
                  page: '1',
                })
              }
              aria-label="Clear all active table filters"
              title="Clear all active table filters"
              variant="secondary-dark"
              icon="filter_alt_off"
              data-testid="runs-clear-filters-btn"
            />
          </div>
        </>
      }
    >
      {/* Error State with Retry Trigger */}
      {isError && (
        <DegradedStateView
          type="api-disconnected"
          title="Failed to Retrieve Execution Runs"
          description={(error as Error)?.message || 'Could not connect to engine service.'}
          actionLabel="Retry Fetching"
          onRetry={() => { refetch(); }}
        />
      )}

      {/* Main Table using PaginatedTable Component */}
      {isLoading ? (
        <div className="p-6 rounded-none bg-[#0F131D]/90 border border-[#464554] shadow-xl">
          <Skeleton count={5} height="2.5rem" />
        </div>
      ) : (
        <PaginatedTable<ExtendedRunItem>
          data={filteredRuns}
          columns={runColumns}
          getRowKey={(run) => run.run_id}
          currentPage={1}
          pageSize={Math.max(1, filteredRuns.length)}
          totalItems={filteredRuns.length}
          onPageChange={() => {}}
          itemLabel="runs"
          testId="runs-table"
          emptyState={
            <DegradedStateView
              type="stale-report"
              title="No Journey Runs Found"
              description="No execution runs match your current status, mode, search, or date range filters."
              actionLabel="Reset Filters"
              onRefresh={() => setParams({ mode: 'all', status: 'all', search: '', startDate: '', endDate: '', page: '1' })}
            />
          }
        />
      )}
    </DirectoryLayout>
  );
};

export default RunListPage;
