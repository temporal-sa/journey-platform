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
        const percentageVal = Number(value);
        targetVariant.weight_basis_points = Math.round(percentageVal * 100);
      } else if (field === 'is_control' && Boolean(value) === true) {
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
    const nextChar = String.fromCharCode(97 + formState.variants.length);
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

  const isAllocationValid = totalBasisPoints === 10000;

  return (
    <div
      data-testid="experiment-inspector"
      className="glass-panel p-6 lg:p-8 bg-[#0F131D]/95 backdrop-blur-xl rounded-2xl border border-[#1F2937] text-[#DFE2F1] font-['Outfit',sans-serif] shadow-2xl space-y-6 max-w-4xl mx-auto"
    >
      {/* Header */}
      <div className="flex justify-between items-center border-b border-[#1F2937] pb-5">
        <div>
          <h2 className="text-xl font-['Outfit'] font-bold text-white flex items-center gap-2.5">
            <span className="material-symbols-outlined text-[#ddb7ff]">science</span>
            <span>Experiment Authoring & Configuration</span>
          </h2>
          <p className="text-xs text-[#908fa0] mt-1 font-['Inter']">
            Define variant allocations, randomization unit, metrics, and attribution rules.
          </p>
        </div>
        <span
          className={`px-3 py-1 rounded-full text-xs font-bold font-mono uppercase tracking-wider shadow-sm border ${
            formState.status === 'active'
              ? 'bg-[#10b981]/20 text-[#10b981] border-[#10b981]/40'
              : 'bg-[#f59e0b]/20 text-[#f59e0b] border-[#f59e0b]/40'
          }`}
        >
          {formState.status}
        </span>
      </div>

      <form onSubmit={handleSave} className="space-y-6">
        {/* Experiment Basic Details */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label htmlFor="exp-id" className="block text-xs font-semibold text-[#c7c4d7] mb-1.5 uppercase tracking-wider font-['Inter']">
              Experiment ID
            </label>
            <input
              id="exp-id"
              type="text"
              value={formState.experiment_id}
              onChange={(e) => handleFieldChange('experiment_id', e.target.value)}
              disabled={readOnly}
              className="w-full px-3.5 py-2.5 rounded-xl bg-[#090D14] border border-[#313540] text-[#DFE2F1] font-mono text-sm focus:outline-none focus:border-[#6366F1] focus:ring-1 focus:ring-[#6366F1] transition-all disabled:opacity-50 disabled:cursor-not-allowed box-border"
            />
          </div>

          <div>
            <label htmlFor="exp-name" className="block text-xs font-semibold text-[#c7c4d7] mb-1.5 uppercase tracking-wider font-['Inter']">
              Experiment Name
            </label>
            <input
              id="exp-name"
              type="text"
              value={formState.name}
              onChange={(e) => handleFieldChange('name', e.target.value)}
              disabled={readOnly}
              className="w-full px-3.5 py-2.5 rounded-xl bg-[#090D14] border border-[#313540] text-[#DFE2F1] font-mono text-sm focus:outline-none focus:border-[#6366F1] focus:ring-1 focus:ring-[#6366F1] transition-all disabled:opacity-50 disabled:cursor-not-allowed box-border"
            />
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Randomization Unit Selector */}
          <div>
            <label htmlFor="randomization-unit-select" className="block text-xs font-semibold text-[#c7c4d7] mb-1.5 uppercase tracking-wider font-['Inter']">
              Randomization Unit
            </label>
            <select
              id="randomization-unit-select"
              value={formState.randomization_unit}
              onChange={(e) => handleFieldChange('randomization_unit', e.target.value as RandomizationUnit)}
              disabled={readOnly}
              className="w-full px-3.5 py-2.5 rounded-xl bg-[#090D14] border border-[#313540] text-[#DFE2F1] font-mono text-sm focus:outline-none focus:border-[#6366F1] focus:ring-1 focus:ring-[#6366F1] transition-all disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer box-border"
            >
              {randomizationUnits.map((unit) => (
                <option key={unit} value={unit} className="bg-[#090D14] text-[#DFE2F1]">
                  {unit}
                </option>
              ))}
            </select>
          </div>

          {/* Attribution Window */}
          <div>
            <label htmlFor="attribution-window-input" className="block text-xs font-semibold text-[#c7c4d7] mb-1.5 uppercase tracking-wider font-['Inter']">
              Attribution Window
            </label>
            <div className="flex gap-2">
              <input
                id="attribution-window-input"
                type="number"
                min="1"
                value={getAttributionValue()}
                onChange={(e) => handleAttributionChange(Number(e.target.value), attributionUnit)}
                disabled={readOnly}
                className="flex-1 min-w-0 px-3.5 py-2.5 rounded-xl bg-[#090D14] border border-[#313540] text-[#DFE2F1] font-mono text-sm focus:outline-none focus:border-[#6366F1] focus:ring-1 focus:ring-[#6366F1] transition-all disabled:opacity-50 disabled:cursor-not-allowed box-border"
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
                className="px-3 py-2.5 rounded-xl bg-[#090D14] border border-[#313540] text-[#DFE2F1] font-mono text-xs focus:outline-none focus:border-[#6366F1] disabled:opacity-50 cursor-pointer"
              >
                <option value="days" className="bg-[#090D14] text-[#DFE2F1]">Days</option>
                <option value="hours" className="bg-[#090D14] text-[#DFE2F1]">Hours</option>
                <option value="seconds" className="bg-[#090D14] text-[#DFE2F1]">Secs</option>
              </select>
            </div>
          </div>

          {/* Salt */}
          <div>
            <label htmlFor="exp-salt" className="block text-xs font-semibold text-[#c7c4d7] mb-1.5 uppercase tracking-wider font-['Inter']">
              Experiment Salt
            </label>
            <input
              id="exp-salt"
              type="text"
              value={formState.salt}
              onChange={(e) => handleFieldChange('salt', e.target.value)}
              disabled={readOnly}
              placeholder="e.g. salt-welcome-v1"
              className="w-full px-3.5 py-2.5 rounded-xl bg-[#090D14] border border-[#313540] text-[#DFE2F1] font-mono text-sm focus:outline-none focus:border-[#6366F1] focus:ring-1 focus:ring-[#6366F1] transition-all disabled:opacity-50 disabled:cursor-not-allowed box-border"
            />
          </div>
        </div>

        {/* Allocation Summary & Validator Banner */}
        <div
          data-testid="allocation-total"
          className={`glass-panel p-4 rounded-xl border flex flex-col sm:flex-row sm:items-center justify-between gap-3 shadow-md transition-all ${
            isAllocationValid
              ? 'border-[#10b981]/40 bg-[#10b981]/10 status-strip-emerald'
              : 'border-[#f43f5e]/40 bg-[#f43f5e]/10 status-strip-rose'
          }`}
        >
          <div>
            <div className={`text-sm font-bold font-['Outfit'] ${isAllocationValid ? 'text-[#10b981]' : 'text-[#ffb4ab]'}`}>
              Total Traffic Allocation: {totalPercentage}% ({totalBasisPoints.toLocaleString()} / 10,000 bps)
            </div>
            <div className={`text-xs mt-1 font-['Inter'] ${isAllocationValid ? 'text-[#c7c4d7]' : 'text-[#ffb4ab]'}`}>
              {isAllocationValid
                ? '✅ Allocation is valid and fully sums to 100% (10,000 basis points).'
                : '⚠️ Save is blocked until total allocation equals exactly 100% (10,000 basis points).'}
            </div>
          </div>
          <div
            className={`px-3 py-1 rounded-full text-xs font-bold font-mono uppercase tracking-wider border shadow-sm shrink-0 ${
              isAllocationValid
                ? 'bg-[#10b981]/20 text-[#10b981] border-[#10b981]/40'
                : 'bg-[#f43f5e]/20 text-[#ffb4ab] border-[#f43f5e]/40 animate-pulse'
            }`}
          >
            {isAllocationValid ? 'VALID ALLOCATION' : 'INVALID ALLOCATION'}
          </div>
        </div>

        {/* 2-to-5 Variant Editor Section */}
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <h3 className="text-sm font-['Outfit'] font-bold text-white flex items-center gap-2">
              <span className="material-symbols-outlined text-[#ddb7ff] text-base">tune</span>
              <span>Variants Configuration (2 - 5 Variants)</span>
            </h3>
            <button
              type="button"
              data-testid="add-variant-btn"
              onClick={handleAddVariant}
              disabled={readOnly || formState.variants.length >= 5}
              className="px-3.5 py-1.5 rounded-xl border border-[#313540] hover:border-[#6366F1] bg-[#171b26] hover:bg-[#262a35] text-[#c0c1ff] font-semibold text-xs transition-all flex items-center gap-1 cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed shadow-sm"
            >
              + Add Variant
            </button>
          </div>

          <div className="flex flex-col gap-3">
            {formState.variants.map((variant, index) => {
              const weightPercentage = (variant.weight_basis_points / 100).toString();

              return (
                <div
                  key={variant.key || `var-idx-${index}`}
                  data-testid={`variant-row-${variant.key}`}
                  className={`glass-panel p-4 rounded-xl border grid grid-cols-1 sm:grid-cols-12 gap-3 items-center shadow-md transition-all ${
                    variant.is_control
                      ? 'border-[#ddb7ff]/40 bg-[#ddb7ff]/10'
                      : 'border-[#1F2937] bg-[#171b26] hover:border-[#313540]'
                  }`}
                >
                  {/* Variant Key (Stable Key) */}
                  <div className="sm:col-span-3">
                    <label className="block text-[11px] font-semibold text-[#908fa0] mb-1 font-['Inter']">
                      Variant Key (Stable)
                    </label>
                    <input
                      type="text"
                      aria-label={`Variant Key ${index + 1}`}
                      value={variant.key}
                      onChange={(e) => handleVariantChange(index, 'key', e.target.value)}
                      disabled={readOnly}
                      className="w-full px-3 py-1.5 rounded-lg bg-[#090D14] border border-[#313540] text-[#ddb7ff] font-mono text-xs font-semibold focus:outline-none focus:border-[#6366F1] box-border"
                    />
                  </div>

                  {/* Display Name */}
                  <div className="sm:col-span-4">
                    <label className="block text-[11px] font-semibold text-[#908fa0] mb-1 font-['Inter']">
                      Display Name
                    </label>
                    <input
                      type="text"
                      aria-label={`Variant Name ${index + 1}`}
                      value={variant.name}
                      onChange={(e) => handleVariantChange(index, 'name', e.target.value)}
                      disabled={readOnly}
                      className="w-full px-3 py-1.5 rounded-lg bg-[#090D14] border border-[#313540] text-[#dfe2f1] font-mono text-xs focus:outline-none focus:border-[#6366F1] box-border"
                    />
                  </div>

                  {/* Weight Basis Points / Percentage Input */}
                  <div className="sm:col-span-2">
                    <label className="block text-[11px] font-semibold text-[#908fa0] mb-1 font-['Inter']">
                      Allocation (%)
                    </label>
                    <div className="flex items-center gap-1.5">
                      <input
                        type="number"
                        min="0"
                        max="100"
                        step="1"
                        aria-label={`Variant Allocation Percentage ${index + 1}`}
                        value={weightPercentage}
                        onChange={(e) => handleVariantChange(index, 'weight_basis_points', e.target.value)}
                        disabled={readOnly}
                        className="w-full px-2.5 py-1.5 rounded-lg bg-[#090D14] border border-[#313540] text-[#dfe2f1] font-mono text-xs font-bold focus:outline-none focus:border-[#6366F1] box-border"
                      />
                      <span className="text-xs text-[#908fa0] font-mono font-bold">%</span>
                    </div>
                    <span className="text-[10px] text-[#908fa0] font-mono block mt-0.5">
                      {variant.weight_basis_points} bps
                    </span>
                  </div>

                  {/* Control Toggle */}
                  <div className="sm:col-span-2 text-center">
                    <label className="block text-[11px] font-semibold text-[#908fa0] mb-1 font-['Inter']">
                      Control?
                    </label>
                    <label className="inline-flex items-center cursor-pointer gap-1.5">
                      <input
                        type="radio"
                        name="controlVariant"
                        checked={variant.is_control}
                        onChange={() => handleVariantChange(index, 'is_control', true)}
                        disabled={readOnly}
                        className="w-4 h-4 accent-[#6366F1] cursor-pointer"
                      />
                      <span className={`text-xs font-mono font-semibold ${variant.is_control ? 'text-[#ddb7ff]' : 'text-[#908fa0]'}`}>
                        {variant.is_control ? 'Control' : 'Treatment'}
                      </span>
                    </label>
                  </div>

                  {/* Delete Button */}
                  <div className="sm:col-span-1 flex justify-end">
                    <button
                      type="button"
                      aria-label={`Remove Variant ${variant.key}`}
                      onClick={() => handleRemoveVariant(index)}
                      disabled={readOnly || formState.variants.length <= 2}
                      className="p-1.5 rounded-lg border border-[#f43f5e]/40 bg-[#f43f5e]/10 text-[#ffb4ab] hover:bg-[#f43f5e]/20 transition-colors disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
                    >
                      <span className="material-symbols-outlined text-sm">delete</span>
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Metric Picker Section */}
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <h3 className="text-sm font-['Outfit'] font-bold text-white flex items-center gap-2">
              <span className="material-symbols-outlined text-[#4cd7f6] text-base">analytics</span>
              <span>Metric Definitions & Pickers</span>
            </h3>
            <button
              type="button"
              onClick={handleAddMetric}
              disabled={readOnly}
              className="px-3.5 py-1.5 rounded-xl border border-[#313540] hover:border-[#6366F1] bg-[#171b26] hover:bg-[#262a35] text-[#c0c1ff] font-semibold text-xs transition-all flex items-center gap-1 cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed shadow-sm"
            >
              + Add Metric
            </button>
          </div>

          <div className="flex flex-col gap-2.5">
            {formState.metrics.map((metric, index) => (
              <div
                key={metric.key || `metric-${index}`}
                className="glass-panel p-3.5 rounded-xl border border-[#1F2937] bg-[#171b26] grid grid-cols-1 sm:grid-cols-12 gap-3 items-center shadow-md"
              >
                <div className="sm:col-span-3">
                  <label htmlFor={`metric-key-${index}`} className="block text-[11px] font-semibold text-[#908fa0] mb-1 font-['Inter']">
                    Metric Key
                  </label>
                  <input
                    id={`metric-key-${index}`}
                    type="text"
                    aria-label={`Metric Key ${index + 1}`}
                    value={metric.key}
                    onChange={(e) => handleMetricChange(index, 'key', e.target.value)}
                    disabled={readOnly}
                    className="w-full px-3 py-1.5 rounded-lg bg-[#090D14] border border-[#313540] text-[#dfe2f1] font-mono text-xs focus:outline-none focus:border-[#6366F1] box-border"
                  />
                </div>

                <div className="sm:col-span-3">
                  <label htmlFor={`metric-name-${index}`} className="block text-[11px] font-semibold text-[#908fa0] mb-1 font-['Inter']">
                    Metric Name
                  </label>
                  <input
                    id={`metric-name-${index}`}
                    type="text"
                    aria-label={`Metric Name ${index + 1}`}
                    value={metric.name}
                    onChange={(e) => handleMetricChange(index, 'name', e.target.value)}
                    disabled={readOnly}
                    className="w-full px-3 py-1.5 rounded-lg bg-[#090D14] border border-[#313540] text-[#dfe2f1] font-mono text-xs focus:outline-none focus:border-[#6366F1] box-border"
                  />
                </div>

                <div className="sm:col-span-3">
                  <label htmlFor={`metric-event-${index}`} className="block text-[11px] font-semibold text-[#908fa0] mb-1 font-['Inter']">
                    Event Type
                  </label>
                  <input
                    id={`metric-event-${index}`}
                    type="text"
                    aria-label={`Metric Event Type ${index + 1}`}
                    value={metric.event_type}
                    onChange={(e) => handleMetricChange(index, 'event_type', e.target.value)}
                    disabled={readOnly}
                    className="w-full px-3 py-1.5 rounded-lg bg-[#090D14] border border-[#313540] text-[#dfe2f1] font-mono text-xs focus:outline-none focus:border-[#6366F1] box-border"
                  />
                </div>

                <div className="sm:col-span-2">
                  <label htmlFor={`metric-type-${index}`} className="block text-[11px] font-semibold text-[#908fa0] mb-1 font-['Inter']">
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
                    className="w-full px-2.5 py-1.5 rounded-lg bg-[#090D14] border border-[#313540] text-[#dfe2f1] font-mono text-xs focus:outline-none focus:border-[#6366F1] cursor-pointer"
                  >
                    <option value="conversion" className="bg-[#090D14] text-[#dfe2f1]">conversion</option>
                    <option value="sum" className="bg-[#090D14] text-[#dfe2f1]">sum</option>
                    <option value="mean" className="bg-[#090D14] text-[#dfe2f1]">mean</option>
                  </select>
                </div>

                <div className="sm:col-span-1 flex justify-end">
                  <button
                    type="button"
                    aria-label={`Remove Metric ${metric.key}`}
                    onClick={() => handleRemoveMetric(index)}
                    disabled={readOnly || formState.metrics.length <= 1}
                    className="p-1.5 rounded-lg border border-[#f43f5e]/40 bg-[#f43f5e]/10 text-[#ffb4ab] hover:bg-[#f43f5e]/20 transition-colors disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
                  >
                    <span className="material-symbols-outlined text-sm">close</span>
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
            className="glass-panel p-4 rounded-xl border border-[#f43f5e]/40 bg-[#f43f5e]/10 text-[#ffb4ab] text-xs space-y-1.5 status-strip-rose shadow-md"
          >
            <div className="font-bold text-white font-['Outfit']">
              Validation Issues (Save Blocked):
            </div>
            <ul className="list-disc list-inside space-y-1 font-['Inter'] text-[#ffb4ab]">
              {validationErrors.map((err, i) => (
                <li key={i}>{err}</li>
              ))}
            </ul>
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex justify-end gap-3 pt-2">
          {onCancel && (
            <button
              type="button"
              onClick={onCancel}
              className="px-5 py-2.5 rounded-xl border border-[#464554] hover:bg-[#262a35] text-[#c7c4d7] hover:text-white font-medium text-sm transition-colors cursor-pointer"
            >
              Cancel
            </button>
          )}

          <button
            type="submit"
            data-testid="save-experiment-btn"
            disabled={!isValid || readOnly}
            className="px-6 py-2.5 rounded-xl bg-gradient-to-r from-[#6366F1] to-[#8b5cf6] hover:opacity-90 text-white font-bold text-sm shadow-lg shadow-indigo-500/25 border border-[#8083ff]/40 transition-all flex items-center gap-2 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed disabled:shadow-none"
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
