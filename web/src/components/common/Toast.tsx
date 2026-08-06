import React, { useEffect } from 'react';
import { CloseButton } from './CloseButton';

export enum ToastMessageType {
  SUCCESS = 'success',
  ERROR = 'error',
  WARNING = 'warning',
  INFO = 'info',
}

export type ToastMessageTypeValue = 'success' | 'error' | 'warning' | 'info' | ToastMessageType;

export interface ToastProps {
  id?: string;
  messageType: ToastMessageTypeValue;
  title: string;
  message?: string;
  isOpen: boolean;
  onClose: () => void;
  duration?: number;
  testId?: string;
}

export function Toast({
  messageType = ToastMessageType.SUCCESS,
  title,
  message,
  isOpen,
  onClose,
  duration = 0,
  testId = 'toast-notification',
}: ToastProps) {
  useEffect(() => {
    if (!isOpen || duration <= 0) return;
    const timer = setTimeout(() => {
      onClose();
    }, duration);
    return () => clearTimeout(timer);
  }, [isOpen, duration, onClose]);

  if (!isOpen) return null;

  const styleMap: Record<string, { bg: string; borderColor: string; borderClass: string; icon: string; iconColor: string; titleColor: string }> = {
    success: {
      bg: 'bg-[#0F131D]/60',
      borderColor: 'rgba(16, 185, 129, 0.6)',
      borderClass: 'border-emerald-500/60',
      icon: 'check_circle',
      iconColor: 'text-emerald-400',
      titleColor: 'text-white',
    },
    error: {
      bg: 'bg-[#0F131D]/60',
      borderColor: 'rgba(239, 68, 68, 0.6)',
      borderClass: 'border-rose-500/60',
      icon: 'error',
      iconColor: 'text-rose-400',
      titleColor: 'text-white',
    },
    warning: {
      bg: 'bg-[#0F131D]/60',
      borderColor: 'rgba(245, 158, 11, 0.6)',
      borderClass: 'border-amber-500/60',
      icon: 'warning',
      iconColor: 'text-amber-400',
      titleColor: 'text-white',
    },
    info: {
      bg: 'bg-[#0F131D]/60',
      borderColor: 'rgba(6, 182, 212, 0.6)',
      borderClass: 'border-cyan-500/60',
      icon: 'info',
      iconColor: 'text-cyan-400',
      titleColor: 'text-white',
    },
  };

  const key = String(messageType).toLowerCase();
  const styles = styleMap[key] || styleMap.success;

  return (
    <div
      role="status"
      aria-live="polite"
      data-testid={testId}
      style={{
        position: 'fixed',
        top: '1.5rem',
        left: '50%',
        transform: 'translateX(-50%)',
        width: '50vw',
        minWidth: '320px',
        zIndex: 9999,
        border: `1px solid ${styles.borderColor}`,
      }}
      className={`p-4 ${styles.bg} backdrop-blur-xl ${styles.borderClass} shadow-2xl flex items-start gap-3 transition-all duration-300`}
    >
      <span className={`material-symbols-outlined text-xl ${styles.iconColor} shrink-0 mt-0.5`}>
        {styles.icon}
      </span>
      <div className="flex-1 min-w-0">
        <h4 className={`text-sm font-['Outfit'] font-bold ${styles.titleColor} leading-tight`}>
          {title}
        </h4>
        {message && (
          <p className="text-xs font-['Outfit'] text-[#908fa0] mt-1 leading-normal break-words">
            {message}
          </p>
        )}
      </div>
      <CloseButton
        onClick={onClose}
        ariaLabel="Close notification"
        size="sm"
      />
    </div>
  );
}
