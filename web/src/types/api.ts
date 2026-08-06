/**
 * Generated / Standard TypeScript API Contract Interfaces
 * Corresponding to OpenAPI 3.0.3 spec at api/openapi.yaml
 */

export interface StandardHeaders {
  'Request-ID'?: string;
  'Idempotency-Key'?: string;
  'If-Match'?: string;
  'X-Tenant-ID'?: string;
}

export interface ErrorDetail {
  field?: string;
  issue?: string;
}

export interface ErrorResponse {
  code: string;
  message: string;
  details?: ErrorDetail[];
  request_id: string;
}

export interface CatalogRecord {
  schema_version: '1.0';
  record_id: string;
  name: string;
  component_type: 'activity' | 'trigger' | 'action' | 'condition';
  version: string;
  description: string;
  schema_definition?: Record<string, unknown>;
  content_hash?: string;
  tags: string[];
  is_deprecated: boolean;
}

export interface GraphNodePosition {
  x?: number;
  y?: number;
}

export interface GraphNode {
  id: string;
  type: string;
  name: string;
  config?: Record<string, unknown>;
  position?: GraphNodePosition;
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
  condition?: string;
  sourceHandle?: string;
  targetHandle?: string;
  label?: string;
}

export interface GraphDraft {
  schema_version: '1.0';
  draft_id: string;
  tenant_id: string;
  name: string;
  description?: string;
  version: number;
  nodes: GraphNode[];
  edges: GraphEdge[];
  content_hash?: string;
  created_at?: string;
  updated_at?: string;
}

export interface ValidationIssue {
  schema_version: '1.0';
  issue_id: string;
  node_id: string;
  severity: 'error' | 'warning' | 'info';
  code: string;
  message: string;
  field_path: string;
}

export interface ValidationResult {
  draft_id: string;
  is_valid: boolean;
  issues: ValidationIssue[];
}

export interface SimulationResult {
  draft_id: string;
  success: boolean;
  execution_path: string[];
  outputs: Record<string, unknown>;
}

export interface DraftVersionItem {
  version: number;
  content_hash: string;
  updated_at: string;
}

export interface DraftVersionList {
  draft_id: string;
  versions: DraftVersionItem[];
}

export interface ExperimentVariant {
  variant_id: string;
  name: string;
  weight_basis_points: number;
  config?: Record<string, unknown>;
}

export interface ExperimentDefinition {
  schema_version: '1.0';
  experiment_id: string;
  name: string;
  description?: string;
  status: 'draft' | 'active' | 'paused' | 'completed';
  variants: ExperimentVariant[];
  target_audience?: string;
  content_hash?: string;
  created_at?: string;
}

export interface ReportMetric {
  metric_name: string;
  total_count: number;
  mean: number;
  weight_basis_points: number;
  confidence_interval_lower?: number;
  confidence_interval_upper?: number;
}

export interface AggregateReport {
  schema_version: '1.0';
  report_id: string;
  tenant_id: string;
  experiment_id: string;
  period_start: string;
  period_end: string;
  metrics: ReportMetric[];
  generated_at: string;
}

export interface StaticList {
  schema_version: '1.0';
  list_id: string;
  name: string;
  description?: string;
  item_count: number;
  data_classification: 'PII' | 'NonPII' | 'Sensitive';
  items: string[];
  records?: Record<string, unknown>[];
  csv_content?: string;
  content_hash?: string;
  created_at?: string;
  updated_at?: string;
  expires_at?: string;
}

export interface StaticListVersionItem {
  version_id: string;
  item_count: number;
  created_at: string;
}

export interface StaticListVersionList {
  list_id: string;
  versions: StaticListVersionItem[];
}

export interface TestRun {
  schema_version: '1.0';
  test_run_id: string;
  draft_id: string;
  ir_id: string;
  static_list_id?: string;
  status: 'pending' | 'passed' | 'failed';
  mock_inputs?: Record<string, unknown>;
  expected_outcomes?: Record<string, unknown>;
  actual_outcomes?: Record<string, unknown>;
  execution_time_ms: number;
  created_at: string;
}

export interface RunProjection {
  schema_version: '1.0';
  run_id: string;
  tenant_id: string;
  workflow_id: string;
  execution_mode?: string;
  status: 'running' | 'completed' | 'failed' | 'terminated';
  current_nodes: string[];
  variables?: Record<string, unknown>;
  started_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface TimelineEvent {
  event_id: string;
  event_type?: string;
  node_id: string;
  status: string;
  payload?: Record<string, unknown>;
  timestamp: string;
}

export interface RunTimeline {
  run_id: string;
  sub_run_id?: string;
  tenant_id?: string;
  workflow_id?: string;
  subject_id?: string;
  status?: string;
  execution_mode?: string;
  started_at?: string;
  updated_at?: string;
  completed_at?: string;
  current_nodes?: string[];
  timeline: TimelineEvent[];
}

export interface SubRunSummary {
  sub_run_id: string;
  subject_id: string;
  recipient: string;
  name: string;
  status: string;
  executed_branch: string;
  completed_at?: string;
}

export interface SubRunListResponse {
  run_id: string;
  total: number;
  page: number;
  limit: number;
  total_pages: number;
  sub_runs: SubRunSummary[];
}

export interface EventEnvelope {
  schema_version: '1.0';
  event_id: string;
  trace_id: string;
  event_type: string;
  source: string;
  subject?: string;
  timestamp: string;
  content_hash?: string;
  data_classification: 'PII' | 'NonPII' | 'Sensitive';
  data: Record<string, unknown>;
}

export interface NormalizedOutcome {
  schema_version: '1.0';
  outcome_id: string;
  run_id: string;
  event_name: string;
  value: number;
  unit: string;
  data_classification: 'PII' | 'NonPII' | 'Sensitive';
  timestamp: string;
  metadata?: Record<string, unknown>;
}

export interface ActionResult {
  schema_version: '1.0';
  action_id: string;
  activity_type: string;
  status: 'success' | 'failure' | 'retrying';
  output?: Record<string, unknown>;
  error_message?: string;
  execution_duration_ms: number;
  completed_at: string;
}

export interface AssignmentExposure {
  schema_version: '1.0';
  assignment_id: string;
  experiment_id: string;
  variant_id: string;
  subject_id: string;
  assigned_at: string;
  exposed_at: string;
  weight_basis_points: number;
  data_classification: 'PII' | 'NonPII' | 'Sensitive';
  context?: Record<string, unknown>;
}

export interface CompiledIRNode {
  id: string;
  type: string;
  activity_name: string;
  params?: Record<string, unknown>;
  timeout_seconds?: number;
  retry_policy?: {
    maximum_attempts?: number;
    initial_interval_seconds?: number;
  };
}

export interface CompiledIREdge {
  id: string;
  source_id: string;
  target_id: string;
  condition_expression?: string;
}

export interface CompiledIR {
  schema_version: '1.0';
  ir_id: string;
  draft_id: string;
  tenant_id: string;
  version: number;
  entry_node_id: string;
  nodes: CompiledIRNode[];
  edges: CompiledIREdge[];
  content_hash: string;
  compiled_at?: string;
}

export interface WorkflowInput {
  schema_version: '1.0';
  workflow_id: string;
  run_id: string;
  tenant_id: string;
  trigger_event_id: string;
  ir_id: string;
  data_classification: 'PII' | 'NonPII' | 'Sensitive';
  input_payload: Record<string, unknown>;
}

export interface EmitEventResponse {
  event_id: string;
  status: string;
  emitted_at: string;
}

export interface ProcessOutcomeResponse {
  outcome_id: string;
  status: string;
  processed_at: string;
}

export interface ActivationResponse {
  draft_id: string;
  status: string;
  activated_at: string;
}

export interface PauseResponse {
  draft_id: string;
  status: string;
  paused_at: string;
}
