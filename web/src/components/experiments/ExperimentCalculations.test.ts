import { describe, it, expect } from 'vitest';

describe('ExperimentCalculations - Basis-Point Calculations & Report Formatting', () => {
  // Basis point conversion: 1% = 100 bps, 100% = 10,000 bps
  describe('Basis-Point & Percentage Transformations', () => {
    it('converts percentages to basis points accurately (1% = 100 bps)', () => {
      const pctToBps = (pct: number) => Math.round(pct * 100);
      expect(pctToBps(50)).toBe(5000);
      expect(pctToBps(33.33)).toBe(3333);
      expect(pctToBps(100)).toBe(10000);
      expect(pctToBps(0)).toBe(0);
    });

    it('converts basis points to formatted percentages accurately', () => {
      const bpsToPct = (bps: number) => (bps / 100).toFixed(2).replace(/\.00$/, '');
      expect(bpsToPct(5000)).toBe('50');
      expect(bpsToPct(3333)).toBe('33.33');
      expect(bpsToPct(10000)).toBe('100');
      expect(bpsToPct(0)).toBe('0');
    });

    it('validates total traffic allocation equals exactly 10,000 basis points (100%)', () => {
      const validateAllocation = (variants: Array<{ weight_basis_points: number }>) => {
        const total = variants.reduce((sum, v) => sum + v.weight_basis_points, 0);
        return {
          totalBps: total,
          isValid: total === 10000,
        };
      };

      const valid5050 = validateAllocation([
        { weight_basis_points: 5000 },
        { weight_basis_points: 5000 },
      ]);
      expect(valid5050.isValid).toBe(true);
      expect(valid5050.totalBps).toBe(10000);

      const valid3Way = validateAllocation([
        { weight_basis_points: 3400 },
        { weight_basis_points: 3300 },
        { weight_basis_points: 3300 },
      ]);
      expect(valid3Way.isValid).toBe(true);

      const invalidSum = validateAllocation([
        { weight_basis_points: 5000 },
        { weight_basis_points: 4000 },
      ]);
      expect(invalidSum.isValid).toBe(false);
      expect(invalidSum.totalBps).toBe(9000);
    });
  });

  describe('Report Metric Aggregations & Confidence Intervals', () => {
    it('computes overall conversion rate across all variants', () => {
      const variants = [
        { delivered: 4920, conversion: 492 }, // 10.0%
        { delivered: 4890, conversion: 635 }, // 13.0%
      ];
      const totalDelivered = variants.reduce((s, v) => s + v.delivered, 0);
      const totalConversion = variants.reduce((s, v) => s + v.conversion, 0);
      const overallRate = totalDelivered > 0 ? totalConversion / totalDelivered : 0;

      expect(totalDelivered).toBe(9810);
      expect(totalConversion).toBe(1127);
      expect((overallRate * 100).toFixed(2)).toBe('11.49');
    });

    it('calculates max treatment lift over control baseline', () => {
      const treatmentLifts = [15.5, 30.0, -5.0];
      const maxLift = Math.max(...treatmentLifts);
      expect(maxLift).toBe(30.0);
    });

    it('formats 95% confidence intervals properly', () => {
      const formatCI = (ci: [number, number]) => `[${ci[0].toFixed(2)}%, ${ci[1].toFixed(2)}%]`;
      expect(formatCI([9.1623, 10.8377])).toBe('[9.16%, 10.84%]');
      expect(formatCI([12.058, 13.942])).toBe('[12.06%, 13.94%]');
    });
  });
});
