import React, { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { JourneyApiClient } from '../../api/client';
import type { CatalogRecord, ExperimentDefinition } from '../../types/api';
import { SafeQueryClientProvider } from '../SafeQueryClientProvider';
import type {
  ExperimentFormState,
  VariantFormState,
  MetricDefinitionState,
  RandomizationUnit,
} from './types';

const apiClient = new JourneyApiClient();
export interface ExperimentInspectorProps {
  initialExperiment?: Partial<ExperimentFormState>;
  onSave?: (experiment: ExperimentFormState) => void;
  onCancel?: () => void;
  readOnly?: boolean;
}


const DEFAULT_INITIAL_EXPERIMENT: ExperimentFormState = {
  schema_version: '1.0',
  tenant_id: 'tenant-default',
  experiment_id: 'exp-101',
  version: 1,
  name: 'New A/B Experiment',
  description: '',
  status: 'draft',
  salt: 'salt-exp-101',
  randomization_unit: 'user_id',
  variants: [
    { key: 'control', name: 'Control (Original)', weight_basis_points: 5000, is_control: true },
    { key: 'variant_a', name: 'Variant A (Treatment)', weight_basis_points: 5000, is_control: false },
  ],
  metrics: [
    { key: 'conversion', name: 'Conversion Rate', event_type: 'user_converted', type: 'conversion' },
  ],
  attribution_window_seconds: 604800, // 7 days
  target_audience: 'all_users',
};

const ExperimentInspectorInner: React.FC<ExperimentInspectorProps> = ({
  initialExperiment,
  onSave,
  onCancel,
  readOnly = false,
}) => {
  const { data: _metricCatalog = [] } = useQuery<CatalogRecord[]>({
    queryKey: ['metricCatalog'],
    queryFn: () => apiClient.getMetricCatalog(),
  });

  const { data: attributeCatalog = [] } = useQuery<CatalogRecord[]>({
    queryKey: ['attributeCatalog'],
    queryFn: () => apiClient.getAttributeCatalog(),
  });

  const randomizationUnits = useMemo(() => {
    if (attributeCatalog.length > 0) {
      return attributeCatalog.map((a) => (a.name || a.record_id) as RandomizationUnit);
    }
    return ['user_id', 'device_id', 'account_id', 'session_id', 'subject_id'] as RandomizationUnit[];
  }, [attributeCatalog]);

  const [formState, setFormState] = useState<ExperimentFormState>(() => ({
    ...DEFAULT_INITIAL_EXPERIMENT,
    ...initialExperiment,
    variants: initialExperiment?.variants?.length
      ? initialExperiment.variants
      : DEFAULT_INITIAL_EXPERIMENT.variants,
    metrics: initialExperiment?.metrics?.length
      ? initialExperiment.metrics
      : DEFAULT_INITIAL_EXPERIMENT.metrics,
  }));
  const [attributionUnit, setAttributionUnit] = useState<'days' | 'hours' | 'seconds'>('days');

  // Allocation calculation in basis points (1% = 100 bps, 100% = 10,000 bps)
  const totalBasisPoints = useMemo(() => {
    return formState.variants.reduce((sum, v) => sum + (Number(v.weight_basis_points) || 0), 0);
  }, [formState.variants]);

  const totalPercentage = (totalBasisPoints / 100).toFixed(2).replace(/\.00$/, '');

  // Continuous Validation
  const validationErrors = useMemo(() => {
    const errors: string[] = [];

    if (!formState.experiment_id.trim()) {
      errors.push('Experiment ID is required.');
    }
    if (!formState.name.trim()) {
      errors.push('Experiment Name is required.');
    }
    if (!formState.salt.trim()) {
      errors.push('Experiment Salt is required.');
    }

    // 2-to-5 variant editor validation
    if (formState.variants.length < 2 || formState.variants.length > 5) {
      errors.push(`Experiment must have between 2 and 5 variants (current: ${formState.variants.length}).`);
    }

    // Key uniqueness and control check
    const keys = formState.variants.map((v) => v.key.trim().toLowerCase());
    const uniqueKeys = new Set(keys);
    if (uniqueKeys.size !== keys.length) {
      errors.push('Variant keys must be unique.');
    }

    const emptyKeys = formState.variants.some((v) => !v.key.trim());
    if (emptyKeys) {
      errors.push('All variant keys must be non-empty.');
    }

    const controlCount = formState.variants.filter((v) => v.is_control).length;
    if (controlCount !== 1) {
      errors.push(`Exactly one control variant must be designated (current: ${controlCount}).`);
    }

    // Weight allocation check (Must equal 10,000 basis points / 100%)
    if (totalBasisPoints !== 10000) {
      errors.push(
        `Total allocation weight must equal 100% (10,000 basis points). Current total: ${totalPercentage}% (${totalBasisPoints} bps).`
      );
    }

    // Metric validation
    if (formState.metrics.length === 0) {
      errors.push('At least one metric definition is required.');
    } else {
      const invalidMetrics = formState.metrics.some((m) => !m.key.trim() || !m.event_type.trim());
      if (invalidMetrics) {
        errors.push('All metrics must have a valid key and event type.');
      }
    }

    // Attribution window check
    if (formState.attribution_window_seconds <= 0) {
      errors.push('Attribution window must be greater than zero.');
    }

    return errors;
  }, [formState, totalBasisPoints, totalPercentage]);

  const isValid = validationErrors.length === 0;

  // Handlers for Top-level Fields
  const handleFieldChange = (field: keyof ExperimentFormState, value: unknown) => {
    if (readOnly) return;
    setFormState((prev) => ({ ...prev, [field]: value }));
  };

  // Handlers for Variants
  const handleVariantChange = (
    index: number,
    field: keyof VariantFormState,
    value: string | number | boolean
  ) => {
    if (readOnly) return;
    setFormState((prev) => {
      const newVariants = [...prev.variants];
      const targetVariant = { ...newVariants[index] };

      if (field === 'weight_basis_points') {
        // Percentage input to basis points conversion
        const percentageVal = Number(value);
        targetVariant.weight_basis_points = Math.round(percentageVal * 100);
      } else if (field === 'is_control' && Boolean(value) === true) {
        // Only 1 variant can be control - unset control for all others
        newVariants.forEach((v, i) => {
          newVariants[i] = { ...v, is_control: i === index };
        });
        return { ...prev, variants: newVariants };
      } else {
        (targetVariant as Record<string, unknown>)[field] = value;
      }

      newVariants[index] = targetVariant;
      return { ...prev, variants: newVariants };
    });
  };

  const handleAddVariant = () => {
    if (readOnly || formState.variants.length >= 5) return;
    const nextChar = String.fromCharCode(97 + formState.variants.length); // 'c', 'd', 'e'
    const newKey = `variant_${nextChar}`;
    const newName = `Variant ${nextChar.toUpperCase()}`;

    setFormState((prev) => ({
      ...prev,
      variants: [
        ...prev.variants,
        {
          key: newKey,
          name: newName,
          weight_basis_points: 0,
          is_control: false,
        },
      ],
    }));
  };

  const handleRemoveVariant = (index: number) => {
    if (readOnly || formState.variants.length <= 2) return;
    setFormState((prev) => {
      const removedVariant = prev.variants[index];
      const newVariants = prev.variants.filter((_, i) => i !== index);

      // If removed variant was control, make the first remaining variant control
      if (removedVariant.is_control && newVariants.length > 0) {
        newVariants[0] = { ...newVariants[0], is_control: true };
      }

      return { ...prev, variants: newVariants };
    });
  };

  // Handlers for Metrics
  const handleMetricChange = (
    index: number,
    field: keyof MetricDefinitionState,
    value: string
  ) => {
    if (readOnly) return;
    setFormState((prev) => {
      const newMetrics = [...prev.metrics];
      newMetrics[index] = { ...newMetrics[index], [field]: value };
      return { ...prev, metrics: newMetrics };
    });
  };

  const handleAddMetric = () => {
    if (readOnly) return;
    const count = formState.metrics.length + 1;
    setFormState((prev) => ({
      ...prev,
      metrics: [
        ...prev.metrics,
        {
          key: `metric_${count}`,
          name: `Metric ${count}`,
          event_type: `event_${count}`,
          type: 'conversion',
        },
      ],
    }));
  };

  const handleRemoveMetric = (index: number) => {
    if (readOnly || formState.metrics.length <= 1) return;
    setFormState((prev) => ({
      ...prev,
      metrics: prev.metrics.filter((_, i) => i !== index),
    }));
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isValid || readOnly) return;

    if (onSave) {
      onSave(formState);
    }

    try {
      const experimentPayload: ExperimentDefinition = {
        schema_version: '1.0',
        experiment_id: formState.experiment_id,
        name: formState.name,
        description: formState.description,
        status: formState.status as 'draft' | 'active' | 'paused' | 'completed',
        target_audience: formState.target_audience,
        variants: formState.variants.map((v) => ({
          variant_id: v.key,
          key: v.key,
          name: v.name,
          weight_basis_points: v.weight_basis_points,
          is_control: v.is_control,
        })),
      };
      await apiClient.createExperiment(experimentPayload);
    } catch (err) {
      console.warn('API sync notice:', err);
    }
  };
  // Attribution window display helper
  const getAttributionValue = () => {
    const secs = formState.attribution_window_seconds;
    if (attributionUnit === 'days') return Math.round(secs / 86400);
    if (attributionUnit === 'hours') return Math.round(secs / 3600);
    return secs;
  };

  const handleAttributionChange = (val: number, unit: 'days' | 'hours' | 'seconds') => {
    let secs = val;
    if (unit === 'days') secs = val * 86400;
    if (unit === 'hours') secs = val * 3600;
    handleFieldChange('attribution_window_seconds', secs);
  };

  return (
    <div
      data-testid="experiment-inspector"
      style={{
        padding: '24px',
        backgroundColor: '#ffffff',
        borderRadius: '12px',
        border: '1px solid #e2e8f0',
        boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.05)',
        maxWidth: '800px',
        fontFamily: 'system-ui, -apple-system, sans-serif',
        color: '#0f172a',
      }}
    >
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <div>
          <h2 style={{ fontSize: '20px', fontWeight: 700, margin: 0, color: '#0f172a' }}>
            🧪 Experiment Authoring & Configuration
          </h2>
          <p style={{ fontSize: '13px', color: '#64748b', margin: '4px 0 0 0' }}>
            Define variant allocations, randomization unit, metrics, and attribution rules.
          </p>
        </div>
        <span
          style={{
            padding: '4px 10px',
            borderRadius: '9999px',
            fontSize: '12px',
            fontWeight: 600,
            backgroundColor: formState.status === 'active' ? '#dcfce7' : '#f1f5f9',
            color: formState.status === 'active' ? '#15803d' : '#475569',
            textTransform: 'uppercase',
          }}
        >
          {formState.status}
        </span>
      </div>

      <form onSubmit={handleSave}>
        {/* Experiment Basic Details */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', marginBottom: '20px' }}>
          <div>
            <label htmlFor="exp-id" style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#334155', marginBottom: '4px' }}>
              Experiment ID
            </label>
            <input
              id="exp-id"
              type="text"
              value={formState.experiment_id}
              onChange={(e) => handleFieldChange('experiment_id', e.target.value)}
              disabled={readOnly}
              style={{
                width: '100%',
                padding: '8px 12px',
                borderRadius: '6px',
                border: '1px solid #cbd5e1',
                fontSize: '14px',
                boxSizing: 'border-box',
              }}
            />
          </div>

          <div>
            <label htmlFor="exp-name" style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#334155', marginBottom: '4px' }}>
              Experiment Name
            </label>
            <input
              id="exp-name"
              type="text"
              value={formState.name}
              onChange={(e) => handleFieldChange('name', e.target.value)}
              disabled={readOnly}
              style={{
                width: '100%',
                padding: '8px 12px',
                borderRadius: '6px',
                border: '1px solid #cbd5e1',
                fontSize: '14px',
                boxSizing: 'border-box',
              }}
            />
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '16px', marginBottom: '20px' }}>
          {/* Randomization Unit Selector */}
          <div>
            <label htmlFor="randomization-unit-select" style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#334155', marginBottom: '4px' }}>
              Randomization Unit
            </label>
            <select
              id="randomization-unit-select"
              value={formState.randomization_unit}
              onChange={(e) => handleFieldChange('randomization_unit', e.target.value as RandomizationUnit)}
              disabled={readOnly}
              style={{
                width: '100%',
                padding: '8px 12px',
                borderRadius: '6px',
                border: '1px solid #cbd5e1',
                fontSize: '14px',
                backgroundColor: '#ffffff',
                boxSizing: 'border-box',
              }}
            >
              {randomizationUnits.map((unit) => (
                <option key={unit} value={unit}>
                  {unit}
                </option>
              ))}
            </select>
          </div>

          {/* Attribution Window */}
          <div>
            <label htmlFor="attribution-window-input" style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#334155', marginBottom: '4px' }}>
              Attribution Window
            </label>
            <div style={{ display: 'flex', gap: '6px' }}>
              <input
                id="attribution-window-input"
                type="number"
                min="1"
                value={getAttributionValue()}
                onChange={(e) => handleAttributionChange(Number(e.target.value), attributionUnit)}
                disabled={readOnly}
                style={{
                  flex: 1,
                  padding: '8px 12px',
                  borderRadius: '6px',
                  border: '1px solid #cbd5e1',
                  fontSize: '14px',
                  boxSizing: 'border-box',
                }}
              />
              <select
                aria-label="Attribution Unit"
                value={attributionUnit}
                onChange={(e) => {
                  const newUnit = e.target.value as 'days' | 'hours' | 'seconds';
                  setAttributionUnit(newUnit);
                  handleAttributionChange(getAttributionValue(), newUnit);
                }}
                disabled={readOnly}
                style={{
                  padding: '8px 8px',
                  borderRadius: '6px',
                  border: '1px solid #cbd5e1',
                  fontSize: '13px',
                  backgroundColor: '#ffffff',
                }}
              >
                <option value="days">Days</option>
                <option value="hours">Hours</option>
                <option value="seconds">Secs</option>
              </select>
            </div>
          </div>

          {/* Salt */}
          <div>
            <label htmlFor="exp-salt" style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#334155', marginBottom: '4px' }}>
              Experiment Salt
            </label>
            <input
              id="exp-salt"
              type="text"
              value={formState.salt}
              onChange={(e) => handleFieldChange('salt', e.target.value)}
              disabled={readOnly}
              placeholder="e.g. salt-welcome-v1"
              style={{
                width: '100%',
                padding: '8px 12px',
                borderRadius: '6px',
                border: '1px solid #cbd5e1',
                fontSize: '14px',
                boxSizing: 'border-box',
              }}
            />
          </div>
        </div>

        {/* Allocation Summary & Validator Banner */}
        <div
          data-testid="allocation-total"
          style={{
            padding: '14px 18px',
            borderRadius: '8px',
            backgroundColor: totalBasisPoints === 10000 ? '#f0fdf4' : '#fef2f2',
            border: totalBasisPoints === 10000 ? '1px solid #bbf7d0' : '1px solid #fecaca',
            marginBottom: '24px',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <div>
            <div
              style={{
                fontSize: '14px',
                fontWeight: 700,
                color: totalBasisPoints === 10000 ? '#15803d' : '#b91c1c',
              }}
            >
              Total Traffic Allocation: {totalPercentage}% ({totalBasisPoints.toLocaleString()} / 10,000 bps)
            </div>
            <div style={{ fontSize: '12px', color: totalBasisPoints === 10000 ? '#166534' : '#991b1b', marginTop: '2px' }}>
              {totalBasisPoints === 10000
                ? '✅ Allocation is valid and fully sums to 100% (10,000 basis points).'
                : '⚠️ Save is blocked until total allocation equals exactly 100% (10,000 basis points).'}
            </div>
          </div>
          <div
            style={{
              padding: '6px 12px',
              borderRadius: '6px',
              fontWeight: 700,
              fontSize: '13px',
              backgroundColor: totalBasisPoints === 10000 ? '#dcfce7' : '#fee2e2',
              color: totalBasisPoints === 10000 ? '#166534' : '#991b1b',
            }}
          >
            {totalBasisPoints === 10000 ? 'VALID ALLOCATION' : 'INVALID ALLOCATION'}
          </div>
        </div>

        {/* 2-to-5 Variant Editor Section */}
        <div style={{ marginBottom: '24px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
            <h3 style={{ fontSize: '16px', fontWeight: 700, margin: 0, color: '#1e293b' }}>
              Variants Configuration (2 - 5 Variants)
            </h3>
            <button
              type="button"
              data-testid="add-variant-btn"
              onClick={handleAddVariant}
              disabled={readOnly || formState.variants.length >= 5}
              style={{
                padding: '6px 12px',
                borderRadius: '6px',
                border: '1px solid #cbd5e1',
                backgroundColor: formState.variants.length >= 5 || readOnly ? '#f1f5f9' : '#ffffff',
                color: formState.variants.length >= 5 || readOnly ? '#94a3b8' : '#0284c7',
                fontWeight: 600,
                fontSize: '13px',
                cursor: formState.variants.length >= 5 || readOnly ? 'not-allowed' : 'pointer',
              }}
            >
              + Add Variant
            </button>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
            {formState.variants.map((variant, index) => {
              const weightPercentage = (variant.weight_basis_points / 100).toString();

              return (
                <div
                  key={variant.key || `var-idx-${index}`}
                  data-testid={`variant-row-${variant.key}`}
                  style={{
                    display: 'grid',
                    gridTemplateColumns: '1.2fr 1.8fr 1fr 1fr auto',
                    gap: '12px',
                    alignItems: 'center',
                    padding: '12px 14px',
                    borderRadius: '8px',
                    border: '1px solid #e2e8f0',
                    backgroundColor: variant.is_control ? '#f8fafc' : '#ffffff',
                  }}
                >
                  {/* Variant Key (Stable Key) */}
                  <div>
                    <label style={{ display: 'block', fontSize: '11px', fontWeight: 600, color: '#64748b', marginBottom: '2px' }}>
                      Variant Key (Stable)
                    </label>
                    <input
                      type="text"
                      aria-label={`Variant Key ${index + 1}`}
                      value={variant.key}
                      onChange={(e) => handleVariantChange(index, 'key', e.target.value)}
                      disabled={readOnly}
                      style={{
                        width: '100%',
                        padding: '6px 10px',
                        borderRadius: '6px',
                        border: '1px solid #cbd5e1',
                        fontSize: '13px',
                        fontFamily: 'monospace',
                        fontWeight: 600,
                        backgroundColor: '#f8fafc',
                        boxSizing: 'border-box',
                      }}
                    />
                  </div>

                  {/* Display Name */}
                  <div>
                    <label style={{ display: 'block', fontSize: '11px', fontWeight: 600, color: '#64748b', marginBottom: '2px' }}>
                      Display Name
                    </label>
                    <input
                      type="text"
                      aria-label={`Variant Name ${index + 1}`}
                      value={variant.name}
                      onChange={(e) => handleVariantChange(index, 'name', e.target.value)}
                      disabled={readOnly}
                      style={{
                        width: '100%',
                        padding: '6px 10px',
                        borderRadius: '6px',
                        border: '1px solid #cbd5e1',
                        fontSize: '13px',
                        boxSizing: 'border-box',
                      }}
                    />
                  </div>

                  {/* Weight Basis Points / Whole-Number Percentage Input */}
                  <div>
                    <label style={{ display: 'block', fontSize: '11px', fontWeight: 600, color: '#64748b', marginBottom: '2px' }}>
                      Allocation (%)
                    </label>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                      <input
                        type="number"
                        min="0"
                        max="100"
                        step="1"
                        aria-label={`Variant Allocation Percentage ${index + 1}`}
                        value={weightPercentage}
                        onChange={(e) => handleVariantChange(index, 'weight_basis_points', e.target.value)}
                        disabled={readOnly}
                        style={{
                          width: '100%',
                          padding: '6px 10px',
                          borderRadius: '6px',
                          border: '1px solid #cbd5e1',
                          fontSize: '13px',
                          fontWeight: 600,
                          boxSizing: 'border-box',
                        }}
                      />
                      <span style={{ fontSize: '12px', color: '#64748b', fontWeight: 600 }}>%</span>
                    </div>
                    <span style={{ fontSize: '10px', color: '#94a3b8', display: 'block', marginTop: '2px' }}>
                      {variant.weight_basis_points} bps
                    </span>
                  </div>

                  {/* Control Toggle */}
                  <div style={{ textAlign: 'center' }}>
                    <label style={{ display: 'block', fontSize: '11px', fontWeight: 600, color: '#64748b', marginBottom: '4px' }}>
                      Control?
                    </label>
                    <label style={{ display: 'inline-flex', alignItems: 'center', cursor: readOnly ? 'default' : 'pointer', gap: '6px' }}>
                      <input
                        type="radio"
                        name="controlVariant"
                        checked={variant.is_control}
                        onChange={() => handleVariantChange(index, 'is_control', true)}
                        disabled={readOnly}
                        style={{ width: '16px', height: '16px', accentColor: '#0284c7' }}
                      />
                      <span style={{ fontSize: '12px', fontWeight: variant.is_control ? 700 : 400, color: variant.is_control ? '#0369a1' : '#475569' }}>
                        {variant.is_control ? 'Control' : 'Treatment'}
                      </span>
                    </label>
                  </div>

                  {/* Delete Button */}
                  <div>
                    <button
                      type="button"
                      aria-label={`Remove Variant ${variant.key}`}
                      onClick={() => handleRemoveVariant(index)}
                      disabled={readOnly || formState.variants.length <= 2}
                      style={{
                        padding: '6px 10px',
                        borderRadius: '6px',
                        border: '1px solid #fecaca',
                        backgroundColor: formState.variants.length <= 2 || readOnly ? '#f1f5f9' : '#fff5f5',
                        color: formState.variants.length <= 2 || readOnly ? '#94a3b8' : '#e11d48',
                        cursor: formState.variants.length <= 2 || readOnly ? 'not-allowed' : 'pointer',
                        fontSize: '13px',
                      }}
                    >
                      🗑️
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Metric Picker Section */}
        <div style={{ marginBottom: '24px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
            <h3 style={{ fontSize: '16px', fontWeight: 700, margin: 0, color: '#1e293b' }}>
              Metric Definitions & Pickers
            </h3>
            <button
              type="button"
              onClick={handleAddMetric}
              disabled={readOnly}
              style={{
                padding: '6px 12px',
                borderRadius: '6px',
                border: '1px solid #cbd5e1',
                backgroundColor: '#ffffff',
                color: '#0284c7',
                fontWeight: 600,
                fontSize: '13px',
                cursor: readOnly ? 'not-allowed' : 'pointer',
              }}
            >
              + Add Metric
            </button>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            {formState.metrics.map((metric, index) => (
              <div
                key={metric.key || `metric-${index}`}
                style={{
                  display: 'grid',
                  gridTemplateColumns: '1fr 1.2fr 1.5fr 1fr auto',
                  gap: '10px',
                  alignItems: 'center',
                  padding: '10px 12px',
                  borderRadius: '6px',
                  border: '1px solid #e2e8f0',
                  backgroundColor: '#f8fafc',
                }}
              >
                <div>
                  <label htmlFor={`metric-key-${index}`} style={{ display: 'block', fontSize: '11px', fontWeight: 600, color: '#64748b' }}>
                    Metric Key
                  </label>
                  <input
                    id={`metric-key-${index}`}
                    type="text"
                    aria-label={`Metric Key ${index + 1}`}
                    value={metric.key}
                    onChange={(e) => handleMetricChange(index, 'key', e.target.value)}
                    disabled={readOnly}
                    style={{
                      width: '100%',
                      padding: '6px 8px',
                      borderRadius: '4px',
                      border: '1px solid #cbd5e1',
                      fontSize: '13px',
                      boxSizing: 'border-box',
                    }}
                  />
                </div>

                <div>
                  <label htmlFor={`metric-name-${index}`} style={{ display: 'block', fontSize: '11px', fontWeight: 600, color: '#64748b' }}>
                    Metric Name
                  </label>
                  <input
                    id={`metric-name-${index}`}
                    type="text"
                    aria-label={`Metric Name ${index + 1}`}
                    value={metric.name}
                    onChange={(e) => handleMetricChange(index, 'name', e.target.value)}
                    disabled={readOnly}
                    style={{
                      width: '100%',
                      padding: '6px 8px',
                      borderRadius: '4px',
                      border: '1px solid #cbd5e1',
                      fontSize: '13px',
                      boxSizing: 'border-box',
                    }}
                  />
                </div>

                <div>
                  <label htmlFor={`metric-event-${index}`} style={{ display: 'block', fontSize: '11px', fontWeight: 600, color: '#64748b' }}>
                    Event Type
                  </label>
                  <input
                    id={`metric-event-${index}`}
                    type="text"
                    aria-label={`Metric Event Type ${index + 1}`}
                    value={metric.event_type}
                    onChange={(e) => handleMetricChange(index, 'event_type', e.target.value)}
                    disabled={readOnly}
                    style={{
                      width: '100%',
                      padding: '6px 8px',
                      borderRadius: '4px',
                      border: '1px solid #cbd5e1',
                      fontSize: '13px',
                      boxSizing: 'border-box',
                    }}
                  />
                </div>

                <div>
                  <label htmlFor={`metric-type-${index}`} style={{ display: 'block', fontSize: '11px', fontWeight: 600, color: '#64748b' }}>
                    Aggregation Type
                  </label>
                  <select
                    id={`metric-type-${index}`}
                    aria-label={`Metric Aggregation Type ${index + 1}`}
                    value={metric.type}
                    onChange={(e) =>
                      handleMetricChange(index, 'type', e.target.value as 'conversion' | 'sum' | 'mean')
                    }
                    disabled={readOnly}
                    style={{
                      width: '100%',
                      padding: '6px 8px',
                      borderRadius: '4px',
                      border: '1px solid #cbd5e1',
                      fontSize: '13px',
                      backgroundColor: '#ffffff',
                      boxSizing: 'border-box',
                    }}
                  >
                    <option value="conversion">conversion</option>
                    <option value="sum">sum</option>
                    <option value="mean">mean</option>
                  </select>
                </div>

                <div>
                  <button
                    type="button"
                    aria-label={`Remove Metric ${metric.key}`}
                    onClick={() => handleRemoveMetric(index)}
                    disabled={readOnly || formState.metrics.length <= 1}
                    style={{
                      padding: '6px 8px',
                      borderRadius: '4px',
                      border: '1px solid #fecaca',
                      backgroundColor: formState.metrics.length <= 1 || readOnly ? '#f1f5f9' : '#fff5f5',
                      color: formState.metrics.length <= 1 || readOnly ? '#94a3b8' : '#e11d48',
                      cursor: formState.metrics.length <= 1 || readOnly ? 'not-allowed' : 'pointer',
                    }}
                  >
                    ✕
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Validation Errors List (if any) */}
        {!isValid && (
          <div
            data-testid="allocation-error"
            style={{
              padding: '12px 16px',
              borderRadius: '8px',
              backgroundColor: '#fff1f2',
              border: '1px solid #fda4af',
              marginBottom: '20px',
            }}
          >
            <div style={{ fontSize: '13px', fontWeight: 700, color: '#9f1239', marginBottom: '4px' }}>
              Validation Issues (Save Blocked):
            </div>
            <ul style={{ margin: 0, paddingLeft: '20px', fontSize: '12px', color: '#be123c' }}>
              {validationErrors.map((err, i) => (
                <li key={i}>{err}</li>
              ))}
            </ul>
          </div>
        )}

        {/* Action Buttons */}
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px' }}>
          {onCancel && (
            <button
              type="button"
              onClick={onCancel}
              style={{
                padding: '8px 16px',
                borderRadius: '6px',
                border: '1px solid #cbd5e1',
                backgroundColor: '#ffffff',
                color: '#475569',
                fontWeight: 600,
                fontSize: '14px',
                cursor: 'pointer',
              }}
            >
              Cancel
            </button>
          )}

          <button
            type="submit"
            data-testid="save-experiment-btn"
            disabled={!isValid || readOnly}
            style={{
              padding: '8px 20px',
              borderRadius: '6px',
              border: 'none',
              backgroundColor: isValid && !readOnly ? '#0284c7' : '#94a3b8',
              color: '#ffffff',
              fontWeight: 700,
              fontSize: '14px',
              cursor: isValid && !readOnly ? 'pointer' : 'not-allowed',
              boxShadow: isValid && !readOnly ? '0 2px 4px rgba(2, 132, 199, 0.2)' : 'none',
            }}
          >
            Save Experiment
          </button>
        </div>
      </form>
    </div>
  );
};

export const ExperimentInspector: React.FC<ExperimentInspectorProps> = (props) => (
  <SafeQueryClientProvider>
    <ExperimentInspectorInner {...props} />
  </SafeQueryClientProvider>
);
