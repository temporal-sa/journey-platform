
export type DegradedStateType =
  | 'api-disconnected'
  | 'stale-draft-revision'
  | 'invalid-workflow'
  | 'failed-simulation'
  | 'failed-test-run'
  | 'stale-report'
  | 'unknown-provider-state';

export interface DegradedStateConfig {
  title: string;
  description: string;
  icon: string;
  defaultActionLabel: string;
  severity: 'error' | 'warning' | 'info';
}

export const DEGRADED_STATE_CONFIGS: Record<DegradedStateType, DegradedStateConfig> = {
  'api-disconnected': {
    title: 'API Disconnected',
    description: 'Connection to the workflow engine API lost. Offline mode active.',
    icon: 'signal_wifi_off',
    defaultActionLabel: 'Reconnect API',
    severity: 'error',
  },
  'stale-draft-revision': {
    title: 'Stale Draft Revision',
    description: 'A newer draft revision exists on the server. Your local edits are out of sync.',
    icon: 'sync_problem',
    defaultActionLabel: 'Fetch Latest Revision',
    severity: 'warning',
  },
  'invalid-workflow': {
    title: 'Invalid Workflow Configuration',
    description: 'The journey graph contains validation errors preventing execution or save.',
    icon: 'warning',
    defaultActionLabel: 'Resolve Errors',
    severity: 'error',
  },
  'failed-simulation': {
    title: 'Simulation Execution Failed',
    description: 'The simulated journey run encountered an unhandled exception or broken expression.',
    icon: 'science',
    defaultActionLabel: 'Rerun Simulation',
    severity: 'error',
  },
  'failed-test-run': {
    title: 'Test Lane Execution Failed',
    description: 'One or more test lane assertions failed during automated evaluation.',
    icon: 'cancel',
    defaultActionLabel: 'Rerun Test Suite',
    severity: 'error',
  },
  'stale-report': {
    title: 'Stale Report Data',
    description: 'The generated analytics report is out of date relative to recent graph updates.',
    icon: 'insert_chart',
    defaultActionLabel: 'Refresh Report',
    severity: 'warning',
  },
  'unknown-provider-state': {
    title: 'Unknown Provider Integration State',
    description: 'Channel provider status (Email/SMS/Push) cannot be verified or is unconfigured.',
    icon: 'help',
    defaultActionLabel: 'Check Provider Status',
    severity: 'warning',
  },
};

export interface DegradedStateViewProps {
  type: DegradedStateType;
  title?: string;
  description?: string;
  onRetry?: () => void;
  onRefresh?: () => void;
  onResolve?: () => void;
  actionLabel?: string;
  details?: string | Record<string, unknown>;
  compact?: boolean;
}

export function DegradedStateView({
  type,
  title,
  description,
  onRetry,
  onRefresh,
  onResolve,
  actionLabel,
  details,
  compact = false,
}: DegradedStateViewProps) {
  const config = DEGRADED_STATE_CONFIGS[type] || DEGRADED_STATE_CONFIGS['api-disconnected'];
  const displayTitle = title || config.title;
  const displayDescription = description || config.description;
  const handleAction = onRetry || onRefresh || onResolve;
  const displayActionLabel = actionLabel || config.defaultActionLabel;

  const isError = config.severity === 'error';
  const isWarning = config.severity === 'warning';

  const containerStyles = isError
    ? 'bg-rose-500/10 border-rose-500/30 text-rose-200'
    : isWarning
    ? 'bg-amber-500/10 border-amber-500/30 text-amber-200'
    : 'bg-[#c0c1ff]/10 border-[#c0c1ff]/30 text-[#dfe2f1]';

  const headingColor = isError
    ? 'text-rose-300'
    : isWarning
    ? 'text-amber-300'
    : 'text-[#c0c1ff]';

  const iconColor = isError
    ? 'text-rose-400 bg-rose-500/20 border-rose-500/30'
    : isWarning
    ? 'text-amber-400 bg-amber-500/20 border-amber-500/30'
    : 'text-[#c0c1ff] bg-[#c0c1ff]/20 border-[#c0c1ff]/30';

  const btnStyles = isError
    ? 'bg-[#171b26] text-rose-300 border border-rose-500/40 hover:bg-rose-500/20 hover:border-rose-400 hover:text-white'
    : isWarning
    ? 'bg-[#171b26] text-amber-300 border border-amber-500/40 hover:bg-amber-500/20 hover:border-amber-400 hover:text-white'
    : 'bg-[#171b26] text-[#ddb7ff] border border-[#ddb7ff]/40 hover:bg-[#b76dff]/20 hover:border-[#ddb7ff] hover:text-white';

  const titleId = `degraded-state-title-${type}`;
  const descId = `degraded-state-desc-${type}`;

  return (
    <div
      role="alert"
      aria-live={isError ? 'assertive' : 'polite'}
      aria-labelledby={titleId}
      aria-describedby={descId}
      data-testid={`degraded-state-${type}`}
      className={`w-full backdrop-blur-xl border rounded-none p-6 flex ${
        compact ? 'flex-row items-center justify-between gap-4 p-4' : 'flex-col sm:flex-row items-start justify-between gap-4'
      } shadow-xl font-['Outfit',sans-serif] ${containerStyles}`}
    >
      <div className="flex items-start gap-4 min-w-0 flex-1">
        <div className={`w-10 h-10 rounded-none border flex items-center justify-center shrink-0 ${iconColor}`}>
          <span className="material-symbols-outlined text-xl">{config.icon}</span>
        </div>

        <div className="min-w-0 flex-1">
          <h3 id={titleId} className={`font-['Outfit'] font-bold text-base leading-snug ${headingColor}`}>
            {displayTitle}
          </h3>
          <p id={descId} className="text-xs text-[#908fa0] mt-1 leading-relaxed">
            {displayDescription}
          </p>

          {details && (
            <div className="mt-3 p-3 rounded-none bg-[#0B0F19] border border-[#464554] text-[#c0c1ff] font-mono text-xs overflow-x-auto">
              {typeof details === 'string' ? details : JSON.stringify(details, null, 2)}
            </div>
          )}
        </div>
      </div>

      {handleAction && (
        <button
          type="button"
          onClick={handleAction}
          aria-label={`${displayActionLabel} for ${displayTitle}`}
          className={`px-4 py-2 rounded-none text-xs font-semibold font-['Outfit'] transition-all cursor-pointer hover:brightness-125 border shrink-0 ${btnStyles}`}
        >
          {displayActionLabel}
        </button>
      )}
    </div>
  );
}
