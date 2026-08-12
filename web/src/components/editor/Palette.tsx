import React, { useState } from 'react';
import { useEditorStore } from '../../stores/editorStore';
import { CloseButton } from '../common/CloseButton';

export interface PaletteItem {
  type: string;
  name: string;
  category: 'Triggers' | 'Decisions' | 'Actions' | 'Utilities';
  icon: string;
  description: string;
  accentColor: string;
  badgeBg: string;
}

export const PALETTE_ITEMS: PaletteItem[] = [
  // Triggers
  {
    type: 'EventStart',
    name: 'Event Start',
    category: 'Triggers',
    icon: 'bolt',
    description: 'Triggers workflow on incoming domain event',
    accentColor: '#4cd7f6',
    badgeBg: 'bg-[#4cd7f6]/20 border-[#4cd7f6]/40 text-[#4cd7f6]',
  },
  // Decisions
  {
    type: 'Condition',
    name: 'Condition',
    category: 'Decisions',
    icon: 'filter_alt',
    description: 'Branch flow based on boolean expression',
    accentColor: '#f59e0b',
    badgeBg: 'bg-[#f59e0b]/20 border-[#f59e0b]/40 text-[#f59e0b]',
  },
  {
    type: 'Delay',
    name: 'Delay',
    category: 'Decisions',
    icon: 'schedule',
    description: 'Pause workflow execution for a duration',
    accentColor: '#f59e0b',
    badgeBg: 'bg-[#f59e0b]/20 border-[#f59e0b]/40 text-[#f59e0b]',
  },
  {
    type: 'WaitForEvent',
    name: 'Wait For Event',
    category: 'Decisions',
    icon: 'hourglass_empty',
    description: 'Pause until event occurs or times out',
    accentColor: '#f59e0b',
    badgeBg: 'bg-[#f59e0b]/20 border-[#f59e0b]/40 text-[#f59e0b]',
  },
  {
    type: 'Experiment',
    name: 'Experiment',
    category: 'Decisions',
    icon: 'science',
    description: 'Split traffic into experiment variants',
    accentColor: '#f59e0b',
    badgeBg: 'bg-[#f59e0b]/20 border-[#f59e0b]/40 text-[#f59e0b]',
  },
  // Actions
  {
    type: 'Email',
    name: 'Email',
    category: 'Actions',
    icon: 'mail',
    description: 'Send transactional or journey email',
    accentColor: '#c0c1ff',
    badgeBg: 'bg-[#c0c1ff]/20 border-[#c0c1ff]/40 text-[#c0c1ff]',
  },
  {
    type: 'SMS',
    name: 'SMS',
    category: 'Actions',
    icon: 'sms',
    description: 'Send SMS message to user phone',
    accentColor: '#c0c1ff',
    badgeBg: 'bg-[#c0c1ff]/20 border-[#c0c1ff]/40 text-[#c0c1ff]',
  },
  {
    type: 'Push',
    name: 'Push',
    category: 'Actions',
    icon: 'notifications_active',
    description: 'Send mobile push notification',
    accentColor: '#c0c1ff',
    badgeBg: 'bg-[#c0c1ff]/20 border-[#c0c1ff]/40 text-[#c0c1ff]',
  },
  {
    type: 'InApp',
    name: 'In-App',
    category: 'Actions',
    icon: 'smartphone',
    description: 'Display in-app message overlay',
    accentColor: '#c0c1ff',
    badgeBg: 'bg-[#c0c1ff]/20 border-[#c0c1ff]/40 text-[#c0c1ff]',
  },
  {
    type: 'Webhook',
    name: 'Webhook',
    category: 'Actions',
    icon: 'webhook',
    description: 'Invoke external HTTP API endpoint',
    accentColor: '#c0c1ff',
    badgeBg: 'bg-[#c0c1ff]/20 border-[#c0c1ff]/40 text-[#c0c1ff]',
  },
  // Utilities
  {
    type: 'Exit',
    name: 'Exit',
    category: 'Utilities',
    icon: 'flag',
    description: 'Terminal node that completes journey',
    accentColor: '#ffb4ab',
    badgeBg: 'bg-rose-500/20 border-rose-500/40 text-rose-300',
  },
];

const CATEGORY_HEADER_STYLES: Record<string, { labelBg: string; text: string }> = {
  Triggers: { labelBg: 'bg-[#4cd7f6]/25 text-[#4cd7f6] border-[#4cd7f6]/50 hover:bg-[#4cd7f6]/35', text: 'text-[#4cd7f6]' },
  Decisions: { labelBg: 'bg-[#f59e0b]/25 text-[#f59e0b] border-[#f59e0b]/50 hover:bg-[#f59e0b]/35', text: 'text-[#f59e0b]' },
  Actions: { labelBg: 'bg-[#c0c1ff]/25 text-[#c0c1ff] border-[#c0c1ff]/50 hover:bg-[#c0c1ff]/35', text: 'text-[#c0c1ff]' },
  Utilities: { labelBg: 'bg-[#ddb7ff]/25 text-[#ddb7ff] border-[#ddb7ff]/50 hover:bg-[#ddb7ff]/35', text: 'text-[#ddb7ff]' },
};

interface PaletteProps {
  onAddNode?: (type: string, name: string) => void;
}

export function Palette({ onAddNode }: PaletteProps) {
  const isCanvasLocked = useEditorStore((s) => s.isCanvasLocked);
  const [searchQuery, setSearchQuery] = useState('');
  const [collapsedCategories, setCollapsedCategories] = useState<Record<string, boolean>>({});

  const categories: Array<'Triggers' | 'Decisions' | 'Actions' | 'Utilities'> = [
    'Triggers',
    'Decisions',
    'Actions',
    'Utilities',
  ];

  const toggleCategory = (category: string) => {
    setCollapsedCategories((prev) => ({
      ...prev,
      [category]: !prev[category],
    }));
  };

  const handleDragStart = (event: React.DragEvent, item: PaletteItem) => {
    event.dataTransfer.setData('application/reactflow/type', item.type);
    event.dataTransfer.setData('application/reactflow/name', item.name);
    event.dataTransfer.effectAllowed = 'move';
  };

  const filteredItems = PALETTE_ITEMS.filter(
    (item) =>
      item.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      item.category.toLowerCase().includes(searchQuery.toLowerCase()) ||
      item.description.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const isAllCollapsed = categories.every((cat) => collapsedCategories[cat]);

  const toggleAll = () => {
    const nextState = !isAllCollapsed;
    const updated: Record<string, boolean> = {};
    categories.forEach((cat) => {
      updated[cat] = nextState;
    });
    setCollapsedCategories(updated);
  };

  return (
    <aside
      role="region"
      aria-label="Node Palette"
      className="w-64 min-w-[16rem] max-w-[16rem] bg-[#171b26] border-r border-[#464554] flex flex-col h-full max-h-full min-h-0 overflow-hidden shrink-0 text-[#dfe2f1] font-['Outfit',sans-serif] shadow-xl z-30 select-none"
      data-testid="palette-drawer"
    >
      {/* Header Section */}
      <div className="p-4 border-b border-[#464554] bg-[#1c1f2a] shrink-0">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="material-symbols-outlined text-[#c0c1ff] text-lg">widgets</span>
            <h2 className="font-['Outfit'] font-bold text-sm text-[#dfe2f1]">Node Palette</h2>
          </div>
          <button
            onClick={toggleAll}
            className="text-[10px] font-['Outfit',sans-serif] font-bold px-2 py-0.5 rounded-none bg-[#c0c1ff]/10 text-[#c0c1ff] hover:bg-[#c0c1ff]/20 border border-[#c0c1ff]/20 transition-colors cursor-pointer"
          >
            {isAllCollapsed ? 'EXPAND ALL' : 'COLLAPSE ALL'}
          </button>
        </div>
        <p className="text-xs text-[#908fa0] mt-1">Drag components onto canvas</p>

        {/* Filter Input */}
        <div className="relative mt-3">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Filter components..."
            data-testid="palette-search-input"
            className="w-full bg-[#11141d] border border-[#464554] focus:border-[#c0c1ff] rounded-none px-3 py-1.5 text-xs text-[#dfe2f1] placeholder-[#64748b] outline-none transition-all"
          />
          {searchQuery && (
            <CloseButton
              onClick={() => setSearchQuery('')}
              ariaLabel="Clear palette search input"
              size="sm"
              className="absolute right-2 top-1.5"
            />
          )}
        </div>
      </div>

      {/* Scrollable Node Categories */}
      <div className="flex-1 min-h-0 overflow-y-auto p-4 space-y-6">
        {categories.map((category) => {
          const items = filteredItems.filter((i) => i.category === category);
          if (items.length === 0) return null;

          const styles = CATEGORY_HEADER_STYLES[category];
          const isCollapsed = Boolean(collapsedCategories[category]);

          return (
            <div key={category} className="space-y-3">
              {/* Collapsible Section Header Pill with Arrow Icon */}
              <div
                onClick={() => toggleCategory(category)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    toggleCategory(category);
                  }
                }}
                tabIndex={0}
                role="button"
                aria-expanded={!isCollapsed}
                aria-label={`Toggle ${category} section`}
                data-testid={`category-header-${category}`}
                className="w-fit cursor-pointer select-none group focus:outline-none"
              >
                <span
                  className={`text-xs font-['Outfit',sans-serif] font-bold uppercase tracking-wider px-3.5 py-1.5 rounded-none border flex items-center gap-2 transition-all hover:opacity-90 ${styles.labelBg}`}
                >
                  <span>
                    {category} ({items.length})
                  </span>
                  <span className="material-symbols-outlined text-sm shrink-0">
                    {isCollapsed ? 'chevron_right' : 'expand_more'}
                  </span>
                </span>
              </div>

              {/* Subpanel Items List */}
              {!isCollapsed && (
                <div className="space-y-3 pl-1">
                  {items.map((item) => (
                    <div
                      key={item.type}
                      draggable={!isCanvasLocked}
                      tabIndex={isCanvasLocked ? -1 : 0}
                      role="button"
                      aria-label={`Add ${item.name} (${item.category}) node`}
                      onDragStart={(e) => {
                        if (isCanvasLocked) {
                          e.preventDefault();
                          return;
                        }
                        handleDragStart(e, item);
                      }}
                      onClick={() => {
                        if (isCanvasLocked) return;
                        onAddNode?.(item.type, item.name);
                      }}
                      onKeyDown={(e) => {
                        if (isCanvasLocked) return;
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault();
                          onAddNode?.(item.type, item.name);
                        }
                      }}
                      title={isCanvasLocked ? 'Canvas is locked (Read Only)' : item.description}
                      className={`group flex items-center gap-4 py-3 px-3.5 rounded-none bg-[#1c1f2a] border transition-all ${
                        isCanvasLocked
                          ? 'opacity-40 border-[#464554] cursor-not-allowed pointer-events-none'
                          : 'border-[#464554] hover:border-[#c0c1ff] hover:bg-[#262a35] hover:shadow-lg cursor-grab active:cursor-grabbing'
                      }`}
                      data-testid={`palette-item-${item.type}`}
                    >
                      {/* Node Icon Badge */}
                      <div
                        className={`w-9 h-9 rounded-none border flex items-center justify-center shrink-0 transition-transform group-hover:scale-105 mr-2 ${item.badgeBg}`}
                      >
                        <span className="material-symbols-outlined text-lg">{item.icon}</span>
                      </div>

                      {/* Node Info */}
                      <div className="flex-1 min-w-0 ml-1">
                        <div className="font-['Outfit'] font-semibold text-xs text-[#dfe2f1] group-hover:text-white transition-colors truncate">
                          {item.name}
                        </div>
                        <div className="text-[10px] text-[#908fa0] mt-0.5 truncate group-hover:text-[#dfe2f1] transition-colors">
                          {item.description}
                        </div>
                      </div>

                      {/* Drag Handle Indicator */}
                      <span className="material-symbols-outlined text-xs text-[#464554] group-hover:text-[#908fa0] transition-colors shrink-0">
                        drag_indicator
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          );
        })}

        {filteredItems.length === 0 && (
          <div className="p-6 text-center text-[#908fa0] text-xs">
            No components matching "{searchQuery}"
          </div>
        )}
      </div>
    </aside>
  );
}
