import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Button } from './Button';

describe('Button Component', () => {
  it('renders button with children and handles click events', () => {
    const handleClick = vi.fn();
    render(<Button onClick={handleClick}>Click Me</Button>);

    const btn = screen.getByRole('button', { name: /Click Me/i });
    expect(btn).toBeInTheDocument();

    fireEvent.click(btn);
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('renders icon when icon prop is passed', () => {
    render(<Button icon="add">New Item</Button>);
    expect(screen.getByRole('button')).toHaveTextContent('add');
    expect(screen.getByRole('button')).toHaveTextContent('New Item');
  });

  it('supports variants: primary-purple, primary-cyan, primary-teal, primary-lavender', () => {
    const { rerender } = render(<Button variant="primary-purple">Purple</Button>);
    expect(screen.getByRole('button')).toHaveClass('bg-[#b76dff]');

    rerender(<Button variant="primary-cyan">Cyan</Button>);
    expect(screen.getByRole('button')).toHaveClass('bg-[#4cd7f6]');
  });

  it('disables button and shows spinner when isLoading is true', () => {
    render(<Button isLoading>Submitting</Button>);
    const btn = screen.getByRole('button');
    expect(btn).toBeDisabled();
    expect(btn).toHaveTextContent('sync');
  });

  it('applies custom hoverColor when hovered', () => {
    render(<Button hoverColor="#38c2e0">Custom Hover</Button>);
    const btn = screen.getByRole('button', { name: /Custom Hover/i });

    fireEvent.mouseEnter(btn);
    expect(btn).toHaveStyle({ backgroundColor: '#38c2e0' });

    fireEvent.mouseLeave(btn);
    expect(btn).not.toHaveStyle({ backgroundColor: '#38c2e0' });
  });
});
