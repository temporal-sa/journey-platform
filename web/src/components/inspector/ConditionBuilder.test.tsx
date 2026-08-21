import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ConditionBuilder, parseExpression, serializeClauses } from './ConditionBuilder';

describe('ConditionBuilder Component', () => {
  it('renders clause fields side-by-side on one line without field labels', () => {
    const handleChange = vi.fn();
    render(<ConditionBuilder value="subject.tier == 'VIP'" onChange={handleChange} />);

    expect(screen.getByTestId('condition-builder')).toBeInTheDocument();
    expect(screen.queryByText('Visual Condition Builder')).not.toBeInTheDocument();
    expect(screen.queryByText('Expression Output')).not.toBeInTheDocument();
    // Verify individual field labels are NOT present
    expect(screen.queryByText('Field Token')).not.toBeInTheDocument();
    expect(screen.queryByText('Operator')).not.toBeInTheDocument();
    expect(screen.queryByText('Value')).not.toBeInTheDocument();

    // Verify side-by-side inputs exist
    expect(screen.getByTestId('clause-field-0')).toBeInTheDocument();
    expect(screen.getByTestId('clause-operator-0')).toBeInTheDocument();
    expect(screen.getByTestId('clause-value-0')).toBeInTheDocument();
  });
  it('renders plain English operator labels without dropdown arrows', () => {
    const handleChange = vi.fn();
    render(<ConditionBuilder value="subject.tier == 'VIP'" onChange={handleChange} />);

    expect(screen.getByText('Equal to')).toBeInTheDocument();
    expect(screen.getByText('Not equal to')).toBeInTheDocument();
    expect(screen.getByText('Greater than')).toBeInTheDocument();
    expect(screen.getByText('Less than')).toBeInTheDocument();
    expect(screen.getByText('Greater than or equal to')).toBeInTheDocument();
    expect(screen.getByText('Less than or equal to')).toBeInTheDocument();
    expect(screen.getByText('In list')).toBeInTheDocument();

    const operatorSelect = screen.getByTestId('clause-operator-0');
    expect(operatorSelect).toHaveClass('appearance-none');
    expect(operatorSelect).toHaveStyle({ backgroundImage: 'none' });
  });

  it('allows editing field, operator, and value', () => {
    const handleChange = vi.fn();
    render(<ConditionBuilder value="subject.tier == 'VIP'" onChange={handleChange} />);

    const operatorSelect = screen.getByTestId('clause-operator-0');
    fireEvent.change(operatorSelect, { target: { value: '!=' } });

    expect(handleChange).toHaveBeenCalledWith(expect.stringContaining('!='));

    const valueInput = screen.getByTestId('clause-value-0');
    fireEvent.change(valueInput, { target: { value: "'Gold'" } });

    expect(handleChange).toHaveBeenCalledWith(expect.stringContaining("'Gold'"));
  });

  it('adds and removes clauses and switches clause logic between AND and OR', () => {
    const handleChange = vi.fn();
    render(<ConditionBuilder value="subject.tier == 'VIP'" onChange={handleChange} />);

    expect(screen.queryByText('Match logic across clauses:')).not.toBeInTheDocument();
    expect(screen.queryByText('Match ALL (AND)')).not.toBeInTheDocument();
    expect(screen.queryByText('Match ANY (OR)')).not.toBeInTheDocument();

    const addClauseBtn = screen.getByTestId('add-clause-btn');
    fireEvent.click(addClauseBtn);

    expect(handleChange).toHaveBeenCalledWith(expect.stringContaining('AND'));

    const clauseLogic1 = screen.getByTestId('clause-logic-1');
    fireEvent.change(clauseLogic1, { target: { value: 'OR' } });

    expect(handleChange).toHaveBeenCalledWith(expect.stringContaining('OR'));

    const removeBtn = screen.getByTestId('remove-clause-btn-1');
    fireEvent.click(removeBtn);

    expect(screen.queryByTestId('clause-row-1')).not.toBeInTheDocument();
  });

  it('toggles between Visual Builder and Raw Code modes', () => {
    const handleChange = vi.fn();
    render(<ConditionBuilder value="subject.tier == 'VIP'" onChange={handleChange} />);

    const modeToggle = screen.getByTestId('condition-builder-mode-toggle');
    fireEvent.click(modeToggle);

    expect(screen.getByText('Raw Expression String')).toBeInTheDocument();
    expect(screen.getByTestId('inspector-input-condition_expression')).toBeInTheDocument();

    fireEvent.click(modeToggle);
    expect(screen.getByTestId('clause-row-0')).toBeInTheDocument();
  });
});

describe('ConditionBuilder parse & serialize helpers', () => {
  it('parses simple and compound expressions', () => {
    const parsedSimple = parseExpression("subject.tier == 'VIP'");
    expect(parsedSimple.clauses.length).toBe(1);
    expect(parsedSimple.clauses[0].field).toBe('subject.tier');
    expect(parsedSimple.clauses[0].operator).toBe('==');
    expect(parsedSimple.clauses[0].value).toBe("'VIP'");

    const parsedCompound = parseExpression("subject.tier == 'VIP' AND event.amount > 100");
    expect(parsedCompound.clauses.length).toBe(2);
    expect(parsedCompound.groupLogic).toBe('AND');
  });

  it('serializes clauses to valid expressions', () => {
    const expr = serializeClauses(
      [
        { id: '1', field: 'subject.tier', operator: '==', value: "'VIP'" },
        { id: '2', field: 'event.amount', operator: '>', value: '100' },
      ],
      'AND'
    );
    expect(expr).toBe("subject.tier == 'VIP' AND event.amount > 100");
  });
});
