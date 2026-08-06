import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { JourneyApiClient } from '../api/client';
import type { StaticList } from '../types/api';
import { Button } from '../components/common/Button';
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
      <div className="p-6">
        <DegradedStateView
          title="Failed to Load Static Lists"
          message={error instanceof Error ? error.message : 'Unable to fetch static list directory'}
          onRetry={refetch}
        />
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6 bg-[#0B0F19] text-[#dfe2f1] font-['Outfit',sans-serif] min-h-screen">
      {/* Header Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-[#0F131D] p-5 border border-[#464554] shadow-lg">
        <div>
          <div className="flex items-center gap-2">
            <span className="material-symbols-outlined text-[#4cd7f6] text-2xl">format_list_bulleted</span>
            <h1 className="text-xl font-bold text-white tracking-wide">Static Lists Directory</h1>
          </div>
          <p className="text-xs text-[#908fa0] mt-1">
            Manage static CSV contact datasets, audience segments, and classification policies.
          </p>
        </div>

        {onOpenUpload && (
          <Button
            variant="primary-cyan"
            size="md"
            icon="upload_file"
            onClick={onOpenUpload}
            data-testid="static-list-upload-trigger"
          >
            Upload Static List
          </Button>
        )}
      </div>

      {/* Summary Stats Bar */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="bg-[#0F131D] p-4 border border-[#464554] flex items-center justify-between">
          <div>
            <p className="text-xs text-[#908fa0] uppercase tracking-wider font-semibold">Total Datasets</p>
            <p className="text-2xl font-bold text-white mt-1 font-mono">{isLoading ? '...' : totalLists}</p>
          </div>
          <span className="material-symbols-outlined text-3xl text-[#4cd7f6]/40">database</span>
        </div>

        <div className="bg-[#0F131D] p-4 border border-[#464554] flex items-center justify-between">
          <div>
            <p className="text-xs text-[#908fa0] uppercase tracking-wider font-semibold">Indexed Contacts / Items</p>
            <p className="text-2xl font-bold text-emerald-400 mt-1 font-mono">{isLoading ? '...' : totalItemsCount.toLocaleString()}</p>
          </div>
          <span className="material-symbols-outlined text-3xl text-emerald-500/40">contacts</span>
        </div>

        <div className="bg-[#0F131D] p-4 border border-[#464554] flex items-center justify-between">
          <div>
            <p className="text-xs text-[#908fa0] uppercase tracking-wider font-semibold">PII Restricted Lists</p>
            <p className="text-2xl font-bold text-amber-400 mt-1 font-mono">{isLoading ? '...' : piiCount}</p>
          </div>
          <span className="material-symbols-outlined text-3xl text-amber-500/40">shield</span>
        </div>
      </div>

      {/* Filter and Control Bar */}
      <div className="bg-[#0F131D] p-4 border border-[#464554] flex flex-wrap items-center justify-between gap-4">
        <div className="flex flex-wrap items-center gap-3 flex-1 min-w-[280px]">
          {/* Search Box */}
          <div className="relative flex-1 min-w-[200px]">
            <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-base text-[#908fa0]">
              search
            </span>
            <input
              type="text"
              placeholder="Search by list name or ID..."
              value={searchTerm}
              onChange={(e) => {
                setSearchTerm(e.target.value);
                setCurrentPage(1);
              }}
              data-testid="static-list-search-input"
              className="w-full pl-9 pr-3 py-1.5 bg-[#171b26] border border-[#464554] text-white text-xs placeholder-[#908fa0] outline-none focus:border-[#4cd7f6] transition-all rounded-none font-['Outfit']"
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

          {(searchTerm || classificationFilter !== 'all') && (
            <Button
              variant="secondary-dark"
              size="sm"
              icon="filter_alt_off"
              onClick={() => {
                setSearchTerm('');
                setClassificationFilter('all');
                setCurrentPage(1);
              }}
            >
              Clear Filters
            </Button>
          )}
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
      </div>

      {/* Main Table */}
      <div className="bg-[#0F131D] border border-[#464554] shadow-xl overflow-x-auto">
        {isLoading ? (
          <div className="p-6 space-y-3">
            <Skeleton height={40} />
            <Skeleton height={40} />
            <Skeleton height={40} />
          </div>
        ) : paginatedLists.length === 0 ? (
          <div className="p-12 text-center space-y-3">
            <span className="material-symbols-outlined text-4xl text-[#908fa0]">format_list_bulleted</span>
            <p className="text-base font-bold text-white">No Static Lists Found</p>
            <p className="text-xs text-[#908fa0]">
              {searchTerm || classificationFilter !== 'all'
                ? 'No lists match your search criteria.'
                : 'No static contact lists have been uploaded yet.'}
            </p>
          </div>
        ) : (
          <table className="w-full text-left border-collapse text-xs">
            <thead>
              <tr className="bg-[#171b26] border-b border-[#464554] text-[#908fa0] font-mono uppercase tracking-wider">
                <th className="p-3.5">List Details</th>
                <th className="p-3.5">Items Count</th>
                <th className="p-3.5">Classification</th>
                <th className="p-3.5">Created Date</th>
                <th className="p-3.5 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#464554]/50">
              {paginatedLists.map((list) => {
                const count = list.item_count || list.items?.length || 0;
                return (
                  <tr
                    key={list.list_id}
                    className="hover:bg-[#171b26]/60 transition-colors"
                    data-testid={`static-list-row-${list.list_id}`}
                  >
                    <td className="p-3.5">
                      <div className="font-bold text-white text-sm">{list.name}</div>
                      <div className="flex items-center gap-2 mt-0.5">
                        <code className="text-[11px] font-mono text-[#4cd7f6] bg-[#4cd7f6]/10 px-1.5 py-0.5 border border-[#4cd7f6]/30">
                          {list.list_id}
                        </code>
                        {list.description && (
                          <span className="text-xs text-[#908fa0] truncate max-w-xs">{list.description}</span>
                        )}
                      </div>
                    </td>

                    <td className="p-3.5 font-mono text-emerald-400 font-bold">
                      {count.toLocaleString()} items
                    </td>

                    <td className="p-3.5">
                      <span
                        className={`inline-block px-2.5 py-1 text-[10px] font-mono font-bold uppercase tracking-wider border rounded-none ${getClassificationBadgeStyles(
                          list.data_classification
                        )}`}
                      >
                        {list.data_classification}
                      </span>
                    </td>

                    <td className="p-3.5 text-[#908fa0] font-mono text-[11px]">
                      {list.created_at ? new Date(list.created_at).toLocaleString() : 'N/A'}
                    </td>

                    <td className="p-3.5 text-right space-x-2">
                      <Button
                        variant="secondary-dark"
                        size="sm"
                        icon="visibility"
                        onClick={() => setSelectedListForInspect(list)}
                        data-testid={`inspect-static-list-${list.list_id}`}
                      >
                        Inspect
                      </Button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}

        {/* Pagination Controls Footer */}
        {!isLoading && filteredLists.length > 0 && (
          <div className="p-4 bg-[#171b26] border-t border-[#464554] flex items-center justify-between">
            <p className="text-xs text-[#908fa0]">
              Showing <span className="text-white font-mono font-bold">{(validPage - 1) * pageSize + 1}</span> to{' '}
              <span className="text-white font-mono font-bold">
                {Math.min(validPage * pageSize, filteredLists.length)}
              </span>{' '}
              of <span className="text-white font-mono font-bold">{filteredLists.length}</span> items
            </p>

            <div className="flex items-center gap-2">
              <Button
                variant="secondary-dark"
                size="sm"
                icon="chevron_left"
                disabled={validPage <= 1}
                onClick={() => handlePageChange(validPage - 1)}
                data-testid="static-list-prev-page"
              >
                Prev
              </Button>

              <span className="text-xs font-mono text-white px-2">
                Page {validPage} of {totalPages}
              </span>

              <Button
                variant="secondary-dark"
                size="sm"
                icon="chevron_right"
                disabled={validPage >= totalPages}
                onClick={() => handlePageChange(validPage + 1)}
                data-testid="static-list-next-page"
              >
                Next
              </Button>
            </div>
          </div>
        )}
      </div>

      {/* Item Inspector Modal */}
      {selectedListForInspect && (
        <div
          className="fixed inset-0 bg-[#0B0F19]/80 backdrop-blur-md flex items-center justify-center z-50 p-4"
          role="dialog"
          aria-modal="true"
          data-testid="static-list-inspect-modal"
        >
          <div className="bg-[#0F131D] border border-[#4cd7f6]/40 w-full max-w-2xl max-h-[85vh] flex flex-col shadow-2xl overflow-hidden glass-modal">
            {/* Modal Header */}
            <div className="p-4 bg-[#171b26] border-b border-[#464554] flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="material-symbols-outlined text-[#4cd7f6]">format_list_bulleted</span>
                <h3 className="font-bold text-white text-base">{selectedListForInspect.name}</h3>
                <code className="text-xs font-mono text-[#4cd7f6] bg-[#4cd7f6]/10 px-2 py-0.5 border border-[#4cd7f6]/30">
                  {selectedListForInspect.list_id}
                </code>
              </div>
              <button
                onClick={() => setSelectedListForInspect(null)}
                className="text-[#908fa0] hover:text-white transition-colors cursor-pointer"
                aria-label="Close modal"
              >
                <span className="material-symbols-outlined">close</span>
              </button>
            </div>

            {/* Modal Content */}
            <div className="p-6 flex-1 min-h-0 overflow-y-auto space-y-4 text-xs">
              <div className="grid grid-cols-2 gap-3 p-3 bg-[#171b26] border border-[#464554]">
                <div>
                  <span className="text-[#908fa0] block">Classification:</span>
                  <span className="font-mono text-white font-bold">{selectedListForInspect.data_classification}</span>
                </div>
                <div>
                  <span className="text-[#908fa0] block">Item Count:</span>
                  <span className="font-mono text-emerald-400 font-bold">
                    {(selectedListForInspect.item_count || selectedListForInspect.items?.length || 0).toLocaleString()}
                  </span>
                </div>
              </div>

              <div>
                <h4 className="font-bold text-white mb-2">Sample Data Records ({selectedListForInspect.items?.length || 0})</h4>
                <pre className="p-4 bg-[#0b0e17] border border-[#464554] text-[#dfe2f1] font-mono text-xs leading-relaxed max-h-60 overflow-y-auto whitespace-pre-wrap">
                  {selectedListForInspect.items?.length
                    ? selectedListForInspect.items.join('\n')
                    : 'No item records returned.'}
                </pre>
              </div>
            </div>

            {/* Modal Footer */}
            <div className="p-4 bg-[#171b26] border-t border-[#464554] flex justify-end">
              <Button
                variant="primary-cyan"
                onClick={() => setSelectedListForInspect(null)}
              >
                Close
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
