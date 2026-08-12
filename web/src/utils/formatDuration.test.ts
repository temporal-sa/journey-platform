import { describe, it, expect } from 'vitest';
import { formatDuration } from './formatDuration';

describe('formatDuration Utility', () => {
  it('formats 0 and sub-second milliseconds correctly', () => {
    expect(formatDuration(0)).toBe('0 ms');
    expect(formatDuration(24)).toBe('24 ms');
    expect(formatDuration(850)).toBe('850 ms');
  });

  it('formats sub-minute seconds with decimals when fractional', () => {
    expect(formatDuration(1200)).toBe('1.2 s');
    expect(formatDuration(18500)).toBe('18.5 s');
    expect(formatDuration(45000)).toBe('45 s');
  });

  it('formats multi-unit minute, hour, and day durations', () => {
    expect(formatDuration(252000)).toBe('4m 12s');
    expect(formatDuration(8130000)).toBe('2h 15m 30s');
    expect(formatDuration(273600000)).toBe('3d 4h');
  });

  it('supports options: compact, verbose, and custom spacing', () => {
    expect(formatDuration(252000, { compact: true })).toBe('4m');
    expect(formatDuration(252000, { verbose: true })).toBe('4 minutes 12 seconds');
    expect(formatDuration(24, { space: false })).toBe('24ms');
  });

  it('handles negative or invalid number fallbacks safely', () => {
    expect(formatDuration(-1200)).toBe('-1.2 s');
    expect(formatDuration(NaN)).toBe('0 ms');
  });
});
