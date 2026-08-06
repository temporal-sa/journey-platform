import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import {
  StaticListUploadModal,
  maskRecipient,
  isFormulaInjection,
  isValidRecipient,
} from './StaticListUploadModal';

describe('StaticListUploadModal Helpers', () => {
  it('correctly masks email and phone recipient addresses', () => {
    expect(maskRecipient('user1@example.com')).toBe('u***1@example.com');
    expect(maskRecipient('ab@test.com')).toBe('a***@test.com');
    expect(maskRecipient('+14155552671')).toBe('+1***71');
    expect(maskRecipient('12345')).toBe('1***');
    expect(maskRecipient('MEMBER123')).toBe('M***3');
  });

  it('correctly identifies CSV formula injection patterns', () => {
    expect(isFormulaInjection('=SUM(A1:A10)')).toBe(true);
    expect(isFormulaInjection('@REF')).toBe(true);
    expect(isFormulaInjection('+10+20')).toBe(true);
    expect(isFormulaInjection('-5+5')).toBe(true);
    expect(isFormulaInjection('+14155552671')).toBe(false); // Valid E.164 phone
    expect(isFormulaInjection('user@example.com')).toBe(false);
  });

  it('validates email and phone recipient addresses', () => {
    expect(isValidRecipient('user@example.com')).toBe(true);
    expect(isValidRecipient('+14155552671')).toBe(true);
    expect(isValidRecipient('invalid-email')).toBe(false);
    expect(isValidRecipient('')).toBe(false);
  });
});

describe('StaticListUploadModal Component', () => {
  it('does not render modal content when isOpen is false', () => {
    const { container } = render(<StaticListUploadModal isOpen={false} onClose={() => {}} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders modal with explicit TEST MODE badge and default form elements when isOpen is true', () => {
    render(<StaticListUploadModal isOpen={true} onClose={() => {}} />);

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Static List CSV Upload' })).toBeInTheDocument();
    expect(screen.getByTestId('test-mode-badge')).toHaveTextContent('TEST MODE');
    expect(screen.getByLabelText('List ID (Target Identifier)')).toBeInTheDocument();
    expect(screen.getByLabelText('Immutable List Version')).toBeInTheDocument();
    expect(screen.getByLabelText('Expiry Duration (TTL)')).toBeInTheDocument();
  });

  it('parses CSV input, displays validation summary, formula rejections, and masked member previews', async () => {
    const onUploadSuccess = vi.fn();
    render(
      <StaticListUploadModal
        isOpen={true}
        onClose={() => {}}
        onUploadSuccess={onUploadSuccess}
      />
    );

    const csvContent = `member_id,recipient
MBR-001,john.doe@example.com
MBR-002,+14155552671
MBR-003,=SUM(A1:A10)
`;

    const file = new File([csvContent], 'test_list.csv', { type: 'text/csv' });
    const fileInput = screen.getByLabelText('Choose CSV File') as HTMLInputElement;

    fireEvent.change(fileInput, { target: { files: [file] } });

    await waitFor(() => {
      expect(screen.getByText('Total Rows')).toBeInTheDocument();
    });

    expect(screen.getByText('3')).toBeInTheDocument(); // total rows
    expect(screen.getByText('2')).toBeInTheDocument(); // verified contacts
    expect(screen.getByText('1')).toBeInTheDocument(); // formula rejections

    // Check masked member preview table
    expect(screen.getByText('j***e@example.com')).toBeInTheDocument();
    expect(screen.getByText('+1***71')).toBeInTheDocument();
    expect(screen.getByText('❌ Formula Rejected')).toBeInTheDocument();

    // Click confirm upload button
    const confirmBtn = screen.getByRole('button', { name: 'Confirm Static List Upload' });
    fireEvent.click(confirmBtn);

    await waitFor(
      () => {
        expect(onUploadSuccess).toHaveBeenCalledWith(
          expect.objectContaining({
            list_id: 'list-static-001',
            rowCount: 3,
            verifiedContacts: 2,
            formulaRejections: 1,
          })
        );
      },
      { timeout: 2000 }
    );
  });

  it('invokes onUploadError callback when upload API call fails', async () => {
    const handleClose = vi.fn();
    const handleUploadError = vi.fn();
    const { setSimulatedApiFailureEnabled } = await import('../../api/simulatedFailure');

    setSimulatedApiFailureEnabled(true);

    try {
      render(
        <StaticListUploadModal
          isOpen={true}
          onClose={handleClose}
          onUploadError={handleUploadError}
        />
      );

      const confirmBtn = screen.getByRole('button', { name: 'Confirm Static List Upload' });
      fireEvent.click(confirmBtn);

      await waitFor(() => {
        expect(handleUploadError).toHaveBeenCalledWith(
          expect.stringContaining('[DEV SIMULATION] Service Unavailable')
        );
      });
    } finally {
      setSimulatedApiFailureEnabled(false);
    }
  });
});
