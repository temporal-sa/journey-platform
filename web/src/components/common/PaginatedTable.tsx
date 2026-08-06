import React from 'react';
import { Button } from './Button';

export interface ColumnDef<T> {
  key: string;
  header: React.ReactNode;
  cell: (item: T) => React.ReactNode;
  headerClassName?: string;
  cellClassName?: string;
}

export interface PaginatedTableProps<T> {
  data: T[];
  columns: ColumnDef<T>[];
  getRowKey: (item: T) => string;
  getRowTestId?: (item: T) => string;
  currentPage: number;
  pageSize: number;
  totalItems: number;
  onPageChange: (page: number) => void;
  itemLabel?: string;
  emptyState?: React.ReactNode;
  isLoading?: boolean;
  className?: string;
  testId?: string;
}

export function PaginatedTable<T>({
  data,
  columns,
  getRowKey,
  getRowTestId,
  currentPage,
  pageSize,
  totalItems,
  onPageChange,
  itemLabel = 'items',
  emptyState,
  isLoading = false,
  className = '',
  testId = 'paginated-table',
}: PaginatedTableProps<T>) {
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize));
  const startItem = totalItems > 0 ? (currentPage - 1) * pageSize + 1 : 0;
  const endItem = Math.min(currentPage * pageSize, totalItems);

  if (!isLoading && data.length === 0 && emptyState) {
    return <>{emptyState}</>;
  }

  return (
    <div
      data-testid={testId}
      className={`w-full bg-[#0F131D]/90 backdrop-blur-xl rounded-none border border-[#464554] shadow-xl overflow-hidden flex-1 flex flex-col justify-between font-['Outfit',sans-serif] ${className}`}
    >
      <div className="overflow-x-auto w-full">
        <table className="w-full text-left text-xs">
          <thead>
            <tr className="bg-[#171b26] border-b border-[#464554] text-[#908fa0] font-mono uppercase tracking-wider text-[10px]">
              {columns.map((col) => (
                <th
                  key={col.key}
                  className={`p-4 font-semibold text-left ${col.headerClassName || ''}`}
                >
                  {col.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-white/5">
            {data.map((item) => (
              <tr
                key={getRowKey(item)}
                data-testid={getRowTestId ? getRowTestId(item) : undefined}
                className="hover:bg-[#171b26]/50 transition-colors"
              >
                {columns.map((col) => (
                  <td key={col.key} className={`p-4 ${col.cellClassName || ''}`}>
                    {col.cell(item)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination Controls */}
      <div className="flex items-center justify-between p-4 border-t border-[#464554] bg-[#171b26]/80 font-mono text-xs w-full shrink-0">
        <div className="text-[#908fa0]">
          Showing {startItem} to {endItem} of {totalItems} {itemLabel}
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="secondary-dark"
            size="sm"
            icon="chevron_left"
            disabled={currentPage <= 1 || totalPages <= 1}
            onClick={() => onPageChange(currentPage - 1)}
            data-testid={`${testId}-prev-page`}
          >
            Previous
          </Button>
          <span className="text-xs text-[#908fa0] px-1 font-mono">
            Page {currentPage} of {totalPages}
          </span>
          <Button
            variant="secondary-dark"
            size="sm"
            icon="chevron_right"
            disabled={currentPage >= totalPages || totalPages <= 1}
            onClick={() => onPageChange(currentPage + 1)}
            data-testid={`${testId}-next-page`}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  );
}
