import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { CloseButton } from './CloseButton';

describe('CloseButton Component', () => {
  it('renders close button with default aria-label and handles click events', () => {
    const handleClick = vi.fn();
    render(<CloseButton onClick={handleClick} testId="test-close-btn" />);

    const button = screen.getByTestId('test-close-btn');
    expect(button).toBeInTheDocument();
    expect(button).toHaveAttribute('aria-label', 'Close');

    fireEvent.click(button);
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('supports custom ariaLabel and size props', () => {
    render(<CloseButton ariaLabel="Dismiss dialog" size="sm" testId="sm-close-btn" />);

    const button = screen.getByTestId('sm-close-btn');
    expect(button).toHaveAttribute('aria-label', 'Dismiss dialog');
    expect(button.className).toContain('w-6 h-6');
  });

  it('applies custom background opacity when bgOpacity prop is supplied', () => {
    render(<CloseButton bgOpacity={60} testId="opacity-close-btn" />);

    const button = screen.getByTestId('opacity-close-btn');
    expect(button.style.backgroundColor).toBe('rgba(28, 31, 42, 0.6)');
  });
});
