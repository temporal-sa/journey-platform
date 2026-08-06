import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { Badge } from './Badge';

describe('Badge Component', () => {
  it('renders badge text and default neutral styles', () => {
    render(<Badge testId="test-badge">PROD</Badge>);

    const badge = screen.getByTestId('test-badge');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveTextContent('PROD');
    expect(badge).toHaveClass('bg-[#1c1f2a]');
  });

  it('renders cyan variant with icon when specified', () => {
    render(
      <Badge variant="cyan" icon="bolt" testId="cyan-badge">
        ACTIVE
      </Badge>
    );

    const badge = screen.getByTestId('cyan-badge');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveTextContent('ACTIVE');
    expect(badge).toHaveClass('text-[#4cd7f6]');
    expect(screen.getByText('bolt')).toBeInTheDocument();
  });

  it('applies custom size and additional className props', () => {
    render(
      <Badge variant="emerald" size="lg" className="shadow-md" testId="lg-badge">
        VERIFIED
      </Badge>
    );

    const badge = screen.getByTestId('lg-badge');
    expect(badge).toHaveClass('px-2.5');
    expect(badge).toHaveClass('shadow-md');
  });
});
