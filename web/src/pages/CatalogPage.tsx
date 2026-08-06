import React, { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { JourneyApiClient } from '../api/client';
import { DegradedStateView } from '../components/DegradedStateView';
import { useRouteParams } from '../hooks/useRouteParams';
import type { CatalogRecord } from '../types/api';
import { PaginatedTable, ColumnDef } from '../components/common/PaginatedTable';
import { DirectoryLayout } from '../components/common/DirectoryLayout';
import { Modal } from '../components/common/Modal';
import { Button } from '../components/common/Button';
import { Badge } from '../components/common/Badge';

const apiClient = new JourneyApiClient();

export type CatalogCategory = 'events' | 'actions' | 'attributes' | 'parameters' | 'metrics' | 'templates';

const DEFAULT_CATALOG_PARAMS = {
  tab: 'events',
  search: '',
  selectedId: '',
};

export const CatalogPage: React.FC = () => {
  const [params, setParams] = useRouteParams(DEFAULT_CATALOG_PARAMS);
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 10;

  const activeTab = (params.tab as CatalogCategory) || 'events';

  // Fetch catalog data based on activeTab
  const {
    data: catalogItems = [],
    isLoading,
    isError,
    refetch,
  } = useQuery<CatalogRecord[]>({
    queryKey: ['catalog', activeTab],
    queryFn: () => apiClient.getCatalog(activeTab),
  });

  const filteredItems = useMemo(() => {
    return catalogItems.filter((item) => {
      if (!params.search) return true;
      const term = params.search.toLowerCase();
      return (
        item.name.toLowerCase().includes(term) ||
        item.record_id.toLowerCase().includes(term) ||
        item.description.toLowerCase().includes(term) ||
        item.tags.some((t) => t.toLowerCase().includes(term))
      );
    });
  }, [catalogItems, params.search]);

  const paginatedItems = useMemo(() => {
    const start = (currentPage - 1) * pageSize;
    return filteredItems.slice(start, start + pageSize);
  }, [filteredItems, currentPage, pageSize]);

  const selectedRecord = useMemo(() => {
    return catalogItems.find((item) => item.record_id === params.selectedId);
  }, [catalogItems, params.selectedId]);

  const handleAddComponentToPage = (record: CatalogRecord) => {
    if (typeof window !== 'undefined') {
      window.dispatchEvent(
        new CustomEvent('add-catalog-component', {
          detail: { record },
        })
      );
    }
    setParams({ selectedId: '' });
  };

  const tabs: { key: CatalogCategory; label: string; icon: string }[] = [
    { key: 'events', label: 'Events Catalog', icon: 'electric_bolt' },
    { key: 'actions', label: 'Actions Catalog', icon: 'mail' },
    { key: 'attributes', label: 'Attributes', icon: 'tune' },
    { key: 'parameters', label: 'Parameters', icon: 'code' },
    { key: 'metrics', label: 'Metrics', icon: 'analytics' },
    { key: 'templates', label: 'Templates', icon: 'dashboard' },
  ];

  const columns: ColumnDef<CatalogRecord>[] = useMemo(
    () => [
      {
        key: 'name',
        header: 'Component Name & ID',
        cell: (item) => (
          <div>
            <div className="font-['Outfit'] font-bold text-sm text-white flex items-center gap-2">
              <span>{item.name}</span>
            </div>
            <div className="text-[11px] font-mono text-[#c0c1ff] mt-0.5">
              {item.record_id}
            </div>
            <p className="text-xs text-[#908fa0] mt-1 line-clamp-1 max-w-md">
              {item.description}
            </p>
          </div>
        ),
      },
      {
        key: 'type',
        header: 'Type',
        headerClassName: 'w-32',
        cell: (item) => <Badge variant="purple">{item.component_type}</Badge>,
      },
      {
        key: 'version',
        header: 'Version',
        headerClassName: 'w-24',
        cell: (item) => <Badge variant="neutral">v{item.version}</Badge>,
      },
      {
        key: 'status',
        header: 'Status',
        headerClassName: 'w-32',
        cell: (item) => (
          <Badge variant={item.is_deprecated ? 'rose' : 'emerald'}>
            {item.is_deprecated ? 'Deprecated' : 'Active'}
          </Badge>
        ),
      },
      {
        key: 'tags',
        header: 'Tags',
        headerClassName: 'w-44',
        cell: (item) => (
          <div className="flex flex-wrap gap-1">
            {item.tags.slice(0, 3).map((tag) => (
              <Badge key={tag} variant="neutral" size="sm">
                #{tag}
              </Badge>
            ))}
          </div>
        ),
      },
      {
        key: 'actions',
        header: 'Actions',
        headerClassName: 'w-36',
        cell: (item) => (
          <Button
            variant="secondary-dark"
            size="sm"
            icon="visibility"
            onClick={() => setParams({ selectedId: item.record_id })}
            className="bg-[#b76dff]/20 hover:bg-[#b76dff]/40 text-[#ddb7ff] border-[#ddb7ff]/30 font-semibold"
          >
            Inspect
          </Button>
        ),
      },
    ],
    [setParams]
  );

  return (
    <DirectoryLayout
      title="Component Catalog & Schemas"
      subtitle="Explore event schemas, side-effect actions, subject attributes, parameters, metrics, and templates."
      icon="inventory_2"
      iconAccentColor="#c0c1ff"
      controls={
        <div className="w-full space-y-4">
          {/* Tabs Navigation Bar */}
          <div className="w-full flex items-center border-b border-[#464554] overflow-x-auto">
            {tabs.map((tab) => {
              const isActive = activeTab === tab.key;
              return (
                <button
                  key={tab.key}
                  onClick={() => {
                    setParams({ tab: tab.key, selectedId: '' });
                    setCurrentPage(1);
                  }}
                  style={{
                    backgroundColor: isActive ? 'rgba(183, 109, 255, 0.2)' : 'transparent',
                    color: isActive ? '#ddb7ff' : '#908fa0',
                    border: 'none',
                    borderBottom: isActive ? '2px solid #ddb7ff' : '2px solid transparent',
                    borderRadius: '0px',
                    outline: 'none',
                    boxShadow: 'none',
                  }}
                  className={`px-4 py-2.5 text-xs font-semibold flex items-center gap-2 transition-all cursor-pointer whitespace-nowrap rounded-none border-t-0 border-l-0 border-r-0 ${
                    isActive
                      ? 'catalog-tab-active text-[#ddb7ff]'
                      : 'catalog-tab-inactive text-[#908fa0] hover:text-[#dfe2f1]'
                  }`}
                >
                  <span className={`material-symbols-outlined text-base ${isActive ? 'text-[#ddb7ff]' : 'text-[#908fa0]'}`}>{tab.icon}</span>
                  <span>{tab.label}</span>
                </button>
              );
            })}
          </div>

          {/* Search Input Bar */}
          <div className="w-full flex items-center gap-4">
            <div className="flex-1 relative">
              <input
                type="text"
                placeholder={`Search ${activeTab} by name, tag, ID, or description...`}
                value={params.search}
                onChange={(e) => {
                  setParams({ search: e.target.value });
                  setCurrentPage(1);
                }}
                className="w-full px-4 py-2 rounded-none bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
              />
            </div>
            <Button
              type="button"
              variant="secondary-dark"
              icon="filter_alt_off"
              aria-label="Clear all active table filters"
              title="Clear all active table filters"
              data-testid="catalog-clear-filters-btn"
              onClick={() => {
                setParams({ search: '' });
                setCurrentPage(1);
              }}
            />
          </div>
        </div>
      }
    >

      {/* Content Area (Paginated Table) */}
      <div className="w-full flex-1 min-w-0">
        {isLoading ? (
          <div className="w-full flex justify-center items-center py-20 bg-[#0F131D]/90 backdrop-blur-xl rounded-none border border-[#464554] shadow-xl">
            <div className="flex items-center gap-3 text-[#c0c1ff] font-mono text-sm">
              <span className="material-symbols-outlined text-xl animate-spin">sync</span>
              <span>Loading Catalog Items...</span>
            </div>
          </div>
        ) : isError ? (
          <DegradedStateView
            type="api-disconnected"
            title={`Failed to Load Catalog (${activeTab})`}
            description="Could not fetch component schemas from the engine service."
            actionLabel="Retry Fetching"
            onRetry={() => { refetch(); }}
          />
        ) : filteredItems.length === 0 ? (
          <DegradedStateView
            type="stale-report"
            title={`No ${activeTab} Components Found`}
            description={
              params.search
                ? `No records matched your search query "${params.search}".`
                : `No records defined in the ${activeTab} catalog yet.`
            }
            actionLabel="Clear Search Filter"
            onRefresh={() => {
              setParams({ search: '' });
              setCurrentPage(1);
            }}
          />
        ) : (
          <PaginatedTable<CatalogRecord>
            data={paginatedItems}
            columns={columns}
            getRowKey={(item) => item.record_id}
            getRowTestId={(item) => `catalog-row-${item.record_id}`}
            currentPage={currentPage}
            pageSize={pageSize}
            totalItems={filteredItems.length}
            onPageChange={(page) => setCurrentPage(page)}
            itemLabel="components"
            testId="catalog-table"
          />
        )}
      </div>

      {/* Inspect Spec Modal */}
      {selectedRecord && (
        <Modal
          isOpen={Boolean(selectedRecord)}
          onClose={() => setParams({ selectedId: '' })}
          title="Component Schema Specification"
          subtitle={`Inspect schema definition and metadata for "${selectedRecord.name}"`}
          icon="code"
          iconAccentColor="#c0c1ff"
          maxWidth="3xl"
          maxHeight="80vh"
          testId="catalog-inspect-modal"
        >
          <div className="space-y-4 font-['Outfit',sans-serif] text-xs">
            <div className="flex flex-col gap-3 p-4 bg-[#11141d] border border-[#464554]">
              <div className="flex items-center gap-3 text-xs">
                <span className="text-[#908fa0] min-w-32">Record ID:</span>
                <code className="text-[#c0c1ff] font-mono font-bold truncate">
                  {selectedRecord.record_id}
                </code>
              </div>

              <div className="flex items-center gap-3 text-xs">
                <span className="text-[#908fa0] min-w-32">Component Type:</span>
                <Badge variant="purple">{selectedRecord.component_type}</Badge>
              </div>

              <div className="flex items-center gap-3 text-xs">
                <span className="text-[#908fa0] min-w-32">Version:</span>
                <Badge variant="neutral">v{selectedRecord.version}</Badge>
              </div>

              <div className="flex items-center gap-3 text-xs">
                <span className="text-[#908fa0] min-w-32">Deprecation Status:</span>
                <Badge variant={selectedRecord.is_deprecated ? 'rose' : 'emerald'}>
                  {selectedRecord.is_deprecated ? 'Deprecated' : 'Active & Supported'}
                </Badge>
              </div>

              <div className="flex items-center gap-3 text-xs">
                <span className="text-[#908fa0] min-w-32">Content Hash:</span>
                <code className="text-[#908fa0] font-mono break-all">
                  {selectedRecord.content_hash || 'hash-sha256-canonical'}
                </code>
              </div>
            </div>

            <div>
              <span className="text-[10px] font-mono uppercase tracking-wider text-[#908fa0] block mb-1">
                Description
              </span>
              <p className="text-sm text-white leading-relaxed">
                {selectedRecord.description}
              </p>
            </div>

            <div>
              <span className="text-[10px] font-mono uppercase tracking-wider text-[#908fa0] block mb-1.5">
                Tags
              </span>
              <div className="flex flex-wrap gap-1.5">
                {selectedRecord.tags.map((tag) => (
                  <Badge key={tag} variant="neutral">
                    #{tag}
                  </Badge>
                ))}
              </div>
            </div>

            <div>
              <span className="text-[10px] font-mono uppercase tracking-wider text-[#908fa0] block mb-1">
                JSON Schema Definition
              </span>
              <pre className="p-4 bg-[#11141d] border border-[#464554] text-[#4cd7f6] font-mono text-[11px] overflow-x-auto max-h-64">
                {JSON.stringify(
                  selectedRecord.schema_definition || { type: 'object', properties: {} },
                  null,
                  2
                )}
              </pre>
            </div>
          </div>
        </Modal>
      )}
    </DirectoryLayout>
  );
};

export default CatalogPage;
