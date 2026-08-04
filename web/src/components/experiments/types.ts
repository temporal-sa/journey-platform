export type RandomizationUnit = 'user_id' | 'device_id' | 'account_id' | 'session_id' | 'subject_id';

export interface VariantFormState {
  key: string;
  name: string;
  weight_basis_points: number;
  is_control: boolean;
  config?: Record<string, unknown>;
}

export interface MetricDefinitionState {
  key: string;
  name: string;
  event_type: string;
  type: 'conversion' | 'sum' | 'mean';
}

export interface ExperimentFormState {
  schema_version: '1.0';
  tenant_id: string;
  experiment_id: string;
  version: number;
  name: string;
  description?: string;
  status: 'draft' | 'active' | 'paused' | 'completed';
  salt: string;
  randomization_unit: RandomizationUnit;
  variants: VariantFormState[];
  metrics: MetricDefinitionState[];
  attribution_window_seconds: number;
  target_audience?: string;
}

export interface VariantAggregateMetric {
  variant_key: string;
  variant_name: string;
  is_control: boolean;
  assigned: number;
  exposed: number;
  attempted: number;
  accepted: number;
  delivered: number;
  unique_open: number;
  unique_click: number;
  conversion: number;
  conversion_rate: number; // ratio 0..1 (e.g. 0.15 = 15%)
  lift_pct: number; // e.g. +12.5%
  confidence_interval_95: [number, number]; // [lower %, upper %]
}

export interface AggregateReportData {
  schema_version: '1.0';
  report_id: string;
  tenant_id: string;
  experiment_id: string;
  experiment_name: string;
  period_start: string;
  period_end: string;
  generated_at: string;
  data_freshness_seconds: number;
  is_filtered: boolean;
  srm_status: 'PASSED' | 'FAILED' | 'PENDING';
  srm_p_value?: number;
  srm_details?: string;
  variant_metrics: VariantAggregateMetric[];
}
