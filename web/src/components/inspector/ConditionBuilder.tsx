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
  { value: '==', label: 'Equal to' },
  { value: '!=', label: 'Not equal to' },
  { value: '>', label: 'Greater than' },
  { value: '<', label: 'Less than' },
  { value: '>=', label: 'Greater than or equal to' },
  { value: '<=', label: 'Less than or equal to' },
  { value: 'IN', label: 'In list' },
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
      <div className="flex items-center justify-end pb-2">
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

        </div>
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
          {/* Clause Rows */}
          <div className="space-y-2">
            {clauses.map((clause, idx) => {
              const isStandard = STANDARD_FIELD_TOKENS.some((t) => t.value === clause.field);
              const showCustomInput = customFields[clause.id] || (!isStandard && clause.field !== '');

              return (
                <div
                  key={clause.id}
                  data-testid={`clause-row-${idx}`}
                  className="bg-[#171b26] p-1.5 space-y-1.5 transition-all"
                >
                  {/* Clause Joiner for row > 0 */}
                  {idx > 0 && (
                    <div className="flex items-center py-0.5">
                      <select
                        value={clause.logicalOperator || groupLogic}
                        onChange={(e) => handleClauseLogicChange(idx, e.target.value as 'AND' | 'OR')}
                        data-testid={`clause-logic-${idx}`}
                        aria-label="Clause Operator"
                        style={{ backgroundImage: 'none', appearance: 'none', WebkitAppearance: 'none', MozAppearance: 'none', fontFamily: "'Outfit', -apple-system, BlinkMacSystemFont, sans-serif" }}
                        className="bg-[#11141d] border-0 hover:bg-[#1c2233] text-[#ddb7ff] hover:text-white text-[11px] font-bold font-['Outfit',sans-serif] px-2 py-0.5 rounded-none outline-none cursor-pointer appearance-none transition-colors duration-150"
                      >
                        <option value="AND" style={{ fontFamily: "'Outfit', sans-serif" }} className="bg-[#11141d] text-white font-['Outfit',sans-serif]">AND</option>
                        <option value="OR" style={{ fontFamily: "'Outfit', sans-serif" }} className="bg-[#11141d] text-white font-['Outfit',sans-serif]">OR</option>
                      </select>
                    </div>
                  )}

                  {/* Field, Operator, Value Controls: Borderless seamless row */}
                  <div className="flex items-stretch w-full bg-[#11141d] rounded-none transition-colors duration-150">
                    {/* Field Selection */}
                    <div className="flex-1 min-w-0">
                      {showCustomInput ? (
                        <div className="flex items-stretch h-full">
                          <input
                            type="text"
                            value={clause.field}
                            onChange={(e) => handleFieldChange(idx, e.target.value)}
                            placeholder="e.g. subject.tier"
                            data-testid={`clause-field-input-${idx}`}
                            className="w-full px-2 py-1.5 clause-control border-0 text-xs font-['Outfit',sans-serif] outline-none"
                          />
                          <button
                            type="button"
                            title="Switch to standard options"
                            onClick={() => {
                              setCustomFields((prev) => ({ ...prev, [clause.id]: false }));
                              handleFieldChange(idx, 'subject.tier');
                            }}
                            className="px-1.5 text-[#908fa0] hover:text-white clause-control border-0 shrink-0 flex items-center justify-center cursor-pointer"
                          >
                            <span className="material-symbols-outlined text-xs">list</span>
                          </button>
                        </div>
                      ) : (
                        <select
                          value={clause.field}
                          onChange={(e) => handleFieldChange(idx, e.target.value)}
                          data-testid={`clause-field-${idx}`}
                          style={{ backgroundImage: 'none', appearance: 'none', WebkitAppearance: 'none', MozAppearance: 'none', fontFamily: "'Outfit', -apple-system, BlinkMacSystemFont, sans-serif" }}
                          className="w-full px-2 py-1.5 clause-control border-0 text-xs font-['Outfit',sans-serif] outline-none cursor-pointer appearance-none"
                        >
                          <optgroup label="Standard Tokens" style={{ fontFamily: "'Outfit', sans-serif" }} className="bg-[#11141d] text-[#908fa0] font-['Outfit',sans-serif]">
                            {STANDARD_FIELD_TOKENS.map((token) => (
                              <option key={token.value} value={token.value} style={{ fontFamily: "'Outfit', sans-serif" }} className="bg-[#11141d] text-white font-['Outfit',sans-serif]">
                                {token.value}
                              </option>
                            ))}
                          </optgroup>
                          <optgroup label="Custom" style={{ fontFamily: "'Outfit', sans-serif" }} className="bg-[#11141d] text-[#908fa0] font-['Outfit',sans-serif]">
                            <option value="__CUSTOM__" style={{ fontFamily: "'Outfit', sans-serif" }} className="bg-[#11141d] text-[#c0c1ff] font-['Outfit',sans-serif]">+ Custom Token...</option>
                          </optgroup>
                        </select>
                      )}
                    </div>

                    {/* Operator Selection */}
                    <div className="w-[130px] shrink-0">
                      <select
                        value={clause.operator}
                        onChange={(e) =>
                          handleOperatorChange(idx, e.target.value as ConditionClause['operator'])
                        }
                        data-testid={`clause-operator-${idx}`}
                        style={{ backgroundImage: 'none', appearance: 'none', WebkitAppearance: 'none', MozAppearance: 'none', fontFamily: "'Outfit', -apple-system, BlinkMacSystemFont, sans-serif" }}
                        className="w-full px-2 py-1.5 clause-operator-control border-0 text-xs font-['Outfit',sans-serif] font-medium outline-none cursor-pointer appearance-none"
                      >
                        {OPERATOR_OPTIONS.map((op) => (
                          <option key={op.value} value={op.value} style={{ fontFamily: "'Outfit', sans-serif" }} className="bg-[#11141d] text-white font-['Outfit',sans-serif]">
                            {op.label}
                          </option>
                        ))}
                      </select>
                    </div>

                    {/* Value Input */}
                    <div className="flex-1 min-w-0">
                      <input
                        type="text"
                        value={clause.value}
                        onChange={(e) => handleValueChange(idx, e.target.value)}
                        placeholder={clause.operator === 'IN' ? "['VIP', 'Pro']" : "'VIP'"}
                        data-testid={`clause-value-${idx}`}
                        className="w-full px-2 py-1.5 clause-control border-0 text-xs font-['Outfit',sans-serif] outline-none placeholder-[#64748b]"
                      />
                    </div>

                    {/* Remove Action - Close Icon without white background */}
                    <button
                      type="button"
                      onClick={() => handleRemoveClause(idx)}
                      data-testid={`remove-clause-btn-${idx}`}
                      aria-label="Remove Clause"
                      title="Remove Clause"
                      className="border-0 bg-transparent text-[#908fa0] hover:text-white px-2 cursor-pointer transition-colors shrink-0 flex items-center justify-center"
                    >
                      <span className="material-symbols-outlined text-sm">close</span>
                    </button>
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
              aria-label="Add Clause"
              title="Add Clause"
              className="p-1 bg-transparent border-0 text-[#908fa0] hover:text-white flex items-center justify-center transition-colors cursor-pointer"
            >
              <span className="material-symbols-outlined text-sm">add</span>
            </button>
          </div>
        </div>
      )}

      {mode === 'builder' && (
        <input
          type="hidden"
          ref={inputRef as unknown as React.Ref<HTMLInputElement>}
          value={value}
          data-testid="inspector-input-condition_expression"
        />
      )}
    </div>
  );
}
