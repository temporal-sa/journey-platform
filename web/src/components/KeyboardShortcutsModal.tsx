import { useRef } from 'react';
import { Button } from './common/Button';
import { Modal } from './common/Modal';
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
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Keyboard Shortcuts"
      subtitle="Quick navigation and editing hotkeys for Journey Control Engine"
      icon="keyboard"
      iconAccentColor="#4cd7f6"
      maxWidth="2xl"
      testId="shortcuts-modal-backdrop"
      ariaLabelledBy="shortcuts-modal-title"
      ariaDescribedBy="shortcuts-modal-desc"
      footer={
        <Button
          onClick={onClose}
          data-testid="close-shortcuts-modal-btn"
          variant="primary-cyan"
        >
          Done
        </Button>
      }
    >
      <div ref={modalRef} tabIndex={-1} className="p-6 focus:outline-none">
        {/* Shortcuts List / Table */}
        <div className="max-h-[60vh] overflow-y-auto rounded-none border border-[#464554] bg-[#0F131D]">
          <table className="text-sm w-full border-collapse text-left">
            <thead>
              <tr className="border-b border-[#464554] bg-[#171b26]">
                <th className="p-3 text-xs font-bold text-[#908fa0] uppercase tracking-wider">Shortcut</th>
                <th className="p-3 text-xs font-bold text-[#908fa0] uppercase tracking-wider">Action</th>
                <th className="p-3 text-xs font-bold text-[#908fa0] uppercase tracking-wider">Category</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#464554]/30">
              {DOCUMENTED_SHORTCUTS.map((sc, index) => (
                <tr
                  key={index}
                  className={index % 2 === 0 ? 'bg-[#0F131D]' : 'bg-[#171b26]'}
                >
                  <td className="p-3">
                    <kbd className="text-xs font-mono font-semibold bg-[#1F2433] text-[#c0c1ff] border border-[#464554] px-2 py-1 rounded-none inline-block shadow-sm">
                      {sc.key}
                    </kbd>
                  </td>
                  <td className="p-3 text-[#dfe2f1] font-medium">
                    {sc.description}
                  </td>
                  <td className="p-3">
                    <span
                      className={`text-xs px-2.5 py-0.5 font-semibold rounded-none inline-block ${
                        sc.category === 'Global'
                          ? 'bg-[#4cd7f6]/10 text-[#4cd7f6] border border-[#4cd7f6]/30'
                          : sc.category === 'Editor'
                          ? 'bg-[#b76dff]/10 text-[#ddb7ff] border border-[#b76dff]/30'
                          : 'bg-[#c0c1ff]/10 text-[#c0c1ff] border border-[#c0c1ff]/30'
                      }`}
                    >
                      {sc.category}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </Modal>
  );
}
