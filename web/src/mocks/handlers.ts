import { http, HttpResponse } from 'msw';
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
  NormalizedOutcome,
} from '../types/api';

const sampleCatalogEvents: CatalogRecord[] = [
  {
    schema_version: '1.0',
    record_id: 'cat-evt-001',
    name: 'user_signup',
    component_type: 'trigger',
    version: '1.0.0',
    description: 'Triggered when a new user signs up',
    tags: ['user', 'onboarding'],
    is_deprecated: false,
  },
];

const sampleCatalogActions: CatalogRecord[] = [
  {
    schema_version: '1.0',
    record_id: 'cat-act-001',
    name: 'send_email',
    component_type: 'action',
    version: '1.0.0',
    description: 'Send email to target user',
    tags: ['notification', 'email'],
    is_deprecated: false,
  },
];

const sampleCatalogAttributes: CatalogRecord[] = [
  {
    schema_version: '1.0',
    record_id: 'cat-attr-001',
    name: 'user_tier',
    component_type: 'condition',
    version: '1.0.0',
    description: 'User subscription tier attribute',
    tags: ['user', 'billing'],
    is_deprecated: false,
  },
];

const sampleCatalogParameters: CatalogRecord[] = [
  {
    schema_version: '1.0',
    record_id: 'cat-param-001',
    name: 'discount_code',
    component_type: 'activity',
    version: '1.0.0',
    description: 'Promo code parameter',
    tags: ['promo'],
    is_deprecated: false,
  },
];

const sampleCatalogMetrics: CatalogRecord[] = [
  {
    schema_version: '1.0',
    record_id: 'cat-met-001',
    name: 'conversion_rate',
    component_type: 'activity',
    version: '1.0.0',
    description: 'Conversion metric definition',
    tags: ['analytics'],
    is_deprecated: false,
  },
];

const sampleCatalogTemplates: CatalogRecord[] = [
  {
    schema_version: '1.0',
    record_id: 'cat-tmpl-001',
    name: 'welcome_series',
    component_type: 'trigger',
    version: '1.0.0',
    description: 'Standard 3-step onboarding email series',
    tags: ['template', 'onboarding'],
    is_deprecated: false,
  },
];

const sampleDraft: GraphDraft = {
  schema_version: '1.0',
  draft_id: 'draft-101',
  tenant_id: 'tenant-default',
  name: 'Welcome Journey Draft',
  description: 'Draft for welcome series flow',
  version: 1,
  nodes: [
    { id: 'node-1', type: 'EventStart', name: 'User Signup Event', position: { x: 100, y: 150 } },
    { id: 'node-2', type: 'Email', name: 'Send Welcome Email', position: { x: 450, y: 150 }, config: { template_id: 'welcome-email' } },
    { id: 'node-3', type: 'SMS', name: 'Send Verification SMS', position: { x: 800, y: 150 } },
    { id: 'node-4', type: 'Push', name: 'Send Push Alert', position: { x: 1150, y: 150 } },
    { id: 'node-5', type: 'InApp', name: 'Show Banner Notification', position: { x: 100, y: 350 } },
    { id: 'node-6', type: 'Webhook', name: 'Sync CRM Webhook', position: { x: 450, y: 350 } },
    { id: 'node-7', type: 'Delay', name: 'Wait 24 Hours', position: { x: 800, y: 350 }, config: { duration: '24h' } },
    { id: 'node-8', type: 'Condition', name: 'plan == "Pro"', position: { x: 1150, y: 350 } },
    { id: 'node-9', type: 'WaitForEvent', name: 'Wait For Click', position: { x: 100, y: 550 } },
    { id: 'node-10', type: 'Experiment', name: 'A/B Test Email', position: { x: 450, y: 550 } },
    { id: 'node-11', type: 'Exit', name: 'Journey Complete', position: { x: 800, y: 550 } },
  ],
  edges: [
    { id: 'edge-1', source: 'node-1', target: 'node-2' },
  ],
  content_hash: 'hash-draft-101-v1',
  created_at: '2026-07-27T10:00:00Z',
  updated_at: '2026-07-27T10:00:00Z',
};

const draftMap = new Map<string, GraphDraft>([['draft-101', sampleDraft]]);

const sampleExperiment: ExperimentDefinition = {
  schema_version: '1.0',
  experiment_id: 'exp-201',
  name: 'Onboarding Subject Line A/B Test',
  description: 'Testing two subject lines for onboarding email',
  status: 'active',
  variants: [
    { variant_id: 'var-a', name: 'Subject A', weight_basis_points: 5000 },
    { variant_id: 'var-b', name: 'Subject B', weight_basis_points: 5000 },
  ],
  target_audience: 'new_users',
  content_hash: 'hash-exp-201',
  created_at: '2026-07-27T10:00:00Z',
};

const sampleAggregateReport: AggregateReport = {
  schema_version: '1.0',
  report_id: 'rep-301',
  tenant_id: 'tenant-default',
  experiment_id: 'exp-201',
  period_start: '2026-07-01T00:00:00Z',
  period_end: '2026-07-27T00:00:00Z',
  metrics: [
    { metric_name: 'open_rate', total_count: 1000, mean: 0.45, weight_basis_points: 5000, confidence_interval_lower: 0.42, confidence_interval_upper: 0.48 },
    { metric_name: 'click_rate', total_count: 1000, mean: 0.18, weight_basis_points: 5000, confidence_interval_lower: 0.15, confidence_interval_upper: 0.21 },
  ],
  generated_at: '2026-07-27T12:00:00Z',
};

const sampleStaticLists: StaticList[] = [
  {
    schema_version: '1.0',
    list_id: 'list-vip-users',
    name: 'VIP Customers Segment',
    description: 'High LTV accounts eligible for priority loyalty rewards',
    item_count: 5,
    data_classification: 'PII',
    records: [
      { id: 'usr_101', email: 'alexander.smith@example.com', name: 'Alexander Smith', tier: 'VIP Platinum', ltv: '$12,450', status: 'Active' },
      { id: 'usr_102', email: 'sophia.martinez@enterprise.org', name: 'Sophia Martinez', tier: 'VIP Gold', ltv: '$8,920', status: 'Active' },
      { id: 'usr_103', email: 'liam.johnson@corporate.co', name: 'Liam Johnson', tier: 'VIP Platinum', ltv: '$15,100', status: 'Active' },
      { id: 'usr_104', email: 'emma.williams@techsolutions.io', name: 'Emma Williams', tier: 'VIP Gold', ltv: '$9,400', status: 'Active' },
      { id: 'usr_105', email: 'noah.brown@cloudops.net', name: 'Noah Brown', tier: 'VIP Silver', ltv: '$6,800', status: 'Active' },
    ],
    items: [
      'alexander.smith@example.com',
      'sophia.martinez@enterprise.org',
      'liam.johnson@corporate.co',
      'emma.williams@techsolutions.io',
      'noah.brown@cloudops.net',
    ],
    content_hash: 'hash-vip-v1',
    created_at: '2026-08-01T10:00:00Z',
    updated_at: '2026-08-01T10:00:00Z',
  },
  {
    schema_version: '1.0',
    list_id: 'list-beta-testers',
    name: 'Beta Feature Opt-Ins',
    description: 'User cohort participating in new canvas UI testing',
    item_count: 4,
    data_classification: 'NonPII',
    records: [
      { id: 'usr_beta_89201', group: 'Cohort A', channel: 'In-App', feature_flag: 'canvas_v2_enabled', opt_in_date: '2026-08-01' },
      { id: 'usr_beta_44102', group: 'Cohort B', channel: 'Email', feature_flag: 'canvas_v2_enabled', opt_in_date: '2026-08-01' },
      { id: 'usr_beta_11093', group: 'Cohort A', channel: 'In-App', feature_flag: 'canvas_v2_enabled', opt_in_date: '2026-08-02' },
      { id: 'usr_beta_77344', group: 'Cohort C', channel: 'Direct', feature_flag: 'canvas_v2_enabled', opt_in_date: '2026-08-02' },
    ],
    items: [
      'usr_beta_89201',
      'usr_beta_44102',
      'usr_beta_11093',
      'usr_beta_77344',
    ],
    content_hash: 'hash-beta-v1',
    created_at: '2026-08-02T14:30:00Z',
    updated_at: '2026-08-02T14:30:00Z',
  },
  {
    schema_version: '1.0',
    list_id: 'list-churn-risk',
    name: 'At-Risk Churn Cohort',
    description: 'Accounts with zero login activity in last 30 days',
    item_count: 5,
    data_classification: 'Sensitive',
    records: [
      { id: 'usr_churn_01', email: 'david.miller@acme.com', company: 'Acme Corp', inactivity_days: 34, risk_level: 'High', arr: '$45,000' },
      { id: 'usr_churn_02', email: 'olivia.davis@globex.org', company: 'Globex Inc', inactivity_days: 41, risk_level: 'Critical', arr: '$62,000' },
      { id: 'usr_churn_03', email: 'james.wilson@initech.co', company: 'Initech', inactivity_days: 30, risk_level: 'Medium', arr: '$28,000' },
      { id: 'usr_churn_04', email: 'isabella.taylor@umbrella.corp', company: 'Umbrella Corp', inactivity_days: 52, risk_level: 'Critical', arr: '$95,000' },
      { id: 'usr_churn_05', email: 'benjamin.anderson@stark.io', company: 'Stark Industries', inactivity_days: 38, risk_level: 'High', arr: '$72,000' },
    ],
    items: [
      'david.miller@acme.com',
      'olivia.davis@globex.org',
      'james.wilson@initech.co',
      'isabella.taylor@umbrella.corp',
      'benjamin.anderson@stark.io',
    ],
    content_hash: 'hash-churn-v1',
    created_at: '2026-08-03T09:15:00Z',
    updated_at: '2026-08-03T09:15:00Z',
  },
];

const sampleStaticList: StaticList = sampleStaticLists[0];

const sampleTestRun: TestRun & Record<string, unknown> = {
  schema_version: '1.0',
  test_run_id: 'tr-501',
  draft_id: 'draft-welcome-v1',
  ir_id: 'ir-901',
  status: 'passed',
  execution_time_ms: 145,
  created_at: '2026-07-27T14:00:00Z',
  totalMembers: 50,
  completedMembers: 46,
  suppressedMembersCount: 4,
  members: [
    {
      memberId: 'MBR-001',
      maskedRecipient: 'u***1@example.com',
      status: 'completed',
      visitSteps: [
        { stepIndex: 1, nodeId: 'node-start', nodeName: 'User Signup Trigger', nodeType: 'trigger', status: 'passed', timestamp: '14:00:00.010' },
        { stepIndex: 2, nodeId: 'node-cond-1', nodeName: 'Is Premium User?', nodeType: 'condition', status: 'passed', timestamp: '14:00:00.025' },
        { stepIndex: 3, nodeId: 'node-action-1', nodeName: 'Send Welcome Email', nodeType: 'action', status: 'passed', timestamp: '14:00:00.080' },
      ],
    },
    {
      memberId: 'MBR-002',
      maskedRecipient: 'j***e@example.com',
      status: 'completed',
      visitSteps: [
        { stepIndex: 1, nodeId: 'node-start', nodeName: 'User Signup Trigger', nodeType: 'trigger', status: 'passed', timestamp: '14:00:00.012' },
        { stepIndex: 2, nodeId: 'node-cond-1', nodeName: 'Is Premium User?', nodeType: 'condition', status: 'passed', timestamp: '14:00:00.028' },
        { stepIndex: 3, nodeId: 'node-action-2', nodeName: 'Send Standard Onboarding Email', nodeType: 'action', status: 'passed', timestamp: '14:00:00.090' },
      ],
    },
    {
      memberId: 'MBR-003',
      maskedRecipient: 's***p@example.com',
      status: 'suppressed',
      visitSteps: [
        { stepIndex: 1, nodeId: 'node-start', nodeName: 'User Signup Trigger', nodeType: 'trigger', status: 'passed', timestamp: '14:00:00.015' },
        { stepIndex: 2, nodeId: 'node-cond-1', nodeName: 'Is Premium User?', nodeType: 'condition', status: 'passed', timestamp: '14:00:00.030' },
        { stepIndex: 3, nodeId: 'node-suppress-check', nodeName: 'Check Frequency Cap & Opt-out', nodeType: 'condition', status: 'failed', timestamp: '14:00:00.045' },
      ],
    },
  ],
  branch_decisions: [
    { memberId: 'MBR-001', nodeId: 'node-cond-1', conditionExpression: 'attributes.user_tier == "premium"', evaluatedResult: true, chosenBranch: 'True Branch (node-action-1)' },
    { memberId: 'MBR-002', nodeId: 'node-cond-1', conditionExpression: 'attributes.user_tier == "premium"', evaluatedResult: false, chosenBranch: 'False Branch (node-action-2)' },
    { memberId: 'MBR-003', nodeId: 'node-cond-1', conditionExpression: 'attributes.user_tier == "premium"', evaluatedResult: false, chosenBranch: 'False Branch (node-suppress-check)' },
  ],
  suppressions: [
    { memberId: 'MBR-003', maskedRecipient: 's***p@example.com', reason: 'frequency_cap', suppressedAtNode: 'node-suppress-check', timestamp: '14:00:00.045' },
    { memberId: 'MBR-014', maskedRecipient: 'o***t@domain.org', reason: 'global_opt_out', suppressedAtNode: 'node-start', timestamp: '14:00:00.018' },
    { memberId: 'MBR-029', maskedRecipient: 'b***d@invalid.com', reason: 'unverified_recipient', suppressedAtNode: 'node-start', timestamp: '14:00:00.020' },
    { memberId: 'MBR-041', maskedRecipient: 'c***t@domain.com', reason: 'control_group', suppressedAtNode: 'node-cond-1', timestamp: '14:00:00.032' },
  ],
  fake_deliveries: [
    { deliveryId: 'del-001', memberId: 'MBR-001', maskedRecipient: 'u***1@example.com', channel: 'email', provider: 'mock-sendgrid', status: 'delivered', responsePayload: '{"statusCode": 202, "messageId": "sg-mock-001"}', timestamp: '14:00:00.082' },
    { deliveryId: 'del-002', memberId: 'MBR-002', maskedRecipient: 'j***e@example.com', channel: 'email', provider: 'mock-sendgrid', status: 'delivered', responsePayload: '{"statusCode": 202, "messageId": "sg-mock-002"}', timestamp: '14:00:00.092' },
    { deliveryId: 'del-003', memberId: 'MBR-005', maskedRecipient: '+1***89', channel: 'sms', provider: 'mock-twilio', status: 'delivered', responsePayload: '{"sid": "SMmock003", "status": "sent"}', timestamp: '14:00:00.110' },
    { deliveryId: 'del-004', memberId: 'MBR-012', maskedRecipient: 'e***r@test.org', channel: 'email', provider: 'mock-sendgrid', status: 'bounced', responsePayload: '{"statusCode": 550, "error": "Simulated hard bounce"}', timestamp: '14:00:00.125' },
  ],
};

const sampleRunProjections: RunProjection[] = [
  {
    schema_version: '1.0',
    run_id: 'run-601',
    tenant_id: 'tenant-default',
    workflow_id: 'wf-draft-101',
    status: 'completed',
    current_nodes: ['node-2'],
    started_at: '2026-07-27T11:00:00Z',
    updated_at: '2026-07-27T11:00:05Z',
    completed_at: '2026-07-27T11:00:05Z',
  },
];

export const handlers = [
  // Catalogs
  http.get('*/api/v1/catalogs/events', ({ request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(sampleCatalogEvents, { headers: { 'Request-ID': reqId } });
  }),

  http.get('*/api/v1/catalogs/actions', ({ request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(sampleCatalogActions, { headers: { 'Request-ID': reqId } });
  }),

  http.get('*/api/v1/catalogs/attributes', ({ request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(sampleCatalogAttributes, { headers: { 'Request-ID': reqId } });
  }),

  http.get('*/api/v1/catalogs/parameters', ({ request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(sampleCatalogParameters, { headers: { 'Request-ID': reqId } });
  }),

  http.get('*/api/v1/catalogs/metrics', ({ request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(sampleCatalogMetrics, { headers: { 'Request-ID': reqId } });
  }),

  http.get('*/api/v1/catalogs/templates', ({ request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(sampleCatalogTemplates, { headers: { 'Request-ID': reqId } });
  }),
  http.post('*/api/v1/catalogs/:type', async ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const body = (await request.json()) as Partial<CatalogRecord>;
    const type = params.type as string;
    const recordId = body.record_id || `cat-${type}-${Date.now().toString(36)}`;
    const newRecord: CatalogRecord = {
      schema_version: '1.0',
      record_id: recordId,
      name: body.name || 'New Component',
      component_type: (body.component_type || type) as CatalogRecord['component_type'],
      version: body.version || '1.0.0',
      description: body.description || '',
      schema_definition: body.schema_definition || {},
      contentHash: body.contentHash || 'hash-mock',
      tags: body.tags || [type],
      is_deprecated: body.is_deprecated || false,
    };
    return HttpResponse.json(newRecord, { status: 201, headers: { 'Request-ID': reqId } });
  }),

  // Journey Drafts
  http.get('*/api/v1/journeys/drafts', ({ request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const drafts = [
      {
        schema_version: '1.0',
        draft_id: 'draft-101',
        tenant_id: 'tenant-default',
        name: 'Welcome Journey Draft',
        description: 'Onboarding customer welcome workflow',
        version: 1,
        nodes: [{ id: 'node-start', type: 'trigger', name: 'User Signup Event' }],
        edges: [],
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
      {
        schema_version: '1.0',
        draft_id: 'draft-102',
        tenant_id: 'tenant-default',
        name: 'KYC Verification Nudge',
        description: 'Fintech compliance verification nudge series',
        version: 2,
        nodes: [{ id: 'node-kyc-start', type: 'trigger', name: 'KYC Pending Event' }],
        edges: [],
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    ];
    return HttpResponse.json(drafts, { headers: { 'Request-ID': reqId } });
  }),

  http.post('*/api/v1/journeys/drafts', async ({ request }) => {
    const body = (await request.json()) as GraphDraft;
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const etag = `etag-v${body.version || 1}-${Date.now()}`;
    const saved = { ...body, content_hash: etag };
    draftMap.set(saved.draft_id, saved);
    return HttpResponse.json(saved, {
      status: 201,
      headers: { ETag: etag, 'Request-ID': reqId },
    });
  }),

  http.get('*/api/v1/journeys/drafts/:draftId', ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const draftId = String(params.draftId);
    const existing = draftMap.get(draftId);
    const draft = existing || { ...sampleDraft, draft_id: draftId };
    return HttpResponse.json(draft, {
      headers: { ETag: draft.content_hash || '"1.0.0-hash"', 'Request-ID': reqId },
    });
  }),

  http.put('*/api/v1/journeys/drafts/:draftId', async ({ params, request }) => {
    const ifMatch = request.headers.get('If-Match');
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const draftId = String(params.draftId);
    const existing = draftMap.get(draftId);

    if (ifMatch && existing?.content_hash && ifMatch !== existing.content_hash) {
      return HttpResponse.json(
        { code: 'PRECONDITION_FAILED', message: 'ETag mismatch / concurrent modification conflict', request_id: reqId },
        { status: 409, headers: { 'Request-ID': reqId } }
      );
    }

    if (ifMatch && ifMatch.includes('etag-hash-server-remote')) {
      return HttpResponse.json(
        { code: 'PRECONDITION_FAILED', message: 'ETag mismatch', request_id: reqId },
        { status: 409, headers: { 'Request-ID': reqId } }
      );
    }

    const body = (await request.json()) as GraphDraft;
    const currentVer = existing ? existing.version : (body.version || 1);
    const nextVersion = currentVer + 1;
    const newETag = `etag-v${nextVersion}-${Date.now()}`;
    const updated: GraphDraft = {
      ...body,
      draft_id: draftId,
      version: nextVersion,
      content_hash: newETag,
    };
    draftMap.set(draftId, updated);
    return HttpResponse.json(updated, {
      headers: { ETag: newETag, 'Request-ID': reqId },
    });
  }),

  http.post('*/api/v1/journeys/drafts/:draftId/validate', ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const result: ValidationResult = {
      draft_id: String(params.draftId),
      is_valid: true,
      issues: [],
    };
    return HttpResponse.json(result, { headers: { 'Request-ID': reqId } });
  }),

  http.post('*/api/v1/journeys/drafts/:draftId/simulate', ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const result: SimulationResult = {
      draft_id: String(params.draftId),
      success: true,
      execution_path: ['node-1', 'node-2'],
      outputs: { email_sent: true },
    };
    return HttpResponse.json(result, { headers: { 'Request-ID': reqId } });
  }),

  http.get('*/api/v1/journeys/drafts/:draftId/versions', ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const versionList: DraftVersionList = {
      draft_id: String(params.draftId),
      versions: [
        { version: 1, content_hash: 'hash-v1', updated_at: '2026-07-27T10:00:00Z' },
        { version: 2, content_hash: 'hash-v2', updated_at: '2026-07-27T11:00:00Z' },
      ],
    };
    return HttpResponse.json(versionList, { headers: { 'Request-ID': reqId } });
  }),

  http.post('*/api/v1/journeys/drafts/:draftId/publish', ({ request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(
      {
        code: 'PUBLISH_DISABLED',
        message: 'Publish capability is disabled in local mode',
        request_id: reqId,
      },
      { status: 403, headers: { 'Request-ID': reqId } }
    );
  }),

  http.post('*/api/v1/journeys/drafts/:draftId/activate-local', ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(
      { draft_id: String(params.draftId), status: 'active', activated_at: new Date().toISOString() },
      { headers: { 'Request-ID': reqId } }
    );
  }),

  http.post('*/api/v1/journeys/drafts/:draftId/pause-local', ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(
      { draft_id: String(params.draftId), status: 'paused', paused_at: new Date().toISOString() },
      { headers: { 'Request-ID': reqId } }
    );
  }),

  // Experiments
  http.post('*/api/v1/experiments', async ({ request }) => {
    const body = (await request.json()) as ExperimentDefinition;
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(body, {
      status: 201,
      headers: { ETag: '"exp-hash-v1"', 'Request-ID': reqId },
    });
  }),

  http.get('*/api/v1/experiments', ({ request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json([sampleExperiment], { headers: { 'Request-ID': reqId } });
  }),

  http.get('*/api/v1/experiments/:experimentId/versions/:versionId', ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const exp = { ...sampleExperiment, experiment_id: String(params.experimentId) };
    return HttpResponse.json(exp, { headers: { 'Request-ID': reqId } });
  }),

  // Reports & Exports
  http.get('*/api/v1/reports/aggregate', ({ request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(sampleAggregateReport, { headers: { 'Request-ID': reqId } });
  }),

  http.get('*/api/v1/exports/csv', ({ request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const csvContent = 'metric_name,total_count,mean,weight_basis_points\nopen_rate,1000,0.45,5000\nclick_rate,1000,0.18,5000\n';
    return new HttpResponse(csvContent, {
      status: 200,
      headers: {
        'Content-Type': 'text/csv',
        'Content-Disposition': 'attachment; filename=report_export.csv',
        'Request-ID': reqId,
      },
    });
  }),

  // Static Lists & Test Runs
  http.get('*/api/v1/static-lists', ({ request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(sampleStaticLists, { headers: { 'Request-ID': reqId } });
  }),

  http.get('*/api/v1/static-lists/:listId', ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const found = sampleStaticLists.find((l) => l.list_id === params.listId) || {
      ...sampleStaticList,
      list_id: String(params.listId),
    };
    return HttpResponse.json(found, { headers: { 'Request-ID': reqId } });
  }),

  http.post('*/api/v1/static-lists/upload', async ({ request }) => {
    const body = (await request.json()) as Partial<StaticList>;
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const list: StaticList = { ...sampleStaticList, ...body };
    return HttpResponse.json(list, {
      status: 201,
      headers: { ETag: '"list-hash-v1"', 'Request-ID': reqId },
    });
  }),

  http.get('*/api/v1/static-lists/:listId/versions', ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const versionList: StaticListVersionList = {
      list_id: String(params.listId),
      versions: [
        { version_id: 'v1', item_count: 2, created_at: '2026-07-27T10:00:00Z' },
      ],
    };
    return HttpResponse.json(versionList, { headers: { 'Request-ID': reqId } });
  }),

  http.post('*/api/v1/test-runs', async ({ request }) => {
    const body = (await request.json()) as TestRun;
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(body, {
      status: 201,
      headers: { 'Request-ID': reqId },
    });
  }),

  http.get('*/api/v1/test-runs/:testRunId', ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const tr = { ...sampleTestRun, test_run_id: String(params.testRunId) };
    return HttpResponse.json(tr, { headers: { 'Request-ID': reqId } });
  }),

  http.post('*/api/v1/test-runs/:testRunId/cancel', ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const cancelled = { ...sampleTestRun, test_run_id: String(params.testRunId), status: 'failed' as const };
    return HttpResponse.json(cancelled, { headers: { 'Request-ID': reqId } });
  }),

  // Journey Runs
  http.get('*/api/v1/journeys/runs', ({ request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(sampleRunProjections, { headers: { 'Request-ID': reqId } });
  }),

  http.get('*/api/v1/journeys/runs/:runId/timeline', ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    const timeline: RunTimeline = {
      run_id: String(params.runId),
      tenant_id: 'tenant-default',
      workflow_id: 'wf-welcome-series',
      status: 'completed',
      timeline: [
        { event_id: 'evt-1', node_id: 'node-1', status: 'completed', timestamp: '2026-07-27T11:00:01Z' },
        { event_id: 'evt-2', node_id: 'node-2', status: 'completed', timestamp: '2026-07-27T11:00:05Z' },
      ],
    };
    return HttpResponse.json(timeline, { headers: { 'Request-ID': reqId } });
  }),

  http.get('*/api/v1/journeys/runs/:runId/sub-runs', ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(
      {
        run_id: String(params.runId),
        total: 1,
        page: 1,
        limit: 10,
        total_pages: 1,
        sub_runs: [
          {
            sub_run_id: `${params.runId}-row-1`,
            subject_id: 'usr_001',
            recipient: 'contact@temporal.io',
            name: 'Audience Contact 1',
            status: 'completed',
            executed_branch: 'email',
          },
        ],
      },
      { headers: { 'Request-ID': reqId } }
    );
  }),

  http.get('*/api/v1/journeys/runs/:runId', ({ params, request }) => {
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(
      {
        run_id: String(params.runId),
        tenant_id: 'tenant-default',
        workflow_id: 'wf-welcome-series',
        execution_mode: 'test',
        status: 'completed',
        current_nodes: ['node-2'],
        actions: [],
        suppressions: [],
      },
      { headers: { 'Request-ID': reqId } }
    );
  }),
  // Kafka Test Event Emission & Outcome Callbacks
  http.post('*/api/v1/events/emit', async ({ request }) => {
    const body = (await request.json()) as EventEnvelope;
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(
      { event_id: body.event_id || 'evt-emitted', status: 'emitted', emitted_at: new Date().toISOString() },
      { status: 202, headers: { 'Request-ID': reqId } }
    );
  }),

  http.post('*/api/v1/callbacks/outcomes', async ({ request }) => {
    const body = (await request.json()) as NormalizedOutcome;
    const reqId = request.headers.get('Request-ID') || 'req-mock';
    return HttpResponse.json(
      { outcome_id: body.outcome_id || 'out-processed', status: 'processed', processed_at: new Date().toISOString() },
      { status: 200, headers: { 'Request-ID': reqId } }
    );
  }),
];
