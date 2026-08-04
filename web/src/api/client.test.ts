import { describe, it, expect, vi } from 'vitest';
import { JourneyApiClient, APIError } from './client';
import type { GraphDraft, ExperimentDefinition, StaticList, TestRun, EventEnvelope, NormalizedOutcome } from '../types/api';

describe('JourneyApiClient', () => {
  const mockFetch = vi.fn();
  const client = new JourneyApiClient({ fetchFn: mockFetch });

  const createMockResponse = (data: unknown, status = 200, headers: Record<string, string> = {}) => {
    const isCsv = headers['content-type']?.includes('text/csv');
    return Promise.resolve(new Response(isCsv ? (data as string) : JSON.stringify(data), {
      status,
      statusText: status === 200 ? 'OK' : 'Error',
      headers: new Headers({
        'content-type': isCsv ? 'text/csv' : 'application/json',
        'Request-ID': 'req-test-123',
        ...headers,
      }),
    }));
  };

  it('fetches catalogs successfully', async () => {
    mockFetch.mockImplementation(() => createMockResponse([{ record_id: '1', name: 'user_signup' }]));

    const events = await client.getEventCatalog({ 'Request-ID': 'req-1' });
    expect(events).toHaveLength(1);
    expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/api/v1/catalogs/events'), expect.objectContaining({
      method: 'GET',
    }));

    await client.getActionCatalog();
    await client.getAttributeCatalog();
    await client.getParameterCatalog();
    await client.getMetricCatalog();
    await client.getJourneyTemplates();
  });

  it('creates and manages journey drafts', async () => {
    const draft: GraphDraft = {
      schema_version: '1.0',
      draft_id: 'draft-test-1',
      tenant_id: 'tenant-test',
      name: 'Test Journey',
      version: 1,
      nodes: [{ id: 'n1', type: 'trigger', name: 'Start' }],
      edges: [],
    };

    mockFetch.mockImplementation(() => createMockResponse(draft, 201, { ETag: '"1.0.0-hash"' }));
    const created = await client.createJourneyDraft(draft, { 'Idempotency-Key': 'idem-1' });
    expect(created.draft.draft_id).toBe('draft-test-1');
    expect(created.etag).toBe('"1.0.0-hash"');

    mockFetch.mockImplementation(() => createMockResponse(draft, 200, { ETag: '"1.0.0-hash"' }));
    const fetched = await client.getJourneyDraft('draft-test-1');
    expect(fetched.draft.draft_id).toBe('draft-test-1');

    mockFetch.mockImplementation(() => createMockResponse({ draft_id: 'draft-test-1', is_valid: true, issues: [] }));
    const validation = await client.validateJourneyDraft('draft-test-1');
    expect(validation.is_valid).toBe(true);

    mockFetch.mockImplementation(() => createMockResponse({ draft_id: 'draft-test-1', success: true, execution_path: [], outputs: {} }));
    const simulation = await client.simulateJourneyDraft('draft-test-1');
    expect(simulation.success).toBe(true);

    mockFetch.mockImplementation(() => createMockResponse({ draft_id: 'draft-test-1', versions: [{ version: 1, content_hash: 'h1', updated_at: '2026' }] }));
    const versions = await client.listJourneyDraftVersions('draft-test-1');
    expect(versions.versions).toHaveLength(1);

    mockFetch.mockImplementation(() => createMockResponse({ draft_id: 'draft-test-1', status: 'active', activated_at: '2026' }));
    const activated = await client.activateLocalJourney('draft-test-1');
    expect(activated.status).toBe('active');

    mockFetch.mockImplementation(() => createMockResponse({ draft_id: 'draft-test-1', status: 'paused', paused_at: '2026' }));
    const paused = await client.pauseLocalJourney('draft-test-1');
    expect(paused.status).toBe('paused');
  });

  it('handles publish-disabled for journey draft', async () => {
    mockFetch.mockImplementation(() => createMockResponse({
      code: 'PUBLISH_DISABLED',
      message: 'Publish capability is disabled in local mode',
      request_id: 'req-err-1',
    }, 403));

    await expect(client.publishJourneyDraft('draft-test-1')).rejects.toThrow(APIError);
  });

  it('creates and lists experiments', async () => {
    const exp: ExperimentDefinition = {
      schema_version: '1.0',
      experiment_id: 'exp-test-1',
      name: 'Test Exp',
      status: 'draft',
      variants: [{ variant_id: 'v1', name: 'V1', weight_basis_points: 10000 }],
    };

    mockFetch.mockImplementation(() => createMockResponse(exp, 201));
    const created = await client.createExperiment(exp);
    expect(created.experiment_id).toBe('exp-test-1');

    mockFetch.mockImplementation(() => createMockResponse([exp]));
    const list = await client.listExperiments();
    expect(list).toHaveLength(1);

    mockFetch.mockImplementation(() => createMockResponse(exp));
    const version = await client.getExperimentVersion('exp-test-1', 'v1');
    expect(version.experiment_id).toBe('exp-test-1');
  });

  it('fetches aggregate report and CSV export', async () => {
    mockFetch.mockImplementation(() => createMockResponse({ report_id: 'rep-1' }));
    const report = await client.getAggregateReport({ experimentId: 'exp-201' });
    expect(report.report_id).toBe('rep-1');

    mockFetch.mockImplementation(() => createMockResponse('metric_name,total_count\nopen_rate,100', 200, { 'content-type': 'text/csv' }));
    const csv = await client.exportCSV('rep-1');
    expect(csv).toContain('metric_name');
  });

  it('handles static lists and test runs', async () => {
    const sl: StaticList = {
      schema_version: '1.0',
      list_id: 'list-1',
      name: 'Test List',
      item_count: 1,
      data_classification: 'NonPII',
      items: ['item1'],
    };

    mockFetch.mockImplementation(() => createMockResponse(sl, 201));
    const uploaded = await client.uploadStaticList(sl);
    expect(uploaded.list_id).toBe('list-1');

    mockFetch.mockImplementation(() => createMockResponse({ list_id: 'list-1', versions: [] }));
    const versions = await client.listStaticListVersions('list-1');
    expect(versions.list_id).toBe('list-1');

    const tr: TestRun = {
      schema_version: '1.0',
      test_run_id: 'tr-1',
      draft_id: 'draft-1',
      ir_id: 'ir-1',
      status: 'pending',
      execution_time_ms: 10,
      created_at: '2026',
    };

    mockFetch.mockImplementation(() => createMockResponse(tr, 201));
    const started = await client.startTestRun(tr);
    expect(started.test_run_id).toBe('tr-1');

    mockFetch.mockImplementation(() => createMockResponse(tr));
    const status = await client.getTestRunStatus('tr-1');
    expect(status.test_run_id).toBe('tr-1');

    mockFetch.mockImplementation(() => createMockResponse({ ...tr, status: 'failed' }));
    const cancelled = await client.cancelTestRun('tr-1');
    expect(cancelled.status).toBe('failed');
  });

  it('fetches journey runs and timeline', async () => {
    mockFetch.mockImplementation(() => createMockResponse([{ run_id: 'run-1' }]));
    const runs = await client.listJourneyRuns();
    expect(runs).toHaveLength(1);

    mockFetch.mockImplementation(() => createMockResponse({ run_id: 'run-1', timeline: [] }));
    const timeline = await client.getJourneyRunTimeline('run-1');
    expect(timeline.timeline).toHaveLength(0);
  });

  it('emits events and processes outcome callbacks', async () => {
    const event: EventEnvelope = {
      schema_version: '1.0',
      event_id: 'evt-100',
      trace_id: 'trace-100',
      event_type: 'user_signup',
      source: 'web',
      timestamp: '2026',
      data_classification: 'NonPII',
      data: { user_id: 'u123' },
    };

    mockFetch.mockImplementation(() => createMockResponse({ event_id: 'evt-100', status: 'emitted', emitted_at: '2026' }, 202));
    const emitRes = await client.emitKafkaTestEvent(event);
    expect(emitRes.status).toBe('emitted');

    const outcome: NormalizedOutcome = {
      schema_version: '1.0',
      outcome_id: 'out-100',
      run_id: 'run-601',
      event_name: 'email_opened',
      value: 1.0,
      unit: 'count',
      data_classification: 'NonPII',
      timestamp: '2026',
    };

    mockFetch.mockImplementation(() => createMockResponse({ outcome_id: 'out-100', status: 'processed', processed_at: '2026' }));
    const outcomeRes = await client.processOutcomeCallback(outcome);
    expect(outcomeRes.status).toBe('processed');
  });
});
