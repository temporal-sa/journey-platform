import { useState, useRef, useEffect } from 'react';

export interface StatusOption {
  status: string;
  label: string;
}

export interface StatusFilterDropdownProps {
  value: string;
  onChange: (newStatus: string) => void;
  options: StatusOption[];
  getStatusStyles: (status: string) => string;
  label?: string;
  dataTestId?: string;
}

export function StatusFilterDropdown({
  value,
  onChange,
  options,
  getStatusStyles,
  label,
  dataTestId = 'status-filter-dropdown',
}: StatusFilterDropdownProps) {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement | null>(null);

  const selectedOption = options.find((o) => o.status === value) || options[0];

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, []);

  return (
    <div className="relative inline-block" ref={containerRef}>
      {label && (
        <label className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1">
          {label}
        </label>
      )}

      {/* Trigger Button */}
      <button
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        aria-label={label || 'Filter by status'}
        data-testid={dataTestId}
        className="bg-[#171b26] border border-[#464554] hover:border-[#c0c1ff] px-2.5 py-1.5 rounded-none flex items-center justify-between gap-3 cursor-pointer transition-all min-w-[160px] font-['Outfit',sans-serif] shadow-sm focus:outline-none"
      >
        <div className="flex items-center gap-2">
          <span
            className={`px-2 py-0.5 rounded-none text-[10px] font-mono font-semibold uppercase border tracking-wider ${getStatusStyles(
              selectedOption.status
            )}`}
          >
            {selectedOption.label}
          </span>
        </div>
        <span className="material-symbols-outlined text-sm text-[#c0c1ff] shrink-0 transition-transform">
          {isOpen ? 'expand_less' : 'expand_more'}
        </span>
      </button>

      {/* Dropdown Menu Overlay */}
      {isOpen && (
        <div
          role="listbox"
          aria-label="Status options"
          className="absolute top-full left-0 mt-1 z-[100] bg-[#171b26] border border-[#464554] shadow-2xl p-1.5 flex flex-col gap-1 min-w-[170px] rounded-none animate-in fade-in slide-in-from-top-1 duration-150"
        >
          {options.map((option) => {
            const isSelected = option.status === value;
            return (
              <button
                key={option.status}
                type="button"
                role="option"
                aria-selected={isSelected}
                onClick={() => {
                  onChange(option.status);
                  setIsOpen(false);
                }}
                className={`w-full text-left px-2.5 py-1.5 rounded-none text-[10px] font-mono font-semibold uppercase border tracking-wider flex items-center justify-between transition-all cursor-pointer ${getStatusStyles(
                  option.status
                )} ${
                  isSelected
                    ? 'ring-1 ring-current font-bold opacity-100 shadow-sm'
                    : 'opacity-70 hover:opacity-100 hover:brightness-125'
                }`}
              >
                <span>{option.label}</span>
                {isSelected && (
                  <span className="material-symbols-outlined text-xs shrink-0">check</span>
                )}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
