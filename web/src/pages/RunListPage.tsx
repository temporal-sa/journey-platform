import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { JourneyApiClient } from '../api/client';
import { Skeleton } from '../components/Skeleton';
import { DegradedStateView } from '../components/DegradedStateView';
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
  return (
    <div className="w-full h-full flex flex-col p-6 bg-[#0B0F19] overflow-y-auto space-y-6 text-[#dfe2f1] font-['Outfit',sans-serif]">
      {/* Top Section Header */}
      <div className="flex flex-wrap items-center justify-between gap-4 shrink-0">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-3">
            <span className="material-symbols-outlined text-emerald-400 text-2xl">route</span>
            Journey Execution Runs
          </h1>
          <p className="text-xs text-[#908fa0] mt-1">
            Inspect active execution state machines, production traces, and test scenario runs.
          </p>
        </div>
      </div>

      {/* Filter Toolbar */}
      <div className="w-full p-4 rounded-2xl bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl flex flex-wrap items-end gap-4 shrink-0">
        {/* Execution Mode Selector */}
        <div>
          <label className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
            Execution Mode Filter
          </label>
          <div className="flex flex-wrap items-center gap-1.5 h-[34px] bg-[#171b26] px-1.5 rounded-xl border border-[#464554]" role="group" aria-label="Execution Mode Filter">
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
                  className={`px-2.5 py-1 rounded-lg text-[10px] font-mono font-semibold uppercase border tracking-wider transition-all cursor-pointer ${getModeStyles(item.mode)} ${
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
        <div>
          <label className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
            Run Status Filter
          </label>
          <div className="flex flex-wrap items-center gap-1.5 h-[34px] bg-[#171b26] px-1.5 rounded-xl border border-[#464554]" role="group" aria-label="Run Status Filter">
            {[
              { status: 'all', label: 'All Statuses' },
              { status: 'running', label: 'Running' },
              { status: 'completed', label: 'Completed' },
              { status: 'failed', label: 'Failed' },
              { status: 'terminated', label: 'Terminated' },
            ].map((item) => {
              const isActive = params.status === item.status;
              return (
                <button
                  key={item.status}
                  type="button"
                  onClick={() => setParams({ status: item.status, page: '1' })}
                  className={`px-2.5 py-1 rounded-lg text-[10px] font-mono font-semibold uppercase border tracking-wider transition-all cursor-pointer ${getStatusStyles(item.status)} ${
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
              className="h-[34px] px-3 rounded-xl bg-[#171b26] border border-[#464554] text-white text-xs focus:outline-none focus:border-[#c0c1ff] font-['Outfit',sans-serif] cursor-pointer"
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
              className="h-[34px] px-3 rounded-xl bg-[#171b26] border border-[#464554] text-white text-xs focus:outline-none focus:border-[#c0c1ff] font-['Outfit',sans-serif] cursor-pointer"
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
            className="w-full h-[34px] px-4 rounded-xl bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
          />
        </div>
      </div>

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

      {/* Main Content State */}
      {isLoading ? (
        <div className="p-6 rounded-2xl bg-[#0F131D]/90 border border-[#464554] shadow-xl">
          <Skeleton count={5} height="2.5rem" />
        </div>
      ) : filteredRuns.length === 0 ? (
        <DegradedStateView
          type="stale-report"
          title="No Journey Runs Found"
          description="No execution runs match your current status, mode, search, or date range filters."
          actionLabel="Reset Filters"
          onRefresh={() => setParams({ mode: 'all', status: 'all', search: '', startDate: '', endDate: '', page: '1' })}
        />
      ) : (
        /* Obsidian Styled Runs Table */
        <div className="w-full rounded-2xl bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl overflow-hidden">
          <table className="w-full text-left text-xs border-collapse">
            <thead>
              <tr className="bg-[#171b26] text-[#908fa0] border-b border-[#464554] font-mono uppercase text-[10px] tracking-wider">
                <th className="p-4 font-semibold">Run ID</th>
                <th className="p-4 font-semibold">Workflow ID</th>
                <th className="p-4 font-semibold">Mode</th>
                <th className="p-4 font-semibold">Status</th>
                <th className="p-4 font-semibold">Current Active Node</th>
                <th className="p-4 font-semibold">Started At</th>
                <th className="p-4 font-semibold text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#464554]/50">
              {filteredRuns.map((run) => {
                const isProd = run.execution_mode === 'production';

                return (
                  <tr key={run.run_id} className="hover:bg-[#171b26]/50 transition-colors">
                    <td className="p-4 font-mono">
                      <code className="px-2 py-1 rounded bg-[#171b26] border border-[#464554] text-[#dfe2f1] font-mono text-xs">
                        {run.run_id}
                      </code>
                    </td>
                    <td className="p-4 text-white font-medium">{run.workflow_id}</td>
                    <td className="p-4 font-mono">
                      <span
                        className={`px-2 py-0.5 rounded text-[10px] font-mono font-semibold uppercase border tracking-wider ${
                          isProd
                            ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                            : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                        }`}
                      >
                        {run.execution_mode}
                      </span>
                    </td>
                    <td className="p-4 font-mono">
                      <span className={`px-2 py-0.5 rounded text-[10px] font-mono font-semibold uppercase border tracking-wider ${getStatusStyles(run.status)}`}>
                        {run.status}
                      </span>
                    </td>
                    <td className="p-4 font-mono text-[#c0c1ff]">
                      {run.current_nodes?.join(', ') || 'N/A'}
                    </td>
                    <td className="p-4 text-[#908fa0] text-xs font-mono">
                      {new Date(run.started_at).toLocaleString()}
                    </td>
                    <td className="p-4 text-right">
                      <button
                        onClick={() => onSelectRun?.(run.run_id)}
                        className="px-3.5 py-1.5 rounded-xl bg-[#b76dff]/20 hover:bg-[#b76dff]/40 hover:brightness-125 text-[#ddb7ff] hover:text-white border border-[#ddb7ff]/30 hover:border-[#ddb7ff] text-xs font-semibold transition-all cursor-pointer flex items-center gap-1.5 shadow-sm"
                      >
                        <span>Inspect Detail</span>
                        <span className="material-symbols-outlined text-sm">arrow_forward</span>
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};

export default RunListPage;
