import React, { useState } from 'react';
import { QueryClient, QueryClientProvider, useQuery, useQueryClient } from '@tanstack/react-query';
import { JourneyApiClient } from '../../api/client';
import { Skeleton } from '../Skeleton';
import { Badge } from '../common/Badge';
import { DegradedStateView } from '../DegradedStateView';

export interface NodeVisitStep {
  stepIndex: number;
  nodeId: string;
  nodeName: string;
  nodeType: string;
  status: 'passed' | 'failed' | 'skipped';
  timestamp: string;
}

export interface MemberBranchDecision {
  memberId: string;
  nodeId: string;
  conditionExpression: string;
  evaluatedResult: boolean;
  chosenBranch: string;
}

export interface MemberSuppression {
  memberId: string;
  maskedRecipient: string;
  reason: 'global_opt_out' | 'frequency_cap' | 'unverified_recipient' | 'control_group';
  suppressedAtNode: string;
  timestamp: string;
}

export interface FakeDeliveryStatus {
  deliveryId: string;
  memberId: string;
  maskedRecipient: string;
  channel: 'email' | 'sms' | 'webhook' | 'push';
  provider: 'mock-sendgrid' | 'mock-twilio' | 'mock-webhook' | 'mock-push';
  status: 'delivered' | 'bounced' | 'queued' | 'failed';
  responsePayload: string;
  timestamp: string;
}

export interface MemberExecutionDetail {
  memberId: string;
  maskedRecipient: string;
  status: 'completed' | 'suppressed' | 'failed';
  visitSteps: NodeVisitStep[];
}

export interface TestRunDetailData {
  testRunId: string;
  draftId: string;
  status: 'passed' | 'failed' | 'running';
  executionMode: 'realistic' | 'forced_variant_coverage';
  fixturePack: string;
  totalMembers: number;
  completedMembers: number;
  suppressedMembersCount: number;
  executionTimeMs: number;
  createdAt: string;
  members: MemberExecutionDetail[];
  branchDecisions: MemberBranchDecision[];
  suppressions: MemberSuppression[];
  fakeDeliveries: FakeDeliveryStatus[];
}


export interface TestRunDetailViewProps {
  testRunId?: string;
  onBackToList?: () => void;
  initialData?: TestRunDetailData;
}

const apiClient = new JourneyApiClient();

function normalizeTestRunData(rawInput: unknown, fallbackId: string): TestRunDetailData {
  const raw = (rawInput && typeof rawInput === 'object' ? rawInput : {}) as Record<string, unknown>;
  const rawMembers = Array.isArray(raw.members) ? raw.members : undefined;
  const rawMemberResults = Array.isArray(raw.member_results) ? raw.member_results : undefined;
  const mockInputs = (raw.mock_inputs && typeof raw.mock_inputs === 'object' ? raw.mock_inputs : {}) as Record<string, unknown>;

  const members: MemberExecutionDetail[] =
    (rawMembers as MemberExecutionDetail[] | undefined) ||
    rawMemberResults?.map((mItem: unknown, idx: number) => {
      const m = (mItem && typeof mItem === 'object' ? mItem : {}) as Record<string, unknown>;
      return {
        memberId: String(m.member_id || m.memberId || `MBR-00${idx + 1}`),
        maskedRecipient: String(m.masked_recipient || m.maskedRecipient || `m***${idx + 1}@example.com`),
        status: m.status === 'passed' ? 'completed' : (m.status as MemberExecutionDetail['status']) || 'completed',
        visitSteps: (m.visit_steps || m.visitSteps || [
          { stepIndex: 1, nodeId: 'node-start', nodeName: 'User Signup Trigger', nodeType: 'trigger', status: 'passed', timestamp: '14:00:00.010' },
        ]) as NodeVisitStep[],
      };
    }) ||
    [];

  const branchDecisions: MemberBranchDecision[] =
    (Array.isArray(raw.branchDecisions) ? (raw.branchDecisions as MemberBranchDecision[]) : undefined) ||
    (Array.isArray(raw.branch_decisions) ? (raw.branch_decisions as MemberBranchDecision[]) : undefined) ||
    [];

  const suppressions: MemberSuppression[] =
    (Array.isArray(raw.suppressions) ? (raw.suppressions as MemberSuppression[]) : undefined) ||
    [];

  const fakeDeliveries: FakeDeliveryStatus[] =
    (Array.isArray(raw.fakeDeliveries) ? (raw.fakeDeliveries as FakeDeliveryStatus[]) : undefined) ||
    (Array.isArray(raw.fake_deliveries) ? (raw.fake_deliveries as FakeDeliveryStatus[]) : undefined) ||
    [];

  return {
    testRunId: String(raw.test_run_id || raw.testRunId || fallbackId),
    draftId: String(raw.draft_id || raw.draftId || 'draft-welcome-v1'),
    status: (raw.status as TestRunDetailData['status']) || 'passed',
    executionMode: (mockInputs.execution_mode || raw.executionMode || 'realistic') as TestRunDetailData['executionMode'],
    fixturePack: String(mockInputs.fixture_pack || raw.fixturePack || 'Standard Fixture Pack (50 contacts)'),
    totalMembers: typeof raw.totalMembers === 'number' ? raw.totalMembers : typeof raw.total_members === 'number' ? raw.total_members : (members.length > 0 ? members.length : 0),
    completedMembers: typeof raw.completedMembers === 'number' ? raw.completedMembers : typeof raw.completed_members === 'number' ? raw.completed_members : members.filter((m) => m.status === 'completed').length,
    suppressedMembersCount: typeof raw.suppressedMembersCount === 'number' ? raw.suppressedMembersCount : typeof raw.suppressed_members_count === 'number' ? raw.suppressed_members_count : suppressions.length,
    executionTimeMs: typeof raw.execution_time_ms === 'number' ? raw.execution_time_ms : typeof raw.executionTimeMs === 'number' ? raw.executionTimeMs : 0,
    createdAt: String(raw.created_at || raw.createdAt || new Date().toISOString()),
    members,
    branchDecisions,
    suppressions,
    fakeDeliveries,
  };
}
const defaultQueryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
    },
  },
});

function SafeQueryClientProvider({ children }: { children: React.ReactNode }) {
  try {
    const client = useQueryClient();
    if (client) {
      return <>{children}</>;
    }
  } catch {
    // Fallback for tests or components rendered outside top-level QueryClientProvider
  }
  return <QueryClientProvider client={defaultQueryClient}>{children}</QueryClientProvider>;
}

function TestRunDetailViewInner({
  testRunId,
  onBackToList,
  initialData,
}: TestRunDetailViewProps) {
  const { data: queryData, isLoading, isError, refetch } = useQuery({
    queryKey: ['test-run', testRunId],
    queryFn: () => apiClient.getTestRunStatus(testRunId || 'tr-static-901'),
    enabled: !initialData,
  });

  const [activeTab, setActiveTab] = useState<'timeline' | 'branches' | 'suppressions' | 'deliveries'>('timeline');
  const [selectedMemberId, setSelectedMemberId] = useState<string | null>(null);

  if (!initialData && isLoading) {
    return (
      <div className="w-full flex flex-col gap-6 p-6">
        <Skeleton count={5} height="3rem" />
      </div>
    );
  }

  if (!initialData && (isError || (!queryData && !initialData))) {
    return (
      <div className="w-full flex flex-col gap-6 p-6">
        <DegradedStateView
          type="failed-test-run"
          title="Test Run Details Unavailable"
          description={`Unable to fetch test run execution details for test run ID "${testRunId}".`}
          onRetry={() => refetch()}
        />
      </div>
    );
  }

  const rawData = initialData || queryData;
  const data = normalizeTestRunData(rawData, testRunId || 'tr-static-901');

  const activeMemberId = selectedMemberId || data.members[0]?.memberId || '';
  const currentMemberDetail = data.members.find((m) => m.memberId === activeMemberId) || data.members[0];
  return (
    <div className="w-full flex flex-col gap-6 text-[#dfe2f1] font-['Outfit',sans-serif]">
      {/* Test Mode Banner Header */}
      <div className="bg-[#0F131D]/90 backdrop-blur-xl p-5 rounded-none border border-[#464554] flex flex-wrap items-center justify-between gap-4 shadow-xl">
        <div className="flex items-center gap-3">
          {onBackToList && (
            <button
              onClick={onBackToList}
              className="px-4 py-2 rounded-none bg-[#171b26] border border-[#464554] text-[#dfe2f1] hover:text-white hover:border-[#c0c1ff] text-xs font-semibold transition-all cursor-pointer"
            >
              ← Back to Runs
            </button>
          )}
          <div>
            <div className="flex items-center gap-3">
              <h2 className="font-['Outfit'] font-bold text-xl text-white tracking-tight">
                Test-Run Results: <code className="text-[#4cd7f6] font-mono">{data.testRunId}</code>
              </h2>
              <Badge variant="cyan" testId="test-mode-badge">
                TEST MODE
              </Badge>
            </div>
            <p className="text-xs text-[#908fa0] mt-1 font-mono">
              Target Draft: <strong className="text-white">{data.draftId}</strong> | Mode: <strong className="text-[#4cd7f6]">{data.executionMode}</strong>
            </p>
          </div>
        </div>

        <div className="text-right">
          <span
            className={`px-3 py-1 rounded-none text-xs font-mono font-bold uppercase tracking-wider border ${
              data.status === 'passed'
                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                : 'bg-rose-500/10 text-rose-400 border-rose-500/20'
            }`}
          >
            {data.status}
          </span>
          <div className="text-xs text-[#908fa0] font-mono mt-1.5">
            Duration: <strong className="text-[#6ee7b7]">{data.executionTimeMs} ms</strong>
          </div>
        </div>
      </div>
      {/* Summary KPI Bar */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="p-4 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border border-white/10 shadow-xl font-mono">
          <div className="text-[10px] text-slate-400 uppercase tracking-wider">Total Test Members</div>
          <div className="text-2xl font-bold text-white mt-1">{data.totalMembers}</div>
        </div>
        <div className="p-4 rounded-none bg-emerald-500/10 backdrop-blur-xl border border-emerald-500/20 shadow-xl font-mono">
          <div className="text-[10px] text-emerald-400 uppercase tracking-wider">Completed Journey</div>
          <div className="text-2xl font-bold text-emerald-300 mt-1">{data.completedMembers}</div>
        </div>
        <div className="p-4 rounded-none bg-amber-500/10 backdrop-blur-xl border border-amber-500/20 shadow-xl font-mono">
          <div className="text-[10px] text-amber-400 uppercase tracking-wider">Suppressions</div>
          <div className="text-2xl font-bold text-amber-300 mt-1">{data.suppressedMembersCount}</div>
        </div>
        <div className="p-4 rounded-none bg-indigo-500/10 backdrop-blur-xl border border-indigo-500/20 shadow-xl font-mono">
          <div className="text-[10px] text-indigo-400 uppercase tracking-wider">Fake Deliveries</div>
          <div className="text-2xl font-bold text-indigo-300 mt-1">{data.fakeDeliveries.length}</div>
        </div>
      </div>

      {/* Main Tabbed Interface */}
      <div className="bg-[#0F131D]/90 backdrop-blur-xl rounded-none border border-white/10 shadow-xl overflow-hidden">
        {/* Navigation Tabs */}
        <div className="flex border-b border-white/10 bg-slate-900/60 px-4 gap-2">
          {[
            { id: 'timeline', label: 'Per-Member Node Visit Steps' },
            { id: 'branches', label: `Branch Decisions (${data.branchDecisions.length})` },
            { id: 'suppressions', label: `Suppressions (${data.suppressions.length})` },
            { id: 'deliveries', label: `Fake Delivery Status (${data.fakeDeliveries.length})` },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`py-3.5 px-4 font-mono text-xs font-semibold transition-all border-b-2 cursor-pointer ${
                activeTab === tab.id
                  ? 'border-[#4cd7f6] text-[#4cd7f6] bg-[#4cd7f6]/10'
                  : 'border-transparent text-[#908fa0] hover:text-white hover:bg-white/5'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Tab 1: Per-Member Node Visit Steps */}
        {activeTab === 'timeline' && (
          <div className="p-6 grid grid-cols-1 md:grid-cols-4 gap-6">
            {/* Member Selector Sidebar */}
            <div className="border-r border-white/10 pr-4 space-y-2 font-mono">
              <h4 className="font-['Outfit'] font-bold text-xs text-slate-400 uppercase tracking-wider mb-2">
                Select Member
              </h4>
              <div className="flex flex-col gap-2">
                {data.members.map((m) => (
                  <button
                    key={m.memberId}
                    onClick={() => setSelectedMemberId(m.memberId)}
                    className={`text-left p-3 rounded-none border transition-all ${
                      activeMemberId === m.memberId
                        ? 'border-indigo-500/50 bg-indigo-500/10 text-white shadow-lg shadow-indigo-500/10'
                        : 'border-white/10 bg-slate-900/60 text-slate-400 hover:text-white hover:bg-white/5'
                    }`}
                  >
                    <div className="font-semibold text-xs text-white">{m.memberId}</div>
                    <div className="text-[10px] text-indigo-300 font-mono mt-0.5">{m.maskedRecipient}</div>
                  </button>
                ))}
              </div>
            </div>

            {/* Member Steps Timeline */}
            <div>
              {currentMemberDetail ? (
                <div className="space-y-4">
                  <div className="flex items-center justify-between pb-3 border-b border-[#464554]">
                    <h3 className="font-['Outfit'] font-bold text-sm text-white">
                      Execution Path for <code className="text-[#4cd7f6] font-mono">{currentMemberDetail.memberId}</code> ({currentMemberDetail.maskedRecipient})
                    </h3>
                    <span
                      className={`px-2.5 py-1 rounded-none text-[10px] font-mono font-bold uppercase tracking-wider border ${
                        currentMemberDetail.status === 'completed'
                          ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                          : 'bg-amber-500/10 text-amber-400 border-amber-500/20'
                      }`}
                    >
                      {currentMemberDetail.status}
                    </span>
                  </div>

                  <div className="space-y-3">
                    {currentMemberDetail.visitSteps.map((step) => (
                      <div
                        key={step.stepIndex}
                        className="flex items-start gap-4 p-4 rounded-none bg-[#171b26] border border-[#464554] shadow-sm"
                      >
                        <div className="w-7 h-7 rounded-none bg-[#4cd7f6]/10 text-[#4cd7f6] border border-[#4cd7f6]/30 flex items-center justify-center font-mono text-xs font-bold shrink-0">
                          {step.stepIndex}
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center justify-between">
                            <div className="font-mono text-xs font-bold text-white">
                              {step.nodeName} <code className="text-[#908fa0] text-[11px]">({step.nodeId})</code>
                            </div>
                            <span className="text-[11px] text-[#908fa0] font-mono">{step.timestamp}</span>
                          </div>
                          <div className="mt-1 text-[11px] text-[#908fa0] font-mono">
                            Type: <strong className="uppercase text-[#dfe2f1]">{step.nodeType}</strong> | Status: <strong className={step.status === 'passed' ? 'text-emerald-400' : 'text-rose-400'}>{step.status}</strong>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ) : (
                <div className="text-[#908fa0] text-xs font-mono p-4">Select a member to view visit steps.</div>
              )}
            </div>
          </div>
        )}

        {/* Tab 2: Branch Decisions */}
        {activeTab === 'branches' && (
          <div className="p-6 space-y-4">
            <h3 className="font-['Outfit'] font-bold text-sm text-white">
              Condition & Experiment Branch Decisions
            </h3>
            <div className="rounded-none border border-[#464554] bg-[#11141d] overflow-hidden">
              <table className="w-full text-left text-xs font-mono">
                <thead>
                  <tr className="bg-[#171b26] border-b border-[#464554] text-[#908fa0] text-[10px] uppercase">
                    <th className="p-3">Member ID</th>
                    <th className="p-3">Node ID</th>
                    <th className="p-3">Condition Expression</th>
                    <th className="p-3">Evaluated Result</th>
                    <th className="p-3">Chosen Branch Path</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#464554]/40">
                  {data.branchDecisions.map((b, idx) => (
                    <tr key={idx} className="hover:bg-[#171b26]/50 transition-colors">
                      <td className="p-3 text-[#dfe2f1] font-bold">{b.memberId}</td>
                      <td className="p-3 text-[#dfe2f1]">{b.nodeId}</td>
                      <td className="p-3 text-[#c0c1ff]">
                        <code>{b.conditionExpression}</code>
                      </td>
                      <td className="p-3">
                        <span
                          className={`px-2 py-0.5 rounded-none text-[10px] font-mono font-bold uppercase border ${
                            b.evaluatedResult
                              ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                              : 'bg-rose-500/10 text-rose-400 border-rose-500/20'
                          }`}
                        >
                          {String(b.evaluatedResult)}
                        </span>
                      </td>
                      <td className="p-3 text-[#4cd7f6] font-semibold">{b.chosenBranch}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Tab 3: Suppressions */}
        {activeTab === 'suppressions' && (
          <div className="p-6 space-y-4">
            <h3 className="font-['Outfit'] font-bold text-sm text-white">
              Target Suppressions & Opt-Out Log
            </h3>
            <div className="rounded-none border border-[#464554] bg-[#11141d] overflow-hidden">
              <table className="w-full text-left text-xs font-mono">
                <thead>
                  <tr className="bg-[#171b26] border-b border-[#464554] text-[#908fa0] text-[10px] uppercase">
                    <th className="p-3">Member ID</th>
                    <th className="p-3">Masked Contact</th>
                    <th className="p-3">Suppression Rule / Reason</th>
                    <th className="p-3">Suppressed At Node</th>
                    <th className="p-3">Timestamp</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#464554]/40">
                  {data.suppressions.map((s, idx) => (
                    <tr key={idx} className="hover:bg-[#171b26]/50 transition-colors">
                      <td className="p-3 text-[#dfe2f1] font-bold">{s.memberId}</td>
                      <td className="p-3 text-[#4cd7f6]">{s.maskedRecipient}</td>
                      <td className="p-3">
                        <span className="px-2 py-0.5 rounded-none text-[10px] font-mono font-bold uppercase tracking-wider bg-amber-500/10 text-amber-300 border border-amber-500/20">
                          {s.reason.replace(/_/g, ' ')}
                        </span>
                      </td>
                      <td className="p-3 text-[#dfe2f1]">{s.suppressedAtNode}</td>
                      <td className="p-3 text-[#908fa0]">{s.timestamp}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Tab 4: Fake Delivery Status */}
        {activeTab === 'deliveries' && (
          <div className="p-6 space-y-4">
            <h3 className="font-['Outfit'] font-bold text-sm text-white">
              Fake Delivery Dispatch Log
            </h3>
            <div className="rounded-none border border-[#464554] bg-[#11141d] overflow-hidden">
              <table className="w-full text-left text-xs font-mono">
                <thead>
                  <tr className="bg-[#171b26] border-b border-[#464554] text-[#908fa0] text-[10px] uppercase">
                    <th className="p-3">Delivery ID</th>
                    <th className="p-3">Member ID</th>
                    <th className="p-3">Mock Provider</th>
                    <th className="p-3">Channel</th>
                    <th className="p-3">Delivery Status</th>
                    <th className="p-3">Response Payload</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#464554]/40">
                  {data.fakeDeliveries.map((d) => (
                    <tr key={d.deliveryId} className="hover:bg-[#171b26]/50 transition-colors">
                      <td className="p-3 text-[#dfe2f1] font-bold">{d.deliveryId}</td>
                      <td className="p-3 text-[#dfe2f1]">{d.memberId}</td>
                      <td className="p-3 text-[#4cd7f6] font-semibold">{d.provider}</td>
                      <td className="p-3 uppercase text-[10px] font-semibold text-[#c0c1ff]">{d.channel}</td>
                      <td className="p-3">
                        <span
                          className={`px-2 py-0.5 rounded-none text-[10px] font-mono font-bold uppercase tracking-wider border ${
                            d.status === 'delivered'
                              ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                              : d.status === 'bounced'
                              ? 'bg-rose-500/10 text-rose-400 border-rose-500/20'
                              : 'bg-amber-500/10 text-amber-300 border border-amber-500/20'
                          }`}
                        >
                          {d.status}
                        </span>
                      </td>
                      <td className="p-3 text-[11px] text-[#908fa0]">{d.responsePayload}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export function TestRunDetailView(props: TestRunDetailViewProps) {
  return (
    <SafeQueryClientProvider>
      <TestRunDetailViewInner {...props} />
    </SafeQueryClientProvider>
  );
}
