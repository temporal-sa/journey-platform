import React from 'react';

export interface DirectoryLayoutProps {
  title: string;
  subtitle?: string;
  icon?: string;
  iconAccentColor?: string;
  headerActions?: React.ReactNode;
  controls?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}

export function DirectoryLayout({
  title,
  subtitle,
  icon = 'table_chart',
  iconAccentColor = '#c0c1ff',
  headerActions,
  controls,
  children,
  className = '',
}: DirectoryLayoutProps) {
  return (
    <div className={`w-full h-full flex flex-col p-6 bg-[#0B0F19] text-[#dfe2f1] font-['Outfit',sans-serif] overflow-y-auto space-y-6 ${className}`}>
      {/* Top Header Section */}
      <div className="w-full flex flex-col sm:flex-row sm:items-center justify-between gap-4 shrink-0">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-2.5 font-['Outfit']">
            {icon && (
              <span
                className="material-symbols-outlined text-2xl"
                style={{ color: iconAccentColor }}
                aria-hidden="true"
              >
                {icon}
              </span>
            )}
            <span>{title}</span>
          </h1>
          {subtitle && (
            <p className="text-xs text-[#908fa0] mt-1 font-['Outfit']">{subtitle}</p>
          )}
        </div>

        {headerActions && (
          <div className="flex items-center gap-3 shrink-0">{headerActions}</div>
        )}
      </div>

      {/* Control / Filter Bar */}
      {controls && (
        <div className="bg-[#0F131D]/90 backdrop-blur-xl p-4 border border-[#464554] shadow-md flex flex-wrap items-center justify-between gap-4 rounded-none shrink-0 relative z-30">
          {controls}
        </div>
      )}

      {/* Main Content Area */}
      <div className="flex-1 w-full min-h-0 flex flex-col">
        {children}
      </div>
    </div>
  );
}
