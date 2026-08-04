import { useState, useMemo } from 'react';
import { createPortal } from 'react-dom';
import { useQuery } from '@tanstack/react-query';
import { JourneyApiClient } from '../../api/client';
import { SafeQueryClientProvider } from '../SafeQueryClientProvider';

const apiClient = new JourneyApiClient();

export interface ParameterItem {
  key: string;
  token: string;
  name: string;
  category: 'event' | 'subject' | 'node' | 'experiment' | 'parameter';
  categoryLabel: string;
  description: string;
  dataClassification: 'PII' | 'NonPII' | 'Sensitive';
  exampleValue?: string;
}

export interface ParameterModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSelectToken: (token: string) => void;
  targetFieldLabel?: string;
  customParameters?: ParameterItem[];
}


function ParameterModalInner({
  isOpen,
  onClose,
  onSelectToken,
  targetFieldLabel,
  customParameters,
}: ParameterModalProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');

  const { data: fetchedParameters = [], isLoading: _isLoading } = useQuery<ParameterItem[]>({
    queryKey: ['parameterLibraryCatalog'],
    enabled: isOpen && !customParameters,
    queryFn: async () => {
      const [params, events, attrs] = await Promise.all([
        apiClient.getParameterCatalog(),
        apiClient.getEventCatalog(),
        apiClient.getAttributeCatalog(),
      ]);

      const items: ParameterItem[] = [];

      events.forEach((rec) => {
        const fieldName = rec.name || rec.record_id;
        items.push({
          key: rec.record_id || fieldName,
          token: (rec.schema_definition?.token as string) || `{{event.${fieldName}}}`,
          name: fieldName,
          category: 'event',
          categoryLabel: 'Event Field',
          description: rec.description || 'Event payload field',
          dataClassification: rec.tags?.includes('pii') ? 'PII' : rec.tags?.includes('sensitive') ? 'Sensitive' : 'NonPII',
          exampleValue: (rec.schema_definition?.exampleValue as string) || undefined,
        });
      });

      attrs.forEach((rec) => {
        const fieldName = rec.name || rec.record_id;
        items.push({
          key: rec.record_id || fieldName,
          token: (rec.schema_definition?.token as string) || `{{subject.${fieldName}}}`,
          name: fieldName,
          category: 'subject',
          categoryLabel: 'Subject Attribute',
          description: rec.description || 'Customer profile attribute',
          dataClassification: rec.tags?.includes('pii') ? 'PII' : rec.tags?.includes('sensitive') ? 'Sensitive' : 'NonPII',
          exampleValue: (rec.schema_definition?.exampleValue as string) || undefined,
        });
      });

      params.forEach((rec) => {
        const fieldName = rec.name || rec.record_id;
        items.push({
          key: rec.record_id || fieldName,
          token: (rec.schema_definition?.token as string) || `{{params.${fieldName}}}`,
          name: fieldName,
          category: 'parameter',
          categoryLabel: 'Shared Parameter',
          description: rec.description || 'Shared system parameter',
          dataClassification: rec.tags?.includes('pii') ? 'PII' : rec.tags?.includes('sensitive') ? 'Sensitive' : 'NonPII',
          exampleValue: (rec.schema_definition?.exampleValue as string) || undefined,
        });
      });

      return items;
    },
  });

  const allParameters = useMemo(() => {
    return customParameters || fetchedParameters;
  }, [customParameters, fetchedParameters]);
  const filteredParameters = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();

    return allParameters.filter((param) => {
      // Category check
      if (selectedCategory !== 'all' && param.category !== selectedCategory) {
        return false;
      }

      // Search query check
      if (!query) return true;

      return (
        param.token.toLowerCase().includes(query) ||
        param.name.toLowerCase().includes(query) ||
        param.description.toLowerCase().includes(query) ||
        param.key.toLowerCase().includes(query) ||
        param.dataClassification.toLowerCase().includes(query)
      );
    });
  }, [allParameters, searchQuery, selectedCategory]);

  if (!isOpen) return null;

  const categories = [
    { id: 'all', label: 'All Items' },
    { id: 'event', label: 'Event Fields' },
    { id: 'subject', label: 'Subject Attributes' },
    { id: 'node', label: 'Node Outputs' },
    { id: 'experiment', label: 'Experiment Context' },
    { id: 'parameter', label: 'Shared Parameters' },
  ];

  const modalContent = (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="parameter-modal-title"
      data-testid="parameter-library-modal"
      className="fixed inset-0 bg-[#0B0F19]/98 backdrop-blur-xl flex items-center justify-center p-4 sm:p-6 overflow-y-auto font-['Outfit',sans-serif]"
      style={{ zIndex: 999999 }}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        className="bg-[#0F131D] border border-[#464554] rounded-none w-full max-w-3xl max-h-[75vh] flex flex-col shadow-2xl overflow-hidden glass-modal shrink-0 relative"
        style={{ zIndex: 1000000 }}
      >
        {/* Header */}
        <div className="p-4 sm:p-5 border-b border-[#464554] bg-[#171b26] flex items-center justify-between shrink-0">
          <div>
            <div className="flex items-center gap-2">
              <span className="material-symbols-outlined text-[#4cd7f6] text-xl">token</span>
              <h2
                id="parameter-modal-title"
                className="font-['Outfit'] font-bold text-lg text-white"
              >
                Parameter Library
              </h2>
            </div>
            <p className="text-xs text-[#908fa0] mt-1">
              {targetFieldLabel
                ? `Insert dynamic parameter token into field "${targetFieldLabel}"`
                : 'Insert dynamic runtime tokens (event, subject, node outputs, experiments)'}
            </p>
          </div>

          <button
            onClick={onClose}
            aria-label="Close parameter library modal"
            data-testid="close-parameter-modal-btn"
            className="w-8 h-8 rounded-none bg-[#1c1f2a] text-[#908fa0] hover:text-white hover:bg-white/10 flex items-center justify-center transition-all border border-[#464554] cursor-pointer shrink-0 shadow-sm"
          >
            <span className="material-symbols-outlined text-lg">close</span>
          </button>
        </div>

        {/* Search & Category Filter Controls */}
        <div className="p-4 sm:p-5 border-b border-[#464554] bg-[#171b26]/80 backdrop-blur-md shrink-0">
          <div className="relative mb-3">
            <input
              type="text"
              placeholder="Search parameters by name, token (e.g. {{event.user_id}}), description..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              data-testid="parameter-search-input"
              autoFocus
              className="w-full px-4 pr-8 py-2.5 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1] placeholder-[#64748b] text-xs focus:outline-none focus:border-[#4cd7f6] focus:ring-1 focus:ring-[#4cd7f6] font-['Outfit',sans-serif]"
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-[#908fa0] hover:text-white transition-colors text-xs"
              >
                ✕
              </button>
            )}
          </div>

          {/* Category Tabs */}
          <div
            className="flex gap-2 overflow-x-auto pb-1 max-w-full"
            role="tablist"
            aria-label="Parameter Categories"
          >
            {categories.map((cat) => (
              <button
                key={cat.id}
                role="tab"
                aria-selected={selectedCategory === cat.id}
                data-testid={`param-category-tab-${cat.id}`}
                onClick={() => setSelectedCategory(cat.id)}
                className={`px-3 py-1.5 rounded-none text-xs font-semibold transition-all cursor-pointer ${
                  selectedCategory === cat.id
                    ? 'bg-[#262a35] text-[#c0c1ff] border border-[#c0c1ff]/40 font-bold shadow-sm'
                    : 'bg-[#1c1f2a] text-[#908fa0] hover:text-white hover:bg-[#262a35] border border-[#464554]'
                }`}
              >
                {cat.label}
              </button>
            ))}
          </div>
        </div>

        {/* Parameter List Content */}
        <div className="flex-1 min-h-0 overflow-y-auto p-4 sm:p-5 space-y-3">
          {filteredParameters.length === 0 ? (
            <div
              className="text-center py-12 text-[#908fa0] text-xs"
              data-testid="no-parameters-found"
            >
              <div className="text-3xl mb-2">🔎</div>
              <div className="font-semibold text-[#dfe2f1]">No parameters match your search</div>
              <div className="text-[11px] text-[#908fa0] mt-1">
                Try searching for a different token or category tab.
              </div>
            </div>
          ) : (
            <div className="flex flex-col gap-3">
              {filteredParameters.map((param) => (
                <div
                  key={param.key}
                  data-testid={`param-item-${param.key}`}
                  className="flex items-center justify-between p-4 rounded-none border border-[#464554] bg-[#11141d] hover:border-[#4cd7f6]/40 hover:bg-[#171b26] transition-all group"
                >
                  <div className="flex-1 pr-4">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="font-semibold text-sm text-white font-['Outfit']">
                        {param.name}
                      </span>
                      <span className="text-[10px] font-mono px-2 py-0.5 rounded-none bg-[#1c1f2a] text-[#c7c4d7] border border-[#464554]">
                        {param.categoryLabel}
                      </span>
                      {param.dataClassification === 'PII' && (
                        <span className="text-[10px] font-mono font-bold px-2 py-0.5 rounded-none bg-amber-500/20 text-amber-300 border border-amber-500/30 uppercase">
                          PII
                        </span>
                      )}
                      {param.dataClassification === 'Sensitive' && (
                        <span className="text-[10px] font-mono font-bold px-2 py-0.5 rounded-none bg-rose-500/20 text-rose-300 border border-rose-500/30 uppercase">
                          Sensitive
                        </span>
                      )}
                    </div>

                    <div className="mt-1 text-xs text-[#908fa0]">
                      {param.description}
                    </div>

                    <div className="mt-2 flex items-center gap-2 font-mono">
                      <code className="text-xs px-2 py-0.5 rounded-none bg-[#4cd7f6]/10 text-[#4cd7f6] border border-[#4cd7f6]/30 font-bold">
                        {param.token}
                      </code>
                      {param.exampleValue && (
                        <span className="text-[11px] text-[#908fa0]">
                          e.g. <em>{param.exampleValue}</em>
                        </span>
                      )}
                    </div>
                  </div>

                  <button
                    onClick={() => {
                      onSelectToken(param.token);
                      onClose();
                    }}
                    data-testid={`insert-token-btn-${param.key}`}
                    className="px-3.5 py-2 rounded-none bg-[#1c1f2a] hover:bg-[#262a35] text-[#c0c1ff] hover:text-white font-semibold text-xs border border-[#464554] hover:border-[#c0c1ff]/40 shadow-sm transition-all cursor-pointer shrink-0 inline-flex items-center gap-1.5"
                  >
                    <span className="material-symbols-outlined text-sm text-[#c0c1ff]">add_circle</span>
                    <span>Insert Token</span>
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-6 py-3.5 border-t border-[#464554] bg-[#171b26] flex items-center justify-between shrink-0">
          <span className="text-xs font-mono text-[#908fa0]">
            Showing {filteredParameters.length} parameters
          </span>
          <button
            onClick={onClose}
            className="px-4 py-2.5 rounded-none bg-[#1c1f2a] hover:bg-[#262a35] text-[#dfe2f1] hover:text-white font-semibold text-xs border border-[#464554] transition-all cursor-pointer"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );

  if (typeof document !== 'undefined') {
    return createPortal(modalContent, document.body);
  }
  return modalContent;
}

export function ParameterModal(props: ParameterModalProps) {
  return (
    <SafeQueryClientProvider>
      <ParameterModalInner {...props} />
    </SafeQueryClientProvider>
  );
}
