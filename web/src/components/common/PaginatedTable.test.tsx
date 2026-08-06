import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PaginatedTable, ColumnDef } from './PaginatedTable';

interface TestRecord {
  id: string;
  name: string;
  role: string;
}

const testData: TestRecord[] = [
  { id: '1', name: 'Alice', role: 'Engineer' },
  { id: '2', name: 'Bob', role: 'Designer' },
];

const testColumns: ColumnDef<TestRecord>[] = [
  { key: 'name', header: 'Name', cell: (row) => row.name },
  { key: 'role', header: 'Role', cell: (row) => row.role },
];

describe('PaginatedTable Component', () => {
  it('renders columns headers and row data correctly', () => {
    const handlePageChange = vi.fn();
    render(
      <PaginatedTable
        data={testData}
        columns={testColumns}
        getRowKey={(row) => row.id}
        currentPage={1}
        pageSize={10}
        totalItems={2}
        onPageChange={handlePageChange}
        itemLabel="users"
      />
    );

    expect(screen.getByText('Name')).toBeInTheDocument();
    expect(screen.getByText('Role')).toBeInTheDocument();
    expect(screen.getByText('Alice')).toBeInTheDocument();
    expect(screen.getByText('Engineer')).toBeInTheDocument();
    expect(screen.getByText('Showing 1 to 2 of 2 users')).toBeInTheDocument();
  });

  it('handles page navigation buttons', () => {
    const handlePageChange = vi.fn();
    render(
      <PaginatedTable
        data={testData}
        columns={testColumns}
        getRowKey={(row) => row.id}
        currentPage={1}
        pageSize={1}
        totalItems={2}
        onPageChange={handlePageChange}
      />
    );

    const nextBtn = screen.getByRole('button', { name: /Next/i });
    fireEvent.click(nextBtn);
    expect(handlePageChange).toHaveBeenCalledWith(2);
  });
});
