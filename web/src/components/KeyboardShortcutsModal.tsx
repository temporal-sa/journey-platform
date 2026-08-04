import { useRef } from 'react';
import { useFocusTrap } from '../hooks/useFocusTrap';

export interface ShortcutItem {
  key: string;
  description: string;
  category: 'Global' | 'Editor' | 'Navigation';
}

export const DOCUMENTED_SHORTCUTS: ShortcutItem[] = [
  { key: '?', description: 'Open Keyboard Shortcuts Help modal', category: 'Global' },
  { key: 'Ctrl+Z / Cmd+Z', description: 'Undo last editor action', category: 'Editor' },
  { key: 'Ctrl+Y / Cmd+Shift+Z', description: 'Redo last undone action', category: 'Editor' },
  { key: 'Cmd+S / Ctrl+S', description: 'Save current draft changes', category: 'Editor' },
  { key: 'Escape', description: 'Close active modal or clear canvas selection', category: 'Global' },
  { key: 'Delete / Backspace', description: 'Delete selected graph node or edge', category: 'Editor' },
];

export interface KeyboardShortcutsModalProps {
  isOpen: boolean;
  onClose: () => void;
  invokingControlRef?: React.RefObject<HTMLElement | null>;
}

export function KeyboardShortcutsModal({
  isOpen,
  onClose,
  invokingControlRef,
}: KeyboardShortcutsModalProps) {
  const modalRef = useRef<HTMLDivElement | null>(null);

  useFocusTrap(modalRef, {
    active: isOpen,
    onClose,
    returnFocusOnDeactivate: true,
    initialFocusRef: invokingControlRef ? undefined : undefined,
  });

  if (!isOpen) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="shortcuts-modal-title"
      aria-describedby="shortcuts-modal-desc"
      style={{
        position: 'fixed',
        inset: 0,
        backgroundColor: 'rgba(15, 23, 42, 0.65)',
        backdropFilter: 'blur(4px)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
        padding: '1rem',
      }}
      data-testid="keyboard-shortcuts-modal"
    >
      <div
        ref={modalRef}
        tabIndex={-1}
        style={{
          backgroundColor: '#ffffff',
          borderRadius: '0px',
          maxWidth: '560px',
          width: '100%',
          boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)',
          overflow: 'hidden',
          border: '1px solid #e2e8f0',
          outline: 'none',
        }}
      >
        {/* Header */}
        <div
          style={{
            padding: '1.25rem 1.5rem',
            borderBottom: '1px solid #e2e8f0',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            backgroundColor: '#f8fafc',
          }}
        >
          <div>
            <h2
              id="shortcuts-modal-title"
              className="text-lg font-bold"
              style={{ margin: 0, color: '#0f172a' }}
            >
              Keyboard Shortcuts
            </h2>
            <p
              id="shortcuts-modal-desc"
              className="text-xs"
              style={{ margin: '0.25rem 0 0 0', color: '#64748b' }}
            >
              Quick navigation and editing hotkeys for Journey Control Engine
            </p>
          </div>
          <button
            onClick={onClose}
            aria-label="Close keyboard shortcuts modal"
            className="w-8 h-8 rounded-none bg-[#262a35] text-[#908fa0] hover:text-white hover:bg-white/10 flex items-center justify-center transition-all border border-white/5 cursor-pointer shrink-0 shadow-sm"
          >
            <span className="material-symbols-outlined text-lg">close</span>
          </button>
        </div>

        {/* Shortcuts List / Table */}
        <div style={{ padding: '1.5rem', maxHeight: '60vh', overflowY: 'auto' }}>
          <table
            className="text-sm w-full"
            style={{ borderCollapse: 'collapse', textAlign: 'left' }}
          >
            <thead>
              <tr style={{ borderBottom: '2px solid #e2e8f0' }}>
                <th style={{ padding: '0.5rem 0.75rem', color: '#475569', fontWeight: 600 }}>Shortcut</th>
                <th style={{ padding: '0.5rem 0.75rem', color: '#475569', fontWeight: 600 }}>Action</th>
                <th style={{ padding: '0.5rem 0.75rem', color: '#475569', fontWeight: 600 }}>Category</th>
              </tr>
            </thead>
            <tbody>
              {DOCUMENTED_SHORTCUTS.map((sc, index) => (
                <tr
                  key={index}
                  style={{
                    borderBottom: '1px solid #f1f5f9',
                    backgroundColor: index % 2 === 0 ? '#ffffff' : '#f8fafc',
                  }}
                >
                  <td style={{ padding: '0.625rem 0.75rem' }}>
                    <kbd
                      className="text-xs font-mono font-semibold"
                      style={{
                        backgroundColor: '#f1f5f9',
                        border: '1px solid #cbd5e1',
                        borderRadius: '0px',
                        padding: '0.2rem 0.4rem',
                        color: '#0f172a',
                        boxShadow: '0 1px 1px rgba(0,0,0,0.05)',
                      }}
                    >
                      {sc.key}
                    </kbd>
                  </td>
                  <td style={{ padding: '0.625rem 0.75rem', color: '#334155', fontWeight: 500 }}>
                    {sc.description}
                  </td>
                  <td style={{ padding: '0.625rem 0.75rem' }}>
                    <span
                      className="text-xs"
                      style={{
                        padding: '0.15rem 0.4rem',
                        borderRadius: '0px',
                        backgroundColor:
                          sc.category === 'Global'
                            ? '#eff6ff'
                            : sc.category === 'Editor'
                            ? '#f0fdf4'
                            : '#fefce8',
                        color:
                          sc.category === 'Global'
                            ? '#1d4ed8'
                            : sc.category === 'Editor'
                            ? '#15803d'
                            : '#a16207',
                        fontWeight: 600,
                      }}
                    >
                      {sc.category}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Footer */}
        <div
          style={{
            padding: '1rem 1.5rem',
            borderTop: '1px solid #e2e8f0',
            backgroundColor: '#f8fafc',
            textAlign: 'right',
          }}
        >
          <button
            onClick={onClose}
            data-testid="close-shortcuts-modal-btn"
            style={{
              padding: '0.5rem 1rem',
              backgroundColor: '#2563eb',
              color: '#ffffff',
              border: 'none',
              borderRadius: '0px',
              fontWeight: 600,
              cursor: 'pointer',
              fontSize: '0.875rem',
            }}
          >
            Done
          </button>
        </div>
      </div>
    </div>
  );
}
