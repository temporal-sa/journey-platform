import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { JourneyApiClient } from '../api/client';
import type { StaticList } from '../types/api';
import { Button } from '../components/common/Button';
import { PaginatedTable, ColumnDef } from '../components/common/PaginatedTable';
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
        header: 'List Details',
        cell: (list) => (
          <div>
            <div className="font-bold text-white text-sm">{list.name}</div>
            <div className="flex items-center gap-2 mt-0.5">
              <code className="text-[11px] font-mono text-[#4cd7f6] bg-[#4cd7f6]/10 px-1.5 py-0.5">
                {list.list_id}
              </code>
              {list.description && (
                <span className="text-xs text-[#908fa0] truncate max-w-xs">
                  {list.description.replace(/uploaded static list\s*/i, '').trim()}
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
        headerClassName: 'text-right',
        cellClassName: 'text-right space-x-2',
        cell: (list) => (
          <Button
            variant="secondary-dark"
            size="sm"
            icon="visibility"
            onClick={() => setSelectedListForInspect(list)}
            data-testid={`inspect-static-list-${list.list_id}`}
          >
            Inspect
          </Button>
        ),
      },
    ],
    []
  );

  return (
    <div className="flex-1 w-full min-w-0 h-full overflow-y-auto bg-[#0B0F19] text-[#dfe2f1] font-['Outfit',sans-serif] p-6 space-y-6">
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

      {/* Filter and Control Bar */}
      <div className="bg-[#0F131D] p-4 border border-[#464554] flex flex-wrap items-center justify-between gap-4">
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
                <code className="text-xs font-mono text-[#4cd7f6] bg-[#4cd7f6]/10 px-2 py-0.5">
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
