import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { StatusFilterDropdown } from './StatusFilterDropdown';

describe('StatusFilterDropdown Component', () => {
  const dummyOptions = [
    { status: 'all', label: 'All Statuses' },
    { status: 'active', label: 'Active' },
    { status: 'draft', label: 'Draft' },
  ];

  const dummyGetStyles = (status: string) => {
    switch (status) {
      case 'active':
        return 'bg-emerald-500/20 text-emerald-300 border-emerald-500/50';
      case 'draft':
        return 'bg-amber-500/20 text-amber-300 border-amber-500/50';
      default:
        return 'bg-[#464554]/30 text-[#c7c4d7] border-[#464554]';
    }
  };

  it('renders selected status badge in trigger button', () => {
    render(
      <StatusFilterDropdown
        value="active"
        onChange={vi.fn()}
        options={dummyOptions}
        getStatusStyles={dummyGetStyles}
      />
    );

    const triggerBtn = screen.getByRole('button', { name: /Filter by status/i });
    expect(triggerBtn).toBeInTheDocument();
    expect(screen.getByText('Active')).toBeInTheDocument();
  });

  it('opens options menu on click and fires onChange when selecting option', () => {
    const handleChange = vi.fn();
    render(
      <StatusFilterDropdown
        value="all"
        onChange={handleChange}
        options={dummyOptions}
        getStatusStyles={dummyGetStyles}
      />
    );

    const triggerBtn = screen.getByRole('button', { name: /Filter by status/i });
    fireEvent.click(triggerBtn);

    expect(screen.getByRole('listbox')).toBeInTheDocument();
    const draftOption = screen.getByRole('option', { name: /Draft/i });
    expect(draftOption).toHaveClass('bg-amber-500/20');

    fireEvent.click(draftOption);
    expect(handleChange).toHaveBeenCalledWith('draft');
  });
});
