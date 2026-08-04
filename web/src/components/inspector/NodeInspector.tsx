import { useState, useEffect, useRef } from 'react';
import { useQuery } from '@tanstack/react-query';
import { JourneyApiClient } from '../../api/client';
import { useEditorStore } from '../../stores/editorStore';
import type { CatalogRecord, GraphNode } from '../../types/api';
import { getNodeIssues, getFieldError, formatIssueCode } from './validationMapping';
import { SafeQueryClientProvider } from '../SafeQueryClientProvider';

const apiClient = new JourneyApiClient();
export interface NodeInspectorProps {
  nodeId: string | null;
  onClose?: () => void;
}

function NodeInspectorInner({ nodeId, onClose }: NodeInspectorProps) {
  const { currentDraft, updateNodes, validationResult, isInspectorOpen, isCanvasLocked } = useEditorStore();

  const { data: actionCatalog = [] } = useQuery<CatalogRecord[]>({
    queryKey: ['actionCatalogNodes'],
    queryFn: () => apiClient.getActionCatalog(),
  });

  // Selected Node from store
  const targetNode = currentDraft?.nodes?.find((n) => n.id === nodeId) || null;

  const catalogItem = actionCatalog.find(
    (item) => item.name === targetNode?.type || item.record_id === targetNode?.type
  );

  const schemaDef = catalogItem?.schema_definition as Record<string, any> | undefined;
  const catalogDefaultDuration =
    (schemaDef?.properties?.duration as { default?: number })?.default ??
    (schemaDef?.duration as number);
  const catalogDefaultUnit =
    (schemaDef?.properties?.unit as { default?: string })?.default ??
    (schemaDef?.unit as string);

  const [isSimulating, setIsSimulating] = useState(false);
  const [simulationStatus, setSimulationStatus] = useState<string | null>(null);

  const handleDryRun = async () => {
    if (!currentDraft?.draft_id || !nodeId) return;
    setIsSimulating(true);
    setSimulationStatus(null);
    try {
      await apiClient.simulateJourneyDraft(currentDraft.draft_id, { node_id: nodeId });
      setSimulationStatus('Dry run simulation succeeded!');
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Dry run simulation failed';
      setSimulationStatus(msg);
    } finally {
      setIsSimulating(false);
    }
  };
  // Isolated Draft Buffer State
  const [localDraft, setLocalDraft] = useState<{
    name: string;
    description: string;
    config: Record<string, unknown>;
  }>({
    name: '',
    description: '',
    config: {},
  });

  const [isDirty, setIsDirty] = useState(false);
  const inputRefs = useRef<Record<string, HTMLInputElement | HTMLTextAreaElement | null>>({});
  const [activeParamField, setActiveParamField] = useState<string | null>(null);
  useEffect(() => {
    if (targetNode) {
      setLocalDraft({
        name: targetNode.name || '',
        description: (targetNode.config?.description as string) || '',
        config: { ...(targetNode.config || {}) },
      });
      setIsDirty(false);
    }
  }, [targetNode?.id, targetNode?.name, JSON.stringify(targetNode?.config), isInspectorOpen]);


  if (!nodeId || !targetNode) {
    return (
      <div
        data-testid="node-inspector-empty"
        className="p-6 text-center text-[#908fa0] text-sm font-['Outfit',sans-serif] h-full flex flex-col items-center justify-center bg-[#171b26] border-l border-[#464554]"
      >
        <div className="w-12 h-12 rounded-none bg-[#c0c1ff]/10 border border-[#c0c1ff]/20 flex items-center justify-center text-[#c0c1ff] mb-3 shadow-lg">
          <span className="material-symbols-outlined text-2xl">edit_note</span>
        </div>
        <div className="font-semibold text-[#dfe2f1] font-['Outfit'] text-base">No Node Selected</div>
        <div className="text-xs text-[#908fa0] mt-1 max-w-[200px]">
          Click any canvas node to inspect and edit its parameters in the drawer.
        </div>
      </div>
    );
  }

  // Node Validation Issues
  const nodeIssues = getNodeIssues(nodeId, validationResult);
  const hasErrors = nodeIssues.some((i) => i.severity === 'error');

  // Handle Field Modifications in Isolated Draft Buffer
  const handleNameChange = (val: string) => {
    setLocalDraft((prev) => ({ ...prev, name: val }));
    setIsDirty(true);
  };

  const handleConfigChange = (field: string, val: unknown) => {
    setLocalDraft((prev) => ({
      ...prev,
      config: {
        ...prev.config,
        [field]: val,
      },
    }));
    setIsDirty(true);
  };

  // Save Semantics: Apply isolated draft buffer to graph node state
  const handleSave = () => {
    if (!currentDraft || !targetNode) return;

    const updatedNodes: GraphNode[] = currentDraft.nodes.map((n) => {
      if (n.id === nodeId) {
        return {
          ...n,
          name: localDraft.name,
          config: {
            ...localDraft.config,
            description: localDraft.description,
          },
        };
      }
      return n;
    });

    updateNodes(updatedNodes);
    setIsDirty(false);
  };

  // Cancel Semantics: Revert isolated draft buffer back to store graph node state
  const handleCancel = () => {
    if (targetNode) {
      setLocalDraft({
        name: targetNode.name || '',
        description: (targetNode.config?.description as string) || '',
        config: { ...(targetNode.config || {}) },
      });
      setIsDirty(false);
    }
  };

  // Parameter Token Insertion at Text Caret
  const handleOpenParamModal = (fieldKey: string) => {
    setActiveParamField(fieldKey);
    window.dispatchEvent(new CustomEvent('open-param-modal', { detail: { fieldKey } }));
  };

  const handleSelectToken = (token: string) => {
    if (!activeParamField) return;

    let currentValue = '';
    if (activeParamField === 'name') {
      currentValue = localDraft.name;
    } else {
      currentValue = (localDraft.config[activeParamField] as string) || '';
    }

    const ref = inputRefs.current[activeParamField];
    let startPos = currentValue.length;
    let endPos = currentValue.length;

    if (ref && typeof ref.selectionStart === 'number' && typeof ref.selectionEnd === 'number') {
      startPos = ref.selectionStart;
      endPos = ref.selectionEnd;
    }

    const updatedValue =
      currentValue.substring(0, startPos) + token + currentValue.substring(endPos);

    if (activeParamField === 'name') {
      handleNameChange(updatedValue);
    } else {
      handleConfigChange(activeParamField, updatedValue);
    }

    setTimeout(() => {
      if (ref) {
        ref.focus();
        const newCaretPos = startPos + token.length;
        ref.setSelectionRange(newCaretPos, newCaretPos);
      }
    }, 50);

  };
  useEffect(() => {
    const handler = (e: Event) => {
      const customEvt = e as CustomEvent<{ token: string; fieldKey: string }>;
      if (customEvt.detail?.token) {
        handleSelectToken(customEvt.detail.token);
      }
    };
    window.addEventListener('select-param-token', handler);
    return () => window.removeEventListener('select-param-token', handler);
  }, [activeParamField, localDraft, targetNode]);
  const renderFieldError = (fieldPath: string) => {
    const issue = getFieldError(fieldPath, nodeIssues);
    if (!issue) return null;

    return (
      <div
        data-testid={`field-error-${fieldPath}`}
        role="alert"
        aria-live="polite"
        className="mt-1.5 p-2 rounded-none bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs flex items-center justify-between"
      >
        <span>{issue.message}</span>
        <span className="font-mono text-[10px] font-bold px-1.5 py-0.5 rounded-none bg-rose-500/20 text-rose-300 uppercase">
          {formatIssueCode(issue.code)}
        </span>
      </div>
    );
  };

  const nodeType = targetNode.type;

  return (
    <aside
      data-testid="node-inspector-panel"
      aria-label="Node Inspector"
      className="flex flex-col h-full max-h-full min-h-0 bg-[#171b26] border-l border-[#464554] text-[#dfe2f1] font-['Outfit',sans-serif] text-sm w-96 shrink-0 shadow-2xl z-40 overflow-hidden"
    >
      {/* Inspector Header */}
      <div className="p-4 border-b border-[#464554] bg-[#1c1f2a] flex items-center justify-between shrink-0">
        <div>
          <div className="flex items-center gap-2">
            <span className="text-[10px] font-mono font-bold uppercase tracking-wider px-2 py-0.5 rounded-none bg-[#c0c1ff]/20 text-[#c0c1ff] border border-[#c0c1ff]/30">
              {nodeType}
            </span>
            {isDirty && (
              <span
                data-testid="unsaved-inspector-badge"
                className="text-[10px] font-semibold px-2 py-0.5 rounded-none bg-amber-500/20 text-amber-300 border border-amber-500/30 animate-pulse"
              >
                Unsaved Edits
              </span>
            )}
          </div>
          <h3 className="font-['Outfit'] font-semibold text-base text-[#dfe2f1] mt-1">
            {localDraft.name || targetNode.name}
          </h3>
        </div>

        {onClose && (
          <button
            onClick={onClose}
            aria-label="Close Node Inspector"
            data-testid="close-inspector-btn"
            className="w-8 h-8 rounded-none bg-[#262a35] text-[#908fa0] hover:text-white hover:bg-white/10 flex items-center justify-center transition-all border border-white/5 cursor-pointer shrink-0 shadow-sm"
          >
            <span className="material-symbols-outlined text-lg">close</span>
          </button>
        )}
      </div>

      {isCanvasLocked && (
        <div
          data-testid="canvas-locked-inspector-banner"
          className="mx-4 mt-3 px-3 py-2 rounded-none bg-amber-500/10 border border-amber-500/30 text-amber-300 text-xs font-semibold flex items-center gap-2"
        >
          <span className="material-symbols-outlined text-sm">lock</span>
          <span>Canvas is Locked (Read Only)</span>
        </div>
      )}


      {/* Node Validation Summary Alert Banner */}
      {nodeIssues.length > 0 && (
        <div
          data-testid="node-validation-alert"
          className={`mx-4 mt-3 p-3 rounded-none border text-xs font-['Outfit',sans-serif] shrink-0 ${
            hasErrors
              ? 'bg-rose-500/10 border-rose-500/30 text-rose-200'
              : 'bg-amber-500/10 border-amber-500/30 text-amber-200'
          }`}
        >
          <div className="font-semibold mb-1 flex items-center gap-1.5 font-['Outfit'] text-sm">
            <span className="material-symbols-outlined text-base">
              {hasErrors ? 'error' : 'warning'}
            </span>
            <span>{hasErrors ? 'Node Validation Errors' : 'Node Validation Warnings'}</span>
          </div>
          {nodeIssues.map((issue) => (
            <div
              key={issue.issue_id}
              className={`text-xs mt-1 flex items-start gap-1 font-mono ${
                issue.severity === 'error' ? 'text-rose-300' : 'text-amber-300'
              }`}
            >
              <span>• [{formatIssueCode(issue.code)}]</span>
              <span>{issue.message}</span>
            </div>
          ))}
        </div>
      )}

      {/* Scrollable Form Body */}
      <div className="flex-1 min-h-0 overflow-y-auto p-4 space-y-4">
        {/* Common Section: General Node Settings */}
        <div>
          <div className="text-[10px] font-mono font-bold uppercase tracking-wider text-[#908fa0] mb-2">
            General Node Info
          </div>

          <div className="mb-3">
            <label className="block text-xs font-medium text-[#c7c4d7] mb-1">
              Node ID (Readonly)
            </label>
            <input
              type="text"
              value={targetNode.id}
              disabled
              className="w-full px-3 py-1.5 bg-[#11141d] border border-[#464554] rounded-none text-[#908fa0] text-xs font-mono outline-none cursor-not-allowed"
            />
          </div>

          <div className="mb-3">
            <div className="flex justify-between items-center mb-1">
              <label className="text-xs font-medium text-[#c7c4d7]">
                Node Label / Display Name
              </label>
              <button
                type="button"
                onClick={() => handleOpenParamModal('name')}
                data-testid="param-btn-name"
                className="px-2 py-0.5 rounded-none bg-[#1c1f2a] hover:bg-[#262a35] border border-[#464554] text-[#c0c1ff] hover:text-white text-[11px] font-mono font-medium transition-all inline-flex items-center gap-1 cursor-pointer shadow-sm"
              >
                <span className="material-symbols-outlined text-xs text-[#c0c1ff]">token</span>
                <span>+ Insert Token</span>
              </button>
            </div>
            <input
              type="text"
              ref={(el) => {
                inputRefs.current['name'] = el;
              }}
              value={localDraft.name}
              onChange={(e) => handleNameChange(e.target.value)}
              data-testid="inspector-input-name"
              className={`w-full px-3 py-1.5 bg-[#11141d] border ${
                getFieldError('name', nodeIssues) ? 'border-rose-500' : 'border-[#464554] focus:border-[#c0c1ff]'
              } rounded-none text-[#dfe2f1] text-xs outline-none transition-all placeholder-[#64748b]`}
            />
            {renderFieldError('name')}
          </div>
        </div>

        {/* Schema-Driven Config Form based on Node Type */}
        <div>
          <div className="text-[10px] font-mono font-bold uppercase tracking-wider text-[#908fa0] mb-2">
            Type Configuration ({nodeType})
          </div>

          {/* TRIGGER / EVENT START */}
          {(nodeType === 'EventStart' || nodeType === 'EventStartNode' || nodeType === 'trigger') && (
            <>
              <div className="mb-3">
                <div className="flex justify-between items-center mb-1">
                  <label className="text-xs font-medium text-[#c7c4d7]">
                    Event Name / Type
                  </label>
                  <button
                    type="button"
                    onClick={() => handleOpenParamModal('event_name')}
                    data-testid="param-btn-event_name"
                    className="px-2 py-0.5 rounded-none bg-[#1c1f2a] hover:bg-[#262a35] border border-[#464554] text-[#c0c1ff] hover:text-white text-[11px] font-mono font-medium transition-all inline-flex items-center gap-1 cursor-pointer shadow-sm"
                  >
                    <span className="material-symbols-outlined text-xs text-[#c0c1ff]">token</span>
                    <span>+ Token</span>
                  </button>
                </div>
                <input
                  type="text"
                  ref={(el) => {
                    inputRefs.current['event_name'] = el;
                  }}
                  value={(localDraft.config.event_name as string) || ''}
                  onChange={(e) => handleConfigChange('event_name', e.target.value)}
                  placeholder="e.g. order_completed, cart_abandoned"
                  data-testid="inspector-input-event_name"
                  className={`w-full px-3 py-1.5 bg-[#11141d] border ${
                    getFieldError('event_name', nodeIssues) ? 'border-rose-500' : 'border-[#464554] focus:border-[#c0c1ff]'
                  } rounded-none text-[#dfe2f1] text-xs outline-none transition-all placeholder-[#64748b]`}
                />
                {renderFieldError('event_name')}
              </div>

              <div className="mb-3">
                <div className="flex justify-between items-center mb-1">
                  <label className="text-xs font-medium text-[#c7c4d7]">
                    Event Filter Expression
                  </label>
                  <button
                    type="button"
                    onClick={() => handleOpenParamModal('event_filter')}
                    data-testid="param-btn-event_filter"
                    className="px-2 py-0.5 rounded-none bg-[#1c1f2a] hover:bg-[#262a35] border border-[#464554] text-[#c0c1ff] hover:text-white text-[11px] font-mono font-medium transition-all inline-flex items-center gap-1 cursor-pointer shadow-sm"
                  >
                    <span className="material-symbols-outlined text-xs text-[#c0c1ff]">token</span>
                    <span>+ Token</span>
                  </button>
                </div>
                <textarea
                  ref={(el) => {
                    inputRefs.current['event_filter'] = el;
                  }}
                  rows={3}
                  value={(localDraft.config.event_filter as string) || ''}
                  onChange={(e) => handleConfigChange('event_filter', e.target.value)}
                  placeholder="e.g. event.data.amount > 100"
                  data-testid="inspector-input-event_filter"
                  className={`w-full px-3 py-1.5 bg-[#11141d] border ${
                    getFieldError('event_filter', nodeIssues) ? 'border-rose-500' : 'border-[#464554] focus:border-[#c0c1ff]'
                  } rounded-none text-[#dfe2f1] text-xs font-mono outline-none transition-all placeholder-[#64748b]`}
                />
                {renderFieldError('event_filter')}
              </div>
            </>
          )}

          {/* EMAIL ACTION */}
          {(nodeType === 'Email' || nodeType === 'EmailNode' || (nodeType === 'action' && localDraft.name.toLowerCase().includes('email'))) && (
            <>
              <div className="mb-3">
                <div className="flex justify-between items-center mb-1">
                  <label className="text-xs font-medium text-[#c7c4d7]">
                    Recipient Address
                  </label>
                  <button
                    type="button"
                    onClick={() => handleOpenParamModal('recipient')}
                    data-testid="param-btn-recipient"
                    className="px-2 py-0.5 rounded-none bg-[#1c1f2a] hover:bg-[#262a35] border border-[#464554] text-[#c0c1ff] hover:text-white text-[11px] font-mono font-medium transition-all inline-flex items-center gap-1 cursor-pointer shadow-sm"
                  >
                    <span className="material-symbols-outlined text-xs text-[#c0c1ff]">token</span>
                    <span>+ Token</span>
                  </button>
                </div>
                <input
                  type="text"
                  ref={(el) => {
                    inputRefs.current['recipient'] = el;
                  }}
                  value={(localDraft.config.recipient as string) || ''}
                  onChange={(e) => handleConfigChange('recipient', e.target.value)}
                  placeholder="e.g. {{subject.email}}"
                  data-testid="inspector-input-recipient"
                  className={`w-full px-3 py-1.5 bg-[#11141d] border ${
                    getFieldError('recipient', nodeIssues) ? 'border-rose-500' : 'border-[#464554] focus:border-[#c0c1ff]'
                  } rounded-none text-[#dfe2f1] text-xs outline-none transition-all placeholder-[#64748b]`}
                />
                {renderFieldError('recipient')}
              </div>

              <div className="mb-3">
                <div className="flex justify-between items-center mb-1">
                  <label className="text-xs font-medium text-[#c7c4d7]">
                    Subject Line
                  </label>
                  <button
                    type="button"
                    onClick={() => handleOpenParamModal('subject')}
                    data-testid="param-btn-subject"
                    className="px-2 py-0.5 rounded-none bg-[#1c1f2a] hover:bg-[#262a35] border border-[#464554] text-[#c0c1ff] hover:text-white text-[11px] font-mono font-medium transition-all inline-flex items-center gap-1 cursor-pointer shadow-sm"
                  >
                    <span className="material-symbols-outlined text-xs text-[#c0c1ff]">token</span>
                    <span>+ Token</span>
                  </button>
                </div>
                <input
                  type="text"
                  ref={(el) => {
                    inputRefs.current['subject'] = el;
                  }}
                  value={(localDraft.config.subject as string) || ''}
                  onChange={(e) => handleConfigChange('subject', e.target.value)}
                  placeholder="e.g. Welcome {{subject.first_name}}!"
                  data-testid="inspector-input-subject"
                  className={`w-full px-3 py-1.5 bg-[#11141d] border ${
                    getFieldError('subject', nodeIssues) ? 'border-rose-500' : 'border-[#464554] focus:border-[#c0c1ff]'
                  } rounded-none text-[#dfe2f1] text-xs outline-none transition-all placeholder-[#64748b]`}
                />
                {renderFieldError('subject')}
              </div>

              <div className="mb-3">
                <label className="block text-xs font-medium text-[#c7c4d7] mb-1">
                  Email Template ID
                </label>
                <input
                  type="text"
                  value={(localDraft.config.template_id as string) || ''}
                  onChange={(e) => handleConfigChange('template_id', e.target.value)}
                  placeholder="e.g. tmpl_welcome_v2"
                  data-testid="inspector-input-template_id"
                  className={`w-full px-3 py-1.5 bg-[#11141d] border ${
                    getFieldError('template_id', nodeIssues) ? 'border-rose-500' : 'border-[#464554] focus:border-[#c0c1ff]'
                  } rounded-none text-[#dfe2f1] text-xs outline-none transition-all placeholder-[#64748b]`}
                />
                {renderFieldError('template_id')}
              </div>
            </>
          )}

          {/* CONDITION NODE */}
          {(nodeType === 'Condition' || nodeType === 'ConditionNode' || nodeType === 'condition') && (
            <div className="mb-3">
              <div className="flex justify-between items-center mb-1">
                <label className="text-xs font-medium text-[#c7c4d7]">
                  Condition Expression
                </label>
                <button
                  type="button"
                  onClick={() => handleOpenParamModal('condition_expression')}
                  data-testid="param-btn-condition_expression"
                  className="px-2 py-0.5 rounded-none bg-[#1c1f2a] hover:bg-[#262a35] border border-[#464554] text-[#c0c1ff] hover:text-white text-[11px] font-mono font-medium transition-all inline-flex items-center gap-1 cursor-pointer shadow-sm"
                >
                  <span className="material-symbols-outlined text-xs text-[#c0c1ff]">token</span>
                  <span>+ Token</span>
                </button>
              </div>
              <textarea
                ref={(el) => {
                  inputRefs.current['condition_expression'] = el;
                }}
                rows={4}
                value={(localDraft.config.condition_expression as string) || ''}
                onChange={(e) => handleConfigChange('condition_expression', e.target.value)}
                placeholder="e.g. {{subject.tier}} == 'VIP' && {{event.data.cart_value}} > 100"
                data-testid="inspector-input-condition_expression"
                className={`w-full px-3 py-1.5 bg-[#11141d] border ${
                  getFieldError('condition_expression', nodeIssues) ? 'border-rose-500' : 'border-[#464554] focus:border-[#c0c1ff]'
                } rounded-none text-[#dfe2f1] text-xs font-mono outline-none transition-all placeholder-[#64748b]`}
              />
              {renderFieldError('condition_expression')}
            </div>
          )}

          {/* DELAY NODE */}
          {(nodeType === 'Delay' || nodeType === 'DelayNode') && (
            <>
              <div className="mb-3">
                <label className="block text-xs font-medium text-[#c7c4d7] mb-1">
                  Duration Value
                </label>
                <input
                  type="number"
                  min={1}
                  value={(localDraft.config.duration as number) ?? catalogDefaultDuration ?? ''}
                  data-testid="inspector-input-duration"
                  className={`w-full px-3 py-1.5 bg-[#11141d] border ${
                    getFieldError('duration', nodeIssues) ? 'border-rose-500' : 'border-[#464554] focus:border-[#c0c1ff]'
                  } rounded-none text-[#dfe2f1] text-xs outline-none transition-all placeholder-[#64748b]`}
                />
                {renderFieldError('duration')}
              </div>

              <div className="mb-3">
                <label className="block text-xs font-medium text-[#c7c4d7] mb-1">
                  Time Unit
                </label>
                <select
                  value={(localDraft.config.unit as string) ?? catalogDefaultUnit ?? ''}
                  data-testid="inspector-input-unit"
                  className="w-full px-3 py-1.5 bg-[#11141d] border border-[#464554] focus:border-[#c0c1ff] rounded-none text-[#dfe2f1] text-xs outline-none transition-all"
                >
                  <option value="seconds">Seconds</option>
                  <option value="minutes">Minutes</option>
                  <option value="hours">Hours</option>
                  <option value="days">Days</option>
                </select>
              </div>
            </>
          )}

          {/* WEBHOOK NODE */}
          {(nodeType === 'Webhook' || nodeType === 'WebhookNode') && (
            <>
              <div className="mb-3">
                <div className="flex justify-between items-center mb-1">
                  <label className="text-xs font-medium text-[#c7c4d7]">
                    Webhook URL
                  </label>
                  <button
                    type="button"
                    onClick={() => handleOpenParamModal('url')}
                    data-testid="param-btn-url"
                    className="px-2 py-0.5 rounded-none bg-[#1c1f2a] hover:bg-[#262a35] border border-[#464554] text-[#c0c1ff] hover:text-white text-[11px] font-mono font-medium transition-all inline-flex items-center gap-1 cursor-pointer shadow-sm"
                  >
                    <span className="material-symbols-outlined text-xs text-[#c0c1ff]">token</span>
                    <span>+ Token</span>
                  </button>
                </div>
                <input
                  type="text"
                  ref={(el) => {
                    inputRefs.current['url'] = el;
                  }}
                  value={(localDraft.config.url as string) || ''}
                  onChange={(e) => handleConfigChange('url', e.target.value)}
                  placeholder="https://api.partner.com/notify"
                  data-testid="inspector-input-url"
                  className={`w-full px-3 py-1.5 bg-[#11141d] border ${
                    getFieldError('url', nodeIssues) ? 'border-rose-500' : 'border-[#464554] focus:border-[#c0c1ff]'
                  } rounded-none text-[#dfe2f1] text-xs outline-none transition-all placeholder-[#64748b]`}
                />
                {renderFieldError('url')}
              </div>

              <div className="mb-3">
                <label className="block text-xs font-medium text-[#c7c4d7] mb-1">
                  HTTP Method
                </label>
                <select
                  value={(localDraft.config.method as string) || 'POST'}
                  onChange={(e) => handleConfigChange('method', e.target.value)}
                  data-testid="inspector-input-method"
                  className="w-full px-3 py-1.5 bg-[#11141d] border border-[#464554] focus:border-[#c0c1ff] rounded-none text-[#dfe2f1] text-xs outline-none transition-all"
                >
                  <option value="GET">GET</option>
                  <option value="POST">POST</option>
                  <option value="PUT">PUT</option>
                  <option value="DELETE">DELETE</option>
                </select>
              </div>
            </>
          )}

          {/* EXPERIMENT NODE */}
          {(nodeType === 'Experiment' || nodeType === 'ExperimentNode') && (
            <>
              <div className="mb-3">
                <label className="block text-xs font-medium text-[#c7c4d7] mb-1">
                  Experiment ID
                </label>
                <input
                  type="text"
                  value={(localDraft.config.experiment_id as string) || ''}
                  onChange={(e) => handleConfigChange('experiment_id', e.target.value)}
                  placeholder="exp_discount_test_v1"
                  data-testid="inspector-input-experiment_id"
                  className={`w-full px-3 py-1.5 bg-[#11141d] border ${
                    getFieldError('experiment_id', nodeIssues) ? 'border-rose-500' : 'border-[#464554] focus:border-[#c0c1ff]'
                  } rounded-none text-[#dfe2f1] text-xs outline-none transition-all placeholder-[#64748b]`}
                />
                {renderFieldError('experiment_id')}
              </div>

              {/* Traffic Flow Split Percentage Controls */}
              <div className="mb-4 p-3.5 rounded-none bg-[#1c1f2a] border border-[#464554] space-y-3 font-['Outfit',sans-serif]">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-bold text-[#ddb7ff] uppercase tracking-wider font-mono">
                    Traffic Split Allocation
                  </span>
                  <span className="text-xs font-mono text-[#c0c1ff] font-bold">
                    {((localDraft.config.variant_a_weight as number) ?? 50)}% / {100 - ((localDraft.config.variant_a_weight as number) ?? 50)}%
                  </span>
                </div>

                {/* Preset Quick-Buttons */}
                <div className="flex gap-2">
                  {[50, 70, 80, 90].map((pct) => (
                    <button
                      key={pct}
                      type="button"
                      onClick={() => {
                        handleConfigChange('variant_a_weight', pct);
                        handleConfigChange('variant_b_weight', 100 - pct);
                      }}
                      className={`flex-1 py-1 rounded-none text-[10px] font-mono font-bold border transition-all cursor-pointer ${
                        ((localDraft.config.variant_a_weight as number) ?? 50) === pct
                          ? 'bg-[#ddb7ff]/20 text-[#ddb7ff] border-[#ddb7ff]'
                          : 'bg-[#11141d] text-[#908fa0] border-[#464554] hover:text-white'
                      }`}
                    >
                      {pct}/{100 - pct}
                    </button>
                  ))}
                </div>

                {/* Interactive Split Slider */}
                <div className="space-y-1 pt-1">
                  <div className="flex justify-between text-[10px] font-mono text-[#908fa0]">
                    <span>Variant A ({((localDraft.config.variant_a_weight as number) ?? 50)}%)</span>
                    <span>Variant B ({100 - ((localDraft.config.variant_a_weight as number) ?? 50)}%)</span>
                  </div>
                  <input
                    type="range"
                    min={0}
                    max={100}
                    step={5}
                    value={((localDraft.config.variant_a_weight as number) ?? 50)}
                    onChange={(e) => {
                      const val = parseInt(e.target.value, 10);
                      handleConfigChange('variant_a_weight', val);
                      handleConfigChange('variant_b_weight', 100 - val);
                    }}
                    data-testid="inspector-slider-variant_split"
                    className="w-full h-2 rounded-none appearance-none cursor-pointer bg-slate-800 accent-[#ddb7ff]"
                  />
                </div>
              </div>
            </>
          )}

          {/* Generic fallback / extra description for all other types */}
          <div className="mb-3">
            <label className="block text-xs font-medium text-[#c7c4d7] mb-1">
              Node Notes / Description
            </label>
            <textarea
              rows={2}
              value={localDraft.description}
              onChange={(e) => {
                setLocalDraft((prev) => ({ ...prev, description: e.target.value }));
                setIsDirty(true);
              }}
              placeholder="Internal documentation note..."
              data-testid="inspector-input-description"
              className="w-full px-3 py-1.5 bg-[#11141d] border border-[#464554] focus:border-[#c0c1ff] rounded-none text-[#dfe2f1] text-xs outline-none transition-all placeholder-[#64748b]"
            />
          </div>

          {/* Dry Run Button */}
          <div className="pt-2 space-y-2">
            <button
              type="button"
              onClick={handleDryRun}
              disabled={isSimulating || !currentDraft?.draft_id}
              data-testid="dry-run-node-btn"
              className="w-full py-2.5 bg-[#c0c1ff]/10 text-[#c0c1ff] border border-[#c0c1ff]/20 rounded-none font-bold hover:bg-[#c0c1ff]/20 transition-all flex items-center justify-center gap-2 text-xs cursor-pointer shadow-md disabled:opacity-50"
            >
              <span className="material-symbols-outlined text-base">
                {isSimulating ? 'sync' : 'play_arrow'}
              </span>
              <span>{isSimulating ? 'Simulating...' : 'Dry Run This Node'}</span>
            </button>
            {simulationStatus && (
              <div className="text-[11px] font-mono text-[#c0c1ff] p-2 rounded-none bg-[#11141d] border border-[#464554]">
                {simulationStatus}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Footer Controls: Isolated Draft Buffer Save and Cancel Semantics */}
      <div className="p-4 bg-[#1c1f2a]/90 border-t border-[#464554] flex items-center gap-3 shrink-0">
        <button
          type="button"
          onClick={handleCancel}
          disabled={!isDirty}
          data-testid="inspector-cancel-btn"
          className={`flex-1 py-2.5 rounded-none font-semibold text-xs border transition-all ${
            !isDirty
              ? 'bg-[#171b26] text-[#908fa0] border-[#464554] cursor-not-allowed opacity-50'
              : 'bg-transparent text-[#c7c4d7] border-[#464554] hover:bg-[#262a35] cursor-pointer'
          }`}
        >
          Discard
        </button>

        <button
          type="button"
          onClick={handleSave}
          disabled={!isDirty}
          data-testid="inspector-save-btn"
          className={`flex-1 py-2.5 rounded-none font-bold text-xs shadow-lg transition-all ${
            !isDirty
              ? 'bg-[#c0c1ff]/30 text-[#1000a9]/50 cursor-not-allowed'
              : 'bg-[#c0c1ff] text-[#1000a9] hover:brightness-110 cursor-pointer'
          }`}
        >
          Save Node
        </button>
      </div>

    </aside>
  );
}

export function NodeInspector(props: NodeInspectorProps) {
  return (
    <SafeQueryClientProvider>
      <NodeInspectorInner {...props} />
    </SafeQueryClientProvider>
  );
}
