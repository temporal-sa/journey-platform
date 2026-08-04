import { useRef, useState } from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { KeyboardShortcutsModal, DOCUMENTED_SHORTCUTS } from './KeyboardShortcutsModal';

function TestInvokerWrapper({ onCloseCallback }: { onCloseCallback?: () => void }) {
  const [isOpen, setIsOpen] = useState(false);
  const buttonRef = useRef<HTMLButtonElement | null>(null);

  const handleClose = () => {
    setIsOpen(false);
    onCloseCallback?.();
  };

  return (
    <div>
      <button
        ref={buttonRef}
        data-testid="invoking-trigger-btn"
        onClick={() => setIsOpen(true)}
      >
        Open Shortcuts
      </button>

      <KeyboardShortcutsModal
        isOpen={isOpen}
        onClose={handleClose}
        invokingControlRef={buttonRef}
      />
    </div>
  );
}

describe('KeyboardShortcutsModal Component', () => {
  it('does not render when isOpen is false', () => {
    render(<KeyboardShortcutsModal isOpen={false} onClose={vi.fn()} />);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('renders modal dialog with documented shortcuts and accessibility attributes', () => {
    const handleClose = vi.fn();
    render(<KeyboardShortcutsModal isOpen={true} onClose={handleClose} />);

    const dialog = screen.getByRole('dialog');
    expect(dialog).toBeInTheDocument();
    expect(dialog).toHaveAttribute('aria-modal', 'true');
    expect(dialog).toHaveAttribute('aria-labelledby', 'shortcuts-modal-title');
    expect(dialog).toHaveAttribute('aria-describedby', 'shortcuts-modal-desc');

    // Check header title
    expect(screen.getByText('Keyboard Shortcuts')).toBeInTheDocument();

    // Check documented shortcuts are present
    DOCUMENTED_SHORTCUTS.forEach((sc) => {
      expect(screen.getByText(sc.key)).toBeInTheDocument();
      expect(screen.getByText(sc.description)).toBeInTheDocument();
    });

    // Check close button
    const closeBtn = screen.getByRole('button', { name: 'Close keyboard shortcuts modal' });
    expect(closeBtn).toBeInTheDocument();

    fireEvent.click(closeBtn);
    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('traps focus inside the modal when pressing Tab and Shift+Tab', async () => {
    render(<KeyboardShortcutsModal isOpen={true} onClose={vi.fn()} />);

    const closeHeaderBtn = screen.getByRole('button', { name: 'Close keyboard shortcuts modal' });
    const doneBtn = screen.getByTestId('close-shortcuts-modal-btn');

    // Focus initial element inside modal
    closeHeaderBtn.focus();
    expect(document.activeElement).toBe(closeHeaderBtn);

    // Tab from first to next
    fireEvent.keyDown(closeHeaderBtn, { key: 'Tab' });

    // Shift+Tab from close header button should wrap around to doneBtn
    fireEvent.keyDown(closeHeaderBtn, { key: 'Tab', shiftKey: true });
    expect(document.activeElement).toBe(doneBtn);
  });

  it('closes on Escape key press', () => {
    const handleClose = vi.fn();
    render(<KeyboardShortcutsModal isOpen={true} onClose={handleClose} />);

    const dialog = screen.getByRole('dialog');
    fireEvent.keyDown(dialog, { key: 'Escape' });

    expect(handleClose).toHaveBeenCalledTimes(1);
  });

  it('returns focus to the invoking control element after modal closes', () => {
    render(<TestInvokerWrapper />);

    const triggerBtn = screen.getByTestId('invoking-trigger-btn');
    triggerBtn.focus();
    expect(document.activeElement).toBe(triggerBtn);

    // Click trigger button to open modal
    fireEvent.click(triggerBtn);
    expect(screen.getByRole('dialog')).toBeInTheDocument();

    // Close modal via Done button
    const doneBtn = screen.getByTestId('close-shortcuts-modal-btn');
    fireEvent.click(doneBtn);

    // Modal closed
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();

    // Focus returned to triggerBtn
    expect(document.activeElement).toBe(triggerBtn);
  });
});
