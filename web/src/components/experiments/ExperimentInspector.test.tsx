import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ExperimentInspector } from './ExperimentInspector';

describe('ExperimentInspector Component', () => {
  it('renders experiment inspector fields including randomization unit, metrics, attribution window, and variants', () => {
    render(<ExperimentInspector />);

    expect(screen.getByTestId('experiment-inspector')).toBeInTheDocument();
    expect(screen.getByLabelText(/Experiment ID/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Experiment Name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Randomization Unit/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Attribution Window/i)).toBeInTheDocument();

    // Check default variants (Control & Variant A)
    expect(screen.getByDisplayValue('Control (Original)')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Variant A (Treatment)')).toBeInTheDocument();
  });

  it('calculates continuous allocation total and allows saving when allocation equals 10,000 basis points (100%)', () => {
    const handleSave = vi.fn();
    render(<ExperimentInspector onSave={handleSave} />);

    const allocationTotal = screen.getByTestId('allocation-total');
    expect(allocationTotal).toHaveTextContent('100%');
    expect(allocationTotal).toHaveTextContent('10,000 / 10,000 bps');

    const saveBtn = screen.getByTestId('save-experiment-btn');
    expect(saveBtn).not.toBeDisabled();

    fireEvent.click(saveBtn);
    expect(handleSave).toHaveBeenCalledTimes(1);
    expect(handleSave.mock.calls[0][0].variants).toHaveLength(2);
    expect(handleSave.mock.calls[0][0].variants[0].weight_basis_points + handleSave.mock.calls[0][0].variants[1].weight_basis_points).toBe(10000);
  });

  it('blocks saving and shows error message when total allocation is not 10,000 basis points', () => {
    const handleSave = vi.fn();
    render(<ExperimentInspector onSave={handleSave} />);

    // Change first variant allocation from 50% to 60% (now total is 110% = 11,000 bps)
    const firstAllocationInput = screen.getByLabelText(/Variant Allocation Percentage 1/i);
    fireEvent.change(firstAllocationInput, { target: { value: '60' } });

    const allocationTotal = screen.getByTestId('allocation-total');
    expect(allocationTotal).toHaveTextContent('110%');
    expect(allocationTotal).toHaveTextContent('11,000 / 10,000 bps');

    const errorContainer = screen.getByTestId('allocation-error');
    expect(errorContainer).toBeInTheDocument();
    expect(errorContainer).toHaveTextContent('Total allocation weight must equal 100%');

    const saveBtn = screen.getByTestId('save-experiment-btn');
    expect(saveBtn).toBeDisabled();

    fireEvent.click(saveBtn);
    expect(handleSave).not.toHaveBeenCalled();
  });

  it('supports 2-to-5 variant editor and preserves stable keys when display names are edited', () => {
    render(<ExperimentInspector />);

    const addBtn = screen.getByTestId('add-variant-btn');

    // Initially 2 variants
    expect(screen.getByTestId('variant-row-control')).toBeInTheDocument();
    expect(screen.getByTestId('variant-row-variant_a')).toBeInTheDocument();

    // Add 3rd variant (variant_c)
    fireEvent.click(addBtn);
    expect(screen.getByTestId('variant-row-variant_c')).toBeInTheDocument();

    // Add 4th variant (variant_d)
    fireEvent.click(addBtn);
    expect(screen.getByTestId('variant-row-variant_d')).toBeInTheDocument();

    // Add 5th variant (variant_e)
    fireEvent.click(addBtn);
    expect(screen.getByTestId('variant-row-variant_e')).toBeInTheDocument();

    // Add button should now be disabled (max 5 variants reached)
    expect(addBtn).toBeDisabled();

    // Edit display name of variant_c and verify key remain "variant_c"
    const nameInputVariantC = screen.getByLabelText(/Variant Name 3/i);
    fireEvent.change(nameInputVariantC, { target: { value: 'New Headline C' } });

    const keyInputVariantC = screen.getByLabelText(/Variant Key 3/i);
    expect(keyInputVariantC).toHaveValue('variant_c');
    expect(screen.getByTestId('variant-row-variant_c')).toBeInTheDocument();
  });

  it('disables remove variant button when only 2 variants remain', () => {
    render(<ExperimentInspector />);

    const removeBtnControl = screen.getByLabelText(/Remove Variant control/i);
    expect(removeBtnControl).toBeDisabled();
  });

  it('updates control toggle correctly so exactly 1 variant is control', () => {
    render(<ExperimentInspector />);

    // Initially variant 1 (control) is control
    const controlRow1 = screen.getByTestId('variant-row-control');
    const controlRadio1 = controlRow1.querySelector('input[type="radio"]') as HTMLInputElement;
    expect(controlRadio1).toBeChecked();

    // Click control for variant 2 (variant_a)
    const variantRowA = screen.getByTestId('variant-row-variant_a');
    const radioA = variantRowA.querySelector('input[type="radio"]') as HTMLInputElement;
    fireEvent.click(radioA);

    expect(radioA).toBeChecked();
    expect(controlRadio1).not.toBeChecked();
  });
});
