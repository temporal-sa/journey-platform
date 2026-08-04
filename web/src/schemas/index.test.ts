import { describe, it, expect } from 'vitest';
import {
  catalogRecordSchema,
  graphDraftSchema,
  validationResultSchema,
  experimentDefinitionSchema,
  eventEnvelopeSchema,
  validateBoundary,
} from './index';

describe('Zod Domain Schemas', () => {
  it('validates CatalogRecord successfully', () => {
    const validRecord = {
      schema_version: '1.0',
      record_id: 'cat-001',
      name: 'User Signup',
      component_type: 'trigger',
      version: '1.0.0',
      description: 'Triggered when user registers',
      tags: ['user', 'auth'],
      is_deprecated: false,
    };

    const parsed = validateBoundary(catalogRecordSchema, validRecord);
    expect(parsed.record_id).toBe('cat-001');

    const invalidRecord = { ...validRecord, component_type: 'invalid-type' };
    expect(() => validateBoundary(catalogRecordSchema, invalidRecord)).toThrow();
  });

  it('validates GraphDraft with nodes and edges', () => {
    const validDraft = {
      schema_version: '1.0',
      draft_id: 'draft-101',
      tenant_id: 'tenant-1',
      name: 'Main Onboarding Flow',
      version: 1,
      nodes: [
        { id: 'n1', type: 'trigger', name: 'Start', position: { x: 10, y: 20 } },
        { id: 'n2', type: 'action', name: 'Send Email' },
      ],
      edges: [{ id: 'e1', source: 'n1', target: 'n2' }],
    };

    const parsed = validateBoundary(graphDraftSchema, validDraft);
    expect(parsed.nodes).toHaveLength(2);
    expect(parsed.edges[0].source).toBe('n1');

    const invalidDraft = { ...validDraft, schema_version: '2.0' };
    expect(() => validateBoundary(graphDraftSchema, invalidDraft)).toThrow();
  });

  it('validates ValidationResult and issues', () => {
    const result = {
      draft_id: 'draft-101',
      is_valid: false,
      issues: [
        {
          schema_version: '1.0',
          issue_id: 'iss-1',
          node_id: 'n2',
          severity: 'error',
          code: 'MISSING_PARAM',
          message: 'Template ID is required',
          field_path: 'nodes[1].config.template_id',
        },
      ],
    };

    const parsed = validateBoundary(validationResultSchema, result);
    expect(parsed.is_valid).toBe(false);
    expect(parsed.issues[0].severity).toBe('error');
  });

  it('validates ExperimentDefinition and EventEnvelope', () => {
    const exp = {
      schema_version: '1.0',
      experiment_id: 'exp-001',
      name: 'Onboarding Subject Line A/B Test',
      status: 'active',
      variants: [
        { variant_id: 'v1', name: 'Control', weight_basis_points: 5000 },
        { variant_id: 'v2', name: 'Treatment', weight_basis_points: 5000 },
      ],
    };
    expect(validateBoundary(experimentDefinitionSchema, exp).status).toBe('active');

    const evt = {
      schema_version: '1.0',
      event_id: 'evt-1001',
      trace_id: 'trace-999',
      event_type: 'user.registered',
      source: 'auth-service',
      timestamp: '2026-07-27T12:00:00Z',
      data_classification: 'PII',
      data: { user_id: 'usr-1' },
    };
    expect(validateBoundary(eventEnvelopeSchema, evt).data_classification).toBe('PII');
  });
});
