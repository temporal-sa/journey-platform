import { z } from 'zod';

/**
 * Zod boundary validation schemas matching OpenAPI domain contracts.
 */

export const catalogRecordSchema = z.object({
  schema_version: z.literal('1.0'),
  record_id: z.string(),
  name: z.string(),
  component_type: z.enum(['activity', 'trigger', 'action', 'condition']),
  version: z.string(),
  description: z.string(),
  schema_definition: z.record(z.unknown()).optional(),
  content_hash: z.string().optional(),
  tags: z.array(z.string()),
  is_deprecated: z.boolean(),
});
export type CatalogRecordSchema = z.infer<typeof catalogRecordSchema>;

export const graphNodePositionSchema = z.object({
  x: z.number().optional(),
  y: z.number().optional(),
});
export type GraphNodePositionSchema = z.infer<typeof graphNodePositionSchema>;

export const graphNodeSchema = z.object({
  id: z.string(),
  type: z.string(),
  name: z.string(),
  config: z.record(z.unknown()).optional(),
  position: graphNodePositionSchema.optional(),
});
export type GraphNodeSchema = z.infer<typeof graphNodeSchema>;

export const graphEdgeSchema = z.object({
  id: z.string(),
  source: z.string(),
  target: z.string(),
  condition: z.string().optional(),
});
export type GraphEdgeSchema = z.infer<typeof graphEdgeSchema>;

export const graphDraftSchema = z.object({
  schema_version: z.literal('1.0'),
  draft_id: z.string(),
  tenant_id: z.string(),
  name: z.string(),
  description: z.string().optional(),
  version: z.number(),
  nodes: z.array(graphNodeSchema),
  edges: z.array(graphEdgeSchema),
  content_hash: z.string().optional(),
  created_at: z.string().optional(),
  updated_at: z.string().optional(),
});
export type GraphDraftSchema = z.infer<typeof graphDraftSchema>;

export const validationIssueSchema = z.object({
  schema_version: z.literal('1.0'),
  issue_id: z.string(),
  node_id: z.string(),
  severity: z.enum(['error', 'warning', 'info']),
  code: z.string(),
  message: z.string(),
  field_path: z.string(),
});
export type ValidationIssueSchema = z.infer<typeof validationIssueSchema>;

export const validationResultSchema = z.object({
  draft_id: z.string(),
  is_valid: z.boolean(),
  issues: z.array(validationIssueSchema),
});
export type ValidationResultSchema = z.infer<typeof validationResultSchema>;

export const simulationResultSchema = z.object({
  draft_id: z.string(),
  success: z.boolean(),
  execution_path: z.array(z.string()),
  outputs: z.record(z.unknown()),
});
export type SimulationResultSchema = z.infer<typeof simulationResultSchema>;

export const draftVersionItemSchema = z.object({
  version: z.number(),
  content_hash: z.string(),
  updated_at: z.string(),
});
export type DraftVersionItemSchema = z.infer<typeof draftVersionItemSchema>;

export const draftVersionListSchema = z.object({
  draft_id: z.string(),
  versions: z.array(draftVersionItemSchema),
});
export type DraftVersionListSchema = z.infer<typeof draftVersionListSchema>;

export const experimentVariantSchema = z.object({
  variant_id: z.string(),
  name: z.string(),
  weight_basis_points: z.number(),
  config: z.record(z.unknown()).optional(),
});
export type ExperimentVariantSchema = z.infer<typeof experimentVariantSchema>;

export const experimentDefinitionSchema = z.object({
  schema_version: z.literal('1.0'),
  experiment_id: z.string(),
  name: z.string(),
  description: z.string().optional(),
  status: z.enum(['draft', 'active', 'paused', 'completed']),
  variants: z.array(experimentVariantSchema),
  target_audience: z.string().optional(),
  content_hash: z.string().optional(),
  created_at: z.string().optional(),
});
export type ExperimentDefinitionSchema = z.infer<typeof experimentDefinitionSchema>;

export const reportMetricSchema = z.object({
  metric_name: z.string(),
  total_count: z.number(),
  mean: z.number(),
  weight_basis_points: z.number(),
  confidence_interval_lower: z.number().optional(),
  confidence_interval_upper: z.number().optional(),
});
export type ReportMetricSchema = z.infer<typeof reportMetricSchema>;

export const aggregateReportSchema = z.object({
  schema_version: z.literal('1.0'),
  report_id: z.string(),
  tenant_id: z.string(),
  experiment_id: z.string(),
  period_start: z.string(),
  period_end: z.string(),
  metrics: z.array(reportMetricSchema),
  generated_at: z.string(),
});
export type AggregateReportSchema = z.infer<typeof aggregateReportSchema>;

export const staticListSchema = z.object({
  schema_version: z.literal('1.0'),
  list_id: z.string(),
  name: z.string(),
  description: z.string().optional(),
  item_count: z.number(),
  data_classification: z.enum(['PII', 'NonPII', 'Sensitive']),
  items: z.array(z.string()),
  content_hash: z.string().optional(),
  created_at: z.string().optional(),
  updated_at: z.string().optional(),
});
export type StaticListSchema = z.infer<typeof staticListSchema>;

export const staticListVersionItemSchema = z.object({
  version_id: z.string(),
  item_count: z.number(),
  created_at: z.string(),
});
export type StaticListVersionItemSchema = z.infer<typeof staticListVersionItemSchema>;

export const staticListVersionListSchema = z.object({
  list_id: z.string(),
  versions: z.array(staticListVersionItemSchema),
});
export type StaticListVersionListSchema = z.infer<typeof staticListVersionListSchema>;

export const testRunSchema = z.object({
  schema_version: z.literal('1.0'),
  test_run_id: z.string(),
  draft_id: z.string(),
  ir_id: z.string(),
  status: z.enum(['pending', 'passed', 'failed']),
  mock_inputs: z.record(z.unknown()).optional(),
  expected_outcomes: z.record(z.unknown()).optional(),
  actual_outcomes: z.record(z.unknown()).optional(),
  execution_time_ms: z.number(),
  created_at: z.string(),
});
export type TestRunSchema = z.infer<typeof testRunSchema>;

export const runProjectionSchema = z.object({
  schema_version: z.literal('1.0'),
  run_id: z.string(),
  tenant_id: z.string(),
  workflow_id: z.string(),
  status: z.enum(['running', 'completed', 'failed', 'terminated']),
  current_nodes: z.array(z.string()),
  variables: z.record(z.unknown()).optional(),
  started_at: z.string(),
  updated_at: z.string(),
  completed_at: z.string().optional(),
});
export type RunProjectionSchema = z.infer<typeof runProjectionSchema>;

export const timelineEventSchema = z.object({
  event_id: z.string(),
  node_id: z.string(),
  status: z.string(),
  payload: z.record(z.unknown()).optional(),
  timestamp: z.string(),
});
export type TimelineEventSchema = z.infer<typeof timelineEventSchema>;

export const runTimelineSchema = z.object({
  run_id: z.string(),
  timeline: z.array(timelineEventSchema),
});
export type RunTimelineSchema = z.infer<typeof runTimelineSchema>;

export const eventEnvelopeSchema = z.object({
  schema_version: z.literal('1.0'),
  event_id: z.string(),
  trace_id: z.string(),
  event_type: z.string(),
  source: z.string(),
  subject: z.string().optional(),
  timestamp: z.string(),
  content_hash: z.string().optional(),
  data_classification: z.enum(['PII', 'NonPII', 'Sensitive']),
  data: z.record(z.unknown()),
});
export type EventEnvelopeSchema = z.infer<typeof eventEnvelopeSchema>;

export const normalizedOutcomeSchema = z.object({
  schema_version: z.literal('1.0'),
  outcome_id: z.string(),
  run_id: z.string(),
  event_name: z.string(),
  value: z.number(),
  unit: z.string(),
  data_classification: z.enum(['PII', 'NonPII', 'Sensitive']),
  timestamp: z.string(),
  metadata: z.record(z.unknown()).optional(),
});
export type NormalizedOutcomeSchema = z.infer<typeof normalizedOutcomeSchema>;

export const compiledIRNodeSchema = z.object({
  id: z.string(),
  type: z.string(),
  activity_name: z.string(),
  params: z.record(z.unknown()).optional(),
  timeout_seconds: z.number().optional(),
  retry_policy: z.object({
    maximum_attempts: z.number().optional(),
    initial_interval_seconds: z.number().optional(),
  }).optional(),
});
export type CompiledIRNodeSchema = z.infer<typeof compiledIRNodeSchema>;

export const compiledIREdgeSchema = z.object({
  id: z.string(),
  source_id: z.string(),
  target_id: z.string(),
  condition_expression: z.string().optional(),
});
export type CompiledIREdgeSchema = z.infer<typeof compiledIREdgeSchema>;

export const compiledIRSchema = z.object({
  schema_version: z.literal('1.0'),
  ir_id: z.string(),
  draft_id: z.string(),
  tenant_id: z.string(),
  version: z.number(),
  entry_node_id: z.string(),
  nodes: z.array(compiledIRNodeSchema),
  edges: z.array(compiledIREdgeSchema),
  content_hash: z.string(),
  compiled_at: z.string().optional(),
});
export type CompiledIRSchema = z.infer<typeof compiledIRSchema>;

export const workflowInputSchema = z.object({
  schema_version: z.literal('1.0'),
  workflow_id: z.string(),
  run_id: z.string(),
  tenant_id: z.string(),
  trigger_event_id: z.string(),
  ir_id: z.string(),
  data_classification: z.enum(['PII', 'NonPII', 'Sensitive']),
  input_payload: z.record(z.unknown()),
});
export type WorkflowInputSchema = z.infer<typeof workflowInputSchema>;

export const errorDetailSchema = z.object({
  field: z.string().optional(),
  issue: z.string().optional(),
});
export type ErrorDetailSchema = z.infer<typeof errorDetailSchema>;

export const errorResponseSchema = z.object({
  code: z.string(),
  message: z.string(),
  details: z.array(errorDetailSchema).optional(),
  request_id: z.string(),
});
export type ErrorResponseSchema = z.infer<typeof errorResponseSchema>;

/**
 * Validation helper function that validates data against a Zod schema
 */
export function validateBoundary<T>(schema: z.ZodSchema<T>, data: unknown): T {
  return schema.parse(data);
}
