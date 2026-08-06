import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AgyContextModal } from './AgyContextModal';

describe('AgyContextModal Component', () => {
  it('does not render when isOpen is false', () => {
    const { container } = render(<AgyContextModal isOpen={false} onClose={() => {}} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders modal with AGY prompt context and copy button when isOpen is true', () => {
    render(<AgyContextModal isOpen={true} onClose={() => {}} />);
    expect(screen.getByTestId('agy-context-modal')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Export Prompt Context' })).toBeInTheDocument();
    expect(screen.getByTestId('agy-copy-context-btn')).toBeInTheDocument();
  });
});
