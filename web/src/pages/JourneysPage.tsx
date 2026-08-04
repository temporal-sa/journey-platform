import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { JourneyApiClient } from '../api/client';
import { useEditorStore } from '../stores/editorStore';
import { DegradedStateView } from '../components/DegradedStateView';
import { Skeleton } from '../components/Skeleton';
import { useRouteParams } from '../hooks/useRouteParams';
import type { GraphDraft } from '../types/api';

const apiClient = new JourneyApiClient();

export interface JourneyItem {
  draft_id: string;
  tenant_id: string;
  name: string;
  description?: string;
  status: 'draft' | 'active' | 'paused' | 'archived';
  version: number;
  node_count: number;
  updated_at: string;
}


const DEFAULT_JOURNEY_PARAMS = {
  status: 'all',
  search: '',
  page: '1',
  pageSize: '5',
};

interface JourneysPageProps {
  onSelectJourney?: (draftId: string) => void;
}

export const JourneysPage: React.FC<JourneysPageProps> = ({ onSelectJourney }) => {
  const queryClient = useQueryClient();
  const currentDraft = useEditorStore((s) => s.currentDraft);
  const [params, setParams] = useRouteParams(DEFAULT_JOURNEY_PARAMS);
  const getStatusStyles = (status: string) => {
    const s = status.toLowerCase();
    if (s === 'active') return 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20';
    if (s === 'draft') return 'bg-amber-500/10 text-amber-400 border-amber-500/20';
    if (s === 'paused') return 'bg-purple-500/10 text-purple-300 border-purple-500/20';
    if (s === 'archived') return 'bg-slate-800 text-slate-400 border-slate-700';
    return 'bg-slate-800 text-slate-300 border-slate-700';
  };
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [newJourneyName, setNewJourneyName] = useState('');
  const [newJourneyDesc, setNewJourneyDesc] = useState('');
  const [newJourneyTenant, setNewJourneyTenant] = useState('tenant-default');
  const [formError, setFormError] = useState<string | null>(null);

  // Fetch list of journey drafts from server / API & merge active store draft
  const { data: journeys = [], isLoading, isError, refetch } = useQuery({
    queryKey: ['journeys', 'list', currentDraft?.draft_id, currentDraft?.name, currentDraft?.version, currentDraft?.updated_at],
    queryFn: async () => {
      const fetchedDrafts = await apiClient.listJourneyDrafts();
      let list: JourneyItem[] = (fetchedDrafts || []).map((d) => ({
        draft_id: d.draft_id,
        tenant_id: d.tenant_id,
        name: d.name,
        description: d.description,
        status: 'active' as const,
        version: d.version,
        node_count: d.nodes?.length || 0,
        updated_at: d.updated_at || new Date().toISOString(),
      }));

      if (currentDraft && currentDraft.draft_id) {
        const localItem: JourneyItem = {
          draft_id: currentDraft.draft_id,
          tenant_id: currentDraft.tenant_id || 'tenant-default',
          name: currentDraft.name || 'Untitled Journey',
          description: currentDraft.description || 'Canvas journey draft',
          status: 'draft' as const,
          version: currentDraft.version || 1,
          node_count: currentDraft.nodes?.length || 0,
          updated_at: currentDraft.updated_at || new Date().toISOString(),
        };
        list = [localItem, ...list.filter((j) => j.draft_id !== localItem.draft_id)];
      }

      return list;
    },
  });

  // Create Draft Mutation
  const createMutation = useMutation({
    mutationFn: async (payload: { name: string; description: string; tenantId: string }) => {
      const draft: GraphDraft = {
        schema_version: '1.0',
        draft_id: `draft-${Date.now().toString().slice(-4)}`,
        tenant_id: payload.tenantId,
        name: payload.name,
        description: payload.description,
        version: 1,
        nodes: [{ id: 'node-start', type: 'trigger', name: 'Start Event' }],
        edges: [],
      };
      const res = await apiClient.createJourneyDraft(draft);
      return res.draft;
    },
    onSuccess: (newDraft) => {
      queryClient.setQueryData(['journeys', 'list'], (old: JourneyItem[] = []) => [
        {
          draft_id: newDraft.draft_id,
          tenant_id: newDraft.tenant_id,
          name: newDraft.name,
          description: newDraft.description,
          status: 'draft' as const,
          version: newDraft.version,
          node_count: newDraft.nodes.length,
          updated_at: new Date().toISOString(),
        },
        ...old,
      ]);
      setIsModalOpen(false);
      setNewJourneyName('');
      setNewJourneyDesc('');
      setFormError(null);
      if (onSelectJourney) {
        onSelectJourney(newDraft.draft_id);
      }
    },
    onError: (err: Error) => {
      setFormError(err.message || 'Failed to create journey draft');
    },
  });

  const handleCreateSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newJourneyName.trim()) {
      setFormError('Journey name is required.');
      return;
    }
    setFormError(null);
    createMutation.mutate({
      name: newJourneyName.trim(),
      description: newJourneyDesc.trim(),
      tenantId: newJourneyTenant.trim() || 'tenant-default',
    });
  };

  // Filter & Search Logic
  const filteredJourneys = journeys.filter((j) => {
    const matchesStatus = params.status === 'all' || j.status === params.status;
    const matchesSearch =
      !params.search ||
      j.name.toLowerCase().includes(params.search.toLowerCase()) ||
      j.draft_id.toLowerCase().includes(params.search.toLowerCase()) ||
      (j.description && j.description.toLowerCase().includes(params.search.toLowerCase()));
    return matchesStatus && matchesSearch;
  });

  // Pagination
  const currentPage = Math.max(1, parseInt(params.page, 10) || 1);
  const pageSize = Math.max(1, parseInt(params.pageSize, 10) || 5);
  const totalPages = Math.ceil(filteredJourneys.length / pageSize) || 1;
  const paginatedJourneys = filteredJourneys.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  return (
    <div className="w-full h-full flex flex-col p-6 bg-[#0B0F19] text-[#DFE2F1] font-['Outfit',sans-serif] overflow-y-auto space-y-6">
      {/* Page Header */}
      <div className="w-full flex flex-col sm:flex-row sm:items-center justify-between gap-4 shrink-0">
        <div>
          <h1 aria-label="Journeys Directory" className="font-['Outfit'] font-bold text-2xl text-white flex items-center gap-2.5">
            <span className="material-symbols-outlined text-[#c0c1ff] text-2xl">map</span>
            <span>Journeys Directory</span>
          </h1>
          <p className="text-xs text-[#908fa0] mt-1">
            Manage lifecycle drafts, active production workflows, and versioned customer journeys.
          </p>
        </div>
      </div>

      {/* Filter & Search Bar */}
      <div className="w-full p-4 rounded-2xl bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl flex flex-wrap items-end gap-4 shrink-0">
        <div className="flex-1 min-w-[240px]">
          <label className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
            Search Journeys
          </label>
          <div className="relative">
            <input
              type="text"
              placeholder="Search by name, ID, or description..."
              value={params.search}
              onChange={(e) => setParams({ search: e.target.value, page: '1' })}
              className="w-full h-[34px] px-4 rounded-xl bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
            />
          </div>
        </div>

        {/* Status Filter Selector */}
        <div>
          <label className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
            Status Filter
          </label>
          <div className="flex flex-wrap items-center gap-1.5 h-[34px] bg-[#171b26] px-1.5 rounded-xl border border-[#464554]" role="group" aria-label="Status Filter">
            {[
              { status: 'all', label: 'All Statuses' },
              { status: 'active', label: 'Active' },
              { status: 'draft', label: 'Draft' },
              { status: 'paused', label: 'Paused' },
              { status: 'archived', label: 'Archived' },
            ].map((item) => {
              const isActive = params.status === item.status;
              return (
                <button
                  key={item.status}
                  type="button"
                  onClick={() => setParams({ status: item.status, page: '1' })}
                  className={`px-2 py-0.5 rounded text-[10px] font-mono font-semibold uppercase border tracking-wider transition-all cursor-pointer ${getStatusStyles(item.status)} ${
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

        <div>
          <label className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
            Per Page
          </label>
          <select
            aria-label="Per Page"
            value={params.pageSize}
            onChange={(e) => setParams({ pageSize: e.target.value, page: '1' })}
            className="h-[34px] px-3 rounded-xl bg-[#171b26] border border-[#464554] text-white text-xs focus:outline-none focus:border-[#c0c1ff] font-['Outfit',sans-serif] cursor-pointer"
          >
            <option value="5">5</option>
            <option value="10">10</option>
            <option value="20">20</option>
            <option value="50">50</option>
          </select>
        </div>

        {(params.search || params.status !== 'all') && (
          <button
            type="button"
            onClick={() => setParams({ search: '', status: 'all', page: '1' })}
            aria-label="Clear Filters"
            className="h-[34px] px-3 rounded-xl bg-[#b76dff]/20 hover:bg-[#b76dff]/40 text-[#ddb7ff] hover:text-white border border-[#ddb7ff]/30 hover:border-[#ddb7ff] text-xs font-semibold font-['Outfit'] transition-all cursor-pointer flex items-center gap-1.5 shadow-sm hover:brightness-125 shrink-0"
          >
            <span className="material-symbols-outlined text-sm">filter_alt_off</span>
            <span>Clear Filters</span>
          </button>
        )}
      </div>

      {/* Main Table / Content Section */}
      {isLoading ? (
        <div className="w-full p-6 rounded-2xl bg-[#0F131D]/90 border border-[#464554] shadow-xl">
          <Skeleton count={5} height="2.5rem" />
        </div>
      ) : isError ? (
        <DegradedStateView
          type="api-disconnected"
          title="Failed to Load Journeys"
          description="Could not connect to journey control API service."
          actionLabel="Retry Loading"
          onRetry={() => { refetch(); }}
        />
      ) : paginatedJourneys.length === 0 ? (
        <DegradedStateView
          type="stale-report"
          title="No Journeys Found"
          description={
            params.search || params.status !== 'all'
              ? 'No journeys matched your search or status filter criteria.'
              : 'There are no journey drafts or published workflows in this tenant yet.'
          }
          actionLabel="Clear Filters"
          onRefresh={() => setParams({ search: '', status: 'all', page: '1' })}
        />
      ) : (
        /* Journey List Table */
        <div className="w-full bg-[#0F131D]/90 backdrop-blur-xl rounded-2xl border border-[#464554] shadow-xl overflow-hidden flex-1 flex flex-col justify-between">
          <div className="overflow-x-auto w-full">
            <table className="w-full text-left text-xs font-['Outfit',sans-serif]">
              <thead>
                <tr className="bg-[#171b26] border-b border-[#464554] text-[#908fa0] font-mono uppercase tracking-wider text-[10px]">
                  <th style={{ textAlign: 'left' }} className="p-4 font-semibold text-left">Journey Name & ID</th>
                  <th style={{ textAlign: 'left' }} className="p-4 font-semibold text-left">Tenant</th>
                  <th style={{ textAlign: 'left' }} className="p-4 font-semibold text-left">Status</th>
                  <th style={{ textAlign: 'left' }} className="p-4 font-semibold text-left">Version</th>
                  <th style={{ textAlign: 'left' }} className="p-4 font-semibold text-left">Nodes</th>
                  <th style={{ textAlign: 'left' }} className="p-4 font-semibold text-left">Last Updated</th>
                  <th style={{ textAlign: 'left' }} className="p-4 font-semibold text-left">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {paginatedJourneys.map((j) => (
                  <tr key={j.draft_id} className="hover:bg-[#171b26]/50 transition-colors">
                    <td className="p-4">
                      <div className="font-semibold text-white text-sm font-['Outfit']">{j.name}</div>
                      <div className="text-[11px] font-mono text-[#c0c1ff] mt-0.5">{j.draft_id}</div>
                      {j.description && <div className="text-xs text-[#908fa0] mt-1">{j.description}</div>}
                    </td>
                    <td className="p-4 font-mono text-[#dfe2f1]">{j.tenant_id}</td>
                    <td className="p-4">
                      <span
                        className={`px-2 py-0.5 rounded text-[10px] font-mono font-semibold uppercase border tracking-wider ${
                          j.status === 'active'
                            ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                            : j.status === 'draft'
                            ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                            : j.status === 'paused'
                            ? 'bg-purple-500/10 text-purple-300 border border-purple-500/20'
                            : 'bg-slate-800 text-slate-400 border-slate-700'
                        }`}
                      >
                        {j.status.toUpperCase()}
                      </span>
                    </td>
                    <td className="p-4 font-mono text-[#dfe2f1]">v{j.version}</td>
                    <td className="p-4 font-mono text-[#dfe2f1]">{j.node_count} nodes</td>
                    <td className="p-4 font-mono text-[#908fa0] text-[11px]">
                      {new Date(j.updated_at).toLocaleString()}
                    </td>
                    <td className="p-4 text-left" style={{ textAlign: 'left' }}>
                      <button
                        onClick={() => onSelectJourney?.(j.draft_id)}
                        className="px-3.5 py-1.5 rounded-xl bg-[#b76dff]/20 hover:bg-[#b76dff]/40 hover:brightness-125 text-[#ddb7ff] hover:text-white border border-[#ddb7ff]/30 hover:border-[#ddb7ff] font-semibold text-xs transition-all cursor-pointer flex items-center gap-1.5 shadow-sm"
                      >
                        <span>Open Canvas</span>
                        <span className="material-symbols-outlined text-sm">arrow_forward</span>
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination Controls */}
          <div className="flex items-center justify-between p-4 border-t border-[#464554] bg-[#171b26]/80 font-mono text-xs w-full shrink-0">
            <div className="text-[#908fa0]">
              Showing {filteredJourneys.length > 0 ? (currentPage - 1) * pageSize + 1 : 0} to{' '}
              {Math.min(currentPage * pageSize, filteredJourneys.length)} of {filteredJourneys.length} journeys
            </div>
            <div className="flex items-center gap-2">
              <button
                disabled={currentPage <= 1 || totalPages <= 1}
                onClick={() => setParams({ page: String(currentPage - 1) })}
                style={{ cursor: (currentPage <= 1 || totalPages <= 1) ? 'not-allowed' : 'pointer' }}
                className="px-3 py-1.5 rounded-lg bg-[#11141d] border border-[#464554] text-[#dfe2f1] disabled:opacity-40 disabled:cursor-not-allowed disabled:text-[#908fa0] disabled:bg-[#171b26]/40 disabled:border-[#464554]/30 enabled:hover:bg-[#262a35] enabled:hover:text-white enabled:cursor-pointer transition-all text-xs font-mono"
              >
                Previous
              </button>
              <span className="text-xs text-[#908fa0] px-1">
                Page {currentPage} of {totalPages}
              </span>
              <button
                disabled={currentPage >= totalPages || totalPages <= 1}
                onClick={() => setParams({ page: String(currentPage + 1) })}
                style={{ cursor: (currentPage >= totalPages || totalPages <= 1) ? 'not-allowed' : 'pointer' }}
                className="px-3 py-1.5 rounded-lg bg-[#11141d] border border-[#464554] text-[#dfe2f1] disabled:opacity-40 disabled:cursor-not-allowed disabled:text-[#908fa0] disabled:bg-[#171b26]/40 disabled:border-[#464554]/30 enabled:hover:bg-[#262a35] enabled:hover:text-white enabled:cursor-pointer transition-all text-xs font-mono"
              >
                Next
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Create Draft Modal */}
      {isModalOpen && (
        <div
          role="dialog"
          aria-labelledby="create-draft-modal-title"
          style={{
            position: 'fixed',
            inset: 0,
            backgroundColor: 'rgba(15, 23, 42, 0.6)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 1000,
          }}
        >
          <div
            style={{
              backgroundColor: '#ffffff',
              padding: '1.5rem',
              borderRadius: '0.5rem',
              width: '100%',
              maxWidth: '440px',
              boxShadow: '0 20px 25px -5px rgba(0,0,0,0.1), 0 10px 10px -5px rgba(0,0,0,0.04)',
            }}
          >
            <h2 id="create-draft-modal-title" style={{ marginTop: 0, fontSize: '1.25rem' }}>
              Create New Journey Draft
            </h2>
            <form onSubmit={handleCreateSubmit}>
              {formError && (
                <div style={{ color: '#dc2626', fontSize: '0.875rem', marginBottom: '1rem' }}>
                  {formError}
                </div>
              )}
              <div style={{ marginBottom: '1rem' }}>
                <label style={{ display: 'block', fontSize: '0.875rem', fontWeight: 600, marginBottom: '0.25rem' }}>
                  Journey Name *
                </label>
                <input
                  type="text"
                  placeholder="e.g. VIP Re-engagement Workflow"
                  value={newJourneyName}
                  onChange={(e) => setNewJourneyName(e.target.value)}
                  style={{
                    width: '100%',
                    padding: '0.4rem 0.75rem',
                    border: '1px solid #cbd5e1',
                    borderRadius: '0.375rem',
                    fontSize: '0.875rem',
                    boxSizing: 'border-box',
                  }}
                />
              </div>

              <div style={{ marginBottom: '1rem' }}>
                <label style={{ display: 'block', fontSize: '0.875rem', fontWeight: 600, marginBottom: '0.25rem' }}>
                  Description
                </label>
                <textarea
                  rows={3}
                  placeholder="Optional description of business logic or goal..."
                  value={newJourneyDesc}
                  onChange={(e) => setNewJourneyDesc(e.target.value)}
                  style={{
                    width: '100%',
                    padding: '0.4rem 0.75rem',
                    border: '1px solid #cbd5e1',
                    borderRadius: '0.375rem',
                    fontSize: '0.875rem',
                    boxSizing: 'border-box',
                  }}
                />
              </div>

              <div style={{ marginBottom: '1.5rem' }}>
                <label style={{ display: 'block', fontSize: '0.875rem', fontWeight: 600, marginBottom: '0.25rem' }}>
                  Tenant ID
                </label>
                <input
                  type="text"
                  value={newJourneyTenant}
                  onChange={(e) => setNewJourneyTenant(e.target.value)}
                  style={{
                    width: '100%',
                    padding: '0.4rem 0.75rem',
                    border: '1px solid #cbd5e1',
                    borderRadius: '0.375rem',
                    fontSize: '0.875rem',
                    boxSizing: 'border-box',
                  }}
                />
              </div>

              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '0.75rem' }}>
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  style={{
                    padding: '0.4rem 0.8rem',
                    backgroundColor: '#f1f5f9',
                    color: '#475569',
                    border: 'none',
                    borderRadius: '0.375rem',
                    cursor: 'pointer',
                  }}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createMutation.isPending}
                  style={{
                    padding: '0.4rem 0.8rem',
                    backgroundColor: '#2563eb',
                    color: '#ffffff',
                    border: 'none',
                    borderRadius: '0.375rem',
                    cursor: 'pointer',
                    fontWeight: 600,
                  }}
                >
                  {createMutation.isPending ? 'Creating...' : 'Create Draft'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
export default JourneysPage;
