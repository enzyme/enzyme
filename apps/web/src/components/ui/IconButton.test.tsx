import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '../../test-utils';
import { IconButton } from './IconButton';

describe('IconButton', () => {
  it('renders with aria-label', () => {
    render(
      <IconButton aria-label="Close">
        <svg data-testid="icon" />
      </IconButton>,
    );

    expect(screen.getByRole('button', { name: 'Close' })).toBeInTheDocument();
    expect(screen.getByTestId('icon')).toBeInTheDocument();
  });

  it('calls onPress when clicked', async () => {
    const handlePress = vi.fn();
    const { userEvent } = await import('../../test-utils');
    const user = userEvent.setup();

    render(
      <IconButton aria-label="Close" onPress={handlePress}>
        <svg />
      </IconButton>,
    );

    await user.click(screen.getByRole('button'));

    expect(handlePress).toHaveBeenCalledTimes(1);
  });

  it('is disabled when isDisabled is true', () => {
    render(
      <IconButton aria-label="Close" isDisabled>
        <svg />
      </IconButton>,
    );

    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('applies ghost variant classes by default', () => {
    render(
      <IconButton aria-label="Close">
        <svg />
      </IconButton>,
    );

    const button = screen.getByRole('button');
    expect(button).toHaveClass('text-gray-500');
    expect(button).toHaveClass('hover:bg-gray-100');
  });

  it('applies danger variant classes', () => {
    render(
      <IconButton aria-label="Delete" variant="danger">
        <svg />
      </IconButton>,
    );

    const button = screen.getByRole('button');
    expect(button).toHaveClass('hover:text-red-600');
  });

  it('applies size classes', () => {
    const { rerender } = render(
      <IconButton aria-label="Close" size="xs">
        <svg />
      </IconButton>,
    );
    expect(screen.getByRole('button')).toHaveClass('p-0.5');

    rerender(
      <IconButton aria-label="Close" size="sm">
        <svg />
      </IconButton>,
    );
    expect(screen.getByRole('button')).toHaveClass('p-1');

    rerender(
      <IconButton aria-label="Close" size="md">
        <svg />
      </IconButton>,
    );
    expect(screen.getByRole('button')).toHaveClass('p-1.5');
  });

  it('merges custom className', () => {
    render(
      <IconButton aria-label="Close" className="my-custom-class">
        <svg />
      </IconButton>,
    );

    expect(screen.getByRole('button')).toHaveClass('my-custom-class');
  });
});
