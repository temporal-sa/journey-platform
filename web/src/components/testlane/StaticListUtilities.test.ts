import { describe, it, expect } from 'vitest';
import {
  maskRecipient,
  isFormulaInjection,
  isValidRecipient,
} from './StaticListUploadModal';

describe('StaticListUtilities - PII Masking & Formula Sanitization Utilities', () => {
  describe('maskRecipient', () => {
    it('masks email addresses preserving first character and domain structure', () => {
      expect(maskRecipient('user@domain.com')).toBe('u***r@domain.com');
      expect(maskRecipient('alexander.smith@company.org')).toBe('a***h@company.org');
      expect(maskRecipient('ab@domain.com')).toBe('a***@domain.com');
    });

    it('masks E.164 and international phone numbers', () => {
      expect(maskRecipient('+14155552671')).toBe('+1***71');
      expect(maskRecipient('+447911123456')).toBe('+4***56');
      expect(maskRecipient('12345')).toBe('1***');
    });

    it('masks plain user IDs or generic identifiers', () => {
      expect(maskRecipient('usr-99201')).toBe('u***1');
      expect(maskRecipient('xy')).toBe('x*');
      expect(maskRecipient('')).toBe('');
    });
  });

  describe('isFormulaInjection', () => {
    it('detects formula injection triggers (=, @, +, -)', () => {
      expect(isFormulaInjection('=SUM(A1:A10)')).toBe(true);
      expect(isFormulaInjection('@cmd|/C calc!A1')).toBe(true);
      expect(isFormulaInjection('+1+1')).toBe(true);
      expect(isFormulaInjection('-100')).toBe(true);
    });

    it('permits valid E.164 phone numbers starting with +', () => {
      expect(isFormulaInjection('+14155552671')).toBe(false);
      expect(isFormulaInjection('+447911123456')).toBe(false);
    });

    it('returns false for safe strings and numbers', () => {
      expect(isFormulaInjection('john.doe@example.com')).toBe(false);
      expect(isFormulaInjection('MBR-10023')).toBe(false);
      expect(isFormulaInjection('')).toBe(false);
    });
  });

  describe('isValidRecipient', () => {
    it('validates email addresses correctly', () => {
      expect(isValidRecipient('user@example.com')).toBe(true);
      expect(isValidRecipient('test.name+tag@sub.domain.co.uk')).toBe(true);
      expect(isValidRecipient('invalid-email-no-at')).toBe(false);
      expect(isValidRecipient('@missing-local.com')).toBe(false);
    });

    it('validates phone numbers correctly', () => {
      expect(isValidRecipient('+14155552671')).toBe(true);
      expect(isValidRecipient('14155552671')).toBe(true);
      expect(isValidRecipient('123')).toBe(false);
      expect(isValidRecipient('abc-phone')).toBe(false);
    });

    it('returns false for empty input', () => {
      expect(isValidRecipient('')).toBe(false);
      expect(isValidRecipient('   ')).toBe(false);
    });
  });
});
