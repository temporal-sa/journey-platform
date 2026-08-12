import React, { useState, useEffect, useCallback } from 'react';
import { Button } from '../common/Button';

export interface ConditionClause {
  id: string;
  field: string;
  operator: '==' | '!=' | '>' | '<' | 'IN' | '>=' | '<=';
  value: string;
  logicalOperator?: 'AND' | 'OR';
}

export interface FieldTokenOption {
  value: string;
  label: string;
  category: string;
  exampleValue: string;
}

export const STANDARD_FIELD_TOKENS: FieldTokenOption[] = [
  { value: 'subject.tier', label: 'subject.tier (Subscription Tier)', category: 'Subject', exampleValue: "'VIP'" },
  { value: 'event.amount', label: 'event.amount (Order Amount)', category: 'Event', exampleValue: '100' },
  { value: 'subject.email', label: 'subject.email (User Email)', category: 'Subject', exampleValue: "'alex@example.com'" },
  { value: 'subject.first_name', label: 'subject.first_name (First Name)', category: 'Subject', exampleValue: "'Alex'" },
  { value: 'subject.lifetime_value', label: 'subject.lifetime_value (Customer LTV)', category: 'Subject', exampleValue: '500' },
  { value: 'event.user_id', label: 'event.user_id (User ID)', category: 'Event', exampleValue: "'usr_8921a'" },
  { value: 'event.email', label: 'event.email (Trigger Email)', category: 'Event', exampleValue: "'alex@example.com'" },
  { value: 'event.timestamp', label: 'event.timestamp (Timestamp)', category: 'Event', exampleValue: "'2026-08-12T00:00:00Z'" },
  { value: 'experiment.variant', label: 'experiment.variant (Variant)', category: 'Experiment', exampleValue: "'variant_b'" },
  { value: 'node.http_status', label: 'node.http_status (HTTP Status)', category: 'Node', exampleValue: '200' },
];

export const OPERATOR_OPTIONS = [
  { value: '==', label: '== (Equals)' },
  { value: '!=', label: '!= (Not Equals)' },
  { value: '>', label: '> (Greater Than)' },
  { value: '<', label: '< (Less Than)' },
  { value: '>=', label: '>= (Greater or Equal)' },
  { value: '<=', label: '<= (Less or Equal)' },
  { value: 'IN', label: 'IN (In List / Array)' },
] as const;

export interface ConditionBuilderProps {
  value: string;
  onChange: (value: string) => void;
  hasError?: boolean;
  inputRef?: (el: HTMLTextAreaElement | HTMLInputElement | null) => void;
  onOpenParamModal?: (fieldKey: string) => void;
}

function generateId(): string {
  return `clause-${Math.random().toString(36).substr(2, 6)}`;
}

export function parseExpression(expr: string): { clauses: ConditionClause[]; groupLogic: 'AND' | 'OR' } {
  if (!expr || !expr.trim()) {
    return {
      clauses: [
        { id: generateId(), field: 'subject.tier', operator: '==', value: "'VIP'", logicalOperator: 'AND' },
      ],
      groupLogic: 'AND',
    };
  }

  const regex = /\s+(AND|OR|&&|\|\|)\s+/gi;
  const parts: string[] = [];
  const joiners: ('AND' | 'OR')[] = [];

  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = regex.exec(expr)) !== null) {
    parts.push(expr.substring(lastIndex, match.index).trim());
    const op = match[1].toUpperCase();
    joiners.push(op === '||' ? 'OR' : op === '&&' ? 'AND' : (op as 'AND' | 'OR'));
    lastIndex = regex.lastIndex;
  }
  parts.push(expr.substring(lastIndex).trim());

  let groupLogic: 'AND' | 'OR' = 'AND';
  if (joiners.length > 0 && joiners.every((j) => j === 'OR')) {
    groupLogic = 'OR';
  }

  const clauses: ConditionClause[] = parts.map((part, index) => {
    const id = generateId();
    const logicalOperator = joiners[index] || (index > 0 ? joiners[index - 1] : groupLogic);

    const opRegex = /\s*(==|!=|>=|<=|>|<|\bIN\b|\bin\b)\s*/i;
    const opMatch = part.match(opRegex);

    if (opMatch && opMatch.index !== undefined) {
      let rawField = part.substring(0, opMatch.index).trim();
      rawField = rawField.replace(/^\{\{\s*/, '').replace(/\s*\}\}$/, '');

      const rawOp = opMatch[1].toUpperCase() as ConditionClause['operator'];

      const rawVal = part.substring(opMatch.index + opMatch[0].length).trim();

      return {
        id,
        field: rawField || 'subject.tier',
        operator: rawOp,
        value: rawVal,
        logicalOperator,
      };
    }

    let cleanField = part.replace(/^\{\{\s*/, '').replace(/\s*\}\}$/, '');
    return {
      id,
      field: cleanField || 'subject.tier',
      operator: '==',
      value: '',
      logicalOperator,
    };
  });

  return {
    clauses: clauses.length > 0 ? clauses : [{ id: generateId(), field: 'subject.tier', operator: '==', value: "'VIP'", logicalOperator: 'AND' }],
    groupLogic,
  };
}

export function serializeClauses(clauses: ConditionClause[], groupLogic: 'AND' | 'OR'): string {
  if (!clauses || clauses.length === 0) return '';

  return clauses
    .map((c, idx) => {
      const fieldStr = c.field.trim() || 'subject.tier';
      const opStr = c.operator || '==';
      const valStr = c.value.trim();
      const exprPart = valStr ? `${fieldStr} ${opStr} ${valStr}` : `${fieldStr} ${opStr}`;

      if (idx === 0) {
        return exprPart;
      }
      const joiner = c.logicalOperator || groupLogic;
      return `${joiner} ${exprPart}`;
    })
    .join(' ');
}

export function ConditionBuilder({
  value,
  onChange,
  hasError,
  inputRef,
  onOpenParamModal,
}: ConditionBuilderProps) {
  const [mode, setMode] = useState<'builder' | 'raw'>('builder');

  const parsed = parseExpression(value);
  const [clauses, setClauses] = useState<ConditionClause[]>(parsed.clauses);
  const [groupLogic, setGroupLogic] = useState<'AND' | 'OR'>(parsed.groupLogic);
  const [customFields, setCustomFields] = useState<Record<string, boolean>>({});

  useEffect(() => {
    const currentSerialized = serializeClauses(clauses, groupLogic);
    if (value !== currentSerialized) {
      const reParsed = parseExpression(value);
      setClauses(reParsed.clauses);
      setGroupLogic(reParsed.groupLogic);
    }
  }, [value]);

  const updateAndNotify = useCallback(
    (newClauses: ConditionClause[], newLogic: 'AND' | 'OR') => {
      setClauses(newClauses);
      setGroupLogic(newLogic);
      const serialized = serializeClauses(newClauses, newLogic);
      onChange(serialized);
    },
    [onChange]
  );

  const handleFieldChange = (index: number, newField: string) => {
    if (newField === '__CUSTOM__') {
      setCustomFields((prev) => ({ ...prev, [clauses[index].id]: true }));
      return;
    }
    setCustomFields((prev) => ({ ...prev, [clauses[index].id]: false }));
    const next = [...clauses];
    next[index] = { ...next[index], field: newField };

    const tokenMatch = STANDARD_FIELD_TOKENS.find((t) => t.value === newField);
    if (tokenMatch && !next[index].value) {
      next[index].value = tokenMatch.exampleValue;
    }

    updateAndNotify(next, groupLogic);
  };

  const handleOperatorChange = (index: number, newOp: ConditionClause['operator']) => {
    const next = [...clauses];
    next[index] = { ...next[index], operator: newOp };

    if (newOp === 'IN' && (!next[index].value || !next[index].value.startsWith('['))) {
      next[index].value = "['VIP', 'Pro']";
    }

    updateAndNotify(next, groupLogic);
  };

  const handleValueChange = (index: number, newVal: string) => {
    const next = [...clauses];
    next[index] = { ...next[index], value: newVal };
    updateAndNotify(next, groupLogic);
  };

  const handleClauseLogicChange = (index: number, newLogic: 'AND' | 'OR') => {
    const next = [...clauses];
    next[index] = { ...next[index], logicalOperator: newLogic };
    updateAndNotify(next, groupLogic);
  };

  const handleGroupLogicChange = (newLogic: 'AND' | 'OR') => {
    const next = clauses.map((c) => ({ ...c, logicalOperator: newLogic }));
    updateAndNotify(next, newLogic);
  };

  const handleAddClause = (fieldValue?: string) => {
    const defaultField = fieldValue || 'event.amount';
    const tokenMatch = STANDARD_FIELD_TOKENS.find((t) => t.value === defaultField);
    const newClause: ConditionClause = {
      id: generateId(),
      field: defaultField,
      operator: '==',
      value: tokenMatch ? tokenMatch.exampleValue : "'VIP'",
      logicalOperator: groupLogic,
    };
    const next = [...clauses, newClause];
    updateAndNotify(next, groupLogic);
  };

  const handleRemoveClause = (index: number) => {
    if (clauses.length <= 1) {
      const resetClause: ConditionClause = {
        id: generateId(),
        field: 'subject.tier',
        operator: '==',
        value: "'VIP'",
        logicalOperator: groupLogic,
      };
      updateAndNotify([resetClause], groupLogic);
      return;
    }
    const next = clauses.filter((_, i) => i !== index);
    updateAndNotify(next, groupLogic);
  };

  const handleRawChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const rawVal = e.target.value;
    onChange(rawVal);
    const reParsed = parseExpression(rawVal);
    setClauses(reParsed.clauses);
    setGroupLogic(reParsed.groupLogic);
  };

  return (
    <div
      data-testid="condition-builder"
      className={`rounded-none border ${
        hasError ? 'border-rose-500' : 'border-[#464554]'
      } bg-[#11141d] p-3 text-xs font-['Outfit',sans-serif] space-y-3`}
    >
      {/* Builder Header Bar */}
      <div className="flex items-center justify-between border-b border-[#2d313f] pb-2">
        <div className="flex items-center gap-2">
          <span className="material-symbols-outlined text-amber-400 text-sm">filter_alt</span>
          <span className="font-bold text-[#dfe2f1] text-xs uppercase tracking-wider font-['Outfit']">
            Visual Condition Builder
          </span>
        </div>

        <div className="flex items-center gap-1.5">
          <button
            type="button"
            data-testid="condition-builder-mode-toggle"
            onClick={() => setMode((m) => (m === 'builder' ? 'raw' : 'builder'))}
            className="px-2 py-0.5 rounded-none bg-[#1c1f2a] border border-[#464554] hover:border-[#c0c1ff] text-[#c0c1ff] hover:text-white text-[11px] flex items-center gap-1 transition-colors cursor-pointer"
          >
            <span className="material-symbols-outlined text-xs">
              {mode === 'builder' ? 'code' : 'tune'}
            </span>
            <span>{mode === 'builder' ? 'Raw Code' : 'Visual Builder'}</span>
          </button>

          {onOpenParamModal && (
            <Button
              type="button"
              variant="secondary-dark"
              size="sm"
              icon="token"
              onClick={() => onOpenParamModal('condition_expression')}
              data-testid="param-btn-condition_expression"
              className="shrink-0 text-[11px] px-2 py-0.5"
            >
              + Token
            </Button>
          )}
        </div>
      </div>

      {/* Field Token Quick Selection Pills */}
      <div className="flex flex-wrap items-center gap-1.5 pt-0.5">
        <span className="text-[10px] text-[#908fa0] font-semibold uppercase tracking-wider">
          Tokens:
        </span>
        {['subject.tier', 'event.amount', 'subject.email', 'experiment.variant'].map((tokenKey) => (
          <button
            key={tokenKey}
            type="button"
            data-testid={`quick-token-${tokenKey}`}
            onClick={() => handleAddClause(tokenKey)}
            className="px-1.5 py-0.5 bg-[#171b26] border border-[#464554] hover:border-[#ddb7ff] text-[#ddb7ff] text-[10px] font-mono rounded-none transition-colors cursor-pointer"
          >
            +{tokenKey}
          </button>
        ))}
      </div>

      {/* Mode View Content */}
      {mode === 'raw' ? (
        <div>
          <label className="block text-[10px] text-[#908fa0] uppercase tracking-wider mb-1 font-semibold">
            Raw Expression String
          </label>
          <textarea
            ref={inputRef as unknown as React.Ref<HTMLTextAreaElement>}
            rows={4}
            value={value}
            onChange={handleRawChange}
            placeholder="e.g. subject.tier == 'VIP' AND event.amount > 100"
            data-testid="inspector-input-condition_expression"
            className="w-full px-3 py-1.5 bg-[#171b26] border border-[#464554] focus:border-[#c0c1ff] rounded-none text-[#dfe2f1] text-xs font-mono outline-none transition-all placeholder-[#64748b]"
          />
        </div>
      ) : (
        <div className="space-y-2.5">
          {/* Match All / Match Any Group Toggle */}
          <div className="flex items-center justify-between bg-[#171b26] px-2.5 py-1.5 border border-[#363945]">
            <span className="text-[11px] text-[#c7c4d7] font-medium">Match logic across clauses:</span>
            <div className="flex gap-1" data-testid="group-logic-toggle">
              <button
                type="button"
                onClick={() => handleGroupLogicChange('AND')}
                className={`px-2 py-0.5 text-[10px] font-bold uppercase font-mono rounded-none border transition-colors cursor-pointer ${
                  groupLogic === 'AND'
                    ? 'bg-[#c0c1ff]/20 text-[#c0c1ff] border-[#c0c1ff]'
                    : 'bg-[#11141d] text-[#908fa0] border-[#464554] hover:text-white'
                }`}
              >
                Match ALL (AND)
              </button>
              <button
                type="button"
                onClick={() => handleGroupLogicChange('OR')}
                className={`px-2 py-0.5 text-[10px] font-bold uppercase font-mono rounded-none border transition-colors cursor-pointer ${
                  groupLogic === 'OR'
                    ? 'bg-[#ddb7ff]/20 text-[#ddb7ff] border-[#ddb7ff]'
                    : 'bg-[#11141d] text-[#908fa0] border-[#464554] hover:text-white'
                }`}
              >
                Match ANY (OR)
              </button>
            </div>
          </div>

          {/* Clause Rows */}
          <div className="space-y-2">
            {clauses.map((clause, idx) => {
              const isStandard = STANDARD_FIELD_TOKENS.some((t) => t.value === clause.field);
              const showCustomInput = customFields[clause.id] || (!isStandard && clause.field !== '');

              return (
                <div
                  key={clause.id}
                  data-testid={`clause-row-${idx}`}
                  className="bg-[#171b26] border border-[#363945] p-2 space-y-2 transition-all"
                >
                  {/* Clause Joiner for row > 0 */}
                  {idx > 0 && (
                    <div className="flex items-center gap-2 pb-1 border-b border-[#272b38]">
                      <select
                        value={clause.logicalOperator || groupLogic}
                        onChange={(e) => handleClauseLogicChange(idx, e.target.value as 'AND' | 'OR')}
                        data-testid={`clause-logic-${idx}`}
                        className="bg-[#11141d] border border-[#464554] text-[#ddb7ff] text-[10px] font-bold font-mono px-1.5 py-0.5 rounded-none outline-none cursor-pointer"
                      >
                        <option value="AND">AND</option>
                        <option value="OR">OR</option>
                      </select>
                      <span className="text-[10px] text-[#908fa0] italic">
                        clause operator
                      </span>
                    </div>
                  )}

                  {/* Field, Operator, Value Controls */}
                  <div className="grid grid-cols-12 gap-1.5 items-center">
                    {/* Field Selection */}
                    <div className="col-span-5">
                      <label className="block text-[9px] text-[#908fa0] uppercase tracking-wider mb-0.5 font-bold">
                        Field Token
                      </label>
                      {showCustomInput ? (
                        <div className="flex items-center gap-1">
                          <input
                            type="text"
                            value={clause.field}
                            onChange={(e) => handleFieldChange(idx, e.target.value)}
                            placeholder="e.g. subject.tier"
                            data-testid={`clause-field-input-${idx}`}
                            className="w-full px-2 py-1 bg-[#11141d] border border-[#464554] focus:border-[#c0c1ff] rounded-none text-[#dfe2f1] text-xs font-mono outline-none"
                          />
                          <button
                            type="button"
                            title="Switch to standard options"
                            onClick={() => {
                              setCustomFields((prev) => ({ ...prev, [clause.id]: false }));
                              handleFieldChange(idx, 'subject.tier');
                            }}
                            className="p-1 text-[#908fa0] hover:text-white bg-[#11141d] border border-[#464554]"
                          >
                            <span className="material-symbols-outlined text-xs">list</span>
                          </button>
                        </div>
                      ) : (
                        <select
                          value={clause.field}
                          onChange={(e) => handleFieldChange(idx, e.target.value)}
                          data-testid={`clause-field-${idx}`}
                          className="w-full px-2 py-1 bg-[#11141d] border border-[#464554] focus:border-[#c0c1ff] rounded-none text-[#dfe2f1] text-xs font-mono outline-none cursor-pointer"
                        >
                          <optgroup label="Standard Tokens">
                            {STANDARD_FIELD_TOKENS.map((token) => (
                              <option key={token.value} value={token.value}>
                                {token.value}
                              </option>
                            ))}
                          </optgroup>
                          <optgroup label="Custom">
                            <option value="__CUSTOM__">+ Custom Token...</option>
                          </optgroup>
                        </select>
                      )}
                    </div>

                    {/* Operator Selection */}
                    <div className="col-span-3">
                      <label className="block text-[9px] text-[#908fa0] uppercase tracking-wider mb-0.5 font-bold">
                        Operator
                      </label>
                      <select
                        value={clause.operator}
                        onChange={(e) =>
                          handleOperatorChange(idx, e.target.value as ConditionClause['operator'])
                        }
                        data-testid={`clause-operator-${idx}`}
                        className="w-full px-1.5 py-1 bg-[#11141d] border border-[#464554] focus:border-[#c0c1ff] rounded-none text-[#dfe2f1] text-xs font-mono font-bold outline-none cursor-pointer"
                      >
                        {OPERATOR_OPTIONS.map((op) => (
                          <option key={op.value} value={op.value}>
                            {op.value}
                          </option>
                        ))}
                      </select>
                    </div>

                    {/* Value Input */}
                    <div className="col-span-3">
                      <label className="block text-[9px] text-[#908fa0] uppercase tracking-wider mb-0.5 font-bold">
                        Value
                      </label>
                      <input
                        type="text"
                        value={clause.value}
                        onChange={(e) => handleValueChange(idx, e.target.value)}
                        placeholder={clause.operator === 'IN' ? "['VIP', 'Pro']" : "'VIP'"}
                        data-testid={`clause-value-${idx}`}
                        className="w-full px-2 py-1 bg-[#11141d] border border-[#464554] focus:border-[#c0c1ff] rounded-none text-[#dfe2f1] text-xs font-mono outline-none placeholder-[#64748b]"
                      />
                    </div>

                    {/* Remove Action */}
                    <div className="col-span-1 flex items-end justify-center pt-3">
                      <button
                        type="button"
                        onClick={() => handleRemoveClause(idx)}
                        data-testid={`remove-clause-btn-${idx}`}
                        title="Remove Clause"
                        className="text-[#908fa0] hover:text-rose-400 p-1 cursor-pointer transition-colors"
                      >
                        <span className="material-symbols-outlined text-sm">delete</span>
                      </button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          {/* Add Clause Button */}
          <div className="flex justify-between items-center pt-1">
            <button
              type="button"
              onClick={() => handleAddClause()}
              data-testid="add-clause-btn"
              className="px-2.5 py-1 bg-[#171b26] border border-[#464554] hover:border-[#c0c1ff] text-[#c0c1ff] hover:text-white text-xs font-medium font-['Outfit'] flex items-center gap-1 transition-colors cursor-pointer"
            >
              <span className="material-symbols-outlined text-xs">add</span>
              <span>+ Add Clause</span>
            </button>
          </div>
        </div>
      )}

      {/* Hidden input to hold string value for ref registration and form tests */}
      <input
        type="hidden"
        ref={inputRef as unknown as React.Ref<HTMLInputElement>}
        value={value}
        data-testid="inspector-input-condition_expression"
      />

      {/* Generated Expression Preview */}
      <div className="pt-2 border-t border-[#2d313f]">
        <div className="text-[10px] text-[#908fa0] font-semibold uppercase tracking-wider mb-1 flex items-center justify-between">
          <span>Expression Output</span>
          <span className="font-mono text-[9px] text-[#c0c1ff] font-normal">Auto-compiled</span>
        </div>
        <div className="px-2.5 py-1.5 bg-[#0c0e14] border border-[#272b38] font-mono text-[11px] text-amber-300 break-all select-all">
          {value || <span className="text-[#64748b] italic">No condition specified</span>}
        </div>
      </div>
    </div>
  );
}
