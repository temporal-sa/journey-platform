import React from 'react';

export type ButtonVariant =
  | 'primary-purple'   // Solid #b76dff fill + #400071 text (New Journey)
  | 'primary-cyan'     // Solid #4cd7f6 fill + #400071 text (AGY Assistant Copy)
  | 'primary-teal'     // Solid #4cd7f6 fill + #003640 text (Test Run / Static List Submit)
  | 'primary-lavender' // Solid #c0c1ff fill + #1000a9 text (Publish / Conflict Overwrite)
  | 'primary-indigo'   // Solid #6366F1 fill + white text (Save Experiment)
  | 'secondary-dark'   // Dark #1c1f2a fill + #dfe2f1 text + #464554 border
  | 'danger'           // Destructive action
  | 'ghost';           // Transparent background

export type ButtonSize = 'sm' | 'md' | 'lg';

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: string;
  fullWidth?: boolean;
  isLoading?: boolean;
  children?: React.ReactNode;
}

const VARIANT_STYLES: Record<ButtonVariant, { className: string; style?: React.CSSProperties }> = {
  'primary-purple': {
    className: 'bg-[#b76dff] text-[#400071] hover:opacity-90 border-none shadow-none font-bold',
    style: { backgroundColor: '#b76dff', color: '#400071' },
  },
  'primary-cyan': {
    className: 'bg-[#4cd7f6] text-[#400071] hover:opacity-90 border-none shadow-none font-bold',
    style: { backgroundColor: '#4cd7f6', color: '#400071' },
  },
  'primary-teal': {
    className: 'bg-[#4cd7f6] text-[#003640] hover:bg-[#38c2e0] border border-[#4cd7f6]/40 shadow-lg shadow-[#4cd7f6]/20 font-bold',
    style: { backgroundColor: '#4cd7f6', color: '#003640' },
  },
  'primary-lavender': {
    className: 'bg-[#c0c1ff] text-[#1000a9] hover:brightness-110 border border-[#c0c1ff]/40 shadow-lg shadow-[#c0c1ff]/20 font-bold',
    style: { backgroundColor: '#c0c1ff', color: '#1000a9' },
  },
  'primary-indigo': {
    className: 'bg-[#6366F1] text-white hover:bg-[#4F46E5] border border-[#8083ff]/40 shadow-md font-bold',
  },
  'secondary-dark': {
    className: 'bg-[#1c1f2a] hover:bg-[#262a35] text-[#dfe2f1] border border-[#464554] font-semibold',
  },
  'danger': {
    className: 'bg-rose-600 hover:bg-rose-700 text-white border border-rose-500/40 font-bold shadow-md',
  },
  'ghost': {
    className: 'bg-transparent text-[#908fa0] hover:text-white hover:bg-white/10 border-none',
  },
};

const SIZE_STYLES: Record<ButtonSize, string> = {
  sm: 'px-3 py-1.5 text-xs',
  md: 'px-4 py-2 text-xs',
  lg: 'px-5 py-2.5 text-sm',
};

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      variant = 'secondary-dark',
      size = 'md',
      icon,
      fullWidth = false,
      isLoading = false,
      disabled,
      className = '',
      style,
      children,
      ...props
    },
    ref
  ) => {
    const variantConfig = VARIANT_STYLES[variant];
    const sizeClass = SIZE_STYLES[size];

    const baseClasses = `rounded-none font-['Outfit'] transition-all cursor-pointer inline-flex items-center justify-center gap-2 select-none ${
      fullWidth ? 'w-full' : ''
    } ${
      disabled || isLoading ? 'opacity-50 cursor-not-allowed pointer-events-none' : ''
    } ${sizeClass} ${variantConfig.className} ${className}`;

    const combinedStyle = {
      ...variantConfig.style,
      ...style,
    };

    return (
      <button
        ref={ref}
        disabled={disabled || isLoading}
        className={baseClasses}
        style={combinedStyle}
        {...props}
      >
        {isLoading ? (
          <span className="material-symbols-outlined text-base animate-spin">sync</span>
        ) : icon ? (
          <span className="material-symbols-outlined text-base">{icon}</span>
        ) : null}
        {children && <span>{children}</span>}
      </button>
    );
  }
);

Button.displayName = 'Button';
