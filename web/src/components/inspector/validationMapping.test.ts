import { describe, it, expect } from 'vitest';
import {
  getNodeIssues,
  getFieldError,
  getNodeValidationCounts,
  getTabErrorCounts,
  formatIssueCode,
} from './validationMapping';
import type { ValidationResult, ValidationIssue } from '../../types/api';

describe('validationMapping - Form Transformations & Error Mapping Utilities', () => {
  const mockIssues: ValidationIssue[] = [
    {
      schema_version: '1.0',
      issue_id: 'issue-1',
      code: 'ERR_MISSING_RECIPIENT',
      node_id: 'node-email-1',
      field_path: 'config.recipient',
      severity: 'error',
      message: 'Recipient email address is required',
    },
    {
      schema_version: '1.0',
      issue_id: 'issue-2',
      code: 'WARN_HIGH_LATENCY',
      node_id: 'node-email-1',
      field_path: 'config.timeout_ms',
      severity: 'warning',
      message: 'Timeout is set above recommended 5000ms threshold',
    },
    {
      schema_version: '1.0',
      issue_id: 'issue-3',
      code: 'ERR_INVALID_CONDITION',
      node_id: 'node-cond-1',
      field_path: 'expression',
      severity: 'error',
      message: 'Condition expression syntax error',
    },
  ];

  const mockValidationResult: ValidationResult = {
    draft_id: 'draft-101',
    is_valid: false,
    issues: mockIssues,
  };

  describe('getNodeIssues', () => {
    it('returns empty array when nodeId or validationResult is null/undefined', () => {
      expect(getNodeIssues(null, mockValidationResult)).toEqual([]);
      expect(getNodeIssues('node-email-1', null)).toEqual([]);
      expect(getNodeIssues(undefined, undefined)).toEqual([]);
    });

    it('filters validation issues for a specific node ID', () => {
      const emailIssues = getNodeIssues('node-email-1', mockValidationResult);
      expect(emailIssues).toHaveLength(2);
      expect(emailIssues[0].code).toBe('ERR_MISSING_RECIPIENT');
      expect(emailIssues[1].code).toBe('WARN_HIGH_LATENCY');

      const condIssues = getNodeIssues('node-cond-1', mockValidationResult);
      expect(condIssues).toHaveLength(1);
      expect(condIssues[0].code).toBe('ERR_INVALID_CONDITION');
    });
  });

  describe('getFieldError', () => {
    it('returns undefined when fieldPath is empty or nodeIssues is empty', () => {
      expect(getFieldError('', mockIssues)).toBeUndefined();
      expect(getFieldError('recipient', [])).toBeUndefined();
    });

    it('matches field path with or without config. prefix', () => {
      const emailNodeIssues = getNodeIssues('node-email-1', mockValidationResult);

      const errWithPrefix = getFieldError('config.recipient', emailNodeIssues);
      expect(errWithPrefix?.code).toBe('ERR_MISSING_RECIPIENT');

      const errWithoutPrefix = getFieldError('recipient', emailNodeIssues);
      expect(errWithoutPrefix?.code).toBe('ERR_MISSING_RECIPIENT');

      const warningField = getFieldError('timeout_ms', emailNodeIssues);
      expect(warningField?.code).toBe('WARN_HIGH_LATENCY');
    });

    it('is case-insensitive when matching field paths', () => {
      const emailNodeIssues = getNodeIssues('node-email-1', mockValidationResult);
      const err = getFieldError('RECIPIENT', emailNodeIssues);
      expect(err?.code).toBe('ERR_MISSING_RECIPIENT');
    });
  });

  describe('getNodeValidationCounts', () => {
    it('returns zero counts for node with no issues or null params', () => {
      expect(getNodeValidationCounts('node-nonexistent', mockValidationResult)).toEqual({
        errors: 0,
        warnings: 0,
      });
      expect(getNodeValidationCounts(null, null)).toEqual({ errors: 0, warnings: 0 });
    });

    it('counts errors and warnings correctly for a target node', () => {
      const counts = getNodeValidationCounts('node-email-1', mockValidationResult);
      expect(counts).toEqual({ errors: 1, warnings: 1 });
    });
  });

  describe('getTabErrorCounts', () => {
    it('calculates total errors and warnings across all graph nodes', () => {
      const totals = getTabErrorCounts(mockValidationResult);
      expect(totals).toEqual({ errors: 2, warnings: 1 });
    });

    it('handles null or empty validation results gracefully', () => {
      expect(getTabErrorCounts(null)).toEqual({ errors: 0, warnings: 0 });
      expect(getTabErrorCounts({ draft_id: 'draft-101', is_valid: true, issues: [] })).toEqual({ errors: 0, warnings: 0 });
    });
  });

  describe('formatIssueCode', () => {
    it('formats machine codes into human-readable capitalized strings', () => {
      expect(formatIssueCode('ERR_MISSING_FIELD')).toBe('Missing Field');
      expect(formatIssueCode('WARN_HIGH_LATENCY')).toBe('High Latency');
      expect(formatIssueCode('ERR_UNCONNECTED_BRANCH')).toBe('Unconnected Branch');
    });

    it('returns default label for empty code', () => {
      expect(formatIssueCode('')).toBe('Validation Error');
    });
  });
});
