import React, { useState, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { JourneyApiClient } from '../api/client';
import { useEditorStore } from '../stores/editorStore';
import { DegradedStateView } from '../components/DegradedStateView';
import { Skeleton } from '../components/Skeleton';
import { Button } from '../components/common/Button';
import { Modal } from '../components/common/Modal';
import { PaginatedTable, ColumnDef } from '../components/common/PaginatedTable';
import { DirectoryLayout } from '../components/common/DirectoryLayout';
import { StatusFilterDropdown } from '../components/common/StatusFilterDropdown';
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

  const journeyColumns: ColumnDef<JourneyItem>[] = useMemo(
    () => [
      {
        key: 'name',
        header: 'Journey Name & ID',
        cell: (j) => (
          <div>
            <div className="font-semibold text-white text-sm font-['Outfit']">{j.name}</div>
            <div className="text-[11px] font-mono text-[#c0c1ff] mt-0.5">{j.draft_id}</div>
            {j.description && <div className="text-xs text-[#908fa0] mt-1">{j.description}</div>}
          </div>
        ),
      },
      {
        key: 'tenant_id',
        header: 'Tenant',
        cell: (j) => <span className="font-mono text-[#dfe2f1]">{j.tenant_id}</span>,
      },
      {
        key: 'status',
        header: 'Status',
        cell: (j) => (
          <span
            className={`px-2 py-0.5 rounded-none text-[10px] font-mono font-semibold uppercase border tracking-wider ${getStatusStyles(
              j.status
            )}`}
          >
            {j.status.toUpperCase()}
          </span>
        ),
      },
      {
        key: 'version',
        header: 'Version',
        cell: (j) => <span className="font-mono text-[#dfe2f1]">v{j.version}</span>,
      },
      {
        key: 'nodes',
        header: 'Nodes',
        cell: (j) => <span className="font-mono text-[#dfe2f1]">{j.node_count} nodes</span>,
      },
      {
        key: 'updated_at',
        header: 'Last Updated',
        cell: (j) => (
          <span className="font-mono text-[#908fa0] text-[11px]">
            {new Date(j.updated_at).toLocaleString()}
          </span>
        ),
      },
      {
        key: 'actions',
        header: 'Actions',
        cell: (j) => (
          <Button
            variant="secondary-dark"
            size="sm"
            icon="arrow_forward"
            onClick={() => onSelectJourney?.(j.draft_id)}
            className="bg-[#b76dff]/20 hover:bg-[#b76dff]/40 text-[#ddb7ff] border-[#ddb7ff]/30 font-semibold"
          >
            Open Canvas
          </Button>
        ),
      },
    ],
    [onSelectJourney]
  );

  return (
    <DirectoryLayout
      title="Journeys Directory"
      subtitle="Browse, search, and manage all event-driven journey workflows."
      icon="account_tree"
      iconAccentColor="#ddb7ff"
      controls={
        <>
          <div className="flex flex-wrap items-center gap-3 flex-1 min-w-[280px]">
            <div className="flex-1 min-w-[200px]">
              <input
                type="text"
                placeholder="Search by journey name, ID, description, or tenant..."
                value={params.search}
                onChange={(e) => setParams({ search: e.target.value, page: '1' })}
                data-testid="journeys-search-input"
                className="w-full px-3 py-1.5 bg-[#171b26] border border-[#464554] text-white text-xs placeholder-[#908fa0] focus:outline-none focus:border-[#ddb7ff] transition-all rounded-none font-['Outfit',sans-serif]"
              />
            </div>

            <StatusFilterDropdown
              value={params.status}
              onChange={(newStatus) => setParams({ status: newStatus, page: '1' })}
              options={[
                { status: 'all', label: 'All Statuses' },
                { status: 'active', label: 'Active' },
                { status: 'draft', label: 'Draft' },
                { status: 'paused', label: 'Paused' },
                { status: 'archived', label: 'Archived' },
              ]}
              getStatusStyles={getStatusStyles}
              dataTestId="journeys-status-filter-dropdown"
            />
          </div>

          <Button
            type="button"
            onClick={() => setParams({ search: '', status: 'all', page: '1' })}
            aria-label="Clear all active table filters"
            title="Clear all active table filters"
            variant="secondary-dark"
            icon="filter_alt_off"
            data-testid="journeys-clear-filters-btn"
          />
        </>
      }
    >
      {/* Main Content Area: Loading, Error, Empty, or Paginated Table */}
      {isLoading ? (
        <div className="p-6 bg-[#0F131D]/90 border border-[#464554] space-y-4">
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
      ) : (
        <PaginatedTable<JourneyItem>
          data={paginatedJourneys}
          columns={journeyColumns}
          getRowKey={(j) => j.draft_id}
          currentPage={currentPage}
          pageSize={pageSize}
          totalItems={filteredJourneys.length}
          onPageChange={(page) => setParams({ page: String(page) })}
          itemLabel="journeys"
          testId="journeys-table"
          emptyState={
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
          }
        />
      )}

      {isModalOpen && (
        <Modal
          isOpen={isModalOpen}
          onClose={() => setIsModalOpen(false)}
          title="Create New Journey Draft"
          subtitle="Configure business logic name and tenant metadata for isolated workflow candidate."
          icon="add_circle"
          iconAccentColor="#b76dff"
          maxWidth="lg"
          ariaLabelledBy="create-draft-modal-title"
          footer={
            <>
              <Button
                type="button"
                onClick={() => setIsModalOpen(false)}
                variant="secondary-dark"
              >
                Cancel
              </Button>
              <Button
                type="submit"
                onClick={handleCreateSubmit}
                disabled={createMutation.isPending}
                isLoading={createMutation.isPending}
                variant="primary-purple"
              >
                {createMutation.isPending ? 'Creating...' : 'Create Draft'}
              </Button>
            </>
          }
        >
          <form onSubmit={handleCreateSubmit} className="space-y-4">
            {formError && (
              <div className="p-3 bg-rose-500/10 border border-rose-500/30 text-rose-400 text-xs">
                {formError}
              </div>
            )}
            <div>
              <label className="block text-xs font-semibold text-[#dfe2f1] mb-1">
                Journey Name *
              </label>
              <input
                type="text"
                placeholder="e.g. VIP Re-engagement Workflow"
                value={newJourneyName}
                onChange={(e) => setNewJourneyName(e.target.value)}
                className="w-full px-3 py-1.5 bg-[#171b26] border border-[#464554] text-white text-xs placeholder-[#908fa0] focus:outline-none focus:border-[#ddb7ff] rounded-none font-['Outfit']"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-[#dfe2f1] mb-1">
                Description
              </label>
              <textarea
                rows={3}
                placeholder="Optional description of business logic or goal..."
                value={newJourneyDesc}
                onChange={(e) => setNewJourneyDesc(e.target.value)}
                className="w-full px-3 py-1.5 bg-[#171b26] border border-[#464554] text-white text-xs placeholder-[#908fa0] focus:outline-none focus:border-[#ddb7ff] rounded-none font-['Outfit']"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-[#dfe2f1] mb-1">
                Tenant ID
              </label>
              <input
                type="text"
                value={newJourneyTenant}
                onChange={(e) => setNewJourneyTenant(e.target.value)}
                className="w-full px-3 py-1.5 bg-[#171b26] border border-[#464554] text-white text-xs placeholder-[#908fa0] focus:outline-none focus:border-[#ddb7ff] rounded-none font-['Outfit']"
              />
            </div>
          </form>
        </Modal>
      )}
    </DirectoryLayout>
  );
};
export default JourneysPage;
