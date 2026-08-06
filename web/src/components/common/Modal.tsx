import React, { useEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import { CloseButton } from './CloseButton';

export interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  icon?: string;
  iconAccentColor?: string;
  badge?: React.ReactNode;
  children: React.ReactNode;
  footer?: React.ReactNode;
  maxWidth?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl';
  testId?: string;
  ariaLabelledBy?: string;
  ariaDescribedBy?: string;
  className?: string;
  bodyClassName?: string;
  maxHeight?: string;
  usePortal?: boolean;
  zIndex?: number;
}

const maxWidthMap = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-xl',
  '2xl': 'max-w-2xl',
  '3xl': 'max-w-3xl',
  '4xl': 'max-w-4xl',
};

export const Modal: React.FC<ModalProps> = ({
  isOpen,
  onClose,
  title,
  subtitle,
  icon,
  iconAccentColor = '#4cd7f6',
  badge,
  children,
  footer,
  maxWidth = '2xl',
  testId = 'modal-backdrop',
  ariaLabelledBy = 'modal-title',
  ariaDescribedBy,
  className = '',
  bodyClassName = "p-6 flex-1 min-h-0 flex flex-col overflow-y-auto text-xs font-['Outfit',sans-serif]",
  maxHeight = '85vh',
  usePortal = true,
  zIndex = 99999,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const modalContent = (
    <div
      className="fixed inset-0 bg-[#0B0F19]/85 backdrop-blur-md flex items-center justify-center p-4 sm:p-6 font-['Outfit',sans-serif]"
      style={{ zIndex }}
      role="dialog"
      aria-modal="true"
      aria-labelledby={ariaLabelledBy}
      aria-describedby={ariaDescribedBy}
      data-testid={testId}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        ref={containerRef}
        style={{ maxHeight }}
        className={`bg-[#0F131D]/95 backdrop-blur-xl border border-[#464554] rounded-none w-full ${maxWidthMap[maxWidth]} flex flex-col shadow-2xl shadow-black/90 overflow-hidden glass-modal shrink-0 relative ${className}`}
      >
        {/* Header */}
        <div className="bg-[#171b26] p-5 border-b border-[#464554] flex items-center justify-between shrink-0 gap-4">
          <div className="flex items-center gap-3 min-w-0">
            {icon && (
              <div
                className="w-10 h-10 rounded-none flex items-center justify-center shrink-0 shadow-md"
                style={{
                  backgroundColor: `${iconAccentColor}1a`,
                  borderColor: `${iconAccentColor}66`,
                  color: iconAccentColor,
                  borderWidth: '1px',
                  borderStyle: 'solid',
                }}
              >
                <span className="material-symbols-outlined text-xl">{icon}</span>
              </div>
            )}
            <div className="min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                {badge}
                <h2 id={ariaLabelledBy} className="font-['Outfit'] font-bold text-lg text-white tracking-wide truncate">
                  {title}
                </h2>
              </div>
              {subtitle && <p id={ariaDescribedBy} className="text-xs text-[#908fa0] mt-0.5">{subtitle}</p>}
            </div>
          </div>
          <CloseButton onClick={onClose} ariaLabel="Close modal" />
        </div>

        {/* Content Body */}
        <div className={bodyClassName}>
          {children}
        </div>

        {/* Footer (Optional) */}
        {footer && (
          <div className="p-4 bg-[#171b26] border-t border-[#464554] flex justify-end items-center gap-3 shrink-0">
            {footer}
          </div>
        )}
      </div>
    </div>
  );

  if (usePortal && typeof document !== 'undefined') {
    return createPortal(modalContent, document.body);
  }

  return modalContent;
};

export default Modal;
