export interface FormatDurationOptions {
  /** Display full unit names ('4 minutes 12 seconds' vs '4m 12s') */
  verbose?: boolean;
  /** Restrict output to the primary largest unit ('4m 12s' -> '4m') */
  compact?: boolean;
  /** Maximum number of unit parts to include (default: Infinity) */
  maxUnits?: number;
  /** Decimal places for sub-minute seconds (default: 1) */
  secondsDecimalDigits?: number;
  /** Include space between number and unit ('24 ms' vs '24ms') */
  space?: boolean;
}

export function formatDuration(ms: number, options: FormatDurationOptions = {}): string {
  const {
    verbose = false,
    compact = false,
    maxUnits = Infinity,
    secondsDecimalDigits = 1,
    space = true,
  } = options;

  if (typeof ms !== 'number' || !Number.isFinite(ms)) return '0 ms';
  if (ms === 0) return verbose ? '0 milliseconds' : space ? '0 ms' : '0ms';

  const isNegative = ms < 0;
  let absMs = Math.abs(ms);

  const sp = space ? ' ' : '';

  // Sub-second handling (< 1000 ms)
  if (absMs < 1000) {
    const roundedMs = Math.round(absMs);
    const unit = verbose ? (roundedMs === 1 ? ' millisecond' : ' milliseconds') : `${sp}ms`;
    return `${isNegative ? '-' : ''}${roundedMs}${unit}`;
  }

  const days = Math.floor(absMs / 86400000);
  absMs %= 86400000;
  const hours = Math.floor(absMs / 3600000);
  absMs %= 3600000;
  const minutes = Math.floor(absMs / 60000);
  absMs %= 60000;
  const seconds = absMs / 1000;

  const parts: Array<{ val: number; strVal?: string; short: string; long: string }> = [];

  if (days > 0) parts.push({ val: days, short: 'd', long: 'day' });
  if (hours > 0) parts.push({ val: hours, short: 'h', long: 'hour' });
  if (minutes > 0) parts.push({ val: minutes, short: 'm', long: 'minute' });

  if (seconds > 0) {
    const isWhole = seconds % 1 === 0 || parts.length > 0;
    const formattedSec = isWhole
      ? Math.floor(seconds).toString()
      : seconds.toFixed(secondsDecimalDigits).replace(/\.0+$/, '');

    if (parseFloat(formattedSec) > 0 || parts.length === 0) {
      parts.push({ val: parseFloat(formattedSec), strVal: formattedSec, short: 's', long: 'second' });
    }
  }

  if (parts.length === 0) return verbose ? '0 milliseconds' : `${sp}0ms`;

  const limit = compact ? 1 : maxUnits;
  const selected = parts.slice(0, limit);

  const formattedStr = selected
    .map((p) => {
      const v = p.strVal ?? p.val;
      if (verbose) {
        const plural = p.val === 1 ? '' : 's';
        return `${v} ${p.long}${plural}`;
      }
      if (selected.length === 1 && (p.short === 's' || p.short === 'ms') && space) {
        return `${v} ${p.short}`;
      }
      return `${v}${p.short}`;
    })
    .join(' ');

  return `${isNegative ? '-' : ''}${formattedStr}`;
}
