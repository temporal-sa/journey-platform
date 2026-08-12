import { useState, useEffect, useRef, useMemo } from 'react';

export interface TemplateOption {
  id: string;
  name: string;
  defaultSubject?: string;
  description?: string;
  category?: string;
}

export type SearchableTemplateOption = TemplateOption;

export interface TemplateSelectorProps {
  value: string;
  onChange: (value: string) => void;
  options: TemplateOption[];
  onSelectTemplate?: (template: TemplateOption) => void;
  hasError?: boolean;
  placeholder?: string;
  disabled?: boolean;
  testId?: string;
  label?: string;
}

export type SearchableTemplateInputProps = TemplateSelectorProps;

export function TemplateSelector({
  value,
  onChange,
  options,
  onSelectTemplate,
  hasError = false,
  placeholder = 'Search catalog templates or enter template ID...',
  disabled = false,
  testId = 'inspector-input-template_id',
}: TemplateSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const containerRef = useRef<HTMLDivElement | null>(null);

  const filteredOptions = useMemo(() => {
    if (!searchQuery.trim()) return options;
    const q = searchQuery.toLowerCase();
    return options.filter(
      (opt) =>
        opt.name.toLowerCase().includes(q) ||
        opt.id.toLowerCase().includes(q) ||
        (opt.description && opt.description.toLowerCase().includes(q))
    );
  }, [options, searchQuery]);

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
    <div className="relative w-full" ref={containerRef}>
      <div className="relative flex items-center">
        <input
          type="text"
          value={isOpen ? searchQuery : value}
          disabled={disabled}
          onFocus={() => {
            setSearchQuery(value);
            setIsOpen(true);
          }}
          onChange={(e) => {
            setSearchQuery(e.target.value);
            onChange(e.target.value);
            setIsOpen(true);
          }}
          placeholder={placeholder}
          data-testid={testId}
          className={`w-full px-3 py-1.5 pr-8 bg-[#11141d] border ${
            hasError ? 'border-rose-500' : 'border-[#464554] focus:border-[#c0c1ff]'
          } rounded-none text-[#dfe2f1] text-xs font-['Outfit',sans-serif] outline-none transition-all placeholder-[#64748b] ${
            disabled ? 'opacity-50 cursor-not-allowed' : ''
          }`}
        />
        <button
          type="button"
          tabIndex={-1}
          disabled={disabled}
          onClick={() => setIsOpen((prev) => !prev)}
          className="absolute right-2 text-[#908fa0] hover:text-[#dfe2f1] cursor-pointer flex items-center justify-center p-0 m-0 bg-transparent border-none"
        >
          <span className="material-symbols-outlined text-sm">
            {isOpen ? 'expand_less' : 'expand_more'}
          </span>
        </button>
      </div>

      {isOpen && (
        <div
          role="listbox"
          className="absolute left-0 right-0 top-full mt-1 z-[100] max-h-56 overflow-y-auto bg-[#171b26] border border-[#464554] shadow-2xl rounded-none py-1 text-xs font-['Outfit',sans-serif]"
        >
          {filteredOptions.length === 0 ? (
            <div className="px-3 py-2 text-[#908fa0] text-xs italic">
              No catalog templates matching &quot;{searchQuery}&quot;. Custom ID &quot;{searchQuery}&quot; will be used.
            </div>
          ) : (
            filteredOptions.map((opt) => {
              const isSelected = opt.id === value;
              return (
                <div
                  key={opt.id}
                  role="option"
                  aria-selected={isSelected}
                  data-testid={`template-option-${opt.id}`}
                  onClick={() => {
                    onChange(opt.id);
                    onSelectTemplate?.(opt);
                    setIsOpen(false);
                    setSearchQuery('');
                  }}
                  className={`px-3 py-2 cursor-pointer flex flex-col justify-center transition-colors ${
                    isSelected
                      ? 'bg-[#c0c1ff]/20 text-[#ddb7ff] font-semibold border-l-2 border-[#ddb7ff]'
                      : 'text-[#dfe2f1] hover:bg-[#262a35] hover:text-white'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <span className="font-bold font-['Outfit']">{opt.name}</span>
                    <code className="text-[10px] font-mono text-[#c0c1ff]">{opt.id}</code>
                  </div>
                  {opt.description && (
                    <div className="text-[10px] text-[#908fa0] truncate mt-0.5">{opt.description}</div>
                  )}
                </div>
              );
            })
          )}
        </div>
      )}
    </div>
  );
}

export const SearchableTemplateInput = TemplateSelector;
