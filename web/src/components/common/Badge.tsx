import React from 'react';

export type BadgeVariant =
  | 'cyan'
  | 'purple'
  | 'emerald'
  | 'amber'
  | 'rose'
  | 'neutral';

export type BadgeSize = 'sm' | 'md' | 'lg';

export interface BadgeProps {
  children: React.ReactNode;
  variant?: BadgeVariant;
  size?: BadgeSize;
  icon?: string;
  testId?: string;
  className?: string;
}

const variantStyles: Record<BadgeVariant, string> = {
  cyan: 'bg-[#4cd7f6]/15 text-[#4cd7f6] border-[#4cd7f6]/30',
  purple: 'bg-[#c0c1ff]/15 text-[#c0c1ff] border-[#c0c1ff]/30',
  emerald: 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30',
  amber: 'bg-amber-500/15 text-amber-300 border-amber-500/30',
  rose: 'bg-rose-500/15 text-rose-300 border-rose-500/30',
  neutral: 'bg-[#1c1f2a] text-[#908fa0] border-[#464554]',
};

const sizeStyles: Record<BadgeSize, string> = {
  sm: 'px-1.5 py-0.5 text-[9px]',
  md: 'px-2 py-0.5 text-[10px]',
  lg: 'px-2.5 py-1 text-xs',
};

export const Badge: React.FC<BadgeProps> = ({
  children,
  variant = 'neutral',
  size = 'md',
  icon,
  testId,
  className = '',
}) => {
  return (
    <span
      data-testid={testId}
      className={`inline-flex items-center gap-1 rounded-none border font-mono font-bold uppercase tracking-wider select-none ${variantStyles[variant]} ${sizeStyles[size]} ${className}`}
    >
      {icon && <span className="material-symbols-outlined text-[11px] leading-none">{icon}</span>}
      <span>{children}</span>
    </span>
  );
};

export default Badge;
