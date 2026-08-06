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
  iconPosition?: 'left' | 'right';
  fullWidth?: boolean;
  isLoading?: boolean;
  hoverColor?: string; // Configurable hover background color
  children?: React.ReactNode;
}

const VARIANT_STYLES: Record<ButtonVariant, { className: string; style?: React.CSSProperties }> = {
  'primary-purple': {
    className: 'bg-[#b76dff] text-[#400071] hover:brightness-110 border-none shadow-none font-bold',
    style: { backgroundColor: '#b76dff', color: '#400071' },
  },
  'primary-cyan': {
    className: 'bg-[#4cd7f6] text-[#400071] hover:brightness-110 border-none shadow-none font-bold',
    style: { backgroundColor: '#4cd7f6', color: '#400071' },
  },
  'primary-teal': {
    className: 'bg-[#4cd7f6] text-[#003640] hover:brightness-110 border border-[#4cd7f6]/40 shadow-lg shadow-[#4cd7f6]/20 font-bold',
    style: { backgroundColor: '#4cd7f6', color: '#003640' },
  },
  'primary-lavender': {
    className: 'bg-[#c0c1ff] text-[#1000a9] hover:brightness-110 border border-[#c0c1ff]/40 shadow-lg shadow-[#c0c1ff]/20 font-bold',
    style: { backgroundColor: '#c0c1ff', color: '#1000a9' },
  },
  'primary-indigo': {
    className: 'bg-[#6366F1] text-white hover:brightness-110 border border-[#8083ff]/40 shadow-md font-bold',
  },
  'secondary-dark': {
    className: 'bg-[#1c1f2a] hover:brightness-110 text-[#dfe2f1] border border-[#464554] font-semibold',
  },
  'secondary': {
    className: 'bg-[#1c1f2a] hover:brightness-110 text-[#dfe2f1] border border-[#464554] font-semibold',
  },
  'danger': {
    className: 'bg-rose-500/20 hover:bg-rose-500/35 text-rose-300 border border-rose-500/50 shadow-md font-bold',
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
      iconPosition = 'left',
      fullWidth = false,
      isLoading = false,
      hoverColor,
      disabled,
      className = '',
      style,
      onMouseEnter,
      onMouseLeave,
      children,
      ...props
    },
    ref
  ) => {
    const [isHovered, setIsHovered] = React.useState(false);
    const variantConfig = VARIANT_STYLES[variant] || VARIANT_STYLES['secondary-dark'];
    const sizeClass = SIZE_STYLES[size] || SIZE_STYLES.md;

    const baseClasses = `rounded-none font-['Outfit'] transition-all cursor-pointer inline-flex items-center justify-center gap-2 select-none ${
      fullWidth ? 'w-full' : ''
    } ${
      disabled || isLoading ? 'opacity-50 cursor-not-allowed pointer-events-none' : ''
    } ${sizeClass} ${variantConfig.className} ${className}`;

    const combinedStyle: React.CSSProperties = {
      ...variantConfig.style,
      ...(isHovered && hoverColor ? { backgroundColor: hoverColor } : {}),
      ...style,
    };

    const handleMouseEnter = (e: React.MouseEvent<HTMLButtonElement>) => {
      setIsHovered(true);
      if (onMouseEnter) onMouseEnter(e);
    };

    const handleMouseLeave = (e: React.MouseEvent<HTMLButtonElement>) => {
      setIsHovered(false);
      if (onMouseLeave) onMouseLeave(e);
    };

    return (
      <button
        ref={ref}
        disabled={disabled || isLoading}
        className={baseClasses}
        style={combinedStyle}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        {...props}
      >
        {isLoading ? (
          <span className="material-symbols-outlined text-base animate-spin">sync</span>
        ) : icon && iconPosition === 'left' ? (
          <span className="material-symbols-outlined text-base">{icon}</span>
        ) : null}
        {children && <span>{children}</span>}
        {!isLoading && icon && iconPosition === 'right' ? (
          <span className="material-symbols-outlined text-base">{icon}</span>
        ) : null}
      </button>
    );
  }
);

Button.displayName = 'Button';
