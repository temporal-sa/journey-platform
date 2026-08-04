import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { JourneyApiClient } from '../api/client';
import { DegradedStateView } from '../components/DegradedStateView';
import { useRouteParams } from '../hooks/useRouteParams';
import type { CatalogRecord } from '../types/api';

const apiClient = new JourneyApiClient();

export type CatalogCategory = 'events' | 'actions' | 'attributes' | 'parameters' | 'metrics' | 'templates';

const DEFAULT_CATALOG_PARAMS = {
  tab: 'events',
  search: '',
  selectedId: '',
};

export const CatalogPage: React.FC = () => {
  const [params, setParams] = useRouteParams(DEFAULT_CATALOG_PARAMS);

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

  const filteredItems = catalogItems.filter((item) => {
    if (!params.search) return true;
    const term = params.search.toLowerCase();
    return (
      item.name.toLowerCase().includes(term) ||
      item.record_id.toLowerCase().includes(term) ||
      item.description.toLowerCase().includes(term) ||
      item.tags.some((t) => t.toLowerCase().includes(term))
    );
  });

  const selectedRecord = catalogItems.find((item) => item.record_id === params.selectedId);

  const tabs: { key: CatalogCategory; label: string; icon: string }[] = [
    { key: 'events', label: 'Events Catalog', icon: 'electric_bolt' },
    { key: 'actions', label: 'Actions Catalog', icon: 'mail' },
    { key: 'attributes', label: 'Attributes', icon: 'tune' },
    { key: 'parameters', label: 'Parameters', icon: 'code' },
    { key: 'metrics', label: 'Metrics', icon: 'analytics' },
    { key: 'templates', label: 'Templates', icon: 'dashboard' },
  ];

  return (
    <div className="w-full h-full flex flex-col p-6 bg-[#0B0F19] text-[#DFE2F1] font-['Outfit',sans-serif] overflow-y-auto space-y-6">
      {/* Page Header */}
      <div className="w-full flex flex-col sm:flex-row sm:items-center justify-between gap-4 shrink-0">
        <div>
          <h1 aria-label="Component Catalog & Schemas" className="font-['Outfit'] font-bold text-2xl text-white flex items-center gap-2.5">
            <span className="material-symbols-outlined text-[#c0c1ff] text-2xl">inventory_2</span>
            <span>Component Catalog & Schemas</span>
          </h1>
          <p className="text-xs text-[#908fa0] mt-1">
            Explore event schemas, side-effect actions, subject attributes, parameters, metrics, and templates.
          </p>
        </div>
      </div>

      {/* Tabs Navigation Bar */}
      <div className="w-full flex items-center border-b border-[#464554] overflow-x-auto shrink-0">
        {tabs.map((tab) => {
          const isActive = activeTab === tab.key;
          return (
            <button
              key={tab.key}
              onClick={() => setParams({ tab: tab.key, selectedId: '' })}
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
      <div className="w-full p-4 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl flex items-center gap-4 shrink-0">
        <div className="flex-1 relative">
          <input
            type="text"
            placeholder={`Search ${activeTab} by name, tag, ID, or description...`}
            value={params.search}
            onChange={(e) => setParams({ search: e.target.value })}
            className="w-full px-4 py-2 rounded-none bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-['Outfit',sans-serif]"
          />
        </div>
        {params.search && (
          <button
            onClick={() => setParams({ search: '' })}
            className="px-3 py-2 rounded-none bg-[#262a35] hover:bg-[#333846] text-[#dfe2f1] text-xs font-semibold transition-colors cursor-pointer"
          >
            Clear Search
          </button>
        )}
      </div>

      {/* Content Area (Grid + Drawer) */}
      <div className="w-full flex-1 flex gap-6 items-start min-w-0">
        {/* Left Side Component Items */}
        <div className="flex-1 min-w-0 w-full">
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
              onRefresh={() => setParams({ search: '' })}
            />
          ) : (
            <div className="grid grid-cols-5 gap-4 w-full min-w-0" style={{ gridTemplateColumns: 'repeat(5, minmax(0, 1fr))' }}>
              {filteredItems.map((item) => {
                const isSelected = item.record_id === params.selectedId;
                return (
                  <div
                    key={item.record_id}
                    onClick={() => setParams({ selectedId: isSelected ? '' : item.record_id })}
                    className={`p-4 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border transition-all cursor-pointer shadow-xl flex flex-col justify-between min-w-0 ${
                      isSelected
                        ? 'border-[#ddb7ff] bg-[#1c1f2a] ring-2 ring-[#ddb7ff]/30'
                        : 'border-[#464554] hover:border-[#c0c1ff] hover:bg-[#171b26]'
                    }`}
                  >
                    <div>
                      <div className="flex items-center justify-between gap-2 mb-2">
                        <span className="px-2 py-0.5 rounded-none bg-[#c0c1ff]/20 text-[#ddb7ff] border border-[#ddb7ff]/30 text-[10px] font-mono font-bold uppercase tracking-wider">
                          {item.component_type}
                        </span>
                        <span className="text-[11px] font-mono text-[#908fa0]">v{item.version}</span>
                      </div>
                      <h3 className="font-['Outfit'] font-bold text-sm text-white mb-1 leading-snug">
                        {item.name}
                      </h3>
                      <p className="text-xs text-[#908fa0] leading-relaxed mb-3 line-clamp-2">
                        {item.description}
                      </p>
                    </div>
                    <div className="flex items-center justify-between gap-2 mt-auto pt-3 border-t border-white/5">
                      <div className="flex flex-wrap gap-1 min-w-0">
                        {item.tags.slice(0, 2).map((tag) => (
                          <span
                            key={tag}
                            className="px-2 py-0.5 rounded-none bg-[#171b26] text-[#908fa0] border border-[#464554] text-[10px] font-mono truncate"
                          >
                            #{tag}
                          </span>
                        ))}
                      </div>
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          setParams({ selectedId: isSelected ? '' : item.record_id });
                        }}
                        className="px-2.5 py-1 rounded-none bg-[#b76dff]/20 hover:bg-[#b76dff]/40 hover:brightness-125 text-[#ddb7ff] hover:text-white border border-[#ddb7ff]/30 hover:border-[#ddb7ff] text-[10px] font-bold transition-all cursor-pointer flex items-center gap-1 shrink-0 shadow-sm"
                      >
                        <span>{isSelected ? 'Active' : 'Spec'}</span>
                        <span className="material-symbols-outlined text-xs">arrow_forward</span>
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Right Side Drawer: Component Spec */}
        {selectedRecord && (
          <aside
            role="region"
            aria-label="Component Specification Drawer"
            className="w-96 bg-[#0F131D]/95 backdrop-blur-xl border border-[#464554] rounded-none p-5 shadow-2xl shrink-0 sticky top-0 space-y-4"
          >
            <div className="flex items-center justify-between border-b border-[#464554] pb-3">
              <h2 className="font-['Outfit'] font-bold text-base text-white flex items-center gap-2">
                <span className="material-symbols-outlined text-[#c0c1ff] text-lg">code</span>
                <span>Schema Specification</span>
              </h2>
              <button
                onClick={() => setParams({ selectedId: '' })}
                className="text-[#908fa0] hover:text-white transition-colors cursor-pointer text-sm"
              >
                ✕
              </button>
            </div>

            <div className="space-y-3 text-xs font-['Outfit',sans-serif]">
              <div>
                <span className="text-[10px] font-mono uppercase tracking-wider text-[#908fa0] block mb-0.5">Record ID</span>
                <code className="px-2 py-1 rounded-none bg-[#11141d] border border-[#464554] text-[#c0c1ff] font-mono text-xs block truncate">
                  {selectedRecord.record_id}
                </code>
              </div>

              <div>
                <span className="text-[10px] font-mono uppercase tracking-wider text-[#908fa0] block mb-0.5">Name</span>
                <strong className="text-white font-['Outfit'] text-sm">{selectedRecord.name}</strong>
              </div>

              <div>
                <span className="text-[10px] font-mono uppercase tracking-wider text-[#908fa0] block mb-0.5">Version</span>
                <span className="font-mono text-[#dfe2f1]">v{selectedRecord.version}</span>
              </div>

              <div>
                <span className="text-[10px] font-mono uppercase tracking-wider text-[#908fa0] block mb-0.5">Content Hash</span>
                <code className="text-[11px] font-mono text-[#908fa0] break-all block">
                  {selectedRecord.content_hash || 'hash-sha256-canonical'}
                </code>
              </div>

              <div>
                <span className="text-[10px] font-mono uppercase tracking-wider text-[#908fa0] block mb-0.5">Deprecation Status</span>
                <span className={`font-mono text-xs font-bold ${selectedRecord.is_deprecated ? 'text-rose-400' : 'text-emerald-400'}`}>
                  {selectedRecord.is_deprecated ? 'Deprecated' : 'Active & Supported'}
                </span>
              </div>

              <div>
                <span className="text-[10px] font-mono uppercase tracking-wider text-[#908fa0] block mb-1">
                  Schema Definition
                </span>
                <pre className="p-3 rounded-none bg-[#11141d] border border-[#464554] text-[#4cd7f6] font-mono text-[11px] overflow-x-auto max-h-56">
                  {JSON.stringify(selectedRecord.schema_definition || { type: 'object', properties: {} }, null, 2)}
                </pre>
              </div>
            </div>
          </aside>
        )}
      </div>
    </div>
  );
};

export default CatalogPage;
