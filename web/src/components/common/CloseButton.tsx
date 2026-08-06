import React from 'react';

export interface CloseButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  onClick?: (e: React.MouseEvent<HTMLButtonElement>) => void;
  ariaLabel?: string;
  testId?: string;
  className?: string;
  size?: 'sm' | 'md' | 'lg';
  bgOpacity?: number;
}

export const CloseButton: React.FC<CloseButtonProps> = ({
  onClick,
  ariaLabel = 'Close',
  testId,
  className = '',
  size = 'md',
  bgOpacity,
  style,
  ...props
}) => {
  const sizeMap: Record<string, string> = {
    sm: 'w-6 h-6',
    md: 'w-8 h-8',
    lg: 'w-10 h-10',
  };

  const iconSizeMap: Record<string, string> = {
    sm: 'text-base',
    md: 'text-lg',
    lg: 'text-xl',
  };

  const sizeClasses = sizeMap[size] || sizeMap.md;
  const iconSizeClasses = iconSizeMap[size] || iconSizeMap.md;

  const bgStyle = bgOpacity !== undefined && bgOpacity > 0
    ? { backgroundColor: `rgba(28, 31, 42, ${bgOpacity <= 1 ? bgOpacity : bgOpacity / 100})` }
    : {};

  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={ariaLabel}
      data-testid={testId}
      style={{
        appearance: 'none',
        WebkitAppearance: 'none',
        background: 'transparent',
        border: 'none',
        outline: 'none',
        boxShadow: 'none',
        textShadow: 'none',
        filter: 'none',
        transform: 'none',
        ...bgStyle,
        ...style,
      }}
      className={`flat-icon-btn text-[#908fa0] hover:text-white flex items-center justify-center transition-colors cursor-pointer shrink-0 p-0 m-0 border-0 outline-none shadow-none ${sizeClasses} ${className}`}
      {...props}
    >
      <span className={`material-symbols-outlined ${iconSizeClasses}`}>close</span>
    </button>
  );
};

export default CloseButton;
