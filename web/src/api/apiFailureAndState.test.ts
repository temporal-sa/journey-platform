import { describe, it, expect, vi, beforeEach } from 'vitest';
import { JourneyApiClient, APIError } from './client';
import type { GraphDraft } from '../types/api';

describe('JourneyApiClient - API State Transitions & Failure State Testing', () => {
  let client: JourneyApiClient;
  let mockFetch: ReturnType<typeof vi.fn>;

  const sampleDraft: GraphDraft = {
    schema_version: '1.0',
    draft_id: 'draft-101',
    tenant_id: 'tenant-default',
    name: 'Welcome Journey',
    version: 2,
    nodes: [],
    edges: [],
    content_hash: 'hash-v2-abc',
  };

  beforeEach(() => {
    mockFetch = vi.fn();
    client = new JourneyApiClient({
      baseUrl: '/api/v1',
      fetchFn: mockFetch as unknown as typeof fetch,
    });
  });

  describe('Headers & ID Injection', () => {
    it('injects Content-Type, Request-ID, Idempotency-Key, and If-Match headers', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'Content-Type': 'application/json', ETag: '"hash-v2-abc"' }),
        json: async () => sampleDraft,
      });

      await client.getJourneyDraft('draft-101', {
        'Request-ID': 'req-custom-123',
        'Idempotency-Key': 'idempotency-xyz-789',
        'If-Match': '"hash-v1-prev"',
      });

      expect(mockFetch).toHaveBeenCalledTimes(1);
      const [, init] = mockFetch.mock.calls[0];
      const headers = init.headers as Headers;

      expect(headers.get('Content-Type')).toBe('application/json');
      expect(headers.get('Request-ID')).toBe('req-custom-123');
      expect(headers.get('Idempotency-Key')).toBe('idempotency-xyz-789');
      expect(headers.get('If-Match')).toBe('"hash-v1-prev"');
    });

    it('generates random Request-ID header if not provided', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'Content-Type': 'application/json' }),
        json: async () => [],
      });

      await client.getEventCatalog();

      const [, init] = mockFetch.mock.calls[0];
      const headers = init.headers as Headers;
      expect(headers.get('Request-ID')).toMatch(/^req-\d+-[a-z0-9]+$/);
    });
  });

  describe('Failure State: Disconnected API', () => {
    it('handles network failure (fetch throws TypeError / NetworkError)', async () => {
      mockFetch.mockRejectedValueOnce(new TypeError('Failed to fetch (NetworkError)'));

      await expect(client.getEventCatalog()).rejects.toThrow(TypeError);
    });
  });

  describe('Failure State: Stale Revision (409 Conflict)', () => {
    it('parses HTTP 409 Conflict error on draft update with stale ETag / revision', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 409,
        headers: new Headers({ 'Content-Type': 'application/json' }),
        json: async () => ({
          code: 'ERR_REVISION_CONFLICT',
          message: 'Draft revision version mismatch: server revision is v3 but client sent v2.',
          request_id: 'req-err-409',
          details: [{ field: 'version', issue: 'STALE_REVISION' }],
        }),
      });

      try {
        await client.updateJourneyDraft('draft-101', sampleDraft, { 'If-Match': '"hash-v1-stale"' });
        expect.unreachable('Should have thrown APIError');
      } catch (err) {
        expect(err).toBeInstanceOf(APIError);
        const apiErr = err as APIError;
        expect(apiErr.status).toBe(409);
        expect(apiErr.code).toBe('ERR_REVISION_CONFLICT');
        expect(apiErr.message).toMatch(/Draft revision version mismatch/i);
        expect(apiErr.details).toEqual([{ field: 'version', issue: 'STALE_REVISION' }]);
        expect(apiErr.requestId).toBe('req-err-409');
      }
    });
  });

  describe('Failure State: Failed Simulation', () => {
    it('parses HTTP 500 error on failed simulation execution', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        headers: new Headers({ 'Content-Type': 'application/json' }),
        json: async () => ({
          code: 'ERR_SIMULATION_FAILED',
          message: 'Simulation failed: Unhandled exception in Condition node (missing user attribute)',
          request_id: 'req-sim-fail-500',
        }),
      });

      try {
        await client.simulateJourneyDraft('draft-101', { user_id: 'usr-invalid' });
        expect.unreachable('Should have thrown APIError');
      } catch (err) {
        expect(err).toBeInstanceOf(APIError);
        const apiErr = err as APIError;
        expect(apiErr.status).toBe(500);
        expect(apiErr.code).toBe('ERR_SIMULATION_FAILED');
        expect(apiErr.message).toMatch(/Simulation failed/i);
      }
    });
  });

  describe('Failure State: Failed Test Run', () => {
    it('parses HTTP 400 error on failed test run execution launch', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        headers: new Headers({ 'Content-Type': 'application/json' }),
        json: async () => ({
          code: 'ERR_TEST_RUN_INVALID',
          message: 'Test run failed: Specified fixture pack contains invalid CSV format',
          request_id: 'req-test-fail-400',
        }),
      });

      try {
        await client.startTestRun({
          schema_version: '1.0',
          test_run_id: 'tr-99',
          draft_id: 'draft-101',
          ir_id: 'ir-101',
          status: 'pending',
          execution_time_ms: 0,
          created_at: new Date().toISOString(),
        });
        expect.unreachable('Should have thrown APIError');
      } catch (err) {
        expect(err).toBeInstanceOf(APIError);
        const apiErr = err as APIError;
        expect(apiErr.status).toBe(400);
        expect(apiErr.code).toBe('ERR_TEST_RUN_INVALID');
        expect(apiErr.message).toMatch(/Test run failed/i);
      }
    });
  });

  describe('Special HTTP Content Responses', () => {
    it('handles HTTP 204 No Content response gracefully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 204,
        headers: new Headers(),
      });

      const res = await client.pauseLocalJourney('draft-101');
      expect(res).toEqual({});
    });

    it('handles text/csv export response correctly', async () => {
      const csvData = 'variant_key,assigned,exposed,conversion\ncontrol,5000,4900,490\nvariant_a,5000,4950,643';
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'Content-Type': 'text/csv' }),
        text: async () => csvData,
      });

      const result = await client.exportCSV('rep-301');
      expect(result).toBe(csvData);
    });
  });
});
