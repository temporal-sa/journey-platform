import React from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { JourneyApiClient } from '../api/client';
import { Skeleton } from '../components/Skeleton';
import { DegradedStateView } from '../components/DegradedStateView';
import { Button } from '../components/common/Button';
import { Badge } from '../components/common/Badge';
import { ToastMessageType } from '../components/common/Toast';
import { PaginatedTable } from '../components/common/PaginatedTable';
import { StatusFilterDropdown } from '../components/common/StatusFilterDropdown';
import { useRouteParams } from '../hooks/useRouteParams';
import type { TimelineEvent, ActionResult, SubRunSummary, GraphNode, GraphEdge } from '../types/api';
import { ExecutionGraphView, NodeVisitStep, SignalSimulationModal } from '../components/testlane';
import { formatDuration } from '../utils/formatDuration';
const apiClient = new JourneyApiClient();

export interface SuppressionRecord {
  suppression_id: string;
  rule_name: string;
  reason: string;
  timestamp: string;
}

interface RunDetailPageProps {
  runId?: string;
  subRunId?: string;
  onBackToList?: () => void;
  onSubRunSelect?: (subRunId: string) => void;
  onFireToast?: (type: ToastMessageType, title: string, message: string) => void;
}

export const RunDetailPage: React.FC<RunDetailPageProps> = ({
  runId: propRunId,
  subRunId: propSubRunId,
  onBackToList,
  onSubRunSelect,
  onFireToast,
}) => {
  const [params, setParams] = useRouteParams({
    runId: propRunId || 'run-601',
    subRunId: propSubRunId || '',
  });

  const activeRunId = propRunId || params.runId || 'run-601';
  const activeSubRunIdParam = propSubRunId || (params.subRunId !== '' ? params.subRunId : undefined);

  const [selectedSubRunId, setSelectedSubRunId] = React.useState<string | undefined>(activeSubRunIdParam);

  React.useEffect(() => {
    if (propRunId && propRunId !== params.runId) {
      setParams({ runId: propRunId, subRunId: propSubRunId || '' });
    }
  }, [propRunId, propSubRunId]);

  React.useEffect(() => {
    if (activeSubRunIdParam && activeSubRunIdParam !== selectedSubRunId) {
      setSelectedSubRunId(activeSubRunIdParam);
    }
  }, [activeSubRunIdParam]);

  const handleSelectSubRun = (id: string) => {
    setSelectedSubRunId(id);
    setParams({ runId: activeRunId, subRunId: id });
    if (onSubRunSelect) {
      onSubRunSelect(id);
    }
  };
  const [viewMode, setViewMode] = React.useState<'graph' | 'timeline'>('graph');
  const [isSignalModalOpen, setIsSignalModalOpen] = React.useState(false);
  const [activeSignalWfId, setActiveSignalWfId] = React.useState<string>('');
  const [activeSignalSubjectId, setActiveSignalSubjectId] = React.useState<string>('');

  const handleOpenSignalModal = (wfId: string, subjectId?: string) => {
    setActiveSignalWfId(wfId);
    setActiveSignalSubjectId(subjectId || '');
    setIsSignalModalOpen(true);
  };
  const [subRunParams, setSubRunParams] = React.useState<{ page: string; search: string; status: string }>({
    page: '1',
    search: '',
    status: 'all',
  });
  const [nowTick, setNowTick] = React.useState(Date.now());

  // Query sub-runs directory for activeRunId with auto-polling
  const { data: subRunsData, isLoading: isSubRunsLoading } = useQuery({
    queryKey: ['sub-runs', activeRunId, subRunParams],
    queryFn: async () => {
      return await apiClient.listJourneyRunSubRuns(activeRunId, {
        page: parseInt(subRunParams.page, 10) || 1,
        limit: 10,
        search: subRunParams.search || undefined,
        status: subRunParams.status !== 'all' ? subRunParams.status : undefined,
      });
    },
    refetchInterval: (query) => {
      const subRuns = query.state.data?.sub_runs || [];
      if (subRuns.length === 0) return 1500;
      const allTerminal = subRuns.every((sr) =>
        ['completed', 'failed', 'suppressed', 'terminated'].includes((sr.status || '').toLowerCase())
      );
      return allTerminal ? false : 1500;
    },
  });

  // Fetch timeline and ledger data for activeRunId and selectedSubRunId with auto-polling
  const {
    data: runDetailData,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['run-detail', activeRunId, selectedSubRunId],
    queryFn: async () => {
      const timelineRes = await apiClient.getJourneyRunTimeline(activeRunId, selectedSubRunId);
      const ledgerRes = await apiClient.getJourneyRun(activeRunId).catch(() => null);
      return {
        timeline: timelineRes,
        ledger: ledgerRes,
      };
    },
    refetchInterval: (query) => {
      const status = (query.state.data?.timeline?.status || query.state.data?.ledger?.status || '').toLowerCase();
      const isTerminal = ['completed', 'failed', 'suppressed', 'terminated'].includes(status);
      return isTerminal ? false : 1500;
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
  const actions: ActionResult[] = React.useMemo(() => {
    if (runDetailData?.ledger?.actions && runDetailData.ledger.actions.length > 0) {
      return runDetailData.ledger.actions;
    }
    if (rawTimeline && 'actions' in rawTimeline && Array.isArray(rawTimeline.actions) && rawTimeline.actions.length > 0) {
      return rawTimeline.actions as ActionResult[];
    }

    return timelineEvents
      .filter((evt) => {
        const type = (evt.event_type || '').toLowerCase();
        const nodeId = (evt.node_id || '').toLowerCase();
        return (
          type.includes('email') ||
          type.includes('sms') ||
          type.includes('push') ||
          type.includes('inapp') ||
          type.includes('webhook') ||
          (type.includes('action') && !type.includes('interaction')) ||
          nodeId.includes('email') ||
          nodeId.includes('sms') ||
          nodeId.includes('push') ||
          nodeId.includes('inapp') ||
          nodeId.includes('webhook')
        );
      })
      .map((evt, idx) => {
        let channelName = 'Email';
        const t = (evt.event_type || '').toLowerCase();
        const n = (evt.node_id || '').toLowerCase();
        if (t.includes('sms') || n.includes('sms')) channelName = 'SMS';
        else if (t.includes('push') || n.includes('push')) channelName = 'Push';
        else if (t.includes('inapp') || n.includes('inapp')) channelName = 'InApp';
        else if (t.includes('webhook') || n.includes('webhook')) channelName = 'Webhook';

        return {
          schema_version: '1.0',
          action_id: evt.event_id || `act-${idx + 1}`,
          activity_type: `ExecuteActionGateway:${channelName}`,
          status: (evt.status === 'completed' || evt.status === 'passed' ? 'success' : evt.status || 'success') as ActionResult['status'],
          execution_duration_ms: evt.payload && typeof evt.payload === 'object' && 'duration_ms' in evt.payload ? Number(evt.payload.duration_ms) : 18,
          completed_at: evt.timestamp || new Date().toISOString(),
        };
      });
  }, [timelineEvents, runDetailData, rawTimeline]);
  const suppressions: SuppressionRecord[] = (runDetailData?.ledger?.suppressions as unknown as SuppressionRecord[]) || (rawTimeline && 'suppressions' in rawTimeline && Array.isArray(rawTimeline.suppressions) ? rawTimeline.suppressions : []);

  // Derive execution status directly from recorded timeline events / projections
  const derivedExecutionStatus = React.useMemo(() => {
    if (!timelineEvents || timelineEvents.length === 0) {
      return runDetailData?.timeline?.status || runDetailData?.ledger?.status || 'running';
    }

    const hasTerminalEvent = timelineEvents.some((evt) => {
      const type = (evt.event_type || '').toLowerCase();
      const status = (evt.status || '').toLowerCase();
      return (
        type.includes('succeeded') ||
        type.includes('completed') ||
        type.includes('failed') ||
        type.includes('suppressed') ||
        status === 'completed' ||
        status === 'passed' ||
        status === 'failed' ||
        status === 'suppressed'
      );
    });

    if (hasTerminalEvent) {
      const lastEvent = timelineEvents[timelineEvents.length - 1];
      const lastStatus = (lastEvent?.status || '').toLowerCase();
      if (lastStatus === 'failed' || (lastEvent?.event_type || '').includes('failed')) return 'failed';
      if (lastStatus === 'suppressed' || (lastEvent?.event_type || '').includes('suppressed')) return 'suppressed';
      return 'completed';
    }

    return 'running';
  }, [timelineEvents, runDetailData]);

  // Live ticking timer for in-progress executions
  React.useEffect(() => {
    if (derivedExecutionStatus === 'running') {
      const interval = setInterval(() => {
        setNowTick(Date.now());
      }, 1000);
      return () => clearInterval(interval);
    }
  }, [derivedExecutionStatus]);

  const derivedDurationMs = React.useMemo(() => {
    if (timelineEvents && timelineEvents.length > 0) {
      const firstTimestamp = new Date(timelineEvents[0].timestamp).getTime();
      const lastEvent = timelineEvents[timelineEvents.length - 1];

      let endTimestamp = new Date(lastEvent.timestamp).getTime();
      if (derivedExecutionStatus === 'running') {
        endTimestamp = nowTick;
      } else if (runDetailData?.timeline?.completed_at) {
        endTimestamp = new Date(runDetailData.timeline.completed_at).getTime();
      }

      if (!isNaN(firstTimestamp) && !isNaN(endTimestamp)) {
        return Math.max(0, endTimestamp - firstTimestamp);
      }
    }

    if (runDetailData?.timeline?.started_at) {
      const start = new Date(runDetailData.timeline.started_at).getTime();
      let end = nowTick;
      if (runDetailData?.timeline?.completed_at) {
        end = new Date(runDetailData.timeline.completed_at).getTime();
      }
      if (!isNaN(start) && !isNaN(end)) {
        return Math.max(0, end - start);
      }
    }

    return 0;
  }, [timelineEvents, runDetailData, derivedExecutionStatus, nowTick]);
  const parameters = {
    tenant_id: runDetailData?.ledger?.tenant_id || runDetailData?.timeline?.tenant_id || 'tenant-default',
    workflow_id: runDetailData?.ledger?.workflow_id || runDetailData?.timeline?.workflow_id || `wf-${activeRunId}`,
    execution_mode: (runDetailData?.ledger?.execution_mode || runDetailData?.timeline?.execution_mode || 'production') as 'production' | 'test',
    data_classification: 'NonPII',
    duration_ms: derivedDurationMs,
    started_at: runDetailData?.ledger?.started_at || runDetailData?.timeline?.started_at || new Date().toISOString(),
    completed_at: runDetailData?.ledger?.completed_at || runDetailData?.timeline?.completed_at,
    status: derivedExecutionStatus,
  };
  const workflowId = runDetailData?.timeline?.workflow_id || runDetailData?.ledger?.workflow_id || parameters.workflow_id;

  const { data: journeyDraft } = useQuery({
    queryKey: ['journey-draft', workflowId],
    queryFn: async () => {
      try {
        if (workflowId) {
          const res = await apiClient.getJourneyDraft(workflowId);
          if (res?.draft?.nodes?.length) return res.draft;
        }
      } catch {
        // ignore
      }
      try {
        const drafts = await apiClient.listJourneyDrafts();
        if (drafts && drafts.length > 0 && drafts[0].nodes?.length) {
          return drafts[0];
        }
      } catch {
        // ignore
      }
      return null;
    },
    enabled: !!workflowId,
  });

  const visitSteps: NodeVisitStep[] = React.useMemo(() => {
    const statusMap: Record<string, NodeVisitStep['status']> = {
      completed: 'completed',
      success: 'completed',
      passed: 'passed',
      failed: 'failed',
      running: 'running',
      suppressed: 'suppressed',
    };

    return timelineEvents.map((evt, idx) => {
      const rawNodeId = evt.node_id || (evt.payload && typeof evt.payload === 'object' && 'node_id' in evt.payload ? String((evt.payload as Record<string, unknown>).node_id) : '');
      const nodeId = rawNodeId || `node-${idx + 1}`;

      return {
        stepIndex: idx + 1,
        nodeId,
        nodeName: nodeId ? `Step ${idx + 1}: ${nodeId}` : `Step ${idx + 1}`,
        nodeType: evt.event_type || 'action',
        status: statusMap[evt.status?.toLowerCase()] || 'completed',
        timestamp: evt.timestamp,
        output: evt.payload,
      };
    });
  }, [timelineEvents]);

  const currentNodeId = React.useMemo(() => {
    if (visitSteps.length === 0) return undefined;
    if (derivedExecutionStatus === 'completed' || derivedExecutionStatus === 'passed') return undefined;

    const latestStep = visitSteps[visitSteps.length - 1];
    return latestStep?.nodeId;
  }, [visitSteps, derivedExecutionStatus]);

  const graphNodesAndEdges = React.useMemo(() => {
    if (journeyDraft?.nodes && journeyDraft.nodes.length > 0) {
      return {
        nodes: journeyDraft.nodes,
        edges: journeyDraft.edges || [],
      };
    }

    if (timelineEvents.length > 0) {
      const derivedNodes: GraphNode[] = timelineEvents.map((evt, idx) => {
        const nodeId = evt.node_id || `node-${idx + 1}`;
        let type = 'Email';
        if (idx === 0) type = 'EventStart';
        else if (evt.event_type?.includes('condition') || nodeId.includes('cond')) type = 'Condition';
        else if (evt.event_type?.includes('sms') || nodeId.includes('sms')) type = 'SMS';
        else if (evt.event_type?.includes('push')) type = 'Push';
        else if (evt.event_type?.includes('delay')) type = 'Delay';

        return {
          id: nodeId,
          type,
          name: evt.node_id ? `Step ${idx + 1}: ${evt.node_id}` : `Step ${idx + 1}`,
          position: { x: 100 + (idx % 3) * 320, y: 80 + Math.floor(idx / 3) * 180 },
        };
      });

      const derivedEdges: GraphEdge[] = [];
      for (let i = 0; i < derivedNodes.length - 1; i++) {
        derivedEdges.push({
          id: `e-${derivedNodes[i].id}-${derivedNodes[i + 1].id}`,
          source: derivedNodes[i].id,
          target: derivedNodes[i + 1].id,
        });
      }

      return { nodes: derivedNodes, edges: derivedEdges };
    }

    const defaultNodes: GraphNode[] = [
      { id: 'node-1', type: 'EventStart', name: 'User Signup Event', position: { x: 50, y: 100 } },
      { id: 'node-2', type: 'Email', name: 'Send Welcome Email', position: { x: 320, y: 100 } },
    ];
    const defaultEdges: GraphEdge[] = [
      { id: 'e-1-2', source: 'node-1', target: 'node-2' },
    ];

    return { nodes: defaultNodes, edges: defaultEdges };
  }, [journeyDraft, timelineEvents]);

  const getStatusBadgeStyles = (status: string) => {
    const s = status.toLowerCase();
    switch (s) {
      case 'completed':
      case 'success':
      case 'succeeded':
        return 'bg-emerald-500/20 text-emerald-300 border-emerald-500/50';
      case 'running':
        return 'bg-amber-500/20 text-amber-300 border-amber-500/50';
      case 'failed':
        return 'bg-rose-500/20 text-rose-300 border-rose-500/50';
      case 'terminated':
        return 'bg-purple-500/20 text-purple-300 border-purple-500/50';
      default:
        return 'bg-[#464554]/30 text-[#c7c4d7] border-[#464554]';
    }
  };

  const getStatusStyles = (s: string) => {
    const st = s.toLowerCase();
    if (st === 'completed' || st === 'success') return { fill: 'bg-emerald-500/20', text: 'text-emerald-300', border: 'border-emerald-500/50' };
    if (st === 'failed') return { fill: 'bg-rose-500/20', text: 'text-rose-300', border: 'border-rose-500/50' };
    if (st === 'running') return { fill: 'bg-amber-500/20', text: 'text-amber-300', border: 'border-amber-500/50' };
    return { fill: 'bg-[#464554]/20', text: 'text-[#c7c4d7]', border: 'border-[#464554]' };
  };

  const subRunColumns = React.useMemo(
    () => [
      {
        key: 'contact',
        header: 'Contact / Recipient',
        cell: (sr: SubRunSummary) => (
          <div className="flex flex-col">
            <span className="font-bold text-white text-xs font-['Outfit']">{sr.name || sr.recipient}</span>
            <span className="font-mono text-[#908fa0] text-[11px]">{sr.recipient || sr.subject_id}</span>
          </div>
        ),
      },
      {
        key: 'sub_run_id',
        header: 'Sub-Run ID',
        cell: (sr: SubRunSummary) => (
          <span className="font-mono text-[#dfe2f1] text-xs">
            {sr.sub_run_id}
          </span>
        ),
      },
      {
        key: 'executed_branch',
        header: 'Evaluated Path',
        cell: (sr: SubRunSummary) => (
          <span className="px-2 py-0.5 rounded-none bg-[#c0c1ff]/15 text-[#c0c1ff] border border-[#c0c1ff]/30 font-mono text-[10px] font-bold uppercase tracking-wider">
            {sr.executed_branch || 'default'}
          </span>
        ),
      },
      {
        key: 'status',
        header: 'Status',
        cell: (sr: SubRunSummary) => (
          <span className={`px-2.5 py-0.5 rounded-none text-[10px] font-mono font-semibold uppercase border tracking-wider ${getStatusBadgeStyles(sr.status)}`}>
            {sr.status}
          </span>
        ),
      },
      {
        key: 'actions',
        header: 'Action',
        cell: (sr: SubRunSummary) => {
          const effectiveActiveSubRunId =
            selectedSubRunId ||
            (subRunsData?.sub_runs && subRunsData.sub_runs.length > 0
              ? subRunsData.sub_runs[0].sub_run_id
              : undefined);
          const isSelected =
            effectiveActiveSubRunId === sr.sub_run_id ||
            (!selectedSubRunId && runDetailData?.timeline?.sub_run_id === sr.sub_run_id);
          const temporalWfId = sr.sub_run_id.startsWith('wf-')
            ? sr.sub_run_id
            : `wf-${parameters.workflow_id}-${sr.sub_run_id}`;

          return (
            <div className="flex items-center gap-2 shrink-0 whitespace-nowrap">
              {isSelected ? (
                <Button
                  type="button"
                  variant="secondary-dark"
                  size="sm"
                  icon="visibility"
                  disabled={true}
                  title="Currently Viewing Active Trace"
                  className="opacity-50 cursor-not-allowed bg-[#171b26] text-[#908fa0] border-[#464554] font-semibold shrink-0 whitespace-nowrap"
                  data-testid={`active-subrun-trace-${sr.sub_run_id}`}
                >
                  Inspect Trace
                </Button>
              ) : (
                <Button
                  type="button"
                  variant="secondary-dark"
                  size="sm"
                  icon="visibility"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleSelectSubRun(sr.sub_run_id);
                    setViewMode('graph');
                  }}
                  className="bg-[#b76dff]/20 hover:bg-[#b76dff]/40 text-[#ddb7ff] border-[#ddb7ff]/30 font-semibold shrink-0 whitespace-nowrap"
                  data-testid={`view-subrun-trace-${sr.sub_run_id}`}
                >
                  Inspect Trace
                </Button>
              )}

              <Button
                type="button"
                variant="secondary-dark"
                size="sm"
                icon="schedule"
                aria-label="Open Temporal Workflow Execution"
                title={`Open Temporal Workflow Execution: ${temporalWfId}`}
                onClick={(e) => {
                  e.stopPropagation();
                  window.open(
                    `http://localhost:8233/namespaces/default/workflows/${encodeURIComponent(temporalWfId)}`,
                    '_blank'
                  );
                }}
                data-testid={`link-subrun-temporal-${sr.sub_run_id}`}
                className="shrink-0 whitespace-nowrap"
              >
                Temporal History
              </Button>

              <Button
                type="button"
                variant="secondary-dark"
                size="sm"
                icon="sensors"
                aria-label="Simulate Signal for Workflow"
                title={`Simulate Signal for Workflow: ${temporalWfId}`}
                onClick={(e) => {
                  e.stopPropagation();
                  handleOpenSignalModal(temporalWfId, sr.recipient || sr.subject_id);
                }}
                data-testid={`signal-subrun-${sr.sub_run_id}`}
                className="bg-[#4cd7f6]/20 hover:bg-[#4cd7f6]/40 text-[#4cd7f6] border-[#4cd7f6]/30 font-semibold shrink-0 whitespace-nowrap"
              >
                Signal
              </Button>
            </div>
          );
        },
      },
    ],
    [selectedSubRunId, subRunsData?.sub_runs, runDetailData?.timeline?.sub_run_id]
  );

  return (
    <div className="w-full h-full flex flex-col p-6 bg-[#0B0F19] overflow-y-auto space-y-6 text-[#dfe2f1] font-['Outfit',sans-serif]">
      {/* Top Action Header Bar */}
      <div className="flex items-center justify-between gap-4 shrink-0 flex-wrap">
        <div className="flex items-center gap-3">
          {onBackToList && (
            <Button
              type="button"
              variant="secondary-dark"
              icon="arrow_back"
              onClick={onBackToList}
              aria-label="Back to Run List"
            >
              Back to Run List
            </Button>
          )}
          <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-3 font-['Outfit']">
            <span className="material-symbols-outlined text-emerald-400 text-2xl">account_tree</span>
            <span>Run Execution Detail:</span>
            <code className="text-[#4cd7f6] font-mono text-base font-semibold">
              {activeRunId}
            </code>
          </h1>
        </div>

        {/* Action Button Bar: External Dev Tools, Outcome Simulators, & Trigger Replay */}
        <div className="flex items-center gap-2">
          {/* External Dev Tool Icons */}
          <Button
            type="button"
            variant="secondary-dark"
            icon="analytics"
            aria-label="Open Jaeger Traces"
            title="Open Jaeger Traces (:16686)"
            data-testid="link-jaeger-ui"
            onClick={() => window.open('http://localhost:16686', '_blank')}
          />
          <Button
            type="button"
            variant="secondary-dark"
            icon="mail"
            aria-label="Open Mailpit Inbox"
            title="Open Mailpit Inbox (:8025)"
            data-testid="link-mailpit-ui"
            onClick={() => window.open('http://localhost:8025', '_blank')}
          />

          <div className="h-4 w-[1px] bg-[#464554] mx-1" />

          {/* Outcome Simulator Icons */}
          <Button
            type="button"
            variant="secondary-dark"
            icon="mark_email_read"
            aria-label="Simulate Email Open"
            title="Simulate Email Open (→ /outcomes)"
            data-testid="simulate-outcome-open"
            className="bg-[#4cd7f6]/20 hover:bg-[#4cd7f6]/30 text-[#4cd7f6] border-[#4cd7f6]/40"
            onClick={() => {
              apiClient.emitKafkaTestEvent({
                schema_version: '1.0',
                event_id: `evt-open-${Date.now().toString().slice(-4)}`,
                trace_id: `trace-${activeRunId}`,
                event_type: 'email_opened',
                source: 'web_control_center',
                timestamp: new Date().toISOString(),
                data_classification: 'NonPII',
                data: { run_id: activeRunId },
              });
            }}
          />
          <Button
            type="button"
            variant="secondary-dark"
            icon="ads_click"
            aria-label="Simulate Link Click"
            title="Simulate Link Click (→ /outcomes)"
            data-testid="simulate-outcome-click"
            className="bg-[#c0c1ff]/20 hover:bg-[#c0c1ff]/30 text-[#c0c1ff] border-[#c0c1ff]/40"
            onClick={() => {
              apiClient.emitKafkaTestEvent({
                schema_version: '1.0',
                event_id: `evt-click-${Date.now().toString().slice(-4)}`,
                trace_id: `trace-${activeRunId}`,
                event_type: 'email_clicked',
                source: 'web_control_center',
                timestamp: new Date().toISOString(),
                data_classification: 'NonPII',
                data: { run_id: activeRunId },
              });
            }}
          />
          <Button
            type="button"
            variant="emerald"
            icon="task_alt"
            aria-label="Simulate Conversion"
            title="Simulate Conversion (→ /outcomes)"
            data-testid="simulate-outcome-conversion"
            onClick={() => {
              apiClient.emitKafkaTestEvent({
                schema_version: '1.0',
                event_id: `evt-conv-${Date.now().toString().slice(-4)}`,
                trace_id: `trace-${activeRunId}`,
                event_type: 'conversion',
                source: 'web_control_center',
                timestamp: new Date().toISOString(),
                data_classification: 'NonPII',
                data: { run_id: activeRunId },
              });
            }}
          />

          <div className="h-4 w-[1px] bg-[#464554] mx-1" />

          {/* Trigger Replay / Retry Icon */}
          <Button
            type="button"
            variant="emerald"
            icon="replay"
            disabled={replayMutation.isPending}
            aria-label={replayMutation.isPending ? 'Replaying Run...' : 'Trigger Replay / Retry'}
            title={replayMutation.isPending ? 'Replaying Run...' : 'Trigger Replay / Retry'}
            onClick={() => replayMutation.mutate()}
          />
        </div>
      </div>

      {replayMutation.isSuccess && (
        <div role="status" className="p-4 rounded-none bg-[#10b981]/15 border border-[#10b981]/40 text-[#6ee7b7] text-xs font-['Outfit']">
          Successfully triggered event emission replay for run <strong className="font-mono">{activeRunId}</strong>! (Event ID:{' '}
          <code className="font-mono">{replayMutation.data?.event_id}</code>)
        </div>
      )}

      {/* Summary Parameter Header Card */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4 p-4 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl shrink-0">
        <div>
          <span className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
            Tenant ID
          </span>
          <strong className="text-xs font-mono text-white">{parameters.tenant_id}</strong>
        </div>
        <div>
          <span className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
            Workflow ID
          </span>
          <strong className="text-xs font-mono text-white">{parameters.workflow_id}</strong>
        </div>
        <div>
          <span className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
            Execution Mode
          </span>
          <span
            className={`px-2 py-0.5 rounded-none text-[10px] font-mono font-semibold uppercase tracking-wider border ${
              parameters.execution_mode === 'production'
                ? 'bg-emerald-500/20 text-emerald-300 border-emerald-500/50'
                : 'bg-amber-500/20 text-amber-300 border-amber-500/50'
            }`}
          >
            {parameters.execution_mode}
          </span>
        </div>
        <div>
          <span className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
            Classification
          </span>
          <Badge variant="rose" size="sm">
            {parameters.data_classification}
          </Badge>
        </div>
        <div>
          <span className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
            Total Duration
          </span>
          <strong className="text-xs font-mono text-[#6ee7b7]">
            {formatDuration(parameters.duration_ms ?? 0)}
          </strong>
        </div>
      </div>

      {/* Audience Sub-Run Execution Directory Panel (Scalable 1,000+ Contacts) */}
      <div className="p-5 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl space-y-4">
        <div className="flex items-center justify-between gap-4 flex-wrap">
          <div className="flex items-center gap-3">
            <span className="material-symbols-outlined text-[#4cd7f6] text-xl">groups</span>
            <div>
              <h2 className="text-base font-bold text-white font-['Outfit']">
                Sub-runs
              </h2>
              <p className="text-xs text-[#908fa0] mt-0.5">
                Search, filter, and inspect individual target contact execution traces across static lists or test runs.
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3 flex-wrap">
            <input
              type="text"
              placeholder="Search contact, ID, recipient, or branch..."
              value={subRunParams.search}
              onChange={(e) => setSubRunParams((prev) => ({ ...prev, search: e.target.value, page: '1' }))}
              data-testid="subrun-search-input"
              className="px-3 py-1.5 bg-[#171b26] border border-[#464554] text-white text-xs placeholder-[#908fa0] focus:outline-none focus:border-[#4cd7f6] transition-all rounded-none font-['Outfit',sans-serif] min-w-[220px]"
            />
            <StatusFilterDropdown
              value={subRunParams.status}
              onChange={(newStatus) => setSubRunParams((prev) => ({ ...prev, status: newStatus, page: '1' }))}
              options={[
                { status: 'all', label: 'All Statuses' },
                { status: 'completed', label: 'Completed' },
                { status: 'failed', label: 'Failed' },
                { status: 'running', label: 'Running' },
              ]}
              getStatusStyles={getStatusStyles}
              dataTestId="subrun-status-filter-dropdown"
            />
          </div>
        </div>

        {isSubRunsLoading ? (
          <Skeleton count={3} height="2.5rem" />
        ) : (
          <PaginatedTable<SubRunSummary>
            data={subRunsData?.sub_runs || []}
            columns={subRunColumns}
            getRowKey={(sr) => sr.sub_run_id}
            getRowStyle={(sr) => {
              const effectiveActiveSubRunId =
                selectedSubRunId ||
                (subRunsData?.sub_runs && subRunsData.sub_runs.length > 0
                  ? subRunsData.sub_runs[0].sub_run_id
                  : undefined);
              return sr.sub_run_id === effectiveActiveSubRunId
                ? { backgroundColor: '#043d4d', color: '#ffffff', boxShadow: 'inset 4px 0 0 #4cd7f6' }
                : {};
            }}
            getRowClassName={(sr) => {
              const effectiveActiveSubRunId =
                selectedSubRunId ||
                (subRunsData?.sub_runs && subRunsData.sub_runs.length > 0
                  ? subRunsData.sub_runs[0].sub_run_id
                  : undefined);
              return sr.sub_run_id === effectiveActiveSubRunId
                ? 'text-white font-medium shadow-xl'
                : '';
            }}
            onRowClick={(sr) => {
              handleSelectSubRun(sr.sub_run_id);
              setViewMode('graph');
            }}
            currentPage={subRunsData?.page || 1}
            pageSize={subRunsData?.limit || 10}
            totalItems={subRunsData?.total || 0}
            onPageChange={(p) => setSubRunParams((prev) => ({ ...prev, page: String(p) }))}
            itemLabel="contacts"
            testId="subruns-paginated-table"
          />
        )}
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
          {/* Left Column: Trace Graph or Sequential Timeline of Node Visits */}
          <div className="lg:col-span-2 p-5 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl space-y-4">
            <div className="flex items-center justify-between border-b border-[#464554] pb-3 flex-wrap gap-2">
              <h2 className="text-base font-bold text-white flex items-center gap-2 font-['Outfit']">
                <span className="material-symbols-outlined text-[#c0c1ff] text-lg">
                  {viewMode === 'graph' ? 'account_tree' : 'schema'}
                </span>
                {viewMode === 'graph'
                  ? 'Trace Graph'
                  : `Node Visits & Outcome Timeline (${timelineEvents.length} Steps)`}
              </h2>

              {/* View Segment Toggle */}
              <div
                className="flex items-center gap-2"
                role="group"
                aria-label="Execution Trace View Mode"
              >
                <Button
                  type="button"
                  variant={viewMode === 'graph' ? 'primary-teal' : 'secondary-dark'}
                  size="sm"
                  icon="account_tree"
                  onClick={() => setViewMode('graph')}
                  data-testid="view-toggle-graph"
                >
                  Trace Graph
                </Button>
                <Button
                  type="button"
                  variant={viewMode === 'timeline' ? 'primary-teal' : 'secondary-dark'}
                  size="sm"
                  icon="schema"
                  onClick={() => setViewMode('timeline')}
                  data-testid="view-toggle-timeline"
                >
                  Timeline List
                </Button>
              </div>
            </div>

            {viewMode === 'graph' ? (
              <div
                className="w-full h-[540px] min-h-[540px] rounded-none border border-[#464554]/50 bg-[#0B0F19] overflow-hidden"
                style={{ height: '540px', minHeight: '540px' }}
                data-testid="execution-graph-container"
              >
                <ExecutionGraphView
                  nodes={graphNodesAndEdges.nodes}
                  edges={graphNodesAndEdges.edges}
                  visitSteps={visitSteps}
                  currentNodeId={currentNodeId}
                  status={
                    parameters.status === 'failed'
                      ? 'failed'
                      : parameters.status === 'running'
                      ? 'running'
                      : 'passed'
                  }
                  height="100%"
                />
              </div>
            ) : (
              <div className="space-y-4">
                {timelineEvents.map((evt, index) => (
                  <div
                    key={evt.event_id}
                    className="flex gap-4 p-4 rounded-none bg-[#171b26] border border-[#464554] shadow-md"
                  >
                    <div className="flex flex-col items-center">
                      <div className="w-7 h-7 rounded-none bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 flex items-center justify-center font-mono text-xs font-bold">
                        {index + 1}
                      </div>
                    </div>

                    <div className="flex-1 space-y-2">
                      <div className="flex items-center justify-between gap-2">
                        <div className="flex items-center gap-2 flex-wrap">
                          {evt.event_type && (
                            <span className="px-2 py-0.5 rounded-none bg-[#c0c1ff]/20 text-[#c0c1ff] border border-[#c0c1ff]/40 text-[10px] font-mono font-bold uppercase tracking-wider">
                              {evt.event_type}
                            </span>
                          )}
                          <strong className="text-sm font-bold text-white font-mono">{evt.node_id || 'workflow_root'}</strong>
                        </div>
                        <span
                          className={`px-2.5 py-0.5 rounded-none text-[10px] font-mono font-semibold uppercase border tracking-wider ${getStatusBadgeStyles(
                            evt.status
                          )}`}
                        >
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
            )}
          </div>

          {/* Right Column: Actions & Suppressions */}
          <div className="lg:col-span-1 space-y-6">

            {/* Actions Executed Panel */}
            <div className="p-5 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl space-y-4">
              <div className="flex items-center justify-between border-b border-[#464554] pb-3">
                <h3 className="text-sm font-bold text-white flex items-center gap-2 font-['Outfit']">
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
                        <span className={`px-2 py-0.5 rounded-none text-[10px] font-mono font-semibold uppercase tracking-wider border shrink-0 ${getStatusBadgeStyles(act.status)}`}>
                          {act.status}
                        </span>
                      </div>
                      <div className="text-[11px] text-[#908fa0] font-mono leading-relaxed px-0.5">
                        Duration: {formatDuration(act.execution_duration_ms)} | Completed: {new Date(act.completed_at).toLocaleTimeString()}
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            {/* Suppressions & Safeguards Panel */}
            <div className="p-5 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl space-y-4">
              <div className="flex items-center justify-between border-b border-[#464554] pb-3">
                <h3 className="text-sm font-bold text-white flex items-center gap-2 font-['Outfit']">
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

      <SignalSimulationModal
        isOpen={isSignalModalOpen}
        onClose={() => setIsSignalModalOpen(false)}
        runId={activeRunId}
        workflowId={activeSignalWfId || `wf-${parameters.workflow_id}-${activeRunId}`}
        subjectId={activeSignalSubjectId || runDetailData?.timeline?.subject_id}
        onFireToast={onFireToast}
      />
    </div>
  );
};

export default RunDetailPage;
