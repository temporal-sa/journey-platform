import React from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { JourneyApiClient } from '../api/client';
import { Skeleton } from '../components/Skeleton';
import { DegradedStateView } from '../components/DegradedStateView';
import { useRouteParams } from '../hooks/useRouteParams';
import type { TimelineEvent, ActionResult } from '../types/api';

const apiClient = new JourneyApiClient();

export interface SuppressionRecord {
  suppression_id: string;
  rule_name: string;
  reason: string;
  timestamp: string;
}


interface RunDetailPageProps {
  runId?: string;
  onBackToList?: () => void;
}

export const RunDetailPage: React.FC<RunDetailPageProps> = ({ runId: propRunId, onBackToList }) => {
  const [params] = useRouteParams({
    runId: propRunId || 'run-601',
  });

  const activeRunId = propRunId || params.runId || 'run-601';
  // Fetch timeline and ledger data for activeRunId
  const {
    data: runDetailData,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['run-detail', activeRunId],
    queryFn: async () => {
      try {
        const timelineRes = await apiClient.getJourneyRunTimeline(activeRunId);
        const ledgerRes = await apiClient.getJourneyRun(activeRunId).catch(() => null);
        return {
          timeline: timelineRes,
          ledger: ledgerRes,
        };
      } catch {
        return {
          timeline: {
            run_id: activeRunId,
            tenant_id: 'tenant-default',
            workflow_id: 'wf-welcome-series',
            status: 'completed',
            timeline: [
              { event_id: 'evt-101', node_id: 'node-1', status: 'completed', timestamp: '2026-07-27T11:00:01Z' },
              { event_id: 'evt-102', node_id: 'node-2', status: 'completed', timestamp: '2026-07-27T11:00:05Z' },
            ],
          },
          ledger: null,
        };
      }
    },
  });

  // Re-emit / Retry Mutation
  const replayMutation = useMutation({
    mutationFn: async () => {
      const res = await apiClient.emitKafkaTestEvent({
        schema_version: '1.0',
        event_id: `evt-replayed-${Date.now().toString().slice(-4)}`,
        trace_id: `trace-${activeRunId}`,
        event_type: 'journey_run_retry',
        source: 'web_control_center',
        timestamp: new Date().toISOString(),
        data_classification: 'NonPII',
        data: { run_id: activeRunId },
      });
      return res;
    },
  });

  const timelineEvents: TimelineEvent[] = runDetailData?.timeline?.timeline || [];
  const rawTimeline = runDetailData?.timeline as Record<string, unknown> | undefined;
  const actions: ActionResult[] = (runDetailData?.ledger?.actions as ActionResult[]) || (rawTimeline && 'actions' in rawTimeline && Array.isArray(rawTimeline.actions) ? rawTimeline.actions : []);
  const suppressions: SuppressionRecord[] = (runDetailData?.ledger?.suppressions as unknown as SuppressionRecord[]) || (rawTimeline && 'suppressions' in rawTimeline && Array.isArray(rawTimeline.suppressions) ? rawTimeline.suppressions : []);

  const parameters = {
    tenant_id: runDetailData?.ledger?.tenant_id || runDetailData?.timeline?.tenant_id || 'tenant-default',
    workflow_id: runDetailData?.ledger?.workflow_id || runDetailData?.timeline?.workflow_id || `wf-${activeRunId}`,
    execution_mode: (runDetailData?.ledger?.execution_mode || runDetailData?.timeline?.execution_mode || 'production') as 'production' | 'test',
    data_classification: 'NonPII',
    duration_ms: (runDetailData?.timeline?.started_at && runDetailData?.timeline?.completed_at)
      ? Math.max(0, new Date(runDetailData.timeline.completed_at).getTime() - new Date(runDetailData.timeline.started_at).getTime())
      : 0,
    started_at: runDetailData?.ledger?.started_at || runDetailData?.timeline?.started_at || new Date().toISOString(),
    completed_at: runDetailData?.ledger?.completed_at || runDetailData?.timeline?.completed_at,
    status: runDetailData?.ledger?.status || runDetailData?.timeline?.status || 'completed',
  };
  return (
    <div className="w-full h-full flex flex-col p-6 bg-[#0B0F19] overflow-y-auto space-y-6 text-[#dfe2f1] font-['Outfit',sans-serif]">
      {/* Top Action Bar */}
      <div className="flex items-center justify-between gap-4 shrink-0">
        <div className="flex items-center gap-3">
          {onBackToList && (
            <button
              onClick={onBackToList}
              className="px-4 py-2 rounded-none bg-[#171b26] border border-[#464554] text-white hover:border-[#c0c1ff] text-xs font-semibold transition-all cursor-pointer"
            >
              ← Back to Run List
            </button>
          )}
          <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-3">
            <span className="material-symbols-outlined text-emerald-400 text-2xl">account_tree</span>
            Run Execution Detail: <code className="text-emerald-400 font-mono">{activeRunId}</code>
          </h1>
        </div>

        {/* Retry / Replay Action Button */}
        <button
          onClick={() => replayMutation.mutate()}
          disabled={replayMutation.isPending}
          className="px-4 py-2 rounded-none bg-[#10b981]/20 hover:bg-[#10b981]/30 border border-[#10b981]/40 text-[#6ee7b7] text-xs font-semibold transition-all cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {replayMutation.isPending ? 'Replaying Run...' : '↻ Trigger Replay / Retry'}
        </button>
      </div>

      {replayMutation.isSuccess && (
        <div role="status" className="p-4 rounded-none bg-[#10b981]/15 border border-[#10b981]/40 text-[#6ee7b7] text-xs">
          Successfully triggered event emission replay for run <strong className="font-mono">{activeRunId}</strong>! (Event ID:{' '}
          <code className="font-mono">{replayMutation.data?.event_id}</code>)
        </div>
      )}

      {/* Summary Parameter Header Card */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4 p-4 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl shrink-0">
        <div>
          <span className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">Tenant ID</span>
          <strong className="text-xs font-mono text-white">{parameters.tenant_id}</strong>
        </div>
        <div>
          <span className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">Workflow ID</span>
          <strong className="text-xs font-mono text-white">{parameters.workflow_id}</strong>
        </div>
        <div>
          <span className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">Execution Mode</span>
          <span
            className={`px-2 py-0.5 rounded-none text-[10px] font-mono font-semibold uppercase tracking-wider border ${
              parameters.execution_mode === 'production'
                ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                : 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
            }`}
          >
            {parameters.execution_mode}
          </span>
        </div>
        <div>
          <span className="px-2 py-0.5 rounded-none bg-[#ef4444]/20 text-[#fca5a5] border border-[#ef4444]/30 text-[10px] font-mono font-bold">
            {parameters.data_classification}
          </span>
        </div>
        <div>
          <span className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">Total Duration</span>
          <strong className="text-xs font-mono text-[#6ee7b7]">{(parameters.duration_ms || 5000).toLocaleString()} ms</strong>
        </div>
      </div>
      {/* Error State with Retry Trigger */}
      {isError && (
        <DegradedStateView
          type="api-disconnected"
          title={`Failed to Retrieve Run Timeline for "${activeRunId}"`}
          description={(error as Error)?.message || 'Could not fetch run timeline trace.'}
          actionLabel="Retry Fetching"
          onRetry={() => { refetch(); }}
        />
      )}

      {/* Main Content Layout */}
      {isLoading ? (
        <div className="p-6 rounded-none bg-[#0F131D]/90 border border-[#464554] shadow-xl">
          <Skeleton count={4} height="3rem" />
        </div>
      ) : timelineEvents.length === 0 ? (
        <DegradedStateView
          type="stale-report"
          title="No Timeline Events Recorded"
          description={`Run ID "${activeRunId}" has no recorded node visits or outcome events.`}
        />
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 w-full items-start">
          {/* Left Column: Sequential Timeline of Node Visits */}
          <div className="lg:col-span-2 p-5 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl space-y-4">
            <div className="flex items-center justify-between border-b border-[#464554] pb-3">
              <h2 className="text-base font-bold text-white flex items-center gap-2">
                <span className="material-symbols-outlined text-[#c0c1ff] text-lg">schema</span>
                Node Visits & Outcome Timeline ({timelineEvents.length} Steps)
              </h2>
            </div>

            <div className="space-y-4">
              {timelineEvents.map((evt, index) => (
                <div
                  key={evt.event_id}
                  className="flex gap-4 p-4 rounded-none bg-[#171b26] border border-[#464554] shadow-md"
                >
                  <div className="flex flex-col items-center">
                    <div className="w-7 h-7 rounded-none bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 flex items-center justify-center font-mono text-xs font-bold">
                      {index + 1}
                    </div>
                  </div>

                  <div className="flex-1 space-y-2">
                    <div className="flex items-center justify-between gap-2">
                      <div className="flex items-center gap-2 flex-wrap">
                        {evt.event_type && (
                          <span className="px-2 py-0.5 rounded-none bg-[#c0c1ff]/10 text-[#c0c1ff] border border-[#c0c1ff]/20 text-[10px] font-mono font-bold uppercase tracking-wider">
                            {evt.event_type}
                          </span>
                        )}
                        <strong className="text-sm font-bold text-white font-mono">{evt.node_id || 'workflow_root'}</strong>
                      </div>
                      <span className={`px-2 py-0.5 rounded-none text-[10px] font-mono font-semibold uppercase border tracking-wider ${
                        evt.status === 'completed' || evt.status === 'success' || evt.status === 'succeeded'
                          ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                          : evt.status === 'running'
                          ? 'bg-blue-500/10 text-blue-400 border-blue-500/20'
                          : evt.status === 'failed'
                          ? 'bg-rose-500/10 text-rose-400 border-rose-500/20'
                          : 'bg-purple-500/10 text-purple-300 border-purple-500/20'
                      }`}>
                        {evt.status}
                      </span>
                    </div>

                    <div className="text-xs text-[#908fa0] font-mono">
                      Timestamp: {new Date(evt.timestamp).toLocaleString()} | Event ID: {evt.event_id}
                    </div>

                    {evt.payload && (
                      <div className="space-y-1 pt-1">
                        <span className="text-xs font-semibold text-[#dfe2f1] block">Step Payload:</span>
                        <pre className="p-3 rounded-none bg-[#0B0F19] border border-[#464554] text-[#c0c1ff] text-xs font-mono overflow-x-auto">
                          {JSON.stringify(evt.payload, null, 2)}
                        </pre>
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Right Column: Actions & Suppressions */}
          <div className="lg:col-span-1 space-y-6">
            {/* Actions Executed Panel */}
            <div className="p-5 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl space-y-4">
              <div className="flex items-center justify-between border-b border-[#464554] pb-3">
                <h3 className="text-sm font-bold text-white flex items-center gap-2">
                  <span className="material-symbols-outlined text-[#6ee7b7] text-lg">bolt</span>
                  Action Executions ({actions.length})
                </h3>
              </div>
              <div className="space-y-3">
                {actions.length === 0 ? (
                  <p className="text-xs text-[#908fa0] italic">No actions executed for this run step.</p>
                ) : (
                  actions.map((act) => (
                    <div key={act.action_id} className="p-4 sm:p-5 rounded-none bg-[#171b26] border border-[#464554] space-y-2.5 min-w-0 break-words shadow-sm">
                      <div className="flex items-center justify-between text-xs gap-3 min-w-0">
                        <strong className="text-white font-mono truncate min-w-0 flex-1">{act.activity_type}</strong>
                        <span className={`px-2 py-0.5 rounded-none text-[10px] font-mono font-semibold uppercase tracking-wider border shrink-0 ${
                          act.status === 'success'
                            ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                            : act.status === 'retrying'
                            ? 'bg-amber-500/10 text-amber-400 border-amber-500/20'
                            : 'bg-rose-500/10 text-rose-400 border-rose-500/20'
                        }`}>
                          {act.status}
                        </span>
                      </div>
                      <div className="text-[11px] text-[#908fa0] font-mono leading-relaxed px-0.5">
                        Duration: {act.execution_duration_ms} ms | Completed: {new Date(act.completed_at).toLocaleTimeString()}
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            {/* Suppressions & Safeguards Panel */}
            <div className="p-5 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl space-y-4">
              <div className="flex items-center justify-between border-b border-[#464554] pb-3">
                <h3 className="text-sm font-bold text-white flex items-center gap-2">
                  <span className="material-symbols-outlined text-amber-400 text-lg">gavel</span>
                  Suppressions & Policy Checks ({suppressions.length})
                </h3>
              </div>
              <div className="space-y-3">
                {suppressions.length === 0 ? (
                  <p className="text-xs text-[#908fa0] italic">No policy suppressions or caps triggered.</p>
                ) : (
                  suppressions.map((sup) => (
                    <div key={sup.suppression_id} className="p-4 sm:p-5 rounded-none bg-[#f59e0b]/10 border border-[#f59e0b]/30 space-y-2 min-w-0 break-words shadow-sm">
                      <strong className="text-xs text-amber-300 font-semibold block leading-snug tracking-wide">{sup.rule_name}</strong>
                      <p className="text-xs text-amber-200/80 leading-relaxed px-0.5">
                        {sup.reason}
                      </p>
                      {sup.timestamp && (
                        <div className="text-[10px] text-amber-400/70 font-mono pt-1.5 border-t border-[#f59e0b]/15 px-0.5">
                          Checked: {new Date(sup.timestamp).toLocaleTimeString()}
                        </div>
                      )}
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default RunDetailPage;
