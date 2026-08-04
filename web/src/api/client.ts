import type {
  CatalogRecord,
  GraphDraft,
  ValidationResult,
  SimulationResult,
  DraftVersionList,
  ExperimentDefinition,
  AggregateReport,
  StaticList,
  StaticListVersionList,
  TestRun,
  RunProjection,
  RunTimeline,
  EventEnvelope,
  ActionResult,
  EmitEventResponse,
  ActivationResponse,
  NormalizedOutcome,
  ProcessOutcomeResponse,
  PauseResponse,
  ErrorResponse,
  StandardHeaders,
} from '../types/api';

export class APIError extends Error {
  public code: string;
  public details?: Array<{ field?: string; issue?: string }>;
  public requestId: string;
  public status: number;

  constructor(status: number, errRes: ErrorResponse) {
    super(errRes.message || `API error (${status})`);
    this.name = 'APIError';
    this.status = status;
    this.code = errRes.code || 'UNKNOWN_ERROR';
    this.details = errRes.details;
    this.requestId = errRes.request_id || '';
  }
}

export interface ClientConfig {
  baseUrl?: string;
  fetchFn?: typeof fetch;
}

export class JourneyApiClient {
  private baseUrl: string;
  private fetchFn: typeof fetch;

  constructor(config?: ClientConfig) {
    this.baseUrl = config?.baseUrl || '/api/v1';
    this.fetchFn = config?.fetchFn || globalThis.fetch.bind(globalThis);
  }

  private buildHeaders(customHeaders?: StandardHeaders, initHeaders?: HeadersInit): Headers {
    const headers = new Headers(initHeaders);
    if (!headers.has('Content-Type')) {
      headers.set('Content-Type', 'application/json');
    }
    if (customHeaders?.['Request-ID']) {
      headers.set('Request-ID', customHeaders['Request-ID']);
    } else if (!headers.has('Request-ID')) {
      headers.set('Request-ID', `req-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`);
    }
    if (customHeaders?.['Idempotency-Key']) {
      headers.set('Idempotency-Key', customHeaders['Idempotency-Key']);
    }
    if (customHeaders?.['If-Match'] && customHeaders['If-Match'] !== 'undefined' && customHeaders['If-Match'] !== 'null') {
      headers.set('If-Match', customHeaders['If-Match']);
    }
    if (customHeaders?.['X-Tenant-ID']) {
      headers.set('X-Tenant-ID', customHeaders['X-Tenant-ID']);
    }
    return headers;
  }

  private async request<T>(path: string, options: RequestInit = {}, headers?: StandardHeaders): Promise<{ data: T; headers: Headers }> {
    let url = `${this.baseUrl}${path}`;
    if (!url.startsWith('http://') && !url.startsWith('https://') && typeof window === 'undefined') {
      url = `http://localhost${url.startsWith('/') ? '' : '/'}${url}`;
    }
    const reqHeaders = this.buildHeaders(headers, options.headers);
    const response = await this.fetchFn(url, { ...options, headers: reqHeaders });

    if (!response.ok) {
      let errPayload: ErrorResponse;
      try {
        errPayload = await response.json();
      } catch {
        errPayload = {
          code: 'HTTP_ERROR',
          message: response.statusText || `HTTP ${response.status}`,
          request_id: reqHeaders.get('Request-ID') || '',
        };
      }
      throw new APIError(response.status, errPayload);
    }

    if (response.status === 204) {
      return { data: {} as T, headers: response.headers };
    }

    const contentType = response.headers.get('content-type') || '';
    if (contentType.includes('text/csv')) {
      const text = await response.text();
      return { data: text as unknown as T, headers: response.headers };
    }

    const data = await response.json();
    return { data, headers: response.headers };
  }

  // Catalogs
  async getEventCatalog(headers?: StandardHeaders): Promise<CatalogRecord[]> {
    const res = await this.request<CatalogRecord[]>('/catalogs/events', { method: 'GET' }, headers);
    return res.data;
  }

  async getActionCatalog(headers?: StandardHeaders): Promise<CatalogRecord[]> {
    const res = await this.request<CatalogRecord[]>('/catalogs/actions', { method: 'GET' }, headers);
    return res.data;
  }

  async getAttributeCatalog(headers?: StandardHeaders): Promise<CatalogRecord[]> {
    const res = await this.request<CatalogRecord[]>('/catalogs/attributes', { method: 'GET' }, headers);
    return res.data;
  }

  async getParameterCatalog(headers?: StandardHeaders): Promise<CatalogRecord[]> {
    const res = await this.request<CatalogRecord[]>('/catalogs/parameters', { method: 'GET' }, headers);
    return res.data;
  }

  async getMetricCatalog(headers?: StandardHeaders): Promise<CatalogRecord[]> {
    const res = await this.request<CatalogRecord[]>('/catalogs/metrics', { method: 'GET' }, headers);
    return res.data;
  }

  async getJourneyTemplates(headers?: StandardHeaders): Promise<CatalogRecord[]> {
    const res = await this.request<CatalogRecord[]>('/catalogs/templates', { method: 'GET' }, headers);
    return res.data;
  }
  async getCatalog(category: string = 'events', headers?: StandardHeaders): Promise<CatalogRecord[]> {
    switch (category) {
      case 'events':
        return await this.getEventCatalog(headers);
      case 'actions':
        return await this.getActionCatalog(headers);
      case 'attributes':
        return await this.getAttributeCatalog(headers);
      case 'parameters':
        return await this.getParameterCatalog(headers);
      case 'metrics':
        return await this.getMetricCatalog(headers);
      case 'templates':
        return await this.getJourneyTemplates(headers);
      default:
        return await this.getEventCatalog(headers);
    }
  }

  // Journey Drafts
  async createJourneyDraft(draft: GraphDraft, headers?: StandardHeaders): Promise<{ draft: GraphDraft; etag: string | null }> {
    const res = await this.request<GraphDraft>('/journeys/drafts', {
      method: 'POST',
      body: JSON.stringify(draft),
    }, headers);
    return { draft: res.data, etag: res.headers.get('ETag') };
  }

  async listJourneyDrafts(headers?: StandardHeaders): Promise<GraphDraft[]> {
    const res = await this.request<GraphDraft[]>('/journeys/drafts', { method: 'GET' }, headers);
    return res.data;
  }

  async getJourneyDraft(draftId: string, headers?: StandardHeaders): Promise<{ draft: GraphDraft; etag: string | null }> {
    const res = await this.request<GraphDraft>(`/journeys/drafts/${draftId}`, { method: 'GET' }, headers);
    return { draft: res.data, etag: res.headers.get('ETag') };
  }
  async updateJourneyDraft(draftId: string, draft: GraphDraft, headers?: StandardHeaders): Promise<{ draft: GraphDraft; etag: string | null }> {
    const res = await this.request<GraphDraft>(`/journeys/drafts/${draftId}`, {
      method: 'PUT',
      body: JSON.stringify(draft),
    }, headers);
    return { draft: res.data, etag: res.headers.get('ETag') };
  }

  async validateJourneyDraft(draftId: string, headers?: StandardHeaders): Promise<ValidationResult> {
    const res = await this.request<ValidationResult>(`/journeys/drafts/${draftId}/validate`, { method: 'POST' }, headers);
    return res.data;
  }

  async simulateJourneyDraft(draftId: string, mockInputs?: Record<string, unknown>, headers?: StandardHeaders): Promise<SimulationResult> {
    const res = await this.request<SimulationResult>(`/journeys/drafts/${draftId}/simulate`, {
      method: 'POST',
      body: JSON.stringify({ mock_inputs: mockInputs }),
    }, headers);
    return res.data;
  }

  async listJourneyDraftVersions(draftId: string, headers?: StandardHeaders): Promise<DraftVersionList> {
    const res = await this.request<DraftVersionList>(`/journeys/drafts/${draftId}/versions`, { method: 'GET' }, headers);
    return res.data;
  }
  async listDraftVersions(draftId: string, headers?: StandardHeaders): Promise<DraftVersionList> {
    return this.listJourneyDraftVersions(draftId, headers);
  }

  async publishJourneyDraft(draftId: string, headers?: StandardHeaders): Promise<{ draft_id: string; status: string }> {
    const res = await this.request<{ draft_id: string; status: string }>(`/journeys/drafts/${draftId}/publish`, { method: 'POST' }, headers);
    return res.data;
  }

  async activateLocalJourney(draftId: string, headers?: StandardHeaders): Promise<ActivationResponse> {
    const res = await this.request<ActivationResponse>(`/journeys/drafts/${draftId}/activate-local`, { method: 'POST' }, headers);
    return res.data;
  }

  async pauseLocalJourney(draftId: string, headers?: StandardHeaders): Promise<PauseResponse> {
    const res = await this.request<PauseResponse>(`/journeys/drafts/${draftId}/pause-local`, { method: 'POST' }, headers);
    return res.data;
  }

  // Experiments
  async createExperiment(experiment: ExperimentDefinition, headers?: StandardHeaders): Promise<ExperimentDefinition> {
    const res = await this.request<ExperimentDefinition>('/experiments', {
      method: 'POST',
      body: JSON.stringify(experiment),
    }, headers);
    return res.data;
  }

  async listExperiments(headers?: StandardHeaders): Promise<ExperimentDefinition[]> {
    const res = await this.request<ExperimentDefinition[]>('/experiments', { method: 'GET' }, headers);
    return res.data;
  }

  async getExperimentVersion(experimentId: string, versionId: string, headers?: StandardHeaders): Promise<ExperimentDefinition> {
    const res = await this.request<ExperimentDefinition>(`/experiments/${experimentId}/versions/${versionId}`, { method: 'GET' }, headers);
    return res.data;
  }

  // Reports & Exports
  async getAggregateReport(params?: { experimentId?: string; tenantId?: string }, headers?: StandardHeaders): Promise<AggregateReport> {
    const query = new URLSearchParams();
    if (params?.experimentId) query.set('experiment_id', params.experimentId);
    if (params?.tenantId) query.set('tenant_id', params.tenantId);
    const qs = query.toString() ? `?${query.toString()}` : '';
    const res = await this.request<AggregateReport>(`/reports/aggregate${qs}`, { method: 'GET' }, headers);
    return res.data;
  }

  async exportCSV(reportId?: string, headers?: StandardHeaders): Promise<string> {
    const qs = reportId ? `?report_id=${encodeURIComponent(reportId)}` : '';
    const res = await this.request<string>(`/exports/csv${qs}`, { method: 'GET' }, headers);
    return res.data;
  }

  // Static Lists & Test Runs
  async uploadStaticList(staticList: StaticList, headers?: StandardHeaders): Promise<StaticList> {
    const res = await this.request<StaticList>('/static-lists/upload', {
      method: 'POST',
      body: JSON.stringify(staticList),
    }, headers);
    return res.data;
  }

  async listStaticListVersions(listId: string, headers?: StandardHeaders): Promise<StaticListVersionList> {
    const res = await this.request<StaticListVersionList>(`/static-lists/${listId}/versions`, { method: 'GET' }, headers);
    return res.data;
  }

  async startTestRun(testRun: Partial<TestRun>, headers?: StandardHeaders): Promise<TestRun> {
    const res = await this.request<TestRun>('/test-runs', {
      method: 'POST',
      body: JSON.stringify(testRun),
    }, headers);
    return res.data;
  }

  async getTestRunStatus(testRunId: string, headers?: StandardHeaders): Promise<TestRun> {
    const res = await this.request<TestRun>(`/test-runs/${testRunId}`, { method: 'GET' }, headers);
    return res.data;
  }

  async cancelTestRun(testRunId: string, headers?: StandardHeaders): Promise<TestRun> {
    const res = await this.request<TestRun>(`/test-runs/${testRunId}/cancel`, { method: 'POST' }, headers);
    return res.data;
  }

  // Journey Runs
  async listJourneyRuns(params?: { workflowId?: string; status?: string }, headers?: StandardHeaders): Promise<RunProjection[]> {
    const query = new URLSearchParams();
    if (params?.workflowId) query.set('workflow_id', params.workflowId);
    if (params?.status) query.set('status', params.status);
    const qs = query.toString() ? `?${query.toString()}` : '';
    const res = await this.request<RunProjection[]>(`/journeys/runs${qs}`, { method: 'GET' }, headers);
    return res.data;
  }

  async getJourneyRunTimeline(runId: string, headers?: StandardHeaders): Promise<RunTimeline> {
    const res = await this.request<RunTimeline>(`/journeys/runs/${runId}/timeline`, { method: 'GET' }, headers);
    return res.data;
  }
  async getJourneyRun(runId: string, headers?: StandardHeaders): Promise<RunProjection & { actions?: ActionResult[]; suppressions?: Record<string, unknown>[] }> {
    const res = await this.request<RunProjection & { actions?: ActionResult[]; suppressions?: Record<string, unknown>[] }>(`/journeys/runs/${runId}`, { method: 'GET' }, headers);
    return res.data;
  }

  // Kafka Test Event Emission & Outcome Callbacks
  async emitKafkaTestEvent(event: EventEnvelope, headers?: StandardHeaders): Promise<EmitEventResponse> {
    const res = await this.request<EmitEventResponse>('/events/emit', {
      method: 'POST',
      body: JSON.stringify(event),
    }, headers);
    return res.data;
  }

  async processOutcomeCallback(outcome: NormalizedOutcome, headers?: StandardHeaders): Promise<ProcessOutcomeResponse> {
    const res = await this.request<ProcessOutcomeResponse>('/callbacks/outcomes', {
      method: 'POST',
      body: JSON.stringify(outcome),
    }, headers);
    return res.data;
  }
}
