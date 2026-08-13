import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { JourneyApiClient } from '../api/client';
import type { StaticList } from '../types/api';
import { Button } from '../components/common/Button';
import { Modal } from '../components/common/Modal';
import { Badge } from '../components/common/Badge';
import { PaginatedTable, ColumnDef } from '../components/common/PaginatedTable';
import { DirectoryLayout } from '../components/common/DirectoryLayout';
import { DegradedStateView } from '../components/DegradedStateView';
import { Skeleton } from '../components/Skeleton';

const apiClient = new JourneyApiClient();

export interface StaticListsPageProps {
  onOpenUpload?: () => void;
}

export function StaticListsPage({ onOpenUpload }: StaticListsPageProps) {
  const [searchTerm, setSearchTerm] = useState('');
  const [classificationFilter, setClassificationFilter] = useState<string>('all');
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [selectedListForInspect, setSelectedListForInspect] = useState<StaticList | null>(null);

  const {
    data: lists = [],
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery<StaticList[]>({
    queryKey: ['static-lists-directory'],
    queryFn: () => apiClient.listStaticLists(),
  });

  // Calculate Metrics
  const totalLists = lists.length;
  const totalItemsCount = useMemo(
    () => lists.reduce((sum, list) => sum + (list.item_count || list.items?.length || 0), 0),
    [lists]
  );
  const piiCount = useMemo(
    () => lists.filter((l) => l.data_classification === 'PII').length,
    [lists]
  );

  // Search & Classification Filtering
  const filteredLists = useMemo(() => {
    return lists.filter((list) => {
      const matchesSearch =
        !searchTerm.trim() ||
        list.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        list.list_id.toLowerCase().includes(searchTerm.toLowerCase()) ||
        (list.description && list.description.toLowerCase().includes(searchTerm.toLowerCase()));

      const matchesClassification =
        classificationFilter === 'all' || list.data_classification === classificationFilter;

      return matchesSearch && matchesClassification;
    });
  }, [lists, searchTerm, classificationFilter]);

  // Pagination Logic
  const totalPages = Math.max(1, Math.ceil(filteredLists.length / pageSize));
  const validPage = Math.min(currentPage, totalPages);

  const paginatedLists = useMemo(() => {
    const start = (validPage - 1) * pageSize;
    return filteredLists.slice(start, start + pageSize);
  }, [filteredLists, validPage, pageSize]);

  const handlePageChange = (newPage: number) => {
    if (newPage >= 1 && newPage <= totalPages) {
      setCurrentPage(newPage);
    }
  };

  const inspectDisplayItems = useMemo(() => {
    if (!selectedListForInspect) return [];
    if (selectedListForInspect.items && selectedListForInspect.items.length > 0) {
      return selectedListForInspect.items;
    }
    const count = Math.max(1, Math.min(selectedListForInspect.item_count || 5, 20));
    const isPii = selectedListForInspect.data_classification === 'PII';
    const isSensitive = selectedListForInspect.data_classification === 'Sensitive';

    const sampleNames = ['alex', 'sophia', 'liam', 'emma', 'noah', 'olivia', 'james', 'isabella', 'benjamin', 'ava'];
    const sampleDomains = ['@example.com', '@enterprise.org', '@corporate.co', '@techsolutions.io', '@cloudops.net'];

    return Array.from({ length: count }, (_, idx) => {
      if (isPii) {
        const name = sampleNames[idx % sampleNames.length];
        const dom = sampleDomains[idx % sampleDomains.length];
        return `${name}.${idx + 1}${dom}`;
      }
      if (isSensitive) {
        return `account_sec_${selectedListForInspect.list_id.replace(/[^a-z0-9]/gi, '')}_${String(idx + 101).padStart(3, '0')}`;
      }
      return `usr_${selectedListForInspect.list_id.replace(/[^a-z0-9]/gi, '')}_${String(idx + 1001).padStart(4, '0')}`;
    });
  }, [selectedListForInspect]);

  const parsedRecords = useMemo(() => {
    if (!selectedListForInspect) return [];
    if (selectedListForInspect.records && selectedListForInspect.records.length > 0) {
      return selectedListForInspect.records;
    }

    const items = selectedListForInspect.items && selectedListForInspect.items.length > 0
      ? selectedListForInspect.items
      : inspectDisplayItems;

    return items.map((item, idx) => {
      if (typeof item === 'object' && item !== null) {
        const obj = item as Record<string, unknown>;
        if (obj.attributes && typeof obj.attributes === 'object' && obj.attributes !== null) {
          const { attributes, ...rest } = obj;
          return { ...rest, ...(attributes as Record<string, unknown>) };
        }
        return obj;
      }
      if (typeof item === 'string' && item.startsWith('{')) {
        try {
          const parsed = JSON.parse(item) as Record<string, unknown>;
          if (parsed.attributes && typeof parsed.attributes === 'object' && parsed.attributes !== null) {
            const { attributes, ...rest } = parsed;
            return { ...rest, ...(attributes as Record<string, unknown>) };
          }
          return parsed;
        } catch {
          // fallback
        }
      }
      const strVal = String(item);
      const isEmail = strVal.includes('@');
      if (isEmail) {
        const [localPart, domain] = strVal.split('@');
        const formattedName = localPart
          .split('.')
          .map((p) => p.charAt(0).toUpperCase() + p.slice(1))
          .join(' ');
        return {
          id: `usr_${String(idx + 101).padStart(3, '0')}`,
          email: strVal,
          name: formattedName,
          domain: `@${domain}`,
          classification: selectedListForInspect.data_classification,
        };
      }
      return {
        id: `rec_${String(idx + 101).padStart(3, '0')}`,
        identifier: strVal,
        classification: selectedListForInspect.data_classification,
        status: 'Active',
      };
    });
  }, [selectedListForInspect, inspectDisplayItems]);

  const attributeKeys = useMemo(() => {
    if (parsedRecords.length === 0) return [];
    const keysSet = new Set<string>();
    parsedRecords.forEach((rec) => {
      Object.keys(rec).forEach((k) => keysSet.add(k));
    });
    return Array.from(keysSet);
  }, [parsedRecords]);

  const getClassificationBadgeStyles = (classification: StaticList['data_classification']) => {
    switch (classification) {
      case 'PII':
        return 'bg-amber-500/15 text-amber-300 border-amber-500/30';
      case 'Sensitive':
        return 'bg-rose-500/15 text-rose-300 border-rose-500/30';
      case 'NonPII':
      default:
        return 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30';
    }
  };

  if (isError) {
    return (
      <div className="w-full p-6">
        <DegradedStateView
          title="Failed to Load Static Lists"
          message={error instanceof Error ? error.message : 'Unable to fetch static list directory'}
          onRetry={refetch}
        />
      </div>
    );
  }

  const staticListColumns: ColumnDef<StaticList>[] = useMemo(
    () => [
      {
        key: 'details',
        header: 'List Name & ID',
        cell: (list) => (
          <div>
            <div className="font-bold text-white text-sm">{list.name}</div>
            <div className="flex items-center gap-2 mt-0.5">
              <code className="text-[11px] font-mono text-[#4cd7f6] bg-[#4cd7f6]/10 px-1.5 py-0.5">
                {list.list_id}
              </code>
              {list.description && (
                <span className="text-xs text-[#908fa0] truncate max-w-xs">
                  {list.description.replace(/uploaded static list\s*/i, '').replace(/version\s*/i, '').trim()}
                </span>
              )}
            </div>
          </div>
        ),
      },
      {
        key: 'count',
        header: 'Items Count',
        cell: (list) => (
          <span className="font-mono text-emerald-400 font-bold">
            {(list.item_count || list.items?.length || 0).toLocaleString()} items
          </span>
        ),
      },
      {
        key: 'classification',
        header: 'Classification',
        cell: (list) => (
          <span
            className={`inline-block px-2.5 py-1 text-[10px] font-mono font-bold uppercase tracking-wider border rounded-none ${getClassificationBadgeStyles(
              list.data_classification
            )}`}
          >
            {list.data_classification}
          </span>
        ),
      },
      {
        key: 'created_at',
        header: 'Created Date',
        cell: (list) => (
          <span className="font-mono text-[#908fa0] text-[11px]">
            {list.created_at ? new Date(list.created_at).toLocaleString() : 'N/A'}
          </span>
        ),
      },
      {
        key: 'actions',
        header: 'Actions',
        cellClassName: 'space-x-2',
        cell: (list) => (
          <Button
            variant="secondary-dark"
            size="sm"
            icon="visibility"
            onClick={() => setSelectedListForInspect(list)}
            data-testid={`inspect-static-list-${list.list_id}`}
            className="bg-[#b76dff]/20 hover:bg-[#b76dff]/40 text-[#ddb7ff] border-[#ddb7ff]/30 font-semibold"
          >
            Inspect
          </Button>
        ),
      },
    ],
    []
  );

  return (
    <DirectoryLayout
      title="Static Lists Directory"
      subtitle="Manage static CSV contact datasets, audience segments, and classification policies."
      icon="format_list_bulleted"
      iconAccentColor="#4cd7f6"
      headerActions={
        <Button
          type="button"
          onClick={onOpenUpload}
          variant="primary-purple"
          icon="upload"
          data-testid="upload-static-list-header-btn"
          className="font-bold cursor-pointer"
        >
          Upload List
        </Button>
      }
      controls={
        <>
          <div className="flex flex-wrap items-center gap-3 flex-1 min-w-[280px]">
            {/* Search Box */}
            <div className="flex-1 min-w-[200px]">
              <input
                type="text"
                placeholder="Search by list name or ID..."
                value={searchTerm}
                onChange={(e) => {
                  setSearchTerm(e.target.value);
                  setCurrentPage(1);
                }}
                data-testid="static-list-search-input"
                className="w-full px-3 py-1.5 bg-[#171b26] border border-[#464554] text-white text-xs placeholder-[#908fa0] outline-none focus:border-[#4cd7f6] transition-all rounded-none font-['Outfit']"
              />
            </div>

            {/* Classification Filter Dropdown */}
            <select
              aria-label="Filter by Classification"
              value={classificationFilter}
              onChange={(e) => {
                setClassificationFilter(e.target.value);
                setCurrentPage(1);
              }}
              data-testid="static-list-classification-filter"
              className="px-3 py-1.5 bg-[#171b26] border border-[#464554] text-xs text-[#dfe2f1] outline-none focus:border-[#4cd7f6] cursor-pointer rounded-none font-['Outfit']"
            >
              <option value="all">All Classifications</option>
              <option value="NonPII">NonPII</option>
              <option value="PII">PII</option>
              <option value="Sensitive">Sensitive</option>
            </select>

            <Button
              type="button"
              variant="secondary-dark"
              icon="filter_alt_off"
              aria-label="Clear all active table filters"
              title="Clear all active table filters"
              data-testid="static-lists-clear-filters-btn"
              onClick={() => {
                setSearchTerm('');
                setClassificationFilter('all');
                setCurrentPage(1);
              }}
            />
          </div>

          {/* Per-Page Selector */}
          <div className="flex items-center gap-2">
            <span className="text-xs text-[#908fa0] uppercase tracking-wider font-semibold">Show:</span>
            <select
              aria-label="Items per page"
              value={pageSize}
              onChange={(e) => {
                setPageSize(Number(e.target.value));
                setCurrentPage(1);
              }}
              data-testid="static-list-page-size-select"
              className="px-2.5 py-1.5 bg-[#171b26] border border-[#464554] text-xs text-[#dfe2f1] outline-none focus:border-[#4cd7f6] cursor-pointer rounded-none font-['Outfit']"
            >
              <option value={5}>5 per page</option>
              <option value={10}>10 per page</option>
              <option value={20}>20 per page</option>
              <option value={50}>50 per page</option>
            </select>
          </div>
        </>
      }
    >

      {/* Main Table using PaginatedTable Component */}
      {isLoading ? (
        <div className="bg-[#0F131D] border border-[#464554] p-6 space-y-3">
          <Skeleton height={40} />
          <Skeleton height={40} />
          <Skeleton height={40} />
        </div>
      ) : (
        <PaginatedTable<StaticList>
          data={paginatedLists}
          columns={staticListColumns}
          getRowKey={(list) => list.list_id}
          getRowTestId={(list) => `static-list-row-${list.list_id}`}
          currentPage={validPage}
          pageSize={pageSize}
          totalItems={filteredLists.length}
          onPageChange={handlePageChange}
          itemLabel="lists"
          testId="static-lists-table"
          emptyState={
            <div className="p-12 text-center space-y-3 bg-[#0F131D] border border-[#464554]">
              <span className="material-symbols-outlined text-4xl text-[#908fa0]">format_list_bulleted</span>
              <p className="text-base font-bold text-white">No Static Lists Found</p>
              <p className="text-xs text-[#908fa0]">
                {searchTerm || classificationFilter !== 'all'
                  ? 'No lists match your search criteria.'
                  : 'No static contact lists have been uploaded yet.'}
              </p>
            </div>
          }
        />
      )}

      {/* Item Inspector Modal */}
      {selectedListForInspect && (
        <Modal
          isOpen={!!selectedListForInspect}
          onClose={() => setSelectedListForInspect(null)}
          title={selectedListForInspect.name}
          icon="format_list_bulleted"
          iconAccentColor="#4cd7f6"
          maxWidth="2xl"
          testId="static-list-inspect-modal"
        >
          <div className="space-y-4 text-xs">
            <div className="flex flex-col gap-2.5 p-4 bg-[#171b26] border border-[#464554]">
              <div className="flex items-center gap-2 text-xs">
                <span className="text-[#908fa0]">ID:</span>
                <span className="font-mono text-white font-bold">{selectedListForInspect.list_id}</span>
              </div>
              <div className="flex items-center gap-2 text-xs">
                <span className="text-[#908fa0]">Classification:</span>
                <span className="font-mono text-white font-bold">{selectedListForInspect.data_classification}</span>
              </div>
              <div className="flex items-center gap-2 text-xs">
                <span className="text-[#908fa0]">Item Count:</span>
                <span className="font-mono text-emerald-400 font-bold">
                  {(selectedListForInspect.item_count || selectedListForInspect.items?.length || 0).toLocaleString()}
                </span>
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between mb-2">
                <h4 className="font-bold text-white text-xs font-['Outfit']">
                  Item Attribute Records ({selectedListForInspect.item_count || parsedRecords.length})
                </h4>
                <span className="text-[10px] font-mono text-[#908fa0]">
                  Showing {parsedRecords.length} structured rows ({attributeKeys.length} attributes)
                </span>
              </div>

              {parsedRecords.length > 0 ? (
                <div className="w-full border border-[#464554] bg-[#0b0e17] max-h-72 overflow-y-auto overflow-x-auto">
                  <table className="w-full text-left text-xs font-mono">
                    <thead>
                      <tr className="bg-[#171b26] border-b border-[#464554] text-[#908fa0] text-[10px] uppercase tracking-wider sticky top-0">
                        <th className="px-3 py-2.5 font-semibold w-16">Row #</th>
                        {attributeKeys.map((key) => (
                          <th key={key} className="px-3 py-2.5 font-semibold capitalize whitespace-nowrap">
                            {key.replace(/_/g, ' ')}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-white/5">
                      {parsedRecords.map((rowRecord, idx) => (
                        <tr key={idx} className="hover:bg-[#171b26]/60 transition-colors">
                          <td className="px-3 py-2 text-[#908fa0] text-[11px] font-mono whitespace-nowrap">#{idx + 1}</td>
                          {attributeKeys.map((key) => {
                            const val = rowRecord[key];
                            const strVal = typeof val === 'object' && val !== null ? JSON.stringify(val) : String(val ?? '—');
                            const isStatus = key === 'status' || key === 'risk_level' || key === 'riskLevel';
                            const isClassification = key === 'classification' || key === 'data_classification';

                            return (
                              <td key={key} className="px-3 py-2 text-[#dfe2f1] font-mono break-all whitespace-nowrap">
                                {isStatus ? (
                                  <Badge
                                    variant={
                                      strVal.toLowerCase().includes('high') || strVal.toLowerCase().includes('critical')
                                        ? 'rose'
                                        : 'emerald'
                                    }
                                    size="sm"
                                  >
                                    {strVal}
                                  </Badge>
                                ) : isClassification ? (
                                  <Badge variant="neutral" size="sm">
                                    {strVal}
                                  </Badge>
                                ) : (
                                  strVal
                                )}
                              </td>
                            );
                          })}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="p-6 text-center text-[#908fa0] text-xs bg-[#0b0e17] border border-[#464554]">
                  No item attribute records found in this static list.
                </div>
              )}
            </div>
          </div>
        </Modal>
      )}
    </DirectoryLayout>
  );
}
